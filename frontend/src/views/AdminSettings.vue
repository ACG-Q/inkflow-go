<template>
  <div class="page">
    <div class="page-header">
      <router-link to="/admin" class="btn-ghost">返回</router-link>
      <h1>全局设置</h1>
    </div>
    <div class="settings-card card">
      <h2>手写效果</h2>
      <p class="desc">配置全局默认的手写效果参数，新建模板时会自动继承此配置。</p>
      <HandwritingPanel
        :model="hw"
        :show-actions="true"
        @update:model-value="hw = $event"
        @reset="resetDefaults"
      />
      <div class="settings-actions">
        <button class="btn-primary" @click="handleSave" :disabled="saving">
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api'
import type { HandwritingConfig } from '@/types'
import { HW_DEFAULTS } from '@/constants/defaults'
import HandwritingPanel from '@/components/editor/HandwritingPanel.vue'

const hw = ref<HandwritingConfig>({})
const saving = ref(false)

function resetDefaults() {
  hw.value = { ...HW_DEFAULTS }
}

onMounted(async () => {
  const res = await api.getHandwritingSettings()
  if (res.code === 0 && res.data) {
    hw.value = res.data
  } else {
    hw.value = { ...HW_DEFAULTS }
  }
})

async function handleSave() {
  saving.value = true
  try {
    await api.updateHandwritingSettings(hw.value)
    alert('保存成功')
  } catch {
    alert('保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.page-header { display: flex; align-items: center; gap: 16px; margin-bottom: 24px; }
.page-header h1 { margin: 0; font-size: 20px; }
.settings-card { max-width: 500px; padding: 24px; }
.settings-card h2 { font-size: 16px; margin: 0 0 8px; }
.desc { font-size: 13px; color: var(--text-secondary); margin-bottom: 16px; }
.settings-actions { margin-top: 16px; text-align: right; }
</style>
