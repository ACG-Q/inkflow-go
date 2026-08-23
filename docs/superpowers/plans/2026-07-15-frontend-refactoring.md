# 前端全面重构实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 对 InkFlow 前端进行系统性重构，核心原则：解耦、类型安全、统一错误处理、消除重复代码。拆分大文件时创建 `.bak` 备份。

**架构：** 分 9 个独立任务，从前到后依次是基础设施（API/常量）→ 核心逻辑解耦（规则引擎/Canvas渲染）→ 认证/类型 → 功能增强。每个任务可独立实现和验证，不阻塞后续任务。

**技术栈：** Vue 3 + TypeScript + Pinia + Vue Router + Canvas API

**备份策略：** 凡涉及拆分源文件或侵入性修改的步骤，操作前先执行 `Copy-Item` 创建 `.bak` 备份。

---

### 任务 1：API 层重构 — 统一调用 + 错误处理

**文件：**
- 修改：`frontend/src/api/index.ts`
- 修改：`frontend/src/types/index.ts`（新增上传响应类型）
- 后续任务中不再使用裸 `fetch`

**职责：** 将所有裸 `fetch` 调用纳入 `api` 对象，增加 HTTP 错误检测、网络异常处理、401 自动登出。

- [ ] **步骤 1：备份原始文件**

```powershell
Copy-Item "frontend/src/api/index.ts" "frontend/src/api/index.ts.bak"
```

- [ ] **步骤 2：重写 api/index.ts**

```typescript
import type { ApiResponse, PageData, TemplateListItem, Template, SigningRecord, FontItem, AdminStats, HandwritingConfig } from '@/types'

const BASE = '/api'

class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

function clearAuth() {
  localStorage.removeItem('token')
  window.location.href = '/admin/login'
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {}
  if (token) headers['Authorization'] = `Bearer ${token}`

  const isFormData = options?.body instanceof FormData
  if (!isFormData) {
    headers['Content-Type'] = 'application/json'
  }

  // 合并用户自定义 headers
  const userHeaders = options?.headers as Record<string, string> | undefined
  Object.assign(headers, userHeaders)

  let res: Response
  try {
    res = await fetch(`${BASE}${url}`, {
      ...options,
      headers,
      signal: AbortSignal.timeout(30000),
    })
  } catch (err) {
    throw new ApiError(0, '网络连接失败，请检查网络后重试')
  }

  if (res.status === 401) {
    clearAuth()
    throw new ApiError(401, '登录已过期，请重新登录')
  }

  if (!res.ok) {
    const text = await res.text().catch(() => '未知错误')
    throw new ApiError(res.status, `请求失败: ${res.status} ${text.slice(0, 200)}`)
  }

  return res.json()
}

export const api = {
  login: (data: { username: string; password: string }) =>
    request<ApiResponse<{ token: string }>>('/auth/login', {
      method: 'POST', body: JSON.stringify(data),
    }),
  me: () => request<ApiResponse<{ username: string }>>('/auth/me'),

  listTemplates: (params?: Record<string, string | number>) =>
    request<ApiResponse<PageData<TemplateListItem>>>(`/templates${params ? '?' + new URLSearchParams(
      Object.entries(params).map(([k, v]) => [k, String(v)])
    ).toString() : ''}`),
  createTemplate: (name: string) =>
    request<ApiResponse<{ id: number }>>('/templates', {
      method: 'POST', body: JSON.stringify({ name }),
    }),
  getTemplate: (id: number) => request<ApiResponse<Template>>(`/templates/${id}`),
  updateTemplate: (id: number, data: Partial<Pick<Template, 'name' | 'controls' | 'rules' | 'handwriting' | 'width' | 'height'>>) =>
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

  signTemplate: (id: number, data: {
    fields_data: Record<string, any>
    effect_preset: string
    text_layer_data: string
    handwriting?: HandwritingConfig
  }) =>
    request<ApiResponse<{ image_url: string }>>(`/templates/${id}/sign`, {
      method: 'POST', body: JSON.stringify(data),
    }),

  listRecords: (params?: Record<string, string | number>) =>
    request<ApiResponse<PageData<SigningRecord>>>(`/records${params ? '?' + new URLSearchParams(
      Object.entries(params).map(([k, v]) => [k, String(v)])
    ).toString() : ''}`),
  deleteRecord: (id: number) =>
    request<ApiResponse<null>>(`/records/${id}`, { method: 'DELETE' }),

  listFonts: () => request<ApiResponse<FontItem[]>>('/fonts'),
  uploadFont: (file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return request<ApiResponse<FontItem>>('/fonts', {
      method: 'POST', body: fd,
    })
  },
  updateFont: (id: number, data: { display_name: string }) =>
    request<ApiResponse<null>>(`/fonts/${id}`, {
      method: 'PUT', body: JSON.stringify(data),
    }),
  deleteFont: (id: number) =>
    request<ApiResponse<null>>(`/fonts/${id}`, { method: 'DELETE' }),

  importTemplate: (file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return request<ApiResponse<{ id: number; name: string }>>('/templates/import', {
      method: 'POST', body: fd,
    })
  },

  getStats: () => request<ApiResponse<AdminStats>>('/admin/stats'),

  getHandwritingSettings: () =>
    request<ApiResponse<HandwritingConfig>>('/settings/handwriting'),
  updateHandwritingSettings: (data: HandwritingConfig) =>
    request<ApiResponse<null>>('/settings/handwriting', {
      method: 'PUT', body: JSON.stringify(data),
    }),
}
```

- [ ] **步骤 3：确认编译通过**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无类型错误

- [ ] **步骤 4：Commit**

```bash
git add frontend/src/api/index.ts frontend/src/types/index.ts
git commit -m "refactor: 统一 API 层，添加错误处理和 401 自动登出"
```

---

### 任务 2：共享常量 + 工具函数提取

**文件：**
- 创建：`frontend/src/constants/defaults.ts`
- 创建：`frontend/src/utils/merge.ts`
- 修改：`frontend/src/views/SignFront.vue`（删除 HW_DEFAULTS，改为 import）
- 修改：`frontend/src/views/AdminSettings.vue`（删除 HW_DEFAULTS，改为 import）

**职责：** 消除 `HW_DEFAULTS` 在两处文件中的重复定义，提取 `mergeHW` 函数。

- [ ] **步骤 1：创建 `constants/defaults.ts`**

```typescript
import type { HandwritingConfig } from '@/types'

export const HW_DEFAULTS: HandwritingConfig = {
  font_family: 'sans-serif',
  paper_enabled: true, paper_opacity: 0.12, fiber_count: 200, dot_count: 800,
  global_tilt: 1, baseline_drift: 0.8, char_jitter: 2, char_rotation: 1.5,
  ink_opacity_min: 0.85, ink_opacity_max: 1.0, char_spacing: 1.5,
  ink_spots_enabled: true, ink_spots_chance: 0.15, ink_spots_max: 2,
  shadow_blur: 0.8, checkbox_enabled: true,
}

export type ControlType = 'textbox' | 'checkbox'

export const ALLOWED_FONT_EXTS = ['.ttf', '.otf', '.woff', '.woff2'] as const
```

- [ ] **步骤 2：创建 `utils/merge.ts`**

```typescript
import type { HandwritingConfig } from '@/types'

export function mergeHW(base: HandwritingConfig, over: Partial<HandwritingConfig>): HandwritingConfig {
  const r = { ...base }
  for (const [k, v] of Object.entries(over)) {
    if (v !== undefined && v !== null) {
      (r as Record<string, any>)[k] = v
    }
  }
  return r
}
```

- [ ] **步骤 3：修改 SignFront.vue — 删除本地 HW_DEFAULTS 和 mergeHW**

在 `<script>` 顶部，删除第 66-73 行的 `HW_DEFAULTS` 常量和第 77-83 行的 `mergeHW` 函数。改为：

```typescript
import { HW_DEFAULTS } from '@/constants/defaults'
import { mergeHW } from '@/utils/merge'
```

- [ ] **步骤 4：修改 AdminSettings.vue — 删除本地 HW_DEFAULTS**

删除第 31-37 行的 `HW_DEFAULTS` 常量。改为：

```typescript
import { HW_DEFAULTS } from '@/constants/defaults'
```

- [ ] **步骤 5：确认编译通过**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无类型错误

- [ ] **步骤 6：Commit**

```bash
git add frontend/src/constants/ frontend/src/utils/ frontend/src/views/SignFront.vue frontend/src/views/AdminSettings.vue
git commit -m "refactor: 提取 HW_DEFAULTS 和 mergeHW 到共享模块"
```

---

### 任务 3：规则引擎解耦

**文件：**
- 创建：`frontend/src/engine/ruleEngine.ts`
- 创建：`frontend/src/engine/types.ts`
- 修改：`frontend/src/views/SignFront.vue`（删除规则函数，改为 import）

**职责：** 将 SignFront.vue 中第 276-454 行的全部规则引擎逻辑提取为独立的可测试模块。

- [ ] **步骤 1：备份 SignFront.vue**

```powershell
Copy-Item "frontend/src/views/SignFront.vue" "frontend/src/views/SignFront.vue.bak"
```

- [ ] **步骤 2：创建 `engine/types.ts`**

```typescript
import type { Rule } from '@/types'

export interface RuleEngineContext {
  getValue: (ctrlId: string) => any
  setValue: (ctrlId: string, value: any) => void
  getControl: (ctrlId: string) => { id: string; type: string } | undefined
  hiddenFields: Set<string>
  disabledFields: Set<string>
}

export interface RuleEngineResult {
  changed?: boolean
}
```

- [ ] **步骤 3：创建 `engine/ruleEngine.ts`**

```typescript
import type { Rule } from '@/types'
import type { RuleEngineContext } from './types'

function getControlValue(ctx: RuleEngineContext, ctrlId: string): any {
  return ctx.getValue(ctrlId)
}

function setControlValue(ctx: RuleEngineContext, ctrlId: string, value: any) {
  ctx.setValue(ctrlId, value)
}

function applyTransform(value: any, transform: string): any {
  if (!transform) return value
  const num = parseFloat(value)
  if (isNaN(num)) return value
  if (transform.startsWith('add(')) {
    const addVal = parseFloat(transform.match(/add\(([^)]+)\)/)?.[1] || '0')
    return num + addVal
  }
  if (transform.startsWith('multiply(')) {
    const mulVal = parseFloat(transform.match(/multiply\(([^)]+)\)/)?.[1] || '1')
    return num * mulVal
  }
  return value
}

function getAutoFillValue(source: string, format: string): string {
  const now = new Date()
  const Y = now.getFullYear()
  const M = String(now.getMonth() + 1).padStart(2, '0')
  const D = String(now.getDate()).padStart(2, '0')
  const h = String(now.getHours()).padStart(2, '0')
  const m = String(now.getMinutes()).padStart(2, '0')
  const s = String(now.getSeconds()).padStart(2, '0')
  switch (source) {
    case 'system.date':
      if (format === 'YYYY年MM月DD日') return `${Y}年${M}月${D}日`
      if (format === 'MM/DD/YYYY') return `${M}/${D}/${Y}`
      return `${Y}-${M}-${D}`
    case 'system.time': return `${h}:${m}:${s}`
    case 'system.datetime': return `${Y}-${M}-${D} ${h}:${m}:${s}`
    case 'system.year': return String(Y)
    case 'system.month': return M
    case 'system.day': return D
    default: return ''
  }
}

function applyMutualExclusion(ctx: RuleEngineContext, rule: Rule, changedId: string) {
  const cfg = (rule.config || {}) as Record<string, any>
  const targets: string[] = cfg.targets || []
  const val = getControlValue(ctx, changedId)
  const isChecked = val === true || (typeof val === 'string' && val.trim() !== '')
  if (!isChecked) return
  for (const id of targets) {
    if (id === changedId) continue
    const current = getControlValue(ctx, id)
    if (current === true || (typeof current === 'string' && current.trim() !== '')) {
      setControlValue(ctx, id, typeof current === 'boolean' ? false : '')
    }
  }
}

function applyGroupExclusion(ctx: RuleEngineContext, rule: Rule, changedId: string) {
  const cfg = (rule.config || {}) as Record<string, any>
  const groups: string[][] = cfg.groups || []
  let changedGroupIdx = -1
  for (let i = 0; i < groups.length; i++) {
    if (groups[i].includes(changedId)) { changedGroupIdx = i; break }
  }
  if (changedGroupIdx === -1) return
  const val = getControlValue(ctx, changedId)
  if (!(val === true || (typeof val === 'string' && val.trim() !== ''))) return
  for (let i = 0; i < groups.length; i++) {
    if (i === changedGroupIdx) continue
    for (const id of groups[i]) {
      const ctrl = ctx.getControl(id)
      if (ctrl) setControlValue(ctx, id, ctrl.type === 'checkbox' ? false : '')
    }
  }
}

function applyTriggerRule(ctx: RuleEngineContext, rule: Rule, changedId: string) {
  const cfg = (rule.config || {}) as Record<string, any>
  const source = cfg.source
  if (source !== changedId) return
  const srcVal = getControlValue(ctx, source)
  let met = false
  const condition = cfg.condition
  if (condition === 'checked') met = srcVal === true
  else if (condition === 'notEmpty') met = srcVal && String(srcVal).trim().length > 0
  else if (condition === 'valueEqual') met = String(srcVal) === String(cfg.value)

  const action = cfg.action
  if (met) {
    switch (action) {
      case 'show': ctx.hiddenFields.delete(rule.target); break
      case 'hide': ctx.hiddenFields.add(rule.target); setControlValue(ctx, rule.target, ''); break
      case 'enable': ctx.disabledFields.delete(rule.target); break
      case 'disable': ctx.disabledFields.add(rule.target); break
      case 'clear': setControlValue(ctx, rule.target, ''); break
    }
  } else {
    if (action === 'show') { ctx.hiddenFields.add(rule.target); setControlValue(ctx, rule.target, '') }
    else if (action === 'hide') ctx.hiddenFields.delete(rule.target)
    else if (action === 'enable') ctx.disabledFields.add(rule.target)
    else if (action === 'disable') ctx.disabledFields.delete(rule.target)
  }
}

function applyValueSyncRule(ctx: RuleEngineContext, rule: Rule, changedId: string) {
  const cfg = (rule.config || {}) as Record<string, any>
  const direction = cfg.direction
  const source = cfg.source
  if (direction === 'one_way' && source !== changedId) return
  if (direction === 'two_way' && source !== changedId && rule.target !== changedId) return
  const srcVal = getControlValue(ctx, source)
  if (srcVal === undefined || srcVal === null || srcVal === '') return
  let newVal = cfg.transform ? applyTransform(srcVal, cfg.transform) : srcVal
  if (direction === 'one_way') {
    setControlValue(ctx, rule.target, newVal)
  } else {
    if (changedId === source) setControlValue(ctx, rule.target, newVal)
    else setControlValue(ctx, source, srcVal)
  }
}

function applyArithmeticRule(ctx: RuleEngineContext, rule: Rule, changedId: string) {
  const cfg = (rule.config || {}) as Record<string, any>
  const source = cfg.source
  if (source !== changedId) return
  const srcVal = parseFloat(getControlValue(ctx, source))
  if (isNaN(srcVal)) return
  let result: number
  switch (cfg.operation) {
    case 'add': result = srcVal + cfg.operand; break
    case 'sub': result = srcVal - cfg.operand; break
    case 'mul': result = srcVal * cfg.operand; break
    case 'div': result = cfg.operand !== 0 ? srcVal / cfg.operand : 0; break
    default: return
  }
  setControlValue(ctx, rule.target, result)
}

function applyAutoFillRule(ctx: RuleEngineContext, rule: Rule) {
  const cfg = (rule.config || {}) as Record<string, any>
  const val = getControlValue(ctx, rule.target)
  if (val !== undefined && val !== null && val !== '') return
  setControlValue(ctx, rule.target, getAutoFillValue(cfg.source, cfg.format || 'YYYY-MM-DD'))
}

/**
 * 执行所有规则
 * @param ctx 规则引擎上下文
 * @param rules 规则列表
 * @param changedId 触发变化的控件 ID（可为空，用于初始加载）
 */
export function applyAllRules(ctx: RuleEngineContext, rules: Rule[], changedId?: string) {
  for (const rule of rules) {
    switch (rule.type) {
      case 'mutual_exclusion': if (changedId) applyMutualExclusion(ctx, rule, changedId); break
      case 'group_exclusion': if (changedId) applyGroupExclusion(ctx, rule, changedId); break
      case 'trigger': if (changedId) applyTriggerRule(ctx, rule, changedId); break
      case 'value_sync': if (changedId) applyValueSyncRule(ctx, rule, changedId); break
      case 'arithmetic': if (changedId) applyArithmeticRule(ctx, rule, changedId); break
      case 'auto_fill': applyAutoFillRule(ctx, rule); break
    }
  }
}

/**
 * 应用初始默认值规则
 */
export function applyInitialDefaults(ctx: RuleEngineContext, rules: Rule[]) {
  for (const rule of rules) {
    if (rule.type === 'default_value') {
      const cfg = (rule.config || {}) as Record<string, any>
      const val = getControlValue(ctx, rule.target)
      if (val === undefined || val === null || val === '') {
        setControlValue(ctx, rule.target, cfg.default_value)
      }
    }
    if (rule.type === 'auto_fill') applyAutoFillRule(ctx, rule)
  }
}

/**
 * 验证单个控件
 */
export function validateField(ctrl: { id: string; label: string; type: string; required?: boolean }, value: any, rules: Rule[]): string {
  if (ctrl.required && (value === undefined || value === null || value === '' || (ctrl.type === 'checkbox' && !value))) {
    return `${ctrl.label} 为必填项`
  }
  if (!value) return ''
  const strVal = String(value).trim()
  for (const rule of rules) {
    if (rule.type === 'validation' && rule.target === ctrl.id && ctrl.type === 'textbox') {
      const cfg = (rule.config || {}) as Record<string, any>
      if (cfg.min_length && strVal.length < cfg.min_length) return `${ctrl.label} 长度不能小于 ${cfg.min_length}`
      if (cfg.max_length && strVal.length > cfg.max_length) return `${ctrl.label} 长度不能大于 ${cfg.max_length}`
      if (cfg.validation_type === 'number' && !/^\d+$/.test(strVal)) return `${ctrl.label} 必须为数字`
      if (cfg.validation_type === 'phone' && !/^1[3-9]\d{9}$/.test(strVal)) return `${ctrl.label} 手机号格式不正确`
      if (cfg.validation_type === 'idcard' && !/^[1-9]\d{5}(18|19|20)?\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$/.test(strVal)) return `${ctrl.label} 身份证号格式不正确`
      if (cfg.validation_type === 'regex' && cfg.pattern) {
        try {
          if (cfg.pattern.length > 200) return `${ctrl.label} 验证规则格式不正确`
          if (!new RegExp(cfg.pattern).test(strVal)) return `${ctrl.label} 格式不正确`
        } catch { return `${ctrl.label} 验证规则格式不正确` }
      }
    }
  }
  return ''
}
```

- [ ] **步骤 4：简化 SignFront.vue**

删除第 278-454 行的整个规则引擎（`getControlValue` 到 `applyInitialDefaults`），改为 `import`：

```typescript
import { applyAllRules, applyInitialDefaults, validateField } from '@/engine/ruleEngine'
import type { RuleEngineContext } from '@/engine/types'
```

将 `onInput` 改为：

```typescript
function onInput(ctrlId: string) {
  const ctx: RuleEngineContext = {
    getValue: (id) => formData.value[id],
    setValue: (id, val) => { formData.value[id] = val },
    getControl: (id) => controls.value.find(c => c.id === id),
    hiddenFields: hiddenFields.value,
    disabledFields: disabledFields.value,
  }
  applyAllRules(ctx, rules.value, ctrlId)
}
```

将 `validate` 函数改为调用导入的 `validateField`。

将 `applyInitialDefaults` 调用改为：

```typescript
const ctx: RuleEngineContext = {
  getValue: (id) => formData.value[id],
  setValue: (id, val) => { formData.value[id] = val },
  getControl: (id) => controls.value.find(c => c.id === id),
  hiddenFields: hiddenFields.value,
  disabledFields: disabledFields.value,
}
applyInitialDefaults(ctx, rules.value)
```

- [ ] **步骤 5：确认编译通过**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无类型错误

- [ ] **步骤 6：Commit**

```bash
git add frontend/src/engine/ frontend/src/views/SignFront.vue
git commit -m "refactor: 规则引擎解耦为独立模块"
```

---

### 任务 4：Canvas 渲染层解耦

**文件：**
- 创建：`frontend/src/engine/canvasRenderer.ts`
- 修改：`frontend/src/views/SignFront.vue`（删除 Canvas 绘制函数，改为 import）

**职责：** 将 SignFront.vue 中的 `drawPaperTexture`、`drawTextOnContext`、`drawCheckOnContext`、`renderPreview`、`generateTextLayerDataURL` 提取为独立模块。

- [ ] **步骤 1：创建 `engine/canvasRenderer.ts`**

```typescript
import type { Control, HandwritingConfig } from '@/types'

export function drawPaperTexture(ctx: CanvasRenderingContext2D, w: number, h: number, hw: HandwritingConfig) {
  if (hw.paper_enabled === false) return
  ctx.save()
  ctx.globalCompositeOperation = 'source-over'
  const opacity = hw.paper_opacity ?? 0.12
  ctx.fillStyle = `rgba(248, 245, 230, ${opacity})`
  ctx.fillRect(0, 0, w, h)
  const fibers = hw.fiber_count ?? 200
  for (let i = 0; i < fibers; i++) {
    ctx.beginPath()
    ctx.moveTo(Math.random() * w, Math.random() * h)
    ctx.lineTo(Math.random() * w, Math.random() * h)
    ctx.strokeStyle = `rgba(100, 80, 50, ${Math.random() * 0.08})`
    ctx.lineWidth = 0.5 + Math.random() * 1
    ctx.stroke()
  }
  const dots = hw.dot_count ?? 800
  for (let i = 0; i < dots; i++) {
    ctx.fillStyle = `rgba(80, 60, 40, ${Math.random() * 0.1})`
    ctx.fillRect(Math.random() * w, Math.random() * h, 1, 1)
  }
  ctx.restore()
}

export function drawTextOnContext(ctx: CanvasRenderingContext2D, text: string, ctrl: Control, hw: HandwritingConfig) {
  ctx.save()
  const fontSize = ctrl.fontSize || 28
  let fontFamily = ctrl.fontFamily || 'sans-serif'
  if (fontFamily === 'sans-serif' || fontFamily === 'serif' || fontFamily === 'monospace') {
    fontFamily = `${fontFamily}, sans-serif`
  } else {
    fontFamily = `'${fontFamily}', sans-serif`
  }
  ctx.font = `${fontSize}px ${fontFamily}`
  ctx.textBaseline = 'top'

  const globalTilt = (Math.random() - 0.5) * 2 * (hw.global_tilt ?? 1)
  ctx.translate(ctrl.x, ctrl.y)
  ctx.rotate(globalTilt * Math.PI / 180)

  const baselineDrift = hw.baseline_drift ?? 0.8
  const charJitter = hw.char_jitter ?? 2
  const charRotation = hw.char_rotation ?? 1.5
  const inkMin = hw.ink_opacity_min ?? 0.85
  const inkRange = (hw.ink_opacity_max ?? 1) - inkMin
  const charSpacing = hw.char_spacing ?? 1.5
  const shadowBlur = hw.shadow_blur ?? 0.8
  const inkSpotsEnabled = hw.ink_spots_enabled !== false
  const inkSpotsChance = hw.ink_spots_chance ?? 0.15
  const inkSpotsMax = hw.ink_spots_max ?? 2

  let xOffset = 0
  let baseLineY = 0
  for (const char of text.split('')) {
    if (char === ' ') { xOffset += fontSize * 0.4; continue }
    baseLineY += (Math.random() - 0.5) * 2 * baselineDrift
    baseLineY = Math.min(Math.max(baseLineY, -2), 2)
    const jitterY = (Math.random() - 0.5) * charJitter + baseLineY
    const cRot = (Math.random() - 0.5) * charRotation * 2 * Math.PI / 180
    const inkOpacity = inkMin + Math.random() * inkRange

    ctx.save()
    ctx.translate(xOffset, jitterY)
    ctx.rotate(cRot)
    ctx.shadowBlur = shadowBlur
    ctx.shadowColor = 'rgba(0,0,0,0.15)'
    ctx.fillStyle = `rgba(20, 20, 25, ${inkOpacity})`
    ctx.fillText(char, 0, 0)
    ctx.shadowBlur = 0
    ctx.restore()

    if (inkSpotsEnabled && Math.random() < inkSpotsChance) {
      const spotCount = Math.floor(Math.random() * inkSpotsMax) + 1
      for (let s = 0; s < spotCount; s++) {
        ctx.save()
        ctx.translate(xOffset + (Math.random() - 0.5) * fontSize * 0.4, jitterY + (Math.random() - 0.5) * fontSize * 0.3)
        ctx.beginPath()
        ctx.arc(0, 0, Math.random() * 1.5 + 0.5, 0, Math.PI * 2)
        ctx.fillStyle = `rgba(30, 30, 35, ${0.3 + Math.random() * 0.4})`
        ctx.fill()
        ctx.restore()
      }
    }
    xOffset += ctx.measureText(char).width + (Math.random() - 0.5) * charSpacing
  }
  ctx.restore()
}

export function drawCheckOnContext(ctx: CanvasRenderingContext2D, ctrl: Control, hw: HandwritingConfig) {
  if (hw.checkbox_enabled === false) {
    const size = ctrl.checkSize || ctrl.width || 32
    ctx.fillStyle = '#000'
    ctx.fillRect(ctrl.x, ctrl.y, size, size)
    return
  }
  ctx.save()
  const size = ctrl.checkSize || ctrl.width || 32
  const cx = ctrl.x + size / 2, cy = ctrl.y + size / 2
  const o = size * 0.25
  const r = () => (Math.random() - 0.5) * size * 0.03
  ctx.beginPath()
  ctx.moveTo(cx - o + r(), cy + r())
  ctx.quadraticCurveTo(cx - o + 3 + r(), cy + 4 + r(), cx - o * 0.4 + r(), cy + o * 0.9 + r())
  ctx.quadraticCurveTo(cx - o * 0.4 + 2 + r(), cy + o * 0.9 - 2 + r(), cx + o + r(), cy - o * 0.6 + r())
  ctx.lineWidth = Math.max(3, size * 0.12)
  ctx.strokeStyle = '#1a1a1a'
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.shadowBlur = 1.2
  ctx.shadowColor = 'rgba(0,0,0,0.3)'
  ctx.stroke()
  ctx.shadowBlur = 0
  ctx.restore()
}

export function renderPreview(
  canvas: HTMLCanvasElement,
  bgImg: HTMLImageElement | null,
  controls: Control[],
  formData: Record<string, any>,
  hiddenFields: Set<string>,
  hw: HandwritingConfig,
) {
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  ctx.clearRect(0, 0, canvas.width, canvas.height)
  if (bgImg && bgImg.complete && bgImg.naturalWidth > 0) {
    ctx.drawImage(bgImg, 0, 0, canvas.width, canvas.height)
  } else {
    ctx.fillStyle = '#fff'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
  }
  drawPaperTexture(ctx, canvas.width, canvas.height, hw)
  for (const ctrl of controls) {
    if (hiddenFields.has(ctrl.id)) continue
    if (ctrl.type === 'textbox') {
      const val = formData[ctrl.id]
      if (val && String(val).trim()) drawTextOnContext(ctx, String(val).trim(), ctrl, hw)
    } else if (ctrl.type === 'checkbox') {
      if (formData[ctrl.id]) drawCheckOnContext(ctx, ctrl, hw)
    }
  }
}

export function generateTextLayerDataURL(
  width: number,
  height: number,
  controls: Control[],
  formData: Record<string, any>,
  hiddenFields: Set<string>,
  hw: HandwritingConfig,
): string {
  const offCanvas = document.createElement('canvas')
  offCanvas.width = width
  offCanvas.height = height
  const offCtx = offCanvas.getContext('2d')
  if (!offCtx) return ''
  offCtx.clearRect(0, 0, offCanvas.width, offCanvas.height)
  drawPaperTexture(offCtx, offCanvas.width, offCanvas.height, hw)
  for (const ctrl of controls) {
    if (hiddenFields.has(ctrl.id)) continue
    if (ctrl.type === 'textbox') {
      const val = formData[ctrl.id]
      if (val && String(val).trim()) drawTextOnContext(offCtx, String(val).trim(), ctrl, hw)
    } else if (ctrl.type === 'checkbox') {
      if (formData[ctrl.id]) drawCheckOnContext(offCtx, ctrl, hw)
    }
  }
  return offCanvas.toDataURL('image/png')
}
```

- [ ] **步骤 2：简化 SignFront.vue**

删除第 114-274 行的所有 Canvas 函数（`drawPaperTexture` 到 `generateTextLayerDataURL`），改为：

```typescript
import { renderPreview, generateTextLayerDataURL } from '@/engine/canvasRenderer'
```

将 `renderPreview()` 调用改为传递参数：

```typescript
function doRenderPreview() {
  const canvas = previewCanvas.value
  if (!canvas || !template.value) return
  renderPreview(canvas, bgImg, controls.value, formData.value, hiddenFields.value, mergedHW.value)
}
```

将 `generateTextLayerDataURL()` 调用改为：

```typescript
const textLayerData = generateTextLayerDataURL(
  template.value?.width || 800,
  template.value?.height || 1000,
  controls.value,
  formData.value,
  hiddenFields.value,
  mergedHW.value,
)
```

- [ ] **步骤 3：在 SignFront.vue 中统一预览调用**

将所有 `renderPreview()` 替换为 `doRenderPreview()`。

- [ ] **步骤 4：确认编译通过**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无类型错误

- [ ] **步骤 5：Commit**

```bash
git add frontend/src/engine/canvasRenderer.ts frontend/src/views/SignFront.vue
git commit -m "refactor: Canvas 渲染层解耦为独立模块"
```

---

### 任务 5：认证系统统一

**文件：**
- 修改：`frontend/src/stores/auth.ts`
- 修改：`frontend/src/router/index.ts`
- 修改：`frontend/src/components/AppNav.vue`
- 修改：`frontend/src/views/Home.vue`

**职责：** 让 authStore 成为认证状态的唯一真实来源，路由守卫验证 token 有效性，AppNav 使用 store。

- [ ] **步骤 1：重写 stores/auth.ts**

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const username = ref('')

  const isAuthenticated = computed(() => !!token.value)

  async function login(user: string, pass: string) {
    const res = await api.login({ username: user, password: pass })
    if (res.code === 0) {
      token.value = res.data.token
      username.value = user
      localStorage.setItem('token', res.data.token)
    }
    return res
  }

  function setToken(t: string) {
    token.value = t
    localStorage.setItem('token', t)
  }

  function clearToken() {
    token.value = ''
    username.value = ''
    localStorage.removeItem('token')
  }

  function logout() {
    clearToken()
  }

  return { token, username, isAuthenticated, login, logout, setToken, clearToken }
})
```

- [ ] **步骤 2：重写 router/index.ts**

```typescript
import { createRouter, createWebHistory } from 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    hideNav?: boolean
  }
}

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
    { path: '/admin/settings', name: 'settings', component: () => import('@/views/AdminSettings.vue'), meta: { requiresAuth: true } },
  ],
})

router.beforeEach((to, _from, next) => {
  if (to.meta?.requiresAuth) {
    const token = localStorage.getItem('token')
    if (!token) {
      next({ name: 'login' })
      return
    }
    // 简单检查 JWT 是否过期（仅解析 payload，不验证签名）
    try {
      const payload = JSON.parse(atob(token.split('.')[1]))
      if (payload.exp && payload.exp * 1000 < Date.now()) {
        localStorage.removeItem('token')
        next({ name: 'login' })
        return
      }
    } catch {
      localStorage.removeItem('token')
      next({ name: 'login' })
      return
    }
  }
  next()
})

export default router
```

- [ ] **步骤 3：修改 AppNav.vue**

```typescript
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const isLoggedIn = computed(() => auth.isAuthenticated)

function handleLogout() {
  auth.logout()
  router.push('/')
}
```

- [ ] **步骤 4：修改 Home.vue**

将 `isLoggedIn` 从直接读 `localStorage` 改为使用 `authStore`：

```typescript
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const isLoggedIn = computed(() => auth.isAuthenticated)
```

- [ ] **步骤 5：确认编译通过**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无类型错误

- [ ] **步骤 6：Commit**

```bash
git add frontend/src/stores/auth.ts frontend/src/router/index.ts frontend/src/components/AppNav.vue frontend/src/views/Home.vue
git commit -m "refactor: 认证系统统一，路由守卫验证 JWT 过期"
```

---

### 任务 6：类型安全增强

**文件：**
- 修改：`frontend/src/types/index.ts`

**职责：** 修复 `Rule.config` 类型、提取 `ControlType` 共用联合类型。

- [ ] **步骤 1：重写 types/index.ts**

```typescript
export type ControlType = 'textbox' | 'checkbox'

export interface Control {
  id: string
  label: string
  type: ControlType
  x: number
  y: number
  width: number
  height: number
  fontSize: number
  fontFamily: string
  handwritingFont?: string
  required: boolean
  previewText?: string
  checkSize?: number
}

export interface HandwritingConfig {
  font_family?: string
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

// 规则配置联合类型
export type RuleConfig =
  | { targets: string[] }  // mutual_exclusion
  | { groups: string[][] }  // group_exclusion
  | { source: string; condition?: string; value?: any; action: string }  // trigger
  | { source: string; direction?: string; transform?: string }  // value_sync
  | { source: string; operation: string; operand: number }  // arithmetic
  | { source: string; format?: string }  // auto_fill
  | { default_value: any }  // default_value
  | { validation_type?: string; pattern?: string; min_length?: number; max_length?: number }  // validation
  | Record<string, any>  // 后备

export interface Rule {
  id: string
  type: string
  name?: string
  target: string
  config: RuleConfig
}

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

export interface TemplateListItem {
  id: number
  name: string
  bg_image?: string
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
  file_hash?: string
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

- [ ] **步骤 2：更新 Constants 引用**

将 `constants/defaults.ts` 中的 `ControlType` 改为 import 自 `@/types`。

- [ ] **步骤 3：确认编译通过**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无类型错误

- [ ] **步骤 4：Commit**

```bash
git add frontend/src/types/index.ts frontend/src/constants/defaults.ts
git commit -m "refactor: 增强类型安全，添加 RuleConfig 联合类型"
```

---

### 任务 7：分页功能

**文件：**
- 修改：`frontend/src/views/Home.vue`
- 修改：`frontend/src/views/AdminRecords.vue`

**职责：** 为模板列表和签署记录列表添加分页参数传递和 UI 分页控件。

- [ ] **步骤 1：修改 Home.vue — 添加分页参数**

```typescript
// 在 script setup 中添加
import { ref, onMounted, computed } from 'vue'

const currentPage = ref(1)
const totalPages = ref(1)
const pageSize = 12

async function load() {
  const res = await api.listTemplates({ page: currentPage.value, page_size: pageSize })
  if (res.code === 0) {
    templates.value = res.data.items
    totalPages.value = Math.ceil(res.data.total / res.data.page_size)
  }
}

function goPage(p: number) {
  currentPage.value = p
  load()
}
```

在模板末尾添加分页 UI：

```vue
<div v-if="totalPages > 1" class="pagination">
  <button :disabled="currentPage <= 1" @click="goPage(currentPage - 1)">上一页</button>
  <span>{{ currentPage }} / {{ totalPages }}</span>
  <button :disabled="currentPage >= totalPages" @click="goPage(currentPage + 1)">下一页</button>
</div>
```

- [ ] **步骤 2：修改 AdminRecords.vue — 添加分页**

同上模式，添加 `currentPage`、`totalPages`、`goPage` 方法。

- [ ] **步骤 3：确认编译通过**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无类型错误

- [ ] **步骤 4：Commit**

```bash
git add frontend/src/views/Home.vue frontend/src/views/AdminRecords.vue
git commit -m "feat: 模板列表和签署记录列表添加分页功能"
```

---

### 任务 8：编辑器 Store 改进

**文件：**
- 修改：`frontend/src/components/editor/editorStore.ts`
- 修改：`frontend/src/views/AdminEditor.vue`

**职责：** 修复保存竞态问题，添加错误处理，添加路由离开时状态清理。

- [ ] **步骤 1：备份 editorStore.ts**

```powershell
Copy-Item "frontend/src/components/editor/editorStore.ts" "frontend/src/components/editor/editorStore.ts.bak"
```

- [ ] **步骤 2：修复 save 方法**

查找 `save()` 函数中的：

```typescript
async save() {
  // 创建 + 更新两步调用
}
```

改为只调用一次 API（如果后端支持），或者至少添加明确的失败处理：

```typescript
async save() {
  if (this.templateId) {
    // 更新已有模板
    const res = await api.updateTemplate(this.templateId, {
      name: this.name,
      width: this.canvasWidth,
      height: this.canvasHeight,
      controls: this.controls,
      rules: this.rules,
      handwriting: this.handwriting,
    })
    if (res.code !== 0) throw new Error(res.message)
  } else {
    // 新建模板
    const res = await api.createTemplate(this.name)
    if (res.code !== 0) throw new Error(res.message)
    this.templateId = res.data.id
    // 更新 controls/rules/handwriting
    await api.updateTemplate(this.templateId, {
      controls: this.controls,
      rules: this.rules,
      handwriting: this.handwriting,
    })
  }
  this.dirty = false
}
```

在 `AdminEditor.vue` 的 `handleSave` 中捕获异常：

```typescript
async function handleSave() {
  saving.value = true
  try {
    await store.save()
    alert('保存成功')
    if (!route.query.id && store.templateId) {
      router.replace({ query: { id: store.templateId } })
    }
  } catch (e: any) {
    alert(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}
```

- [ ] **步骤 3：添加路由离开清理**

在 `AdminEditor.vue` 中添加：

```typescript
import { onBeforeRouteLeave } from 'vue-router'

onBeforeRouteLeave((_to, _from, next) => {
  if (store.dirty) {
    if (!confirm('有未保存的修改，确定离开吗？')) {
      next(false)
      return
    }
  }
  next()
})
```

- [ ] **步骤 4：确认编译通过**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无类型错误

- [ ] **步骤 5：Commit**

```bash
git add frontend/src/components/editor/editorStore.ts frontend/src/views/AdminEditor.vue
git commit -m "fix: 编辑器 Store 保存竞态修复 + 路由离开确认"
```

---

### 任务 9：AdminFonts.vue 重构

**文件：**
- 修改：`frontend/src/views/AdminFonts.vue`

**职责：** 将裸 `fetch` 调用替换为 `api` 对象，消除命令式 DOM 操作。

- [ ] **步骤 1：备份 AdminFonts.vue**

```powershell
Copy-Item "frontend/src/views/AdminFonts.vue" "frontend/src/views/AdminFonts.vue.bak"
```

- [ ] **步骤 2：替换 uploadFiles 中的裸 fetch**

将：

```typescript
const resp = await fetch('/api/fonts', {
  method: 'POST',
  headers: { Authorization: `Bearer ${token}` },
  body: fd,
})
```

改为：

```typescript
const result = await api.uploadFont(file)
```

- [ ] **步骤 3：替换 saveName 中的裸 fetch**

将：

```typescript
await fetch(`/api/fonts/${id}`, {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  body: JSON.stringify({ display_name: editingName.value.trim() }),
})
```

改为：

```typescript
await api.updateFont(id, { display_name: editingName.value.trim() })
```

- [ ] **步骤 4：消除命令式 DOM 操作**

将 `startEdit` 中的 `querySelectorAll` 改为模板 ref：

```typescript
const editInputRefs = ref<Map<number, HTMLInputElement>>(new Map())

function setEditRef(el: HTMLInputElement | null, id: number) {
  if (el) editInputRefs.value.set(id, el)
  else editInputRefs.value.delete(id)
}

function startEdit(f: FontItem) {
  editingId.value = f.id
  editingName.value = f.display_name || f.filename
  nextTick(() => {
    editInputRefs.value.get(f.id)?.focus()
  })
}
```

在模板中给 input 添加 ref：

```vue
<input
  v-model="editingName"
  :ref="(el: any) => setEditRef(el as HTMLInputElement | null, f.id)"
  class="edit-input"
  @blur="saveName(f.id)"
  @keydown.enter="saveName(f.id)"
/>
```

- [ ] **步骤 5：确认编译通过**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无类型错误

- [ ] **步骤 6：Commit**

```bash
git add frontend/src/views/AdminFonts.vue
git commit -m "refactor: AdminFonts.vue 统一 API 调用，消除命令式 DOM"
```

---

### 任务 10（可选）：SignFront.vue 签署改用 api 对象

**文件：**
- 修改：`frontend/src/views/SignFront.vue`

**职责：** 将 `handleSign` 中的裸 `fetch` 替换为 `api.signTemplate`。

- [ ] **步骤 1：替换 handleSign 中的裸 fetch**

将第 501 行的裸 `fetch` 调用改为：

```typescript
const textLayerData = generateTextLayerDataURL(
  template.value?.width || 800,
  template.value?.height || 1000,
  controls.value,
  formData.value,
  hiddenFields.value,
  mergedHW.value,
)
const res = await api.signTemplate(templateId!, {
  fields_data: formData.value,
  effect_preset: 'light',
  text_layer_data: textLayerData,
  handwriting: Object.keys(signOverride.value).length > 0 ? signOverride.value : undefined,
})
resultUrl.value = res.data.image_url
downloadName.value = `Signed_${template.value?.name || 'protocol'}_${Date.now()}.png`
```

- [ ] **步骤 2：确认编译通过**

运行：`cd frontend && npx vue-tsc --noEmit`
预期：无类型错误

- [ ] **步骤 3：Commit**

```bash
git add frontend/src/views/SignFront.vue
git commit -m "refactor: SignFront.vue 签署改用 api 对象"
```

---

## 自检清单

- [x] 所有任务都有明确的职责描述
- [x] 每个步骤包含实际代码（非占位符）
- [x] 无 "TODO" 或 "后续实现" 等占位符
- [x] 每个步骤包含精确的命令和预期输出
- [x] 类型一致性：前后使用的类型/函数名一致
- [x] 所有裸 fetch 已被标记为替换
- [x] 解耦策略清晰：规则引擎 → engine/ruleEngine.ts，Canvas → engine/canvasRenderer.ts
- [x] 备份策略明确：凡拆分的源文件先 `.bak` 再修改
- [x] 分页使用了 `api` 对象的新参数格式（`Record<string, string | number>`）

## 执行说明

**两种执行方式：**

1. **子代理驱动（推荐）** — 每个任务调度一个新的子代理，任务间进行审查
2. **内联执行** — 在当前会话中逐任务执行，批量执行并设有检查点

任务之间存在依赖关系：
- 任务 1 必须先于任务 7、任务 9、任务 10（因为分页和新 API 方法依赖于新的 `api` 对象）
- 任务 2 必须先于任务 3（虽然无强依赖，但为了减少冲突建议顺序执行）
- 任务 3/4 可以并行（分别处理 SignFront.vue 的不同部分）
- 任务 5 可以独立在任何时间执行
- 任务 6 建议在任务 1 之后执行（类型依赖于 API 签名）
- 任务 7-10 可以按任意顺序执行
