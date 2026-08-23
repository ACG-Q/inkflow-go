# InkFlow-Go 存储架构文档

## 1. 目录布局

所有运行时数据存储在 `DATA_DIR`（默认 `./data`）下：

```
data/
├── templates.db                # SQLite 数据库
├── bg_images/                  # 模板背景图
│   ├── {id}/                   # 按模板 ID 分目录
│   │   ├── bg.png              # 上传的图片文件
│   │   └── pages/              # PDF 转换结果
│   │       ├── page_1.png
│   │       ├── page_2.png
│   │       └── ...
│   └── imported/               # 导入模板的背景图
│       └── {filename}
├── output/                     # 签名合成输出
│   └── signed_{id}_{pid36}.png
└── fonts/                      # 字体文件
    ├── {sha256}.ttf
    ├── {sha256}.otf
    ├── {sha256}.woff
    └── {sha256}.woff2
```

启动时自动创建所有子目录（`main.go:19-25`）。

---

## 2. 数据库

### 2.1 PRAGMA 配置

连接建立后立即执行以下 PRAGMA（`database/db.go:25-36`）：

| PRAGMA | 值 | 说明 |
|--------|-----|------|
| `journal_mode` | `WAL` | Write-Ahead Logging，允许读写并发，崩溃恢复更快 |
| `busy_timeout` | `5000` | 写锁等待 5 秒超时，避免高并发下立即报错 |
| `foreign_keys` | `ON` | 强制外键约束，`signing_records.template_id` 必须引用 `templates.id` |
| `synchronous` | `NORMAL` | WAL 模式下安全且比 `FULL` 快，崩溃最多丢 1 帧 |
| `temp_store` | `MEMORY` | 临时表（排序、GROUP BY）使用内存，减少磁盘 I/O |

### 2.2 连接池

```go
DB.SetMaxOpenConns(4)   // 最大打开连接数
DB.SetMaxIdleConns(2)   // 最大空闲连接数
```

SQLite 写锁是串行的，4 连接足够处理读并发 + 偶发写操作。过高会增加锁竞争。

### 2.3 表结构

#### templates — 模板表

| 字段 | 类型 | 默认值 | 约束 | 说明 |
|------|------|--------|------|------|
| `id` | INTEGER | — | PK AUTOINCREMENT | 主键 |
| `name` | TEXT | `'未命名模板'` | NOT NULL | 模板名称 |
| `bg_image` | TEXT | NULL | — | 背景图相对路径，如 `bg_images/1/pages/page_1.png` |
| `width` | INTEGER | `800` | — | 画布宽度（px） |
| `height` | INTEGER | `1000` | — | 画布高度（px） |
| `config_json` | TEXT | `'{}'` | NOT NULL | 控件配置 JSON（TemplateConfig 序列化） |
| `created_at` | DATETIME | `CURRENT_TIMESTAMP` | — | 创建时间 |
| `deleted_at` | DATETIME | NULL | — | 软删除时间戳 |

#### signing_records — 签署记录表

| 字段 | 类型 | 默认值 | 约束 | 说明 |
|------|------|--------|------|------|
| `id` | INTEGER | — | PK AUTOINCREMENT | 主键 |
| `template_id` | INTEGER | — | NOT NULL, FK → templates(id) | 关联模板 |
| `fields_data` | TEXT | NULL | — | 用户填写的字段值 JSON |
| `image_url` | TEXT | NULL | — | 签名图 URL，如 `/static/output/signed_1_abc.png` |
| `ip_address` | TEXT | NULL | — | 签署者 IP |
| `created_at` | DATETIME | `CURRENT_TIMESTAMP` | — | 签署时间 |
| `deleted_at` | DATETIME | NULL | — | 软删除时间戳 |

#### fonts — 字体表

| 字段 | 类型 | 默认值 | 约束 | 说明 |
|------|------|--------|------|------|
| `id` | INTEGER | — | PK AUTOINCREMENT | 主键 |
| `filename` | TEXT | — | NOT NULL | 存储文件名，如 `a1b2c3.ttf` |
| `display_name` | TEXT | NULL | — | 从字体 name table 提取的显示名 |
| `original_filename` | TEXT | NULL | — | 上传时的原始文件名 |
| `file_hash` | TEXT | NULL | — | SHA-256 哈希，用于去重 |
| `created_at` | DATETIME | `CURRENT_TIMESTAMP` | — | 创建时间 |
| `deleted_at` | DATETIME | NULL | — | 软删除时间戳 |

#### admins — 管理员表

| 字段 | 类型 | 默认值 | 约束 | 说明 |
|------|------|--------|------|------|
| `id` | INTEGER | — | PK AUTOINCREMENT | 主键 |
| `username` | TEXT | `'admin'` | NOT NULL UNIQUE | 用户名 |
| `password` | TEXT | — | NOT NULL | bcrypt 哈希 |
| `created_at` | DATETIME | `CURRENT_TIMESTAMP` | — | 创建时间 |

> admins 表不支持软删除。

### 2.4 索引

| 索引名 | 列 | 用途 |
|--------|-----|------|
| `idx_records_template_id` | `signing_records(template_id)` | 按模板查询签署记录 |
| `idx_records_created_at` | `signing_records(created_at)` | 按时间排序/范围查询 |
| `idx_templates_deleted_at` | `templates(deleted_at)` | 软删除过滤 |
| `idx_records_deleted_at` | `signing_records(deleted_at)` | 软删除过滤 |
| `idx_fonts_deleted_at` | `fonts(deleted_at)` | 软删除过滤 |
| `idx_fonts_file_hash` | `fonts(file_hash)` | 字体去重查询 |
| `idx_templates_name` | `templates(name)` | 按名称搜索/排序 |

### 2.5 软删除策略

所有表（admins 除外）通过 `deleted_at` 字段实现软删除：

- **删除操作**：`UPDATE ... SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL`
- **查询过滤**：所有 SELECT 语句加 `WHERE deleted_at IS NULL`
- **数据恢复**：需手动执行 `UPDATE ... SET deleted_at = NULL WHERE id = ?`

---

## 3. 文件存储

### 3.1 背景图 — `bg_images/`

**命名规则**：

| 来源 | 路径格式 | 示例 |
|------|---------|------|
| 图片上传 | `bg_images/{id}/bg.{ext}` | `bg_images/5/bg.png` |
| PDF 上传 | `bg_images/{id}/pages/page_N.png` | `bg_images/5/pages/page_1.png` |
| 导入 ZIP | `bg_images/imported/{filename}` | `bg_images/imported/contract.png` |

**PDF 转换流程**（`UploadTemplateFileHandler`）：

1. 接收 PDF → 写入临时文件 `bg_images/{id}/_tmp_upload.pdf`
2. 调用 `core.PDFToImages(tmpPath, outDir)` 转换为 PNG
3. 删除临时 PDF
4. 数据库写入 `bg_image = "bg_images/{id}/pages/page_1.png"`
5. DB 失败时 `os.RemoveAll(outDir)` 清理

**路由映射**：

```
GET /static/bg_images/{id}/pages/page_1.png
    → 服务端: data/bg_images/{id}/pages/page_1.png
```

**签章时查找逻辑**（`SignHandler`）：

```go
bgPath := filepath.Join(bgImageDir, bgImage)     // 直接拼接
if _, err := os.Stat(bgPath); os.IsNotExist(err) {
    altPath := filepath.Join(bgImageDir, strings.TrimPrefix(bgImage, "bg_images/"))
    // 兼容旧数据：尝试去掉 "bg_images/" 前缀
}
```

### 3.2 输出图片 — `output/`

**命名规则**：`signed_{template_id}_{pid_base36}.png`

```go
outFileName := fmt.Sprintf("signed_%d_%s.png", id, strconv.FormatInt(int64(os.Getpid()), 36))
```

- `pid_base36`：当前进程 PID 的 base36 编码，避免并发冲突
- 例：PID=12345 → `signed_5_9n9.png`

**路由映射**：

```
GET /static/output/signed_5_9n9.png
    → 服务端: data/output/signed_5_9n9.png
```

**生命周期**：

- 签署时创建，无自动清理机制
- 数据库记录保留 `image_url` 字段指向该文件
- 删除记录不会自动删除输出文件

### 3.3 字体文件 — `fonts/`

**命名规则**：`{sha256}.{ext}`

- `sha256`：文件内容的 SHA-256 哈希
- `ext`：原始扩展名（`.ttf`、`.otf`、`.woff`、`.woff2`）

**上传流程**（`CreateFontHandler`）：

1. 校验扩展名（仅允许 `.ttf`、`.otf`、`.woff`、`.woff2`）
2. 写入临时文件 `_tmp_{filename}`
3. 计算 SHA-256 哈希
4. 查询 `file_hash` 去重 → 若已存在则拒绝
5. `os.Rename` 移动到最终路径（跨文件系统会失败，需同分区）
6. DB 失败时 `os.Remove(destPath)` 清理

**路由映射**：

```
GET /static/fonts/a1b2c3.ttf
    → 服务端: data/fonts/a1b2c3.ttf
```

---

## 4. 数据一致性

### 4.1 文件-数据库事务策略

由于 SQLite 文件操作和 `os.Create`/`os.WriteFile` 无法在同一个事务中完成，采用**补偿清理**模式：

| 操作 | 创建文件 | DB 写入 | 失败回滚 |
|------|---------|---------|---------|
| 签署 | `os.Create(outPath)` | `INSERT signing_records` | `os.Remove(outPath)` |
| 字体上传 | `os.Rename(tmp, dest)` | `INSERT fonts` | `os.Remove(destPath)` |
| PDF 上传 | `PDFToImages()` 写入 `pages/` | `UPDATE templates.bg_image` | `os.RemoveAll(outDir)` |
| 图片上传 | `os.Create(outPath)` | `UPDATE templates.bg_image` | `os.Remove(outPath)` |

`sign.go:153`、`fonts.go:94`、`templates.go:275` 均实现了失败清理。

### 4.2 孤立文件处理

- **`--test` 模式**：`os.RemoveAll(dataDir)` 完全删除测试数据目录
- **生产模式**：无自动清理。以下情况可能产生孤立文件：
  - 模板被软删除，背景图仍保留在 `bg_images/{id}/`
  - 签署记录被软删除，输出图片仍保留在 `output/`
  - 字体被软删除，字体文件仍保留在 `fonts/`

**清理建议**：

```bash
# 查找无 DB 记录的孤立背景图目录
ls data/bg_images/ | while read id; do
  [ -f "data/bg_images/$id/.gitkeep" ] && continue
  sqlite3 data/templates.db "SELECT COUNT(*) FROM templates WHERE id=$id AND deleted_at IS NULL" | grep -q 0 && echo "orphan: bg_images/$id"
done
```

---

## 5. 运维指南

### 5.1 备份

SQLite 数据库在 WAL 模式下备份流程：

```bash
# 1. 执行 CHECKPOINT，将 WAL 合并到主库
sqlite3 data/templates.db "PRAGMA wal_checkpoint(TRUNCATE);"

# 2. 备份数据库文件
cp data/templates.db data/templates.db.bak

# 3. 备份文件目录
tar czf data-files-$(date +%Y%m%d).tar.gz data/bg_images/ data/output/ data/fonts/
```

**完整恢复步骤**：

1. 停止服务
2. 恢复 `templates.db` + `templates.db-wal` + `templates.db-shm`
3. 恢复 `bg_images/`、`output/`、`fonts/` 目录
4. 启动服务

### 5.2 数据库迁移

当前使用 `CREATE TABLE IF NOT EXISTS` 无版本管理。如需结构性迁移：

```bash
# 查看当前表结构
sqlite3 data/templates.db ".schema"

# 查看表版本（如已添加 user_version）
sqlite3 data/templates.db "PRAGMA user_version;"
```

建议在 `migrate()` 中增加版本号检查：

```go
func migrate() error {
    var version int
    database.DB.QueryRow("PRAGMA user_version").Scan(&version)
    if version < 1 {
        // 执行 v1 迁移...
        database.DB.Exec("PRAGMA user_version = 1")
    }
    return nil
}
```

### 5.3 性能调优

| 参数 | 默认值 | 调优建议 |
|------|--------|---------|
| `MaxOpenConns` | 4 | 读密集型可增至 8；写密集型保持 4 |
| `MaxIdleConns` | 2 | 保持 2 即可 |
| `busy_timeout` | 5000 | 并发写入多可增至 10000 |
| `synchronous` | `NORMAL` | 数据安全优先改 `FULL`，性能下降约 2x |
| `journal_mode` | `WAL` | 保持，不建议改 `DELETE` |
| `temp_store` | `MEMORY` | 保持，足够内存时可存大量临时数据 |

### 5.4 磁盘空间估算

| 资源 | 单个大小 | 1000 模板估算 |
|------|---------|-------------|
| 模板背景图（单页 A4 200DPI） | ~200KB | ~200MB |
| 签署输出图 | ~300KB | ~300MB |
| 字体文件 | ~500KB | ~50MB |
| SQLite 数据库 | <1MB | ~5MB |
| **总计** | — | **~555MB** |
