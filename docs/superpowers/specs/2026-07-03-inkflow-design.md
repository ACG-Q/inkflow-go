# InkFlow-Go 电子签章系统 — 设计规格

## 概述

从零开发的 Go 版电子签章系统，基于 Go 版本的使用经验，保留 Node 版的功能完整性，重新设计交互流程，采用纯 Go 技术栈实现单文件可执行程序。

## 技术栈

| 层次 | 选择 | 理由 |
|------|------|------|
| 后端框架 | Gin v1.x | 成熟稳定，当前 Go 版已验证 |
| 数据库 | modernc.org/sqlite | 纯 Go SQLite，支持 SQL 查询 |
| PDF 渲染 | go-pdfium + Wazero WASM | Chrome 级渲染质量，纯 Go 无外部依赖 |
| 图片处理 | 自定义 Go 管线 | 物理效果预设 + 简易调节 |
| 日志 | log/slog（标准库） | 零依赖，JSON/Text 双格式 |
| 前端框架 | Vue 3 + TypeScript | 组件化开发，良好的开发体验 |
| 状态管理 | Pinia | 编辑器等复杂页面共享状态 |
| Canvas 引擎 | Fabric.js 5.x | HTML5 Canvas 交互编辑 |
| 构建工具 | Vite 8.x | 快速开发构建 |
| 测试 | Python (pytest + requests + Playwright) | 按要求使用 Python 编写测试 |
| 认证 | JWT | 无状态，适合单 exe 部署 |

## 项目结构

```
inkflow-go/
├── main.go                        # 入口：路由注册、DB 初始化
├── go.mod
├── cmd/
│   └── init.go                    # 初始化管理员密码、JWT 密钥
├── config/
│   └── config.go                  # 配置（端口、路径、常量）
├── database/
│   └── db.go                      # modernc.org/sqlite 连接 + 表迁移
├── models/
│   ├── template.go                # 模板模型
│   ├── signing_record.go          # 签署记录模型
│   ├── font.go                    # 字体模型
│   └── admin.go                   # 管理员模型
├── handlers/
│   ├── auth.go                    # 登录/认证
│   ├── middleware.go              # JWT 中间件 + 限流中间件
│   ├── templates.go               # 模板 CRUD + 上传/导出/导入
│   ├── sign.go                    # 签名合成
│   ├── records.go                 # 签署记录 CRUD
│   ├── fonts.go                   # 字体管理
│   └── admin.go                   # 统计数据
├── core/
│   ├── converter.go               # go-pdfium WASM → PNG
│   ├── processor.go               # 物理效果管线（预设 + 调节）
│   └── fontutil.go                # 字体读取/校验
├── web/
│   └── embed.go                   //go:embed frontend/dist/*
├── frontend/                      # Vue 3 源码
│   ├── src/
│   │   ├── views/                 # 7 个页面
│   │   ├── components/            # 通用组件
│   │   ├── api/                   # API 客户端
│   │   ├── stores/                # Pinia 状态管理
│   │   ├── types/                 # TypeScript 接口
│   │   ├── router/                # 路由 + 守卫
│   │   └── styles/                # 全局样式
│   ├── package.json
│   └── vite.config.ts
├── tests/                         # Python 测试
│   ├── requirements.txt
│   ├── run_tests.py               # 总控：启动 Go 服务器 → 测试 → 清理
│   ├── conftest.py
│   ├── test_api/
│   └── test_e2e/
├── build.bat                      # Windows 构建
├── build.sh                       # Linux/macOS 构建
├── test.bat                       # Windows 构建 + 测试
└── test.sh                        # Linux/macOS 构建 + 测试
```

## 数据库设计

```sql
-- 模板表
CREATE TABLE templates (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL DEFAULT '未命名模板',
    bg_image    TEXT,                    -- data/bg_images/{uuid}/page_N.png
    width       INTEGER DEFAULT 800,
    height      INTEGER DEFAULT 1000,
    config_json TEXT NOT NULL DEFAULT '{}',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME DEFAULT NULL    -- 软删除
);

-- 签署记录表
CREATE TABLE signing_records (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id INTEGER NOT NULL REFERENCES templates(id),
    fields_data TEXT,                    -- 用户填写的字段值 JSON
    image_url   TEXT,                    -- data/output/{uuid}.png
    ip_address  TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME DEFAULT NULL
);

-- 字体表
CREATE TABLE fonts (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    filename          TEXT NOT NULL,         -- data/fonts/{hash}.ttf
    display_name      TEXT,                  -- 从字体元数据读取
    original_filename TEXT,                  -- 上传时的原始文件名
    file_hash         TEXT,                  -- SHA-256 排重
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at        DATETIME DEFAULT NULL
);

-- 管理员表
CREATE TABLE admins (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    username  TEXT NOT NULL UNIQUE DEFAULT 'admin',
    password  TEXT NOT NULL,             -- bcrypt hash
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_records_template_id ON signing_records(template_id);
CREATE INDEX idx_records_created_at ON signing_records(created_at);
CREATE INDEX idx_templates_deleted_at ON templates(deleted_at);
CREATE INDEX idx_records_deleted_at ON signing_records(deleted_at);
```

### 存储路径

```
data/
├── bg_images/{uuid}/page_N.png    # PDF 转换的背景图
├── output/{uuid}.png              # 签名合成输出
├── fonts/{file_hash}.ttf          # 字体文件
└── templates.db                   # SQLite 数据库文件
```

### 密码管理

- admins.password 使用 bcrypt 哈希存储，绝不存明文
- 首次启动自动检测：无管理员 → 尝试环境变量 `ADMIN_PASSWORD` → 若空则生成随机密码打印到控制台
- 命令行初始化：`inkflow.exe init <password>`
- JWT 密钥：未设置 `JWT_SECRET` 环境变量时自动生成 32 字节随机密钥（仅存内存，重启后旧 token 失效）

### 查询规范

所有查询默认加 `WHERE deleted_at IS NULL`，实现软删除。

## RESTful API 规范

### 统一响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 分页响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [],
    "total": 42,
    "page": 1,
    "page_size": 20
  }
}
```

### 错误码

| code | 含义 |
|------|------|
| 0 | 成功 |
| 40001 | 参数校验失败 |
| 40002 | 文件格式不支持 |
| 40003 | 资源不存在 |
| 40004 | 权限不足 |
| 40005 | 文件过大 |
| 40101 | 未登录或 token 过期 |
| 40102 | 用户名或密码错误 |
| 50001 | 数据库错误 |
| 50002 | 文件写入失败 |
| 50003 | PDF 渲染失败 |
| 50004 | 签名合成失败 |
| 50005 | 物理效果处理失败 |
| 50006 | 外部服务不可用 |

### 路由表

```
# 认证
POST /api/auth/login            # 管理员登录 → JWT
GET  /api/auth/me               # 当前用户信息
## 登出由客户端清理 token，无服务端接口

# 模板
GET    /api/templates                    # 列表（page, page_size, keyword）
POST   /api/templates                    # 创建
GET    /api/templates/:id                # 详情
PUT    /api/templates/:id                # 更新
DELETE /api/templates/:id                # 删除（软删除）
POST   /api/templates/:id/upload         # 上传 PDF/图片
GET    /api/templates/:id/export         # 导出 ZIP
POST   /api/templates/import             # 导入 ZIP

# 签署
POST   /api/templates/:id/sign           # 生成签名图

# 签署记录
GET    /api/records                      # 列表（page, page_size, template_id, start, end）
DELETE /api/records/:id                  # 删除

# 字体
GET    /api/fonts
POST   /api/fonts
PUT    /api/fonts/:id
DELETE /api/fonts/:id

# 统计
GET    /api/admin/stats
```

### 签名接口

```
POST /api/templates/:id/sign
Content-Type: application/json

Request:
{
  "fields_data": { "field_1": "张三", "field_2": true },
  "effect_preset": "moderate",           // none | light | moderate | heavy
  "effect_params": {                     // 可选，覆盖预设
    "noise": 30,
    "yellow": 40,
    "crease": 20
  },
  "text_layer_data": "data:image/png;base64,..."
}

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "record_id": 42,
    "image_url": "/static/output/abc123.png"
  }
}
```

### ZIP 导入导出规范

```
template-export.zip
├── template.json     # { name, width, height, config_json }
└── bg_image.png      # 背景原图
```

### 权限

| 路由 | 权限 |
|------|------|
| 签署相关 | 公开（游客） |
| 模板列表/详情 | 公开 |
| 模板增删改/上传/导出导入 | 管理员 |
| 字体管理 | 管理员 |
| 记录管理 | 管理员 |
| 统计 | 管理员 |

### 速率限制

基于 IP 的内存限流中间件，默认 60 请求/分钟，通过配置 `RATE_LIMIT` 调整。

## 前端架构

### 路由

| 路径 | 页面 | 权限 |
|------|------|------|
| `/` | Home | 公开 |
| `/sign?template_id=X` | SignFront | 公开 |
| `/admin/login` | AdminLogin | 公开 |
| `/admin` | AdminDashboard | 管理员 |
| `/admin/editor?id=X` | AdminEditor | 管理员 |
| `/admin/records` | AdminRecords | 管理员 |
| `/admin/fonts` | AdminFonts | 管理员 |

### 组件树

```
App.vue
├── AppNav.vue             # 导航栏
├── AppToast.vue           # 通知
├── ConfirmDialog.vue      # 确认框
├── Home.vue               # 首页
├── SignFront.vue          # 签署页
├── AdminLogin.vue         # 登录
├── AdminDashboard.vue     # 管理首页
├── AdminEditor.vue        # 编辑器
│   ├── ControlPanel.vue   # 控件面板
│   ├── RuleManager.vue    # 规则管理
│   └── EffectPresets.vue  # 物理效果预设
├── AdminRecords.vue       # 签署记录
└── AdminFonts.vue         # 字体管理
```

### 状态管理（Pinia）

| Store | 职责 |
|-------|------|
| `useTemplateStore` | 当前编辑模板的 controls/rules/背景 |
| `useFontStore` | 字体列表缓存 |
| `useSettingsStore` | 物理效果预设配置 |

### Fabric.js 封装

自定义 `CanvasEditor.vue` 组件：
- `onMounted` 初始化 `Fabric.Canvas`
- `onUnmounted` 释放资源 `canvas.dispose()`
- 通过 props 传入 controls，emit 事件通知父组件
- 不直接操作 DOM，完全通过 Vue 生命周期管理

### 移动端适配

签署页响应式布局：
- 大屏（>=768px）：左侧表单 + 右侧 Canvas 预览
- 小屏（<768px）：仅表单，预览可折叠展开

## 物理效果管线（core/processor.go）

### 输入输出

| 输入 | 类型 | 说明 |
|------|------|------|
| 背景图 | `image.Image` | PDF 渲染的页面 PNG |
| 文本图层 | `image.Image` | 用户填写内容，前端 Canvas 生成 |
| 预设名称 | `string` | `none` / `light` / `moderate` / `heavy` |
| 微调参数 | `EffectParams` | `{noise, yellow, crease: 0-100}` |
| 随机种子 | `int64` | 可选，保证可复现性 |

| 输出 | 类型 | 说明 |
|------|------|------|
| 签名图 | `*image.RGBA` | 合成后的最终图片 |
| 图片路径 | `string` | 保存到 `data/output/{uuid}.png` |

### 管线流程

```
背景图 → 与文本图层叠合 → 加载预设参数
  → 1. 随机旋转 (±0.5°)
  → 2. 高斯模糊 (radius 0.3)
  → 3. 加噪点 (按 noise 参数)
  → 4. 泛黄 (按 yellow 参数)
  → 5. 折痕 (按 crease 参数)
  → 6. 纤维纹理
  → 7. 暗角
  → 输出 RGBA
```

### 前端一致性

前端 Canvas 预览使用同一套算法参数，效果参数序列化为 JSON 传递给生成接口，前后端管线严格一致。

## 日志模块

使用 Go 标准库 `log/slog`：

```go
slog.Info("template created", "id", id, "admin", user)
slog.Error("pdf rendering failed", "error", err, "template_id", id)
```

- 生产环境 `LOG_FORMAT=json`，便于日志采集
- 开发环境 `LOG_FORMAT=text`，可读性好
- Info 级别记录操作审计（谁 + 什么操作 + 什么资源）
- Error 级别记录异常堆栈

## 构建方案

### build.bat (Windows)

```batch
cd frontend && npm run build
cd ..
go build -ldflags="-s -w" -o dist/inkflow.exe .
xcopy /E /I frontend\dist dist\static
```

### build.sh (Linux/macOS)

```sh
cd frontend && npm run build && cd ..
go build -ldflags="-s -w" -o dist/inkflow .
```

### go:embed

```go
//go:embed frontend/dist/*
var embeddedFS embed.FS
```

PDFium WASM 由 go-pdfium/webassembly 依赖自动管理，无需手动嵌入。

## 部署与配置

### 配置项

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `PORT` | 8080 | 监听端口 |
| `DATA_DIR` | ./data | 数据存储目录 |
| `JWT_SECRET` | 自动生成 | JWT 签名密钥 |
| `ADMIN_PASSWORD` | 随机生成 | 初始管理员密码 |
| `RATE_LIMIT` | 60 | 每分钟每 IP 最大请求数 |
| `LOG_LEVEL` | info | 日志级别 |
| `LOG_FORMAT` | text | 日志格式（text/json） |
| `CORS_ORIGINS` | * | 允许的跨域来源 |
| `TLS_CERT` | — | HTTPS 证书路径 |
| `TLS_KEY` | — | HTTPS 密钥路径 |

### 初始化流程

```bash
# 首次运行：自动检测无管理员 → 生成随机密码 → 打印到控制台
inkflow.exe

# 手动初始化
inkflow.exe init --password=myadmin123

# 指定端口
inkflow.exe --port=3000

# 启用 HTTPS
inkflow.exe --tls-cert=cert.pem --tls-key=key.pem
```

### CORS

- 开发模式：允许 `localhost:5173`（Vite dev server）
- 生产模式：允许同域访问，通过 `CORS_ORIGINS` 环境变量配置

## 性能预期

| 场景 | 预期 |
|------|------|
| PDF 渲染（单页 A4, 200 DPI） | < 1s |
| 签名合成（含物理效果） | < 2s |
| 并发签署 | ~10 QPS |
| 编辑器 Canvas 操作 | 60 FPS |

## 测试方案（Python）

`tests/run_tests.py` 统一管控：

1. `go build` 构建测试用二进制
2. 启动 `--test --port=9876`（独立测试库）
3. 轮询 `/api/health` 等待就绪
4. 运行 pytest
5. 终止服务器，返回测试结果

API 测试：requests 验证每个接口的状态码和响应格式。
E2E 测试：Playwright 模拟浏览器交互。

## 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| PDF 渲染 | go-pdfium WASM | 纯 Go，无外部依赖，Chrome 级质量 |
| 数据库 | modernc.org/sqlite | 纯 Go，SQL 查询灵活 |
| 日志 | slog | 标准库无依赖 |
| 状态管理 | Pinia | 编辑器复杂状态共享 |
| Fabric 集成 | 封装自定义组件 | 生命周期安全，避免 DOM 冲突 |
| 物理效果 | 预设 + 3 参数微调 | 简化操作，保留灵活性 |
| 认证 | JWT + bcrypt | 无状态密码安全 |
| 软删除 | deleted_at 字段 | 数据可恢复 |
| 测试 | Python | 按要求执行 |
| 前端嵌入 | go:embed | 单文件发布 |
