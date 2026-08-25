package main

import (
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"inkflow-go/internal/config"
	"inkflow-go/internal/server"
)

//go:embed frontend/dist
var embeddedFS embed.FS

var version = "dev"

const usage = `inkflow - 协议签署系统

用法:
  inkflow [flags] [command]

命令:
  无参数              启动服务器
  init <password>     初始化管理员账号
  help                显示帮助信息
  version             显示版本信息

全局标志:
  -p <port>           服务端口 (默认: 8080)
  -d <data_dir>       数据目录 (默认: ./data)
  -s <jwt_secret>     JWT 密钥
  -l <log_level>      日志级别: debug/info/warn/error (默认: info)
  -r <rate_limit>     速率限制 (默认: 60)
  -cors <origins>     CORS 允许来源 (默认: 同源)
  -tls-cert <path>    TLS 证书路径
  -tls-key <path>     TLS 私钥路径

示例:
  inkflow                          # 默认启动
  inkflow -p 9090                  # 指定端口启动
  inkflow -p 9090 -d /data         # 指定端口和数据目录
  inkflow -s mysecret -l debug     # 自定义 JWT 密钥和日志级别
  inkflow -tls-cert cert.pem -tls-key key.pem  # 启用 HTTPS
  inkflow init mypassword          # 初始化管理员
  inkflow version                  # 查看版本`

func main() {
	fs := flag.NewFlagSet("inkflow", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	port := fs.Int("p", 0, "服务端口")
	dataDir := fs.String("d", "", "数据目录")
	jwtSecret := fs.String("s", "", "JWT 密钥")
	logLevel := fs.String("l", "", "日志级别")
	rateLimit := fs.Int("r", 0, "速率限制")
	corsOrigins := fs.String("cors", "", "CORS 允许来源")
	tlsCert := fs.String("tls-cert", "", "TLS 证书路径")
	tlsKey := fs.String("tls-key", "", "TLS 私钥路径")
	showVersion := fs.Bool("v", false, "显示版本")
	showHelp := fs.Bool("h", false, "显示帮助")

	args := os.Args[1:]
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	remaining := fs.Args()

	if *showHelp {
		fmt.Println(usage)
		return
	}

	if *showVersion {
		fmt.Printf("inkflow %s\n", version)
		return
	}

	if len(remaining) > 0 {
		cmd := remaining[0]
		switch cmd {
		case "help", "--help", "-h":
			fmt.Println(usage)
			return
		case "version", "-v", "--version":
			fmt.Printf("inkflow %s\n", version)
			return
		case "init":
			if len(remaining) < 2 {
				fmt.Fprintln(os.Stderr, "用法: inkflow init <password>")
				os.Exit(1)
			}
			password := remaining[1]
			if len(password) < 6 {
				fmt.Fprintln(os.Stderr, "密码长度不能少于 6 个字符")
				os.Exit(1)
			}
			cfg := buildConfig(port, dataDir, jwtSecret, logLevel, rateLimit, corsOrigins, tlsCert, tlsKey)
			if err := server.InitAdmin(cfg.DBPath(), password); err != nil {
				slog.Error("初始化管理员失败", "error", err)
				os.Exit(1)
			}
			fmt.Println("管理员初始化成功")
			return
		default:
			fmt.Fprintf(os.Stderr, "未知命令: %s\n运行 'inkflow help' 查看帮助\n", cmd)
			os.Exit(1)
		}
	}

	cfg := buildConfig(port, dataDir, jwtSecret, logLevel, rateLimit, corsOrigins, tlsCert, tlsKey)

	dirs := []string{cfg.DataDir, cfg.BgImageDir(), cfg.OutputDir(), cfg.FontsDir()}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			slog.Error("failed to create directory", "path", d, "error", err)
			os.Exit(1)
		}
	}

	if err := server.Run(cfg, embeddedFS); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func buildConfig(port *int, dataDir, jwtSecret, logLevel *string, rateLimit *int, corsOrigins, tlsCert, tlsKey *string) *config.Config {
	cfg := config.Load()

	if *port != 0 {
		cfg.Port = *port
	}
	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}
	if *jwtSecret != "" {
		cfg.JWTSecret = *jwtSecret
	}
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}
	if *rateLimit != 0 {
		cfg.RateLimit = *rateLimit
	}
	if *corsOrigins != "" {
		cfg.CorsOrigins = *corsOrigins
	}
	if *tlsCert != "" {
		cfg.TLSCert = *tlsCert
	}
	if *tlsKey != "" {
		cfg.TLSKey = *tlsKey
	}

	return cfg
}
