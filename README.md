<p align="center">
  <img src="docs/logo.svg" alt="Inkflow Logo" width="128" />
</p>

<h1 align="center">Inkflow (墨流)</h1>

<p align="center">轻量级协议签署系统。单二进制部署，内嵌前端，开箱即用。</p>

## 功能

- **模板管理** — 创建/编辑/删除协议模板，上传 PDF 或图片作为背景
- **手写签名** — 支持上传自定义手写字体，签署效果自然逼真
- **规则引擎** — 验证、默认值、互斥、触发器、值同步、自动填充
- **签署记录** — 一键签署，自动生成签署图片并归档
- **导入导出** — 模板打包为 `.zip`，跨实例迁移
- **JWT 认证** — 管理端登录鉴权
- **CLI 支持** — 命令行参数 + 子命令，适合脚本化部署
- **TLS** — 可选 HTTPS 支持
- **多架构** — 预编译 Linux (amd64/arm64/armv7/armv6/386)、Windows

## 快速开始

### 下载

从 [GitHub Releases](https://github.com/ACG-Q/inkflow-go/releases) 下载对应平台的二进制文件。

### 启动

```bash
# 初始化管理员
./inkflow init <password>

# 启动服务（默认 :8080）
./inkflow
```

浏览器访问 `http://localhost:8080`。

## 从源码构建

需要 Go 1.25+ 和 Node.js 20+。

```bash
# Linux / macOS
./build.sh

# Windows
build.bat
```

产物位于 `dist/` 目录。

## CLI 用法

```
inkflow [flags] [command]

命令:
  无参数              启动服务器
  init <password>     初始化管理员账号
  help                显示帮助信息
  version             显示版本信息

标志:
  -p <port>           服务端口 (默认: 8080)
  -d <data_dir>       数据目录 (默认: ./data)
  -s <jwt_secret>     JWT 密钥
  -l <log_level>      日志级别: debug/info/warn/error (默认: info)
  -r <rate_limit>     速率限制 (默认: 60 req/min)
  --cors <origins>    CORS 允许来源
  --tls-cert <path>   TLS 证书路径
  --tls-key <path>    TLS 私钥路径
```

### 示例

```bash
inkflow                            # 默认启动
inkflow -p 9090                    # 指定端口
inkflow -p 9090 -d /data           # 指定端口和数据目录
inkflow -s mysecret -l debug       # 自定义 JWT 密钥和日志级别
inkflow --tls-cert cert.pem --tls-key key.pem  # 启用 HTTPS
inkflow init mypassword            # 初始化管理员
```

## 环境变量

所有命令行参数均可通过环境变量设置，命令行优先级更高：

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `PORT` | 服务端口 | `8080` |
| `DATA_DIR` | 数据目录 | `./data` |
| `JWT_SECRET` | JWT 密钥 | 空 |
| `RATE_LIMIT` | 速率限制 | `60` |
| `LOG_LEVEL` | 日志级别 | `info` |
| `LOG_FORMAT` | 日志格式 (text/json) | `text` |
| `CORS_ORIGINS` | CORS 来源 | `*` |
| `TLS_CERT` | TLS 证书路径 | 空 |
| `TLS_KEY` | TLS 私钥路径 | 空 |

## 项目结构

```
inkflow-go/
├── main.go                  # 入口，CLI 解析
├── internal/
│   ├── config/              # 配置加载
│   ├── server/              # HTTP 服务器 + 路由
│   ├── handlers/            # API 处理器
│   ├── models/              # 数据模型
│   ├── database/            # SQLite 初始化
│   └── core/                # PDFium 等核心能力
├── frontend/                # Vue 3 前端 (git submodule)
│   └── dist/                # 构建产物，嵌入二进制
├── data/                    # 运行时数据 (gitignored)
├── build.sh / build.bat     # 构建脚本
└── .github/workflows/       # CI/CD
```

## 技术栈

- **后端**: Go 1.25 + Gin
- **数据库**: SQLite (modernc.org/sqlite, 纯 Go 无 CGO)
- **PDF**: PDFium (go-pdfium)
- **认证**: JWT (golang-jwt)
- **前端**: Vue 3 + Vite (内嵌于二进制)

## 部署方式

### 直接运行

```bash
./inkflow init admin123
./inkflow -p 8080 -d /var/lib/inkflow
```

### Systemd 服务

```ini
[Unit]
Description=Inkflow
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/inkflow -d /var/lib/inkflow -s your-jwt-secret
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### Docker（手动）

```dockerfile
FROM alpine:3.20
COPY inkflow-linux-amd64 /usr/local/bin/inkflow
RUN chmod +x /usr/local/bin/inkflow
EXPOSE 8080
VOLUME /data
CMD ["inkflow", "-d", "/data"]
```

## License

MIT
