<template>
  <div class="page">
    <div class="page-header">
      <h1>签署记录</h1>
    </div>
    <div class="card">
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>模板</th>
            <th>填写内容</th>
            <th>IP</th>
            <th>时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in records" :key="r.id">
            <td>{{ r.id }}</td>
            <td>{{ r.template_name }}</td>
            <td>
              <button class="btn-ghost btn-sm" @click="showFields(r)">查看详情</button>
            </td>
            <td>{{ r.ip || '-' }}</td>
            <td>{{ formatDate(r.created_at) }}</td>
            <td class="actions">
              <button class="btn-ghost btn-sm" @click="previewImage(r.image_url)">预览</button>
              <a :href="r.image_url" download class="btn-ghost btn-sm">下载</a>
              <button class="btn-danger btn-sm" @click="handleDelete(r.id)">删除</button>
            </td>
          </tr>
          <tr v-if="records.length === 0">
            <td colspan="6" style="text-align:center;color:var(--text-secondary)">暂无记录</td>
          </tr>
        </tbody>
      </table>
      <div v-if="totalPages > 1" class="pagination">
        <button :disabled="currentPage <= 1" @click="goPage(currentPage - 1)">上一页</button>
        <span>{{ currentPage }} / {{ totalPages }}</span>
        <button :disabled="currentPage >= totalPages" @click="goPage(currentPage + 1)">下一页</button>
      </div>
    </div>

    <!-- 填写详情模态框 -->
    <div v-if="showFieldsModal" class="modal-overlay" @click.self="showFieldsModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3>填写详情</h3>
          <button class="close-btn" @click="showFieldsModal = false">&times;</button>
        </div>
        <div class="modal-body">
          <div v-if="fieldsEntries.length === 0" class="empty-fields">暂无填写内容</div>
          <table v-else class="fields-table">
            <tr v-for="[key, val] in fieldsEntries" :key="key">
              <td class="field-key">{{ getFieldLabel(key) }}</td>
              <td class="field-val">{{ formatFieldValue(val) }}</td>
            </tr>
          </table>
        </div>
      </div>
    </div>

    <!-- 图片预览模态框 -->
    <div v-if="previewUrl" class="modal-overlay" @click.self="previewUrl = ''">
      <div class="modal-content modal-image">
        <button class="close-btn floating" @click="previewUrl = ''">&times;</button>
        <img :src="previewUrl" class="preview-img" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api } from '@/api'
import type { SigningRecord } from '@/types'

const records = ref<SigningRecord[]>([])
const showFieldsModal = ref(false)
const currentRecord = ref<SigningRecord | null>(null)
const previewUrl = ref('')
const currentPage = ref(1)
const totalPages = ref(1)
const pageSize = 20
const templateControlsCache = new Map<number, { id: string; label: string }[]>()

async function load() {
  const res = await api.listRecords({ page: currentPage.value, page_size: pageSize })
  if (res.code === 0) {
    records.value = res.data.items
    totalPages.value = Math.ceil(res.data.total / res.data.page_size)
  }
}

function goPage(p: number) {
  currentPage.value = p
  load()
}

function formatDate(d: string) {
  return new Date(d).toLocaleString('zh-CN')
}

async function fetchTemplateControls(templateId: number) {
  if (templateControlsCache.has(templateId)) return templateControlsCache.get(templateId)!
  try {
    const res = await api.getTemplate(templateId)
    if (res.code === 0) {
      const controls = (res.data.controls || []).map(c => ({ id: c.id, label: c.label || c.id }))
      templateControlsCache.set(templateId, controls)
      return controls
    }
  } catch {}
  return []
}

function getFieldLabel(ctrlId: string): string {
  if (!currentRecord.value) return ctrlId
  const controls = templateControlsCache.get(currentRecord.value.template_id) || []
  const ctrl = controls.find(c => c.id === ctrlId)
  if (ctrl && ctrl.label !== ctrl.id) return `${ctrl.label} (${ctrlId})`
  return ctrlId
}

function formatFieldValue(val: any): string {
  if (val === undefined || val === null) return ''
  if (typeof val === 'boolean') return val ? '✓ 是' : '✗ 否'
  return String(val)
}

const fieldsEntries = computed(() => {
  if (!currentRecord.value) return []
  try {
    const fields = typeof currentRecord.value.fields_data === 'string'
      ? JSON.parse(currentRecord.value.fields_data)
      : currentRecord.value.fields_data
    return Object.entries(fields || {})
  } catch { return [] }
})

async function showFields(record: SigningRecord) {
  currentRecord.value = record
  await fetchTemplateControls(record.template_id)
  showFieldsModal.value = true
}

function previewImage(url: string) {
  previewUrl.value = url
}

async function handleDelete(id: number) {
  if (!confirm('确认删除？')) return
  await api.deleteRecord(id)
  load()
}

onMounted(load)
</script>

<style scoped>
.actions { display: flex; gap: 6px; align-items: center; }
.btn-sm { padding: 4px 8px; font-size: 12px; }
.btn-ghost {
  background: none;
  border: 1px solid var(--border);
  padding: 4px 8px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  color: var(--text);
  text-decoration: none;
}
.btn-ghost:hover { background: var(--bg); }
.btn-danger { padding: 4px 8px; font-size: 12px; }
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  padding: 16px;
}
.pagination button {
  padding: 6px 16px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg-card);
  cursor: pointer;
  font-size: 13px;
}
.pagination button:disabled { opacity: 0.4; cursor: not-allowed; }
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-content {
  background: var(--bg-card);
  border-radius: 12px;
  width: 420px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0,0,0,0.2);
}
.modal-image {
  width: auto;
  max-width: 90vw;
  background: transparent;
  position: relative;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}
.modal-header h3 { margin: 0; font-size: 16px; }
.close-btn {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: var(--text-secondary);
}
.close-btn.floating {
  position: absolute;
  top: 8px;
  right: 8px;
  color: #fff;
  background: rgba(0,0,0,0.5);
  border-radius: 50%;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1;
}
.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}
.empty-fields { text-align: center; color: var(--text-secondary); padding: 20px; }
.fields-table { width: 100%; border-collapse: collapse; }
.fields-table td { padding: 8px 12px; border-bottom: 1px solid var(--border); font-size: 13px; }
.field-key { color: var(--text-secondary); white-space: nowrap; width: 120px; }
.field-val { color: var(--text); word-break: break-all; }
.preview-img { max-width: 100%; max-height: 80vh; border-radius: 8px; }
</style>
