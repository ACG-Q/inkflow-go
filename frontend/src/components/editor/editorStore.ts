import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api'
import type { Control, Rule, HandwritingConfig } from '@/types'

function snapToGrid(v: number, grid: number): number {
  return Math.round(v / grid) * grid
}

function genId(): string {
  return `ctrl_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`
}

function genRuleId(): string {
  return `rule_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`
}

export const useEditorStore = defineStore('editor', () => {
  const templateId = ref<number | null>(null)
  const name = ref('')
  const bgImage = ref('')
  const canvasWidth = ref(800)
  const canvasHeight = ref(1000)
  const controls = ref<Control[]>([])
  const rules = ref<Rule[]>([])
  const handwriting = ref<HandwritingConfig>({})
  const selectedId = ref<string | null>(null)
  const zoom = ref(1)
  const showGrid = ref(true)
  const snapSize = ref(10)
  const effectPreset = ref('none')
  const dirty = ref(false)
  const saving = ref(false)

  const selectedControl = computed(() =>
    controls.value.find(c => c.id === selectedId.value) || null
  )

  const zoomPercent = computed(() => Math.round(zoom.value * 100))

  function selectControl(id: string | null) {
    selectedId.value = id
  }

  function addControl(type: 'textbox' | 'checkbox') {
    const id = genId()
    const cx = 100 + Math.floor((controls.value.length % 3) * 60)
    const cy = 100 + Math.floor((controls.value.length / 3) * 60)
    const ctrl: Control = type === 'textbox'
      ? { id, type: 'textbox', label: '文字', x: cx, y: cy, width: 200, height: 40, fontSize: 20, fontFamily: 'sans-serif', handwritingFont: 'default', required: false, previewText: '请输入' }
      : { id, type: 'checkbox', label: '选项', x: cx, y: cy, width: 24, height: 24, fontSize: 14, fontFamily: 'sans-serif', handwritingFont: 'default', required: false, checkSize: 24 }
    controls.value.push(ctrl)
    selectedId.value = id
    dirty.value = true
  }

  function removeControl(id: string) {
    controls.value = controls.value.filter(c => c.id !== id)
    rules.value = rules.value.filter(r => {
      if (r.target === id) return false
      const cfg = r.config as Record<string, any>
      if (r.type === 'mutual_exclusion' && cfg.targets?.includes(id)) return false
      if (r.type === 'group_exclusion' && cfg.groups?.some((g: string[]) => g.includes(id))) return false
      if (r.type === 'trigger' && (cfg.source === id || r.target === id)) return false
      if (r.type === 'value_sync' && (cfg.source === id || r.target === id)) return false
      if (r.type === 'arithmetic' && (cfg.source === id || r.target === id)) return false
      return true
    })
    if (selectedId.value === id) selectedId.value = null
    dirty.value = true
  }

  function updateControl(id: string, patch: Partial<Control>) {
    const idx = controls.value.findIndex(c => c.id === id)
    if (idx !== -1) {
      controls.value[idx] = { ...controls.value[idx], ...patch }
      dirty.value = true
    }
  }

  function moveControl(id: string, dx: number, dy: number) {
    const ctrl = controls.value.find(c => c.id === id)
    if (ctrl) {
      ctrl.x = snapToGrid(Math.max(0, ctrl.x + dx), snapSize.value)
      ctrl.y = snapToGrid(Math.max(0, ctrl.y + dy), snapSize.value)
      dirty.value = true
    }
  }

  function resizeControl(id: string, handle: string, dx: number, dy: number) {
    const ctrl = controls.value.find(c => c.id === id)
    if (!ctrl) return
    const s = snapSize.value
    let { x, y, width, height } = ctrl
    switch (handle) {
      case 'tl': x = snapToGrid(x + dx, s); y = snapToGrid(y + dy, s); width = snapToGrid(width - dx, s); height = snapToGrid(height - dy, s); break
      case 'tr': y = snapToGrid(y + dy, s); width = snapToGrid(width + dx, s); height = snapToGrid(height - dy, s); break
      case 'bl': x = snapToGrid(x + dx, s); width = snapToGrid(width - dx, s); height = snapToGrid(height + dy, s); break
      case 'br': width = snapToGrid(width + dx, s); height = snapToGrid(height + dy, s); break
      case 'tm': y = snapToGrid(y + dy, s); height = snapToGrid(height - dy, s); break
      case 'bm': height = snapToGrid(height + dy, s); break
      case 'ml': x = snapToGrid(x + dx, s); width = snapToGrid(width - dx, s); break
      case 'mr': width = snapToGrid(width + dx, s); break
    }
    ctrl.width = Math.max(20, width)
    ctrl.height = Math.max(20, height)
    ctrl.x = x
    ctrl.y = y
    dirty.value = true
  }

  function addRule(rule: Omit<Rule, 'id'>) {
    rules.value.push({ ...rule, id: genRuleId() } as Rule)
    dirty.value = true
  }

  function updateRule(id: string, patch: Partial<Rule>) {
    const idx = rules.value.findIndex(r => r.id === id)
    if (idx !== -1) {
      rules.value[idx] = { ...rules.value[idx], ...patch } as Rule
      dirty.value = true
    }
  }

  function removeRule(id: string) {
    rules.value = rules.value.filter(r => r.id !== id)
    dirty.value = true
  }

  function updateHandwriting(patch: Partial<HandwritingConfig>) {
    handwriting.value = { ...handwriting.value, ...patch }
    dirty.value = true
  }

  async function loadTemplate(id: number) {
    const res = await api.getTemplate(id)
    if (res.code === 0) {
      templateId.value = res.data.id
      name.value = res.data.name
      bgImage.value = res.data.bg_image
      canvasWidth.value = res.data.width
      canvasHeight.value = res.data.height
      controls.value = res.data.controls || []
      rules.value = res.data.rules || []
      handwriting.value = res.data.handwriting || {}
      selectedId.value = null
      dirty.value = false
    }
  }

  async function save() {
    saving.value = true
    try {
      const body: Record<string, any> = {
        controls: controls.value,
        rules: rules.value.map(r => ({
          id: r.id,
          type: r.type,
          name: r.name || '',
          target: r.target,
          config: r.config,
        })),
        handwriting: handwriting.value,
        width: canvasWidth.value,
        height: canvasHeight.value,
      }
      if (templateId.value) {
        body.name = name.value
        await api.updateTemplate(templateId.value, body)
      } else {
        const res = await api.createTemplate(name.value)
        if (res.code === 0) {
          templateId.value = res.data.id
          await api.updateTemplate(templateId.value, body)
        }
      }
      dirty.value = false
    } finally {
      saving.value = false
    }
  }

  function zoomIn() { zoom.value = Math.min(2, +(zoom.value + 0.25).toFixed(2)) }
  function zoomOut() { zoom.value = Math.max(0.25, +(zoom.value - 0.25).toFixed(2)) }

  return {
    templateId, name, bgImage, canvasWidth, canvasHeight,
    controls, rules, handwriting, selectedId, selectedControl,
    zoom, zoomPercent, showGrid, snapSize, effectPreset, dirty, saving,
    selectControl, addControl, removeControl, updateControl, moveControl, resizeControl,
    addRule, updateRule, removeRule, updateHandwriting,
    loadTemplate, save, zoomIn, zoomOut,
  }
})
