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

export function validateField(
  ctrl: { id: string; label: string; type: string; required?: boolean },
  value: any,
  rules: Rule[],
): string {
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
