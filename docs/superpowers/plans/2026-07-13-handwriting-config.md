# 手写效果三级配置 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将手写效果从硬编码改为可配置，支持三级覆盖：全局 → 模板 → 签署（临时）

**架构：** 全局配置存 `settings` 表（键值），模板配置存 `template_handwriting` 表（1:1），签署临时配置通过 sign 请求传递（不持久化）

**优先级：** 签署临时配置 > 模板配置 > 全局配置（逐字段覆盖，`undefined`/`null` 表示沿用上级）

---

### 任务 1：数据层定义（types + models）

**文件：**
- 创建：`internal/models/handwriting.go`
- 修改：`internal/models/template.go`
- 修改：`frontend/src/types/index.ts`

- [ ] **步骤 1.1：定义 `HandwritingConfig` 模型**

`internal/models/handwriting.go`：

```go
package models

type HandwritingConfig struct {
    PaperEnabled     *bool    `json:"paper_enabled,omitempty"`
    PaperOpacity     *float64 `json:"paper_opacity,omitempty"`
    FiberCount       *int     `json:"fiber_count,omitempty"`
    DotCount         *int     `json:"dot_count,omitempty"`
    GlobalTilt       *float64 `json:"global_tilt,omitempty"`
    BaselineDrift    *float64 `json:"baseline_drift,omitempty"`
    CharJitter       *float64 `json:"char_jitter,omitempty"`
    CharRotation     *float64 `json:"char_rotation,omitempty"`
    InkOpacityMin    *float64 `json:"ink_opacity_min,omitempty"`
    InkOpacityMax    *float64 `json:"ink_opacity_max,omitempty"`
    CharSpacing      *float64 `json:"char_spacing,omitempty"`
    InkSpotsEnabled  *bool    `json:"ink_spots_enabled,omitempty"`
    InkSpotsChance   *float64 `json:"ink_spots_chance,omitempty"`
    InkSpotsMax      *int     `json:"ink_spots_max,omitempty"`
    ShadowBlur       *float64 `json:"shadow_blur,omitempty"`
    CheckboxEnabled  *bool    `json:"checkbox_enabled,omitempty"`
}

func DefaultHandwriting() HandwritingConfig {
    t := true
    return HandwritingConfig{
        PaperEnabled:    &t,
        PaperOpacity:    float64Ptr(0.12),
        FiberCount:      intPtr(200),
        DotCount:        intPtr(800),
        GlobalTilt:      float64Ptr(1),
        BaselineDrift:   float64Ptr(0.8),
        CharJitter:      float64Ptr(2),
        CharRotation:    float64Ptr(1.5),
        InkOpacityMin:   float64Ptr(0.85),
        InkOpacityMax:   float64Ptr(1.0),
        CharSpacing:     float64Ptr(1.5),
        InkSpotsEnabled: &t,
        InkSpotsChance:  float64Ptr(0.15),
        InkSpotsMax:     intPtr(2),
        ShadowBlur:      float64Ptr(0.8),
        CheckboxEnabled: &t,
    }
}

// Merge overlays non-nil fields from `other` onto `base`
func (base HandwritingConfig) Merge(other HandwritingConfig) HandwritingConfig {
    result := base
    if other.PaperEnabled != nil { result.PaperEnabled = other.PaperEnabled }
    if other.PaperOpacity != nil { result.PaperOpacity = other.PaperOpacity }
    if other.FiberCount != nil { result.FiberCount = other.FiberCount }
    if other.DotCount != nil { result.DotCount = other.DotCount }
    if other.GlobalTilt != nil { result.GlobalTilt = other.GlobalTilt }
    if other.BaselineDrift != nil { result.BaselineDrift = other.BaselineDrift }
    if other.CharJitter != nil { result.CharJitter = other.CharJitter }
    if other.CharRotation != nil { result.CharRotation = other.CharRotation }
    if other.InkOpacityMin != nil { result.InkOpacityMin = other.InkOpacityMin }
    if other.InkOpacityMax != nil { result.InkOpacityMax = other.InkOpacityMax }
    if other.CharSpacing != nil { result.CharSpacing = other.CharSpacing }
    if other.InkSpotsEnabled != nil { result.InkSpotsEnabled = other.InkSpotsEnabled }
    if other.InkSpotsChance != nil { result.InkSpotsChance = other.InkSpotsChance }
    if other.InkSpotsMax != nil { result.InkSpotsMax = other.InkSpotsMax }
    if other.ShadowBlur != nil { result.ShadowBlur = other.ShadowBlur }
    if other.CheckboxEnabled != nil { result.CheckboxEnabled = other.CheckboxEnabled }
    return result
}

func float64Ptr(v float64) *float64 { return &v }
func intPtr(v int) *int { return &v }
```

- [ ] **步骤 1.2：修改 `template.go` 模型**

删除旧 `Rule` 和 `TemplateConfig` 结构体，新增 `ControlRow`、`RuleRow`、`TemplateWithHandwriting`：

```go
// 删除以下内容（已被新表替代）:
// type Rule struct { ... }
// type TemplateConfig struct { ... }

// ControlRow 表示 template_controls 表的一行
type ControlRow struct {
    ID           string  `json:"id"`
    TemplateID   int64   `json:"-"`
    Label        string  `json:"label"`
    Type         string  `json:"type"`
    X            float64 `json:"x"`
    Y            float64 `json:"y"`
    Width        float64 `json:"width"`
    Height       float64 `json:"height"`
    FontSize     int     `json:"font_size"`
    FontFamily   string  `json:"font_family"`
    Required     bool    `json:"required"`
    PreviewText  string  `json:"preview_text"`
    CheckSize    int     `json:"check_size"`
    SortOrder    int     `json:"-"`
}

// RuleRow 表示 template_rules 表的一行
type RuleRow struct {
    ID           string      `json:"id"`
    TemplateID   int64       `json:"-"`
    Type         string      `json:"type"`
    Name         string      `json:"name"`
    Target       string      `json:"target"`
    ConfigJSON   string      `json:"-"`
    Config       interface{} `json:"config"`
    SortOrder    int         `json:"-"`
}

// TemplateWithHandwriting 扩展 Template，携带手写效果配置
type TemplateWithHandwriting struct {
    Template
    Rules       []RuleRow          `json:"rules"`
    Handwriting *HandwritingConfig `json:"handwriting,omitempty"`
}
```

- [ ] **步骤 1.3：前端类型定义**

`frontend/src/types/index.ts` 新增：

```typescript
export interface HandwritingConfig {
  paper_enabled?: boolean
  paper_opacity?: number
  fiber_count?: number
  dot_count?: number
  global_tilt?: number
  baseline_drift?: number
  char_jitter?: number
  char_rotation?: number
  ink_opacity_min?: number
  ink_opacity_max?: number
  char_spacing?: number
  ink_spots_enabled?: boolean
  ink_spots_chance?: number
  ink_spots_max?: number
  shadow_blur?: number
  checkbox_enabled?: boolean
}

export interface Rule {
  id: string
  type: string
  name?: string
  target: string
  config: Record<string, any>
}

// 修改 Template 接口
export interface Template {
  id: number
  name: string
  bg_image: string
  width: number
  height: number
  controls: Control[]
  rules?: Rule[]
  handwriting?: HandwritingConfig
  created_at: string
}
```

---

### 任务 2：数据库迁移

**文件：**
- 修改：`internal/database/db.go`

- [ ] **步骤 2.1：新增 migration**

在 `migrate()` 函数的 `tables` 切片中追加：

```sql
CREATE TABLE IF NOT EXISTS template_controls (
    id             TEXT PRIMARY KEY,
    template_id    INTEGER NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    label          TEXT NOT NULL DEFAULT '',
    type           TEXT NOT NULL DEFAULT 'textbox',
    x              REAL DEFAULT 0,
    y              REAL DEFAULT 0,
    width          REAL DEFAULT 200,
    height         REAL DEFAULT 40,
    font_size      INTEGER DEFAULT 20,
    font_family    TEXT DEFAULT 'sans-serif',
    required       INTEGER DEFAULT 0,
    preview_text   TEXT DEFAULT '',
    check_size     INTEGER DEFAULT 24,
    sort_order     INTEGER DEFAULT 0,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
),

CREATE TABLE IF NOT EXISTS template_rules (
    id             TEXT PRIMARY KEY,
    template_id    INTEGER NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    type           TEXT NOT NULL,
    name           TEXT DEFAULT '',
    target         TEXT DEFAULT '',
    config_json    TEXT NOT NULL DEFAULT '{}',
    sort_order     INTEGER DEFAULT 0,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
),

CREATE TABLE IF NOT EXISTS template_handwriting (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id       INTEGER NOT NULL UNIQUE REFERENCES templates(id) ON DELETE CASCADE,
    paper_enabled     INTEGER DEFAULT 1,
    paper_opacity     REAL DEFAULT 0.12,
    fiber_count       INTEGER DEFAULT 200,
    dot_count         INTEGER DEFAULT 800,
    global_tilt       REAL DEFAULT 1,
    baseline_drift    REAL DEFAULT 0.8,
    char_jitter       REAL DEFAULT 2,
    char_rotation     REAL DEFAULT 1.5,
    ink_opacity_min   REAL DEFAULT 0.85,
    ink_opacity_max   REAL DEFAULT 1.0,
    char_spacing      REAL DEFAULT 1.5,
    ink_spots_enabled INTEGER DEFAULT 1,
    ink_spots_chance  REAL DEFAULT 0.15,
    ink_spots_max     INTEGER DEFAULT 2,
    shadow_blur       REAL DEFAULT 0.8,
    checkbox_enabled  INTEGER DEFAULT 1,
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME DEFAULT CURRENT_TIMESTAMP
),

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '{}'
),

CREATE INDEX IF NOT EXISTS idx_tc_template_id ON template_controls(template_id),
CREATE INDEX IF NOT EXISTS idx_tr_template_id ON template_rules(template_id),
CREATE INDEX IF NOT EXISTS idx_tr_type ON template_rules(type)
```

---

### 任务 3：全局配置 API

**文件：**
- 创建：`internal/handlers/settings.go`

- [ ] **步骤 3.1：实现 settings handler**

```go
package handlers

// GET /api/settings/handwriting
// 返回全局手写效果配置，不存在则返回默认值

// PUT /api/settings/handwriting
// 保存全局手写效果配置，upsert 到 settings 表 key='handwriting'
```

---

### 任务 4：模板 API 重写（使用新表）

**文件：**
- 重写：`internal/handlers/templates.go`

- [ ] **步骤 4.1：重写 `CreateTemplateHandler`**

创建模板后，插入默认手写效果到 `template_handwriting`。

- [ ] **步骤 4.2：重写 `GetTemplateHandler`**

JOIN 三个新表返回完整数据：controls + rules + handwriting。

- [ ] **步骤 4.3：重写 `UpdateTemplateHandler`**

接收 `controls`、`rules`、`handwriting` 字段，分别写入对应表（事务操作）。

- [ ] **步骤 4.4：重写 `DeleteTemplateHandler`**

依赖 `ON DELETE CASCADE` 自动清理，无需手动删除子表。

---

### 任务 5：签署 API 三级合并

**文件：**
- 修改：`internal/handlers/sign.go`

- [ ] **步骤 5.1：修改 `SignRequest`**

```go
type SignRequest struct {
    FieldsData    map[string]interface{}      `json:"fields_data"`
    EffectPreset  string                      `json:"effect_preset"`
    EffectParams  map[string]int              `json:"effect_params,omitempty"`
    TextLayerData string                      `json:"text_layer_data"`
    Handwriting   *models.HandwritingConfig   `json:"handwriting,omitempty"`
}
```

- [ ] **步骤 5.2：实现三级合并逻辑**

```go
func (h SignHandler) mergeHandwriting(templateID int64, signOverride *models.HandwritingConfig) models.HandwritingConfig {
    // 1. 全局默认
    global := loadGlobalHandwriting(h.db)
    // 2. 模板覆盖
    templateHW := loadTemplateHandwriting(h.db, templateID)
    merged := global.Merge(templateHW)
    // 3. 签署临时覆盖
    if signOverride != nil {
        merged = merged.Merge(*signOverride)
    }
    return merged
}
```

---

### 任务 6：后端路由注册

**文件：**
- 修改：`internal/server/server.go`

- [ ] **步骤 6.1：注册 settings 路由**

```go
auth.GET("/settings/handwriting", handlers.GetHandwritingSettings(database.DB))
auth.PUT("/settings/handwriting", handlers.UpdateHandwritingSettings(database.DB))
```

---

### 任务 7：前端基础（API + Store）

**文件：**
- 修改：`frontend/src/api/index.ts`
- 修改：`frontend/src/components/editor/editorStore.ts`

- [ ] **步骤 7.1：新增 settings API**

```typescript
getHandwritingSettings: () =>
  request<ApiResponse<HandwritingConfig>>('/settings/handwriting'),
updateHandwritingSettings: (data: HandwritingConfig) =>
  request<ApiResponse<null>>('/settings/handwriting', {
    method: 'PUT', body: JSON.stringify(data),
  }),
```

- [ ] **步骤 7.2：扩展 editorStore**

新增 state：
```typescript
const handwriting = ref<HandwritingConfig>({})
const useGlobalHandwriting = ref(true)
```

新增方法：
```typescript
async function loadHandwriting()  // 从模板加载
function resetHandwriting()       // 恢复默认
```

`save()` 方法增加 `handwriting` 字段到请求 body。

---

### 任务 8：HandwritingPanel 组件

**文件：**
- 创建：`frontend/src/components/editor/HandwritingPanel.vue`
- 修改：`frontend/src/components/editor/PropertiesPanel.vue`
- 修改：`frontend/src/views/AdminEditor.vue`

- [ ] **步骤 8.1：创建 HandwritingPanel.vue**

配置面板，包含：
- 纸张纹理：启用开关、不透明度滑块、纤维数量、斑点数量
- 文字渲染：倾斜角度、基线漂移、字符抖动、字符旋转、墨水不透明度范围、字间距
- 墨点：启用开关、出现概率、每字最多数
- 阴影模糊滑块
- 勾选手绘开关
- "恢复默认"按钮 + "预览效果"按钮

- [ ] **步骤 8.2：PropertiesPanel 底部集成**

属性面板底部增加"手写效果"折叠区域，点击展开 HandwritingPanel。

- [ ] **步骤 8.3：AdminEditor 集成**

EditorToolbar 或 PropertiesPanel 中增加手写效果入口。

---

### 任务 9：全局设置页面

**文件：**
- 创建：`frontend/src/views/AdminSettings.vue`
- 修改：`frontend/src/views/AdminDashboard.vue`
- 修改：`frontend/src/router/index.ts`

- [ ] **步骤 9.1：创建 AdminSettings.vue**

页面内容：
- 标题"全局设置 - 手写效果"
- 复用 HandwritingPanel 组件（全局模式）
- "保存"按钮 → 调用 `api.updateHandwritingSettings()`

- [ ] **步骤 9.2：AdminDashboard 增加入口**

Dashboard 页面增加"全局设置"按钮，路由跳转 `/admin/settings`。

- [ ] **步骤 9.3：注册路由**

```typescript
{ path: '/admin/settings', name: 'AdminSettings', component: () => import('@/views/AdminSettings.vue'), meta: { requiresAuth: true } }
```

---

### 任务 10：SignFront 三级合并 + 绘制参数化

**文件：**
- 重写：`frontend/src/views/SignFront.vue`

- [ ] **步骤 10.1：三级合并逻辑**

```typescript
// 模板加载后
const globalHW = await api.getHandwritingSettings()
const templateHW = template.value?.handwriting || {}
const baseHW = mergeHandwriting(globalHW.data, templateHW)

// 签署时
const signHW = signOverride.value || {}
const finalHW = mergeHandwriting(baseHW, signHW)
```

`mergeHandwriting` 实现逐字段覆盖。

- [ ] **步骤 10.2：绘制函数参数化**

修改 `drawPaperTexture`、`drawTextOnContext`、`drawCheckOnContext`，接收 `HandwritingConfig` 参数替代硬编码值。

- [ ] **步骤 10.3：临时配置 UI**

签署页表单底部增加折叠区域"调整手写效果"，展开后显示 HandwritingPanel（签署模式，不持久化）。

- [ ] **步骤 10.4：sign 请求携带**

```typescript
body: JSON.stringify({
  fields_data: formData.value,
  effect_preset: 'light',
  text_layer_data: textLayerData,
  handwriting: signOverride.value,  // 新增
})
```

---

### 任务 11：验证

- [ ] **步骤 11.1：前端 typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

- [ ] **步骤 11.2：前端构建**

```bash
npx vite build
```

- [ ] **步骤 11.3：后端编译**

```bash
cd inkflow-go && go build ./...
```

- [ ] **步骤 11.4：后端测试**

```bash
go test ./...
```

---

### 验收标准

| 检查项 | 通过条件 |
|--------|---------|
| 全局配置 | AdminSettings 页保存后，新建模板自动继承 |
| 模板配置 | 编辑器可自定义，保存后独立于全局 |
| 签署临时配置 | 签署页调整后实时预览，签署完成后不保存 |
| 三级合并 | 签署页正确叠加 全局→模板→临时 |
| 默认值 | 未配置时使用 `DefaultHandwriting()` |
| 级联删除 | 删模板时子表数据自动清理 |
| 构建 | `vue-tsc` + `vite build` + `go build` + `go test` 全部通过 |
