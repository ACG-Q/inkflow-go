package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"inkflow-go/internal/config"
	"inkflow-go/internal/core"
	"inkflow-go/internal/database"
	"inkflow-go/internal/handlers"
)

func Run(cfg *config.Config, frontendFS embed.FS) error {
	setupLogging(cfg)

	if err := database.Init(cfg.DBPath()); err != nil {
		return fmt.Errorf("database init: %w", err)
	}
	slog.Info("database initialized")

	if err := core.InitPDFium(); err != nil {
		slog.Warn("pdfium init failed, PDF upload will be unavailable", "error", err)
	} else {
		slog.Info("pdfium initialized")
		defer core.ClosePDFium()
	}

	handlers.InitJWT(cfg.JWTSecret)

	initAdminAuto(cfg.DBPath())

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	if cfg.CorsOrigins != "" {
		r.Use(corsMiddleware(cfg.CorsOrigins))
	}

	r.Static("/static/bg_images", cfg.BgImageDir())
	r.Static("/static/output", cfg.OutputDir())
	r.Static("/static/fonts", cfg.FontsDir())

	frontendSub, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		return fmt.Errorf("frontend dist not found: %w", err)
	}
	assetsSub, err := fs.Sub(frontendSub, "assets")
	if err != nil {
		return fmt.Errorf("frontend assets not found: %w", err)
	}
	r.StaticFS("/assets", http.FS(assetsSub))
	r.GET("/", func(c *gin.Context) {
		data, err := fs.ReadFile(frontendSub, "index.html")
		if err != nil {
			c.JSON(500, gin.H{"code": 500, "message": "frontend not built", "data": nil})
			return
		}
		c.Data(200, "text/html; charset=utf-8", data)
	})

	api := r.Group("/api")
	api.Use(handlers.RateLimit(cfg.RateLimit))
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
		})
		api.POST("/auth/login", handlers.LoginHandler(database.DB))

		api.GET("/templates", handlers.ListTemplatesHandler(database.DB))
		api.GET("/templates/:id", handlers.GetTemplateHandler(database.DB))
		api.GET("/records/:id", handlers.GetRecordHandler(database.DB))
		api.GET("/fonts", handlers.ListFontsHandler(database.DB))

		auth := api.Group("")
		auth.Use(handlers.AuthRequired())
		{
			auth.GET("/auth/me", handlers.MeHandler())
			auth.POST("/templates", handlers.CreateTemplateHandler(database.DB))
			auth.PUT("/templates/:id", handlers.UpdateTemplateHandler(database.DB))
			auth.DELETE("/templates/:id", handlers.DeleteTemplateHandler(database.DB))
			auth.POST("/templates/:id/upload", handlers.UploadTemplateFileHandler(database.DB, cfg.BgImageDir()))
			auth.GET("/templates/:id/export", handlers.ExportTemplateHandler(database.DB, cfg.BgImageDir(), cfg.FontsDir()))
			auth.POST("/templates/import", handlers.ImportTemplateHandler(database.DB, cfg.BgImageDir()))
			auth.GET("/records", handlers.ListRecordsHandler(database.DB))
			auth.DELETE("/records/:id", handlers.DeleteRecordHandler(database.DB))
			auth.POST("/fonts", handlers.CreateFontHandler(database.DB, cfg.FontsDir()))
			auth.POST("/fonts/batch", handlers.CreateFontsBatchHandler(database.DB, cfg.FontsDir()))
			auth.PUT("/fonts/:id", handlers.UpdateFontHandler(database.DB))
			auth.DELETE("/fonts/:id", handlers.DeleteFontHandler(database.DB))
			auth.GET("/admin/stats", handlers.StatsHandler(database.DB))
			auth.GET("/settings/handwriting", handlers.GetHandwritingSettings(database.DB))
			auth.PUT("/settings/handwriting", handlers.UpdateHandwritingSettings(database.DB))
			auth.GET("/settings/font_batch_limit", handlers.GetFontBatchLimit(database.DB))
			auth.PUT("/settings/font_batch_limit", handlers.UpdateFontBatchLimit(database.DB))
		}
	}

	r.POST("/api/templates/:id/sign", handlers.SignHandler(database.DB, cfg.BgImageDir(), cfg.OutputDir()))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(404, gin.H{"code": 404, "message": "not found", "data": nil})
			return
		}
		if len(path) >= 7 && path[:7] == "/static" {
			c.JSON(404, gin.H{"code": 404, "message": "not found", "data": nil})
			return
		}
		data, err := fs.ReadFile(frontendSub, "index.html")
		if err != nil {
			c.JSON(404, gin.H{"code": 404, "message": "not found", "data": nil})
			return
		}
		c.Data(200, "text/html; charset=utf-8", data)
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	slog.Info("server starting", "addr", addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if cfg.TLSCert != "" && cfg.TLSKey != "" {
			if err := srv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey); err != nil && err != http.ErrServerClosed {
				slog.Error("server listen error", "error", err)
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("server listen error", "error", err)
			}
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	database.DB.Close()
	slog.Info("server stopped")
	return nil
}

func setupLogging(cfg *config.Config) {
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: logLevel}
	var handler slog.Handler
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}

func corsMiddleware(allowedOrigins string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowed := false
		if allowedOrigins == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
			allowed = true
		} else {
			for _, a := range strings.Split(allowedOrigins, ",") {
				if strings.TrimSpace(a) == origin {
					c.Header("Access-Control-Allow-Origin", origin)
					allowed = true
					break
				}
			}
		}

		if !allowed {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if c.Request.Method == "OPTIONS" {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Max-Age", "86400")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
