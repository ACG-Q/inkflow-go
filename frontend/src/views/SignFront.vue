<template>
  <div class="page sign-page">
    <div v-if="loading" class="loading-state">加载中...</div>
    <div v-else-if="error" class="error-state">{{ error }}</div>
    <div v-else class="sign-layout">
      <div class="form-section card">
        <h2>{{ template?.name }}</h2>
        <div v-for="control in controls" :key="control.id" class="field" :class="{ hidden: hiddenFields.has(control.id), disabled: disabledFields.has(control.id) }">
          <label>
            {{ control.label }}
            <span v-if="control.required" class="required">*</span>
          </label>
          <input
            v-if="control.type === 'textbox'"
            v-model="formData[control.id]"
            :placeholder="control.previewText || '请输入'"
            :disabled="disabledFields.has(control.id)"
            @input="onInput(control.id)"
          />
          <label v-else class="checkbox-label">
            <input type="checkbox" v-model="formData[control.id]" :disabled="disabledFields.has(control.id)" @change="onInput(control.id)" />
            {{ control.label }}
          </label>
          <div v-if="errors[control.id]" class="field-error">{{ errors[control.id] }}</div>
        </div>

        <div class="hw-section">
          <button class="hw-toggle-btn" @click="showHandwritingModal = true">
            <span>调整手写效果</span>
          </button>
          <HandwritingModal
            v-model:visible="showHandwritingModal"
            :model="signOverride"
            :show-actions="true"
            :fonts="fontList"
            title="手写效果临时配置"
            @update:model-value="signOverride = $event"
            @reset="signOverride = {}"
          />
        </div>

        <div class="form-actions">
          <button class="btn-primary" style="width:100%" @click="handleSign" :disabled="signing">
            {{ signing ? '签署中...' : '确认签署' }}
          </button>
        </div>
      </div>
      <div class="preview-section card">
        <canvas ref="previewCanvas" class="preview-canvas"></canvas>
        <div v-if="resultUrl" class="result-actions">
          <img :src="resultUrl" class="result-image" />
          <a :href="resultUrl" :download="downloadName" class="btn-primary btn-download">下载图片</a>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/api'
import type { Template, Control, Rule, HandwritingConfig, FontItem } from '@/types'
import { HW_DEFAULTS } from '@/constants/defaults'
import { mergeHW } from '@/utils/merge'
import { applyAllRules, applyInitialDefaults, validateField } from '@/engine/ruleEngine'
import type { RuleEngineContext } from '@/engine/types'
import { renderPreview, generateTextLayerDataURL } from '@/engine/canvasRenderer'
import HandwritingModal from '@/components/editor/HandwritingModal.vue'

const fontList = ref<FontItem[]>([])

const route = useRoute()
const previewCanvas = ref<HTMLCanvasElement>()
const template = ref<Template>()
const controls = ref<Control[]>([])
const rules = ref<Rule[]>([])
const formData = ref<Record<string, any>>({})
const errors = ref<Record<string, string>>({})
const hiddenFields = ref(new Set<string>())
const disabledFields = ref(new Set<string>())
const loading = ref(true)
const error = ref('')
const signing = ref(false)
const resultUrl = ref('')
const downloadName = ref('signed.png')
const showHandwritingModal = ref(false)
const signOverride = ref<HandwritingConfig>({})
const mergedHW = ref<HandwritingConfig>(HW_DEFAULTS)

let bgImg: HTMLImageElement | null = null
let templateId: number | null = null

function recomputeMerged() {
  const globalHW = HW_DEFAULTS
  const templateHW = template.value?.handwriting || {}
  const base = mergeHW(globalHW, templateHW)
  mergedHW.value = mergeHW(base, signOverride.value)
}

function doRenderPreview() {
  const canvas = previewCanvas.value
  if (!canvas) return
  renderPreview(canvas, bgImg, controls.value, formData.value, hiddenFields.value, mergedHW.value)
}

function createRuleContext(): RuleEngineContext {
  return {
    getValue: (id) => formData.value[id],
    setValue: (id, val) => { formData.value[id] = val },
    getControl: (id) => controls.value.find(c => c.id === id),
    hiddenFields: hiddenFields.value,
    disabledFields: disabledFields.value,
  }
}

function validate(): boolean {
  const errs: Record<string, string> = {}
  let valid = true
  for (const ctrl of controls.value) {
    if (hiddenFields.value.has(ctrl.id)) continue
    const val = formData.value[ctrl.id]
    const err = validateField(ctrl, val, rules.value)
    if (err) { errs[ctrl.id] = err; valid = false }
    else delete errs[ctrl.id]
  }
  errors.value = errs
  return valid
}

function onInput(ctrlId: string) {
  const ctx = createRuleContext()
  applyAllRules(ctx, rules.value, ctrlId)
  doRenderPreview()
}

async function handleSign() {
  if (!validate()) return
  signing.value = true
  try {
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
  } catch (e: any) {
    alert(e.message || '签署请求失败')
  } finally { signing.value = false }
}

onMounted(async () => {
  const fontsRes = await api.listFonts()
  if (fontsRes.code === 0) fontList.value = fontsRes.data as FontItem[]

  const id = route.query.template_id
  if (!id) { error.value = '缺少模板 ID'; loading.value = false; return }
  templateId = Number(id)
  const res = await api.getTemplate(templateId)
  if (res.code !== 0) { error.value = res.message; loading.value = false; return }
  template.value = res.data
  controls.value = res.data.controls || []
  rules.value = res.data.rules || []
  controls.value.forEach(c => { formData.value[c.id] = '' })
  loading.value = false
  await nextTick()
  const canvas = previewCanvas.value
  if (canvas && template.value) {
    canvas.width = template.value.width || 800
    canvas.height = template.value.height || 1000
  }
  recomputeMerged()
  const initCtx = createRuleContext()
  applyInitialDefaults(initCtx, rules.value)
  applyAllRules(initCtx, rules.value)
  if (template.value?.bg_image) {
    bgImg = new Image()
    bgImg.crossOrigin = 'anonymous'
    bgImg.src = `/static/${template.value.bg_image}`
    bgImg.onload = () => doRenderPreview()
    bgImg.onerror = () => doRenderPreview()
  } else { doRenderPreview() }
})
</script>

<style scoped>
.sign-layout {
  display: grid;
  grid-template-columns: 360px 1fr;
  gap: 24px;
  margin-top: 24px;
  align-items: start;
}
.form-section h2 { font-size: 18px; margin-bottom: 16px; }
.field { margin-bottom: 16px; }
.field.hidden { display: none; }
.field.disabled { opacity: 0.5; pointer-events: none; }
.field label { display: block; margin-bottom: 4px; font-size: 14px; }
.required { color: var(--danger); }
.checkbox-label { display: flex; align-items: center; gap: 8px; cursor: pointer; }
.field-error { color: var(--danger); font-size: 12px; margin-top: 4px; }
.form-actions { margin-top: 20px; }
.preview-section { display: flex; flex-direction: column; align-items: center; padding: 16px; }
.preview-canvas { max-width: 100%; border: 1px solid var(--border); border-radius: var(--radius); background: #fff; }
.result-actions { margin-top: 16px; text-align: center; }
.result-image { max-width: 100%; border-radius: var(--radius); margin-bottom: 12px; }
.btn-download { display: inline-block; text-decoration: none; }
.loading-state, .error-state { text-align: center; padding: 60px 20px; color: var(--text-secondary); font-size: 16px; }
.error-state { color: var(--danger); }
.hw-section { border-top: 1px solid var(--border); padding-top: 12px; margin-top: 12px; }
.hw-toggle-btn { display: flex; align-items: center; justify-content: space-between; width: 100%; padding: 6px 0; background: none; border: none; cursor: pointer; font-size: 13px; font-weight: 600; color: var(--text-secondary); }
@media (max-width: 768px) { .sign-layout { grid-template-columns: 1fr; } }
</style>
