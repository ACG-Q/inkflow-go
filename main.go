package main

import (
	"embed"
	"fmt"
	"log/slog"
	"os"

	"inkflow-go/internal/config"
	"inkflow-go/internal/server"
)

//go:embed frontend/dist
var embeddedFS embed.FS

func main() {
	cfg := config.Load()

	dirs := []string{cfg.DataDir, cfg.BgImageDir(), cfg.OutputDir(), cfg.FontsDir()}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			slog.Error("failed to create directory", "path", d, "error", err)
			os.Exit(1)
		}
	}

	if len(os.Args) > 1 && os.Args[1] == "init" {
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: inkflow init <password>")
			os.Exit(1)
		}
		password := os.Args[2]
		if len(password) < 6 {
			fmt.Fprintln(os.Stderr, "password must be at least 6 characters")
			os.Exit(1)
		}
		if err := server.InitAdmin(cfg.DBPath(), password); err != nil {
			slog.Error("init admin failed", "error", err)
			os.Exit(1)
		}
		fmt.Println("admin initialized successfully")
		return
	}

	if err := server.Run(cfg, embeddedFS); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
