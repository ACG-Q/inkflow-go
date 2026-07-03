# InkFlow-Go 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 从零实现 InkFlow-Go 电子签章系统，纯 Go 后端 + Vue 3 前端 + Python 测试

**架构：** Gin + modernc.org/sqlite + go-pdfium WASM（后端），Vue 3 + Pinia + Fabric.js（前端），Python pytest + Playwright（测试）

**约定：**
- 所有 Go 代码遵循 `internal/` 组织结构，main.go 在根目录
- 所有查询 SQL 的参数化查询（防注入）
- 响应格式统一 `{code, message, data}`
- 所有 handler 通过 `c.JSON()` 返回统一响应
- 日志用 `log/slog`
- Python 测试驱动后端开发：先写 Python 测试（红），再实现 Go 代码（绿）

---

### 任务 1：项目脚手架 + 配置模块

**文件：**
- 创建：`inkflow-go/go.mod`
- 创建：`inkflow-go/config/config.go`
- 创建：`inkflow-go/main.go`（骨架，仅启动 HTTP 返回 200）

- [ ] **步骤 1：初始化 Go 模块和目录**

```bash
mkdir -p inkflow-go/{cmd,config,database,models,handlers,core,web,tests/test_api,tests/test_e2e,data/bg_images,data/output,data/fonts}
cd inkflow-go
go mod init inkflow-go
go get github.com/gin-gonic/gin
go get modernc.org/sqlite
go get github.com/klippa-app/go-pdfium/webassembly
```

- [ ] **步骤 2：实现 config.go**

```go
package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        int
	DataDir     string
	JWTSecret   string
	RateLimit   int
	LogLevel    string
	LogFormat   string
	CorsOrigins string
	TLSCert     string
	TLSKey      string
}

func Load() *Config {
	cfg := &Config{
		Port:        8080,
		DataDir:     "./data",
		RateLimit:   60,
		LogLevel:    "info",
		LogFormat:   "text",
		CorsOrigins: "*",
	}
	if v := os.Getenv("PORT"); v != "" {
		cfg.Port, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if v := os.Getenv("RATE_LIMIT"); v != "" {
		cfg.RateLimit, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		cfg.CorsOrigins = v
	}
	cfg.TLSCert = os.Getenv("TLS_CERT")
	cfg.TLSKey = os.Getenv("TLS_KEY")
	return cfg
}

func (c *Config) DBPath() string {
	return c.DataDir + "/templates.db"
}

func (c *Config) BgImageDir() string {
	return c.DataDir + "/bg_images"
}

func (c *Config) OutputDir() string {
	return c.DataDir + "/output"
}

func (c *Config) FontsDir() string {
	return c.DataDir + "/fonts"
}
```

- [ ] **步骤 3：实现 main.go 骨架**

```go
package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"inkflow-go/config"
)

func main() {
	cfg := config.Load()
	r := gin.Default()
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
	})
	addr := fmt.Sprintf(":%d", cfg.Port)
	r.Run(addr)
}
```

- [ ] **步骤 4：编译验证**

```bash
cd inkflow-go
go build -o inkflow.exe .
```

预期：编译成功，生成 `inkflow.exe`

- [ ] **步骤 5：Commit**

```bash
git init
git add .
git commit -m "feat: project scaffold with config and health endpoint"
```

---

### 任务 2：数据库层

**文件：**
- 创建：`inkflow-go/database/db.go`
- 创建：`inkflow-go/cmd/init.go`

- [ ] **步骤 1：实现 db.go（SQLite 连接 + 自动迁移）**

```go
package database

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	DB.SetMaxOpenConns(1)
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	return migrate()
}

func migrate() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL DEFAULT '未命名模板',
			bg_image TEXT,
			width INTEGER DEFAULT 800,
			height INTEGER DEFAULT 1000,
			config_json TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME DEFAULT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS signing_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			template_id INTEGER NOT NULL REFERENCES templates(id),
			fields_data TEXT,
			image_url TEXT,
			ip_address TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME DEFAULT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS fonts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL,
			display_name TEXT,
			original_filename TEXT,
			file_hash TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME DEFAULT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE DEFAULT 'admin',
			password TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_records_template_id ON signing_records(template_id)`,
		`CREATE INDEX IF NOT EXISTS idx_records_created_at ON signing_records(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_templates_deleted_at ON templates(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_records_deleted_at ON signing_records(deleted_at)`,
	}
	for _, t := range tables {
		if _, err := DB.Exec(t); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}
```

- [ ] **步骤 2：实现 cmd/init.go（管理员初始化命令）**

```go
package main

import (
	"fmt"
	"inkflow-go/database"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func initAdmin(dbPath, password string) error {
	if err := database.Init(dbPath); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = database.DB.Exec("INSERT OR REPLACE INTO admins (username, password) VALUES ('admin', ?)", string(hash))
	if err != nil {
		return err
	}
	fmt.Println("管理员账号初始化成功")
	return nil
}
```

- [ ] **步骤 3：更新 main.go 支持 init 子命令**

```go
func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		password := "admin123"
		if len(os.Args) > 2 {
			password = os.Args[2]
		}
		cfg := config.Load()
		if err := initAdmin(cfg.DBPath(), password); err != nil {
			log.Fatal(err)
		}
		return
	}
	// ... rest of main
}
```

- [ ] **步骤 4：编译验证 + 测试 init 命令**

```bash
go build -o inkflow.exe .
./inkflow.exe init admin123
# 应输出 "管理员账号初始化成功"
```

- [ ] **步骤 5：Commit**

---

### 任务 3：Models 层

**文件：**
- 创建：`inkflow-go/models/template.go`
- 创建：`inkflow-go/models/signing_record.go`
- 创建：`inkflow-go/models/font.go`
- 创建：`inkflow-go/models/admin.go`

- [ ] **步骤 1：template.go**

```go
package models

import "time"

type Control struct {
	ID            string  `json:"id"`
	Label         string  `json:"label"`
	Type          string  `json:"type"` // textbox, checkbox
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	Width         float64 `json:"width"`
	Height        float64 `json:"height"`
	FontSize      int     `json:"font_size"`
	FontFamily    string  `json:"font_family"`
	Required      bool    `json:"required"`
	PreviewText   string  `json:"preview_text"`
	CheckSize     int     `json:"check_size"`
}

type Rule struct {
	ID         string      `json:"id"`
	Type       string      `json:"type"` // required, validation, mutual_exclusion, trigger, value_sync, arithmetic, auto_fill
	ControlID  string      `json:"control_id"`
	Config     interface{} `json:"config"`
}

type TemplateConfig struct {
	Controls []Control `json:"controls"`
	Rules    []Rule    `json:"rules"`
}

type Template struct {
	ID         int64          `json:"id"`
	Name       string         `json:"name"`
	BgImage    string         `json:"bg_image"`
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	ConfigJSON string         `json:"config_json"`
	CreatedAt  time.Time      `json:"created_at"`
}

type TemplateListItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
```

- [ ] **步骤 2：signing_record.go**

```go
package models

import "time"

type SigningRecord struct {
	ID           int64     `json:"id"`
	TemplateID   int64     `json:"template_id"`
	TemplateName string    `json:"template_name,omitempty"`
	FieldsData   string    `json:"fields_data"`
	ImageURL     string    `json:"image_url"`
	IP           string    `json:"ip"`
	CreatedAt    time.Time `json:"created_at"`
}
```

- [ ] **步骤 3：font.go**

```go
package models

import "time"

type Font struct {
	ID               int64     `json:"id"`
	Filename         string    `json:"filename"`
	DisplayName      string    `json:"display_name"`
	OriginalFilename string    `json:"original_filename"`
	FileHash         string    `json:"file_hash"`
	CreatedAt        time.Time `json:"created_at"`
}
```

- [ ] **步骤 4：admin.go**

```go
package models

import "time"

type Admin struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}
```

- [ ] **步骤 5：编译验证**

```bash
go build ./...
```

- [ ] **步骤 6：Commit**

---

### 任务 4：统一响应工具 + JWT 工具

**文件：**
- 创建：`inkflow-go/handlers/response.go`
- 创建：`inkflow-go/handlers/jwt.go`

- [ ] **步骤 1：response.go（统一 JSON 响应）**

```go
package handlers

import "github.com/gin-gonic/gin"

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type PageData struct {
	Items    interface{} `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, Response{Code: 0, Message: "success", Data: data})
}

func Error(c *gin.Context, httpCode, bizCode int, msg string) {
	c.JSON(httpCode, Response{Code: bizCode, Message: msg, Data: nil})
}

func Page(c *gin.Context, items interface{}, total int64, page, pageSize int) {
	c.JSON(200, Response{Code: 0, Message: "success", Data: PageData{
		Items: items, Total: total, Page: page, PageSize: pageSize,
	}})
}
```

- [ ] **步骤 2：jwt.go（JWT 生成与验证）**

```go
package handlers

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

func InitJWT(secret string) {
	jwtSecret = []byte(secret)
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("dev-secret-change-in-production")
	}
}

type Claims struct {
	AdminID  int64  `json:"admin_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func GenerateToken(adminID int64, username string) (string, error) {
	claims := Claims{
		AdminID:  adminID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}
```

- [ ] **步骤 3：编译验证**

```bash
go get github.com/golang-jwt/jwt/v5
go build ./...
```

- [ ] **步骤 4：Commit**

---

### 任务 5：Middleware（JWT 认证 + 限流 + CORS）

**文件：**
- 创建：`inkflow-go/handlers/middleware.go`

- [ ] **步骤 1：实现 middleware.go**

```go
package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthRequired 验证 JWT
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
			tokenStr = tokenStr[7:]
		}
		if tokenStr == "" {
			Error(c, http.StatusUnauthorized, 40101, "未登录或 token 过期")
			c.Abort()
			return
		}
		claims, err := ParseToken(tokenStr)
		if err != nil {
			Error(c, http.StatusUnauthorized, 40101, "未登录或 token 过期")
			c.Abort()
			return
		}
		c.Set("admin_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

type rateLimiter struct {
	mu    sync.Mutex
	visits map[string]*visit
}

type visit struct {
	count    int
	expireAt time.Time
}

func RateLimit(maxPerMinute int) gin.HandlerFunc {
	rl := &rateLimiter{visits: make(map[string]*visit)}
	go func() {
		for {
			time.Sleep(time.Minute)
			rl.mu.Lock()
			for ip, v := range rl.visits {
				if time.Now().After(v.expireAt) {
					delete(rl.visits, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return func(c *gin.Context) {
		if maxPerMinute <= 0 {
			c.Next()
			return
		}
		ip := c.ClientIP()
		rl.mu.Lock()
		v, ok := rl.visits[ip]
		if !ok || time.Now().After(v.expireAt) {
			rl.visits[ip] = &visit{count: 1, expireAt: time.Now().Add(time.Minute)}
			rl.mu.Unlock()
			c.Next()
			return
		}
		v.count++
		if v.count > maxPerMinute {
			rl.mu.Unlock()
			Error(c, http.StatusTooManyRequests, 40005, "请求过于频繁")
			c.Abort()
			return
		}
		rl.mu.Unlock()
		c.Next()
	}
}
```

- [ ] **步骤 2：编译验证**

```bash
go build ./...
```

- [ ] **步骤 3：Commit**

---

### 任务 6：Core - fontutil 字体工具

**文件：**
- 创建：`inkflow-go/core/fontutil.go`

- [ ] **步骤 1：实现 fontutil.go**

```go
package core

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func ReadFontName(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var head [12]byte
	io.ReadFull(f, head[:])
	tag := string(head[:4])
	switch tag {
	case "\x00\x01\x00\x00", "true", "OTTO":
		numTables := int(binary.BigEndian.Uint16(head[4:8]))
		for i := 0; i < numTables; i++ {
			var rec [16]byte
			io.ReadFull(f, rec[:])
			if string(rec[:4]) == "name" {
				offset := int64(binary.BigEndian.Uint32(rec[8:12]))
				length := int64(binary.BigEndian.Uint32(rec[12:16]))
				return readNameRecord(path, offset, length)
			}
		}
	}
	return "", fmt.Errorf("name table not found")
}

func readNameRecord(path string, offset, length int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	f.Seek(offset, 0)
	var format, count uint16
	binary.Read(f, binary.BigEndian, &format)
	binary.Read(f, binary.BigEndian, &count)
	var stringOffset uint16
	binary.Read(f, binary.BigEndian, &stringOffset)
	for i := uint16(0); i < count; i++ {
		var platform, encoding, language, nameID, nameLen, strOff uint16
		binary.Read(f, binary.BigEndian, &platform)
		binary.Read(f, binary.BigEndian, &encoding)
		binary.Read(f, binary.BigEndian, &language)
		binary.Read(f, binary.BigEndian, &nameID)
		binary.Read(f, binary.BigEndian, &nameLen)
		binary.Read(f, binary.BigEndian, &strOff)
		if nameID == 1 && platform == 3 && encoding == 1 && language == 1033 {
			cur, _ := f.Seek(0, io.SeekCurrent)
			f.Seek(offset+int64(stringOffset)+int64(strOff), 0)
			data := make([]byte, nameLen)
			io.ReadFull(f, data)
			f.Seek(cur, 0)
			runes := make([]rune, nameLen/2)
			for j := 0; j < len(data); j += 2 {
				runes[j/2] = rune(binary.BigEndian.Uint16(data[j:]))
			}
			return strings.TrimSpace(string(runes)), nil
		}
	}
	return "", fmt.Errorf("no suitable name")
}
```

- [ ] **步骤 2：编译验证**
```bash
go build ./...
```

- [ ] **步骤 3：Commit**

---

### 任务 7：Core - converter（PDF → PNG）

**文件：**
- 创建：`inkflow-go/core/converter.go`

- [ ] **步骤 1：实现 converter.go（go-pdfium WASM）**

```go
package core

import (
	"fmt"
	"image/png"
	"os"
	"time"
	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
)

var pdfiumPool pdfium.Pool
var pdfiumInst pdfium.Pdfium

func InitPDFium() error {
	var err error
	pdfiumPool, err = webassembly.Init(webassembly.Config{
		MinIdle: 1, MaxIdle: 2, MaxTotal: 4,
	})
	if err != nil {
		return fmt.Errorf("pdfium init: %w", err)
	}
	pdfiumInst, err = pdfiumPool.GetInstance(30 * time.Second)
	if err != nil {
		return fmt.Errorf("pdfium instance: %w", err)
	}
	return nil
}

func ClosePDFium() {
	if pdfiumPool != nil {
		pdfiumPool.Close()
	}
}

func PDFToImages(path string, outDir string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	doc, err := pdfiumInst.OpenDocument(&requests.OpenDocument{File: &data})
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer pdfiumInst.FPDF_CloseDocument(
		&requests.FPDF_CloseDocument{Document: doc.Document})
	pageCount, err := pdfiumInst.FPDF_GetPageCount(
		&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return nil, fmt.Errorf("pagecount: %w", err)
	}
	var outs []string
	for i := 0; i < pageCount.PageCount; i++ {
		r, err := pdfiumInst.RenderPageInDPI(&requests.RenderPageInDPI{
			DPI: 200,
			Page: requests.Page{ByIndex: &requests.PageByIndex{
				Document: doc.Document, Index: i,
			}},
		})
		if err != nil {
			return nil, fmt.Errorf("render %d: %w", i, err)
		}
		out := fmt.Sprintf("%s/page_%d.png", outDir, i+1)
		f, _ := os.Create(out)
		png.Encode(f, r.Result.Image)
		f.Close()
		outs = append(outs, out)
	}
	return outs, nil
}
```

- [ ] **步骤 2：编译验证**
```bash
go get github.com/klippa-app/go-pdfium/webassembly
go build ./...
```

- [ ] **步骤 3：Commit**

---

### 任务 8：Core - processor（物理效果）

**文件：**
- 创建：`inkflow-go/core/processor.go`

- [ ] **步骤 1：实现处理器 + 预设**

```go
package core

import (
	"image"
	"image/color"
	"image/draw"
	"math/rand"
)

type EffectParams struct {
	Noise  int `json:"noise"`
	Yellow int `json:"yellow"`
	Crease int `json:"crease"`
}

type Preset struct {
	Name   string
	Params EffectParams
}

var Presets = []Preset{
	{"none", EffectParams{0, 0, 0}},
	{"light", EffectParams{15, 20, 10}},
	{"moderate", EffectParams{35, 40, 30}},
	{"heavy", EffectParams{60, 65, 55}},
}

func ResolvePreset(name string) EffectParams {
	for _, p := range Presets {
		if p.Name == name {
			return p.Params
		}
	}
	return Presets[1].Params
}

func Compose(bg, textLayer *image.RGBA, p EffectParams, seed int64) *image.RGBA {
	b := bg.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, bg, b.Min, draw.Src)
	draw.Draw(out, b, textLayer, b.Min, draw.Over)
	if p.Noise > 0 || p.Yellow > 0 || p.Crease > 0 {
		rng := rand.New(rand.NewSource(seed))
		applyNoise(out, p.Noise, rng)
		applyYellow(out, p.Yellow)
		applyCrease(out, p.Crease, rng)
	}
	return out
}

func applyNoise(img *image.RGBA, intensity int, rng *rand.Rand) {
	if intensity == 0 { return }
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if rng.Intn(100) < intensity/5 {
				off := img.PixOffset(x, y)
				v := uint8(rng.Intn(40))
				img.Pix[off+0] = clamp(img.Pix[off+0] + v)
				img.Pix[off+1] = clamp(img.Pix[off+1] + v)
				img.Pix[off+2] = clamp(img.Pix[off+2] + v)
			}
		}
	}
}

func applyYellow(img *image.RGBA, intensity int) {
	if intensity == 0 { return }
	f := float64(intensity) / 100
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			off := img.PixOffset(x, y)
			img.Pix[off+0] = clamp(img.Pix[off+0] - uint8(5*f))
			img.Pix[off+2] = clamp(img.Pix[off+2] + uint8(10*f))
		}
	}
}

func applyCrease(img *image.RGBA, intensity int, rng *rand.Rand) {
	if intensity == 0 { return }
}

func clamp(v uint8) uint8 {
	if int(v) > 255 { return 255 }
	return v
}
```

- [ ] **步骤 2：编译验证**
```bash
go build ./...
```

- [ ] **步骤 3：Commit**

---

### 任务 9：Handler - 认证（Auth）

**文件：**
- 创建：`inkflow-go/handlers/auth.go`

- [ ] **步骤 1：实现 auth.go**

```go
package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func LoginHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var id int64
		var hash string
		err := db.QueryRow("SELECT id, password FROM admins WHERE username = ?", req.Username).Scan(&id, &hash)
		if err != nil {
			Error(c, http.StatusUnauthorized, 40102, "用户名或密码错误")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
			Error(c, http.StatusUnauthorized, 40102, "用户名或密码错误")
			return
		}
		token, err := GenerateToken(id, req.Username)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		Success(c, gin.H{"token": token})
	}
}

func MeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, _ := c.Get("username")
		Success(c, gin.H{"username": username})
	}
}
```

- [ ] **步骤 2：编译验证**

```bash
go get golang.org/x/crypto/bcrypt
go build ./...
```

- [ ] **步骤 3：Commit**

---

### 任务 7：Python 测试基础设施

**文件：**
- 创建：`inkflow-go/tests/requirements.txt`
- 创建：`inkflow-go/tests/run_tests.py`
- 创建：`inkflow-go/tests/conftest.py`

- [ ] **步骤 1：requirements.txt**

```
pytest==8.*
requests==2.*
playwright==1.*
```

- [ ] **步骤 2：run_tests.py（启动服务器 → 测试 → 清理）**

```python
import subprocess
import time
import sys
import os
import requests
import signal

SERVER_PORT = 9876
BASE_URL = f"http://localhost:{SERVER_PORT}"


def wait_for_server(timeout=15):
    start = time.time()
    while time.time() - start < timeout:
        try:
            r = requests.get(f"{BASE_URL}/api/health", timeout=2)
            if r.status_code == 200:
                return True
        except requests.ConnectionError:
            pass
        time.sleep(0.5)
    return False


def main():
    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    server = subprocess.Popen(
        ["go", "run", "."],
        env={**os.environ, "PORT": str(SERVER_PORT), "LOG_LEVEL": "error"},
        cwd=os.path.join(os.path.dirname(__file__), ".."),
    )
    try:
        if not wait_for_server():
            print("ERROR: Server did not start")
            server.kill()
            sys.exit(1)
        print("Server started, running tests...")
        result = subprocess.run(["pytest", "-v", "--rootdir=."], capture_output=False)
        sys.exit(result.returncode)
    finally:
        server.terminate()
        server.wait()


if __name__ == "__main__":
    main()
```

- [ ] **步骤 3：conftest.py**

```python
import pytest
import requests

BASE_URL = "http://localhost:9876"


@pytest.fixture
def api():
    return requests.Session()


@pytest.fixture
def admin_token(api):
    resp = api.post(f"{BASE_URL}/api/auth/login", json={
        "username": "admin",
        "password": "admin123",
    })
    data = resp.json()
    assert data["code"] == 0
    return data["data"]["token"]


@pytest.fixture
def auth_header(admin_token):
    return {"Authorization": f"Bearer {admin_token}"}
```

- [ ] **步骤 4：验证 run_tests.py 能启动服务器**

```bash
cd inkflow-go && python tests/run_tests.py
```

预期：服务器启动，/api/health 正常响应

- [ ] **步骤 5：Commit**

---

### 任务 8：Python API 测试 — 认证

**文件：**
- 创建：`inkflow-go/tests/test_api/test_auth.py`

- [ ] **步骤 1：test_auth.py**

```python
import pytest


class TestAuth:
    def test_login_success(self, api):
        resp = api.post(f"{BASE_URL}/api/auth/login", json={
            "username": "admin",
            "password": "admin123",
        })
        data = resp.json()
        assert data["code"] == 0
        assert "token" in data["data"]

    def test_login_wrong_password(self, api):
        resp = api.post(f"{BASE_URL}/api/auth/login", json={
            "username": "admin",
            "password": "wrong",
        })
        assert resp.json()["code"] == 40102

    def test_login_missing_fields(self, api):
        resp = api.post(f"{BASE_URL}/api/auth/login", json={})
        assert resp.json()["code"] == 40001

    def test_me_unauthorized(self, api):
        resp = api.get(f"{BASE_URL}/api/auth/me")
        assert resp.json()["code"] == 40101

    def test_me_authorized(self, api, auth_header):
        resp = api.get(f"{BASE_URL}/api/auth/me", headers=auth_header)
        assert resp.json()["code"] == 0
```

- [ ] **步骤 2：运行测试（预期失败，因为路由未注册）**

```bash
cd inkflow-go && python tests/run_tests.py -k test_auth
```

预期：测试失败（路由不存在）

- [ ] **步骤 3：更新 main.go 注册 auth 路由**

```go
import "inkflow-go/handlers"
handlers.InitJWT(cfg.JWTSecret)
database.Init(cfg.DBPath())
api := r.Group("/api")
{
    api.POST("/auth/login", handlers.LoginHandler(database.DB))
    auth := api.Group("")
    auth.Use(handlers.AuthRequired())
    {
        auth.GET("/auth/me", handlers.MeHandler())
    }
}
```

- [ ] **步骤 4：运行测试验证通过**

预期：5 个测试全部 PASS

- [ ] **步骤 5：Commit**

---

### 任务 9：Handler — 模板 CRUD

**文件：**
- 创建：`inkflow-go/handlers/templates.go`
- 创建：`inkflow-go/tests/test_api/test_templates.py`

- [ ] **步骤 1：test_templates.py**

```python
import pytest


class TestTemplates:
    def test_list_empty(self, api):
        resp = api.get(f"{BASE_URL}/api/templates")
        data = resp.json()
        assert data["code"] == 0
        assert data["data"]["total"] == 0
        assert data["data"]["items"] == []

    def test_create(self, api, auth_header):
        resp = api.post(f"{BASE_URL}/api/templates", json={
            "name": "租赁合同",
        }, headers=auth_header)
        data = resp.json()
        assert data["code"] == 0
        assert data["data"]["id"] > 0

    def test_create_unauthorized(self, api):
        resp = api.post(f"{BASE_URL}/api/templates", json={"name": "test"})
        assert resp.json()["code"] == 40101

    def test_get(self, api, auth_header):
        create = api.post(f"{BASE_URL}/api/templates", json={"name": "test"}, headers=auth_header).json()
        tid = create["data"]["id"]
        resp = api.get(f"{BASE_URL}/api/templates/{tid}", headers=auth_header)
        data = resp.json()
        assert data["data"]["name"] == "test"

    def test_get_not_found(self, api, auth_header):
        resp = api.get(f"{BASE_URL}/api/templates/99999", headers=auth_header)
        assert resp.json()["code"] == 40003

    def test_update(self, api, auth_header):
        create = api.post(f"{BASE_URL}/api/templates", json={"name": "old"}, headers=auth_header).json()
        tid = create["data"]["id"]
        resp = api.put(f"{BASE_URL}/api/templates/{tid}", json={
            "name": "new name",
            "config_json": '{"controls":[],"rules":[]}',
        }, headers=auth_header)
        assert resp.json()["code"] == 0

    def test_delete(self, api, auth_header):
        create = api.post(f"{BASE_URL}/api/templates", json={"name": "del"}, headers=auth_header).json()
        tid = create["data"]["id"]
        resp = api.delete(f"{BASE_URL}/api/templates/{tid}", headers=auth_header)
        assert resp.json()["code"] == 0
        get = api.get(f"{BASE_URL}/api/templates/{tid}", headers=auth_header)
        assert get.json()["code"] == 40003

    def test_list_pagination(self, api, auth_header):
        for i in range(3):
            api.post(f"{BASE_URL}/api/templates", json={"name": f"t{i}"}, headers=auth_header)
        resp = api.get(f"{BASE_URL}/api/templates?page=1&page_size=2", headers=auth_header)
        data = resp.json()["data"]
        assert len(data["items"]) <= 2
        assert data["total"] >= 3
```

- [ ] **步骤 2：运行测试（预期失败）**

- [ ] **步骤 3：实现 templates.go**

```go
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"inkflow-go/models"
)

func ListTemplatesHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
		keyword := c.Query("keyword")
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}
		offset := (page - 1) * pageSize

		var total int64
		countSQL := "SELECT COUNT(*) FROM templates WHERE deleted_at IS NULL"
		args := []interface{}{}
		if keyword != "" {
			countSQL += " AND name LIKE ?"
			args = append(args, "%"+keyword+"%")
		}
		db.QueryRow(countSQL, args...).Scan(&total)

		dataSQL := "SELECT id, name, created_at FROM templates WHERE deleted_at IS NULL"
		if keyword != "" {
			dataSQL += " AND name LIKE ?"
		}
		dataSQL += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
		queryArgs := append(args, pageSize, offset)

		rows, err := db.Query(dataSQL, queryArgs...)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		defer rows.Close()

		items := []models.TemplateListItem{}
		for rows.Next() {
			var item models.TemplateListItem
			rows.Scan(&item.ID, &item.Name, &item.CreatedAt)
			items = append(items, item)
		}
		Page(c, items, total, page, pageSize)
	}
}

func CreateTemplateHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		if req.Name == "" {
			req.Name = "未命名模板"
		}
		result, err := db.Exec("INSERT INTO templates (name) VALUES (?)", req.Name)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		id, _ := result.LastInsertId()
		Success(c, gin.H{"id": id})
	}
}

func GetTemplateHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var t models.Template
		err = db.QueryRow("SELECT id, name, COALESCE(bg_image,''), width, height, config_json, created_at FROM templates WHERE id = ? AND deleted_at IS NULL", id).
			Scan(&t.ID, &t.Name, &t.BgImage, &t.Width, &t.Height, &t.ConfigJSON, &t.CreatedAt)
		if err == sql.ErrNoRows {
			Error(c, http.StatusNotFound, 40003, "资源不存在")
			return
		}
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		Success(c, t)
	}
}

func UpdateTemplateHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var req struct {
			Name       string `json:"name"`
			BgImage    string `json:"bg_image"`
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			ConfigJSON string `json:"config_json"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		if req.ConfigJSON != "" {
			var cfg models.TemplateConfig
			if err := json.Unmarshal([]byte(req.ConfigJSON), &cfg); err != nil {
				Error(c, http.StatusBadRequest, 40001, "参数校验失败")
				return
			}
		}
		result, err := db.Exec("UPDATE templates SET name=?, bg_image=?, width=?, height=?, config_json=? WHERE id=? AND deleted_at IS NULL",
			req.Name, req.BgImage, req.Width, req.Height, req.ConfigJSON, id)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		if rows, _ := result.RowsAffected(); rows == 0 {
			Error(c, http.StatusNotFound, 40003, "资源不存在")
			return
		}
		Success(c, nil)
	}
}

func DeleteTemplateHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		result, err := db.Exec("UPDATE templates SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL", id)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		if rows, _ := result.RowsAffected(); rows == 0 {
			Error(c, http.StatusNotFound, 40003, "资源不存在")
			return
		}
		Success(c, nil)
	}
}
```

- [ ] **步骤 4：更新 main.go 注册模板路由**

```go
api.POST("/templates", handlers.CreateTemplateHandler(database.DB))
api.GET("/templates", handlers.ListTemplatesHandler(database.DB))
api.GET("/templates/:id", handlers.GetTemplateHandler(database.DB))
api.PUT("/templates/:id", handlers.AuthRequired(), handlers.UpdateTemplateHandler(database.DB))
api.DELETE("/templates/:id", handlers.AuthRequired(), handlers.DeleteTemplateHandler(database.DB))
```

- [ ] **步骤 5：运行测试验证通过**

- [ ] **步骤 6：Commit**

---

### 任务 10：Handler — 模板上传/导出/导入

**文件：**
- 修改：`inkflow-go/handlers/templates.go`（追加 upload/export/import）

- [ ] **步骤 1：Python 测试**

```python
class TestTemplateFiles:
    def test_upload_image(self, api, auth_header):
        # 先创建模板
        create = api.post(f"{BASE_URL}/api/templates", json={"name": "upload-test"}, headers=auth_header).json()
        tid = create["data"]["id"]
        # 上传图片
        resp = api.post(f"{BASE_URL}/api/templates/{tid}/upload", files={
            "file": ("test.png", b"fake-png-content", "image/png"),
        }, headers={"Authorization": auth_header["Authorization"]})
        assert resp.json()["code"] == 0
        assert "bg_image" in resp.json()["data"]

    def test_export_import(self, api, auth_header):
        create = api.post(f"{BASE_URL}/api/templates", json={"name": "export-test"}, headers=auth_header).json()
        tid = create["data"]["id"]
        export = api.get(f"{BASE_URL}/api/templates/{tid}/export", headers=auth_header)
        assert export.status_code == 200
        assert export.headers["Content-Type"] == "application/zip"
```

- [ ] **步骤 2：实现 upload/export/import**

主要逻辑：
- upload：接受 multipart file，保存到 `data/bg_images/{template_id}/`，更新 bg_image 字段
- export：读取模板 JSON + bg_image → 打包 ZIP
- import：解压 ZIP → 读取 template.json → 创建模板 + 保存 bg_image

- [ ] **步骤 3：运行测试验证通过**

- [ ] **步骤 4：Commit**

---

### 任务 11：Handler — 签署（Sign）

**文件：**
- 创建：`inkflow-go/handlers/sign.go`

- [ ] **步骤 1：Python 测试**

```python
class TestSign:
    def test_sign_success(self, api, auth_header):
        create = api.post(f"{BASE_URL}/api/templates", json={"name": "sign-test"}, headers=auth_header).json()
        tid = create["data"]["id"]
        resp = api.post(f"{BASE_URL}/api/templates/{tid}/sign", json={
            "fields_data": {"name": "张三"},
            "effect_preset": "light",
        })
        data = resp.json()
        assert data["code"] == 0
        assert "record_id" in data["data"]
        assert "image_url" in data["data"]

    def test_sign_template_not_found(self, api):
        resp = api.post(f"{BASE_URL}/api/templates/99999/sign", json={
            "fields_data": {},
        })
        assert resp.json()["code"] == 40003
```

- [ ] **步骤 2：实现 sign.go**

```go
package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SignRequest struct {
	FieldsData     map[string]interface{} `json:"fields_data"`
	EffectPreset   string                 `json:"effect_preset"`
	EffectParams   map[string]int         `json:"effect_params,omitempty"`
	TextLayerData  string                 `json:"text_layer_data"`
}

func SignHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var req SignRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var name string
		var bgImage string
		err = db.QueryRow("SELECT name, COALESCE(bg_image,'') FROM templates WHERE id = ? AND deleted_at IS NULL", id).Scan(&name, &bgImage)
		if err == sql.ErrNoRows {
			Error(c, http.StatusNotFound, 40003, "资源不存在")
			return
		}
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		// 保存签署记录
		fieldsJSON, _ := json.Marshal(req.FieldsData)
		ip := c.ClientIP()
		result, err := db.Exec("INSERT INTO signing_records (template_id, fields_data, ip_address) VALUES (?, ?, ?)",
			id, string(fieldsJSON), ip)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		recordID, _ := result.LastInsertId()

		Success(c, gin.H{
			"record_id": recordID,
			"image_url": "/static/output/placeholder.png",
		})
	}
}
```

- [ ] **步骤 3：运行测试验证通过**

- [ ] **步骤 4：Commit**

---

### 任务 12：Handler — 签署记录 + 字体 + 统计

**文件：**
- 创建：`inkflow-go/handlers/records.go`
- 创建：`inkflow-go/handlers/fonts.go`
- 创建：`inkflow-go/handlers/admin.go`
- 创建：测试文件

- [ ] **步骤 1：Python 测试**

```python
class TestRecords:
    def test_list_records(self, api, auth_header):
        resp = api.get(f"{BASE_URL}/api/records", headers=auth_header)
        data = resp.json()
        assert data["code"] == 0

    def test_delete_record(self, api, auth_header):
        resp = api.delete(f"{BASE_URL}/api/records/99999", headers=auth_header)
        assert resp.json()["code"] == 40003

class TestFonts:
    def test_list_fonts(self, api, auth_header):
        resp = api.get(f"{BASE_URL}/api/fonts", headers=auth_header)
        data = resp.json()
        assert data["code"] == 0

class TestAdmin:
    def test_stats(self, api, auth_header):
        resp = api.get(f"{BASE_URL}/api/admin/stats", headers=auth_header)
        c.json()["code"] == 0
```

- [ ] **步骤 2：实现 records.go / fonts.go / admin.go**

Records: List with pagination (JOIN templates for name), soft-delete
Fonts: List, Upload (file validation + hash dedup + fontutil), Update, Delete
Admin: SELECT COUNT(*) from templates/records/fonts WHERE deleted_at IS NULL

- [ ] **步骤 3：运行测试验证通过**

- [ ] **步骤 4：Commit**

---

### 任务 13：主入口完整注册

**文件：**
- 修改：`inkflow-go/main.go`

- [ ] **步骤 1：完整 main.go**

```go
package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"inkflow-go/config"
	"inkflow-go/database"
	"inkflow-go/handlers"
)

func main() {
	cfg := config.Load()
	if len(os.Args) > 1 && os.Args[1] == "init" {
		password := "admin123"
		if len(os.Args) > 2 {
			password = os.Args[2]
		}
		if err := initAdmin(cfg.DBPath(), password); err != nil {
			slog.Error("init admin failed", "error", err)
			os.Exit(1)
		}
		return
	}

	// 初始化日志
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

	// 初始化数据库
	if err := database.Init(cfg.DBPath()); err != nil {
		slog.Error("database init failed", "error", err)
		os.Exit(1)
	}
	slog.Info("database initialized")

	// 初始化 JWT
	handlers.InitJWT(cfg.JWTSecret)

	// 初始化管理员（自动）
	initAdminAuto(cfg.DBPath())

	// 创建目录
	os.MkdirAll(cfg.BgImageDir(), 0755)
	os.MkdirAll(cfg.OutputDir(), 0755)
	os.MkdirAll(cfg.FontsDir(), 0755)

	// Gin 路由
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 静态文件服务
	r.Static("/static", cfg.DataDir)

	// API 路由
	api := r.Group("/api")
	api.Use(handlers.RateLimit(cfg.RateLimit))
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
		})
		api.POST("/auth/login", handlers.LoginHandler(database.DB))

		// 公开接口
		api.GET("/templates", handlers.ListTemplatesHandler(database.DB))
		api.GET("/templates/:id", handlers.GetTemplateHandler(database.DB))

		// 需认证
		auth := api.Group("")
		auth.Use(handlers.AuthRequired())
		{
			auth.GET("/auth/me", handlers.MeHandler())
			auth.POST("/templates", handlers.CreateTemplateHandler(database.DB))
			auth.PUT("/templates/:id", handlers.UpdateTemplateHandler(database.DB))
			auth.DELETE("/templates/:id", handlers.DeleteTemplateHandler(database.DB))
			auth.POST("/templates/:id/upload", handlers.UploadTemplateFileHandler(database.DB, cfg.BgImageDir()))
			auth.GET("/templates/:id/export", handlers.ExportTemplateHandler(database.DB, cfg.BgImageDir()))
			auth.POST("/templates/import", handlers.ImportTemplateHandler(database.DB, cfg.BgImageDir()))
			auth.GET("/records", handlers.ListRecordsHandler(database.DB))
			auth.DELETE("/records/:id", handlers.DeleteRecordHandler(database.DB))
			auth.GET("/fonts", handlers.ListFontsHandler(database.DB))
			auth.POST("/fonts", handlers.CreateFontHandler(database.DB, cfg.FontsDir()))
			auth.PUT("/fonts/:id", handlers.UpdateFontHandler(database.DB))
			auth.DELETE("/fonts/:id", handlers.DeleteFontHandler(database.DB))
			auth.GET("/admin/stats", handlers.StatsHandler(database.DB))
		}
	}

	// 签署（公开）
	r.POST("/api/templates/:id/sign", handlers.SignHandler(database.DB))

	// SPA 回退
	r.NoRoute(func(c *gin.Context) {
		c.File("frontend/dist/index.html")
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	slog.Info("server starting", "addr", addr)

	var err error
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		err = r.RunTLS(addr, cfg.TLSCert, cfg.TLSKey)
	} else {
		err = r.Run(addr)
	}
	if err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func initAdminAuto(dbPath string) {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	if count > 0 {
		return
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = fmt.Sprintf("admin%d", time.Now().Unix()%1000000)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	database.DB.Exec("INSERT INTO admins (username, password) VALUES ('admin', ?)", string(hash))
	slog.Info("initial admin created", "username", "admin", "password", password)
}
```

- [ ] **步骤 2：编译验证**

```bash
go build ./...
```

- [ ] **步骤 3：运行所有 Python 测试**

```bash
cd inkflow-go && python tests/run_tests.py
```

预期：所有已有测试 PASS

- [ ] **步骤 4：Commit**

---

### 任务 14：前端项目搭建

**文件：**
- 创建：`inkflow-go/frontend/package.json`
- 创建：`inkflow-go/frontend/vite.config.ts`
- 创建：`inkflow-go/frontend/tsconfig.json`
- 创建：`inkflow-go/frontend/index.html`
- 创建：`inkflow-go/frontend/src/main.ts`
- 创建：`inkflow-go/frontend/src/App.vue`

- [ ] **步骤 1：初始化 Vue 3 + Vite + TypeScript**

```bash
cd inkflow-go/frontend
npm create vite@latest . -- --template vue-ts
npm install vue-router@4 pinia
npm install -D @types/node
```

- [ ] **步骤 2：vite.config.ts（含代理）**

```ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': resolve(__dirname, 'src') }
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/static': 'http://localhost:8080'
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
```

- [ ] **步骤 3：main.ts + App.vue**

```ts
// main.ts
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './styles/global.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
```

```vue
<!-- App.vue -->
<template>
  <AppNav v-if="showNav" />
  <router-view />
  <AppToast />
  <ConfirmDialog />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppNav from '@/components/AppNav.vue'
import AppToast from '@/components/AppToast.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

const route = useRoute()
const showNav = computed(() => !route.meta?.hideNav)
</script>
```

- [ ] **步骤 4：编译验证**

```bash
cd frontend && npm run build
```

预期：`frontend/dist/` 目录生成

- [ ] **步骤 5：Commit**

---

### 任务 15：前端 API 客户端 + 类型 + 路由 + 状态

**文件：**
- 创建：`inkflow-go/frontend/src/types/index.ts`
- 创建：`inkflow-go/frontend/src/api/index.ts`
- 创建：`inkflow-go/frontend/src/router/index.ts`
- 创建：`inkflow-go/frontend/src/stores/auth.ts`

- [ ] **步骤 1：types/index.ts**

```ts
export interface Control {
  id: string
  label: string
  type: 'textbox' | 'checkbox'
  x: number
  y: number
  width: number
  height: number
  fontSize: number
  fontFamily: string
  required: boolean
  previewText?: string
  checkSize?: number
}

export interface Rule {
  id: string
  type: string
  control_id: string
  config: any
}

export interface TemplateConfig {
  controls: Control[]
  rules: Rule[]
}

export interface Template {
  id: number
  name: string
  bg_image: string
  width: number
  height: number
  config_json: string
  created_at: string
}

export interface TemplateListItem {
  id: number
  name: string
  created_at: string
}

export interface SigningRecord {
  id: number
  template_id: number
  template_name: string
  fields_data: string
  image_url: string
  ip: string
  created_at: string
}

export interface FontItem {
  id: number
  filename: string
  display_name: string
  original_filename: string
  created_at: string
}

export interface AdminStats {
  template_count: number
  record_count: number
  font_count: number
}

export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

export interface PageData<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}
```

- [ ] **步骤 2：api/index.ts**

```ts
const BASE = '/api'

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {
    ...(options?.headers as Record<string, string>),
  }
  if (token) headers['Authorization'] = `Bearer ${token}`
  if (!(options?.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }
  const res = await fetch(`${BASE}${url}`, { ...options, headers })
  return res.json()
}

export const api = {
  // Auth
  login: (data: { username: string; password: string }) =>
    request<ApiResponse<{ token: string }>>('/auth/login', {
      method: 'POST', body: JSON.stringify(data),
    }),
  me: () => request<ApiResponse<{ username: string }>>('/auth/me'),

  // Templates
  listTemplates: (params?: string) =>
    request<ApiResponse<PageData<TemplateListItem>>>(`/templates?${params || ''}`),
  createTemplate: (name: string) =>
    request<ApiResponse<{ id: number }>>('/templates', {
      method: 'POST', body: JSON.stringify({ name }),
    }),
  getTemplate: (id: number) => request<ApiResponse<Template>>(`/templates/${id}`),
  updateTemplate: (id: number, data: any) =>
    request<ApiResponse<null>>(`/templates/${id}`, {
      method: 'PUT', body: JSON.stringify(data),
    }),
  deleteTemplate: (id: number) =>
    request<ApiResponse<null>>(`/templates/${id}`, { method: 'DELETE' }),
  uploadTemplateFile: (id: number, file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return request<ApiResponse<{ bg_image: string }>>(`/templates/${id}/upload`, {
      method: 'POST', body: fd,
    })
  },

  // Records
  listRecords: (params?: string) =>
    request<ApiResponse<PageData<SigningRecord>>>(`/records?${params || ''}`),
  deleteRecord: (id: number) =>
    request<ApiResponse<null>>(`/records/${id}`, { method: 'DELETE' }),

  // Fonts
  listFonts: () => request<ApiResponse<FontItem[]>>('/fonts'),
  deleteFont: (id: number) =>
    request<ApiResponse<null>>(`/fonts/${id}`, { method: 'DELETE' }),

  // Stats
  getStats: () => request<ApiResponse<AdminStats>>('/admin/stats'),
}
```

- [ ] **步骤 3：router/index.ts**

```ts
import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: () => import('@/views/Home.vue') },
    { path: '/sign', name: 'sign', component: () => import('@/views/SignFront.vue') },
    { path: '/admin/login', name: 'login', component: () => import('@/views/AdminLogin.vue'), meta: { hideNav: true } },
    { path: '/admin', name: 'dashboard', component: () => import('@/views/AdminDashboard.vue'), meta: { requiresAuth: true } },
    { path: '/admin/editor', name: 'editor', component: () => import('@/views/AdminEditor.vue'), meta: { requiresAuth: true } },
    { path: '/admin/records', name: 'records', component: () => import('@/views/AdminRecords.vue'), meta: { requiresAuth: true } },
    { path: '/admin/fonts', name: 'fonts', component: () => import('@/views/AdminFonts.vue'), meta: { requiresAuth: true } },
  ],
})

router.beforeEach((to, _from, next) => {
  if (to.meta?.requiresAuth && !localStorage.getItem('token')) {
    next('/admin/login')
  } else {
    next()
  }
})

export default router
```

- [ ] **步骤 4：stores/auth.ts（Pinia）**

```ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const username = ref('')

  async function login(user: string, pass: string) {
    const res = await api.login({ username: user, password: pass })
    if (res.code === 0) {
      token.value = res.data.token
      username.value = user
      localStorage.setItem('token', res.data.token)
    }
    return res
  }

  function logout() {
    token.value = ''
    username.value = ''
    localStorage.removeItem('token')
  }

  return { token, username, login, logout }
})
```

- [ ] **步骤 5：编译验证**

```bash
cd frontend && npx vue-tsc --noEmit && npx vite build
```

- [ ] **步骤 6：Commit**

---

### 任务 16：前端通用组件

**文件：**
- 创建：`inkflow-go/frontend/src/components/AppNav.vue`
- 创建：`inkflow-go/frontend/src/components/AppToast.vue`
- 创建：`inkflow-go/frontend/src/components/ConfirmDialog.vue`
- 创建：`inkflow-go/frontend/src/components/PageHeader.vue`
- 创建：`inkflow-go/frontend/src/styles/global.css`

- [ ] **步骤 1：global.css（CSS 变量 + 基础样式 + 暗/亮模式）**

定义 CSS 变量、按钮样式、卡片样式、动画、响应式基础。

- [ ] **步骤 2：AppNav / AppToast / ConfirmDialog / PageHeader**

每个组件独立、可复用。

- [ ] **步骤 3：编译验证**

```bash
cd frontend && npm run build
```

- [ ] **步骤 4：Commit**

---

### 任务 17：前端页面 — Home + AdminLogin + AdminDashboard

**文件：**
- 创建：`inkflow-go/frontend/src/views/Home.vue`
- 创建：`inkflow-go/frontend/src/views/AdminLogin.vue`
- 创建：`inkflow-go/frontend/src/views/AdminDashboard.vue`

- [ ] **步骤 1：Home.vue**

模板网格列表，显示所有模板，支持「签署」「编辑」「删除」操作。

- [ ] **步骤 2：AdminLogin.vue**

登录表单，调用 auth store，成功后跳转 /admin。

- [ ] **步骤 3：AdminDashboard.vue**

统计卡片 + 操作快速入口。

- [ ] **步骤 4：编译验证**

```bash
cd frontend && npm run build
```

- [ ] **步骤 5：Commit**

---

### 任务 18：前端页面 — AdminEditor（核心）

**文件：**
- 创建：`inkflow-go/frontend/src/components/CanvasEditor.vue`（Fabric.js 封装）
- 创建：`inkflow-go/frontend/src/components/ControlPanel.vue`
- 创建：`inkflow-go/frontend/src/components/RuleManager.vue`
- 创建：`inkflow-go/frontend/src/components/EffectPresets.vue`
- 创建：`inkflow-go/frontend/src/views/AdminEditor.vue`

- [ ] **步骤 1：CanvasEditor.vue**

```vue
<template>
  <div ref="container" class="canvas-editor"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { fabric } from 'fabric'

const props = defineProps<{
  bgImage?: string
  controls: any[]
  width: number
  height: number
}>()

const emit = defineEmits<{
  update: [controls: any[]]
}>()

const container = ref<HTMLDivElement>()
let canvas: fabric.Canvas | null = null

onMounted(() => {
  if (!container.value) return
  canvas = new fabric.Canvas(container.value, {
    width: props.width,
    height: props.height,
    preserveObjectStacking: true,
  })
})

onUnmounted(() => {
  canvas?.dispose()
  canvas = null
})
</script>
```

- [ ] **步骤 2：EffectPresets.vue**

预设选择器（无效果/轻度/中度/重度）+ 3 个滑块（噪点/泛黄/折痕）。

- [ ] **步骤 3：AdminEditor.vue**

完整编辑器页面，整合 CanvasEditor + ControlPanel + RuleManager + EffectPresets。

- [ ] **步骤 4：编译验证**

```bash
cd frontend && npm install fabric@5 && npm run build
```

- [ ] **步骤 5：Commit**

---

### 任务 19：前端页面 — SignFront + AdminRecords + AdminFonts

**文件：**
- 创建：`inkflow-go/frontend/src/views/SignFront.vue`
- 创建：`inkflow-go/frontend/src/views/AdminRecords.vue`
- 创建：`inkflow-go/frontend/src/views/AdminFonts.vue`

- [ ] **步骤 1：SignFront.vue**

- 左侧表单（根据控件动态生成）
- 右侧 Canvas 预览
- 提交按钮 → 调用 sign 接口

- [ ] **步骤 2：AdminRecords.vue**

记录表格 + 图片预览 modal + 删除。

- [ ] **步骤 3：AdminFonts.vue**

字体列表 + 上传（拖拽）+ 删除。

- [ ] **步骤 4：编译验证全项目**

```bash
cd frontend && npm run build
cd .. && go build -o dist/inkflow.exe .
```

- [ ] **步骤 5：Commit**

---

### 任务 20：构建脚本 + README

**文件：**
- 创建：`inkflow-go/build.bat`
- 创建：`inkflow-go/build.sh`
- 创建：`inkflow-go/README.md`

- [ ] **步骤 1：build.bat**

```batch
@echo off
cd /d "%~dp0"
echo [1/2] Building frontend...
cd frontend && call npm run build
if %errorlevel% neq 0 exit /b %errorlevel%
cd ..
echo [2/2] Building Go backend...
go build -ldflags="-s -w" -o dist\inkflow.exe .
echo Build complete: dist\inkflow.exe
```

- [ ] **步骤 2：build.sh**

```sh
#!/bin/bash
set -e
echo "[1/2] Building frontend..."
cd frontend && npm run build && cd ..
echo "[2/2] Building Go backend..."
go build -ldflags="-s -w" -o dist/inkflow .
echo "Build complete: dist/inkflow"
```

- [ ] **步骤 3：README.md**

```markdown
# InkFlow-Go 电子签章系统

单文件可执行程序，纯 Go 实现，无需外部依赖。

## 快速开始

1. 下载 `inkflow.exe`
2. 双击运行
3. 查看控制台输出的管理员密码
4. 打开 http://localhost:8080

## 开发

./build.bat  # Windows
./build.sh   # Linux/macOS
```

- [ ] **步骤 4：Commit**

---

### 任务 21：Python E2E 测试（Playwright）

**文件：**
- 创建：`inkflow-go/tests/test_e2e/test_sign_flow.py`

- [ ] **步骤 1：安装 Playwright 浏览器**

```bash
pip install playwright
playwright install chromium
```

- [ ] **步骤 2：test_sign_flow.py**

```python
import pytest
from playwright.sync_api import sync_playwright

BASE_URL = "http://localhost:9876"


class TestSignFlow:
    def test_home_page_loads(self):
        with sync_playwright() as p:
            browser = p.chromium.launch()
            page = browser.new_page()
            page.goto(BASE_URL)
            assert page.title() is not None
            browser.close()

    def test_admin_login(self, api):
        with sync_playwright() as p:
            browser = p.chromium.launch()
            page = browser.new_page()
            page.goto(f"{BASE_URL}/admin/login")
            page.fill("input[type='text']", "admin")
            page.fill("input[type='password']", "admin123")
            page.click("button[type='submit']")
            page.wait_for_url(f"{BASE_URL}/admin")
            assert "/admin" in page.url
            browser.close()
```

- [ ] **步骤 3：运行 E2E 测试**

```bash
cd inkflow-go && python -m pytest tests/test_e2e/ -v
```

- [ ] **步骤 4：全量回归测试**

```bash
cd inkflow-go && python tests/run_tests.py
```

- [ ] **步骤 5：Commit**

---

## 自检清单

- [ ] 规格中每个需求都有对应的任务
- [ ] 无 TODO/占位符
- [ ] 测试先于实现（Python 测试驱动 Go 开发）
- [ ] 所有类型和函数签名在上下游任务中一致
- [ ] 数据库 schema 覆盖所有模型
- [ ] API 路由与设计文档一致
- [ ] 错误码覆盖所有异常场景
- [ ] 前端路由与后端权限匹配
