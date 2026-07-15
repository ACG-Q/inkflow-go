<template>
  <div
    class="fonts-page"
    @dragenter.prevent="onDragEnter"
    @dragover.prevent="onDragOver"
    @dragleave.prevent="onDragLeave"
    @drop.prevent="onDrop"
  >
    <div class="page-header">
      <h1>字体管理</h1>
      <span class="font-count">{{ fonts.length }} 个字体</span>
    </div>

    <Transition name="overlay-fade">
      <div v-if="dragging" class="drop-overlay">
        <div class="drop-overlay-inner">
          <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M12 5v14M5 12l7 7 7-7" />
          </svg>
          <p>释放以上传字体文件</p>
          <span>支持 .ttf / .otf / .woff / .woff2，可同时拖入多个文件</span>
        </div>
      </div>
    </Transition>

    <div v-if="uploadQueue.length > 0" class="upload-queue">
      <div v-for="item in uploadQueue" :key="item.id" class="upload-item">
        <span class="upload-filename">{{ item.file.name }}</span>
        <span v-if="item.status === 'uploading'" class="upload-status uploading">上传中...</span>
        <span v-else-if="item.status === 'done'" class="upload-status done">完成</span>
        <span v-else-if="item.status === 'error'" class="upload-status error">{{ item.error }}</span>
      </div>
    </div>

    <div class="font-grid">
      <div v-for="f in fonts" :key="f.id" class="font-card">
        <div class="font-preview" :style="{ fontFamily: `'${f.display_name}', sans-serif` }">
          预览文字：AaBbCc 123
        </div>
        <div class="font-info">
          <div class="font-name">
            <template v-if="editingId === f.id">
              <input
                v-model="editingName"
                :ref="(el: any) => setEditRef(el as HTMLInputElement | null, f.id)"
                class="edit-input"
                @blur="saveName(f.id)"
                @keydown.enter="saveName(f.id)"
              />
            </template>
            <template v-else>
              <span class="name-display" @dblclick="startEdit(f)">{{ f.display_name || f.filename }}</span>
              <button class="btn-icon" @click="startEdit(f)" title="编辑名称">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
              </button>
            </template>
          </div>
          <div class="font-file">{{ f.original_filename }}</div>
          <button class="btn-danger btn-sm" @click="handleDelete(f.id)">删除</button>
        </div>
      </div>

      <div v-if="fonts.length === 0 && uploadQueue.length === 0" class="empty-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" style="opacity: 0.3">
          <path d="M9 12h6M12 9v6M21 20V8a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2h14a2 2 0 002-2z"/>
        </svg>
        <p>暂无字体</p>
        <span>拖拽字体文件到页面任意位置，或点击下方按钮</span>
      </div>
    </div>

    <label class="fab-add" :class="{ uploading: overallUploading }">
      <input
        ref="fileInput"
        type="file"
        accept=".ttf,.otf,.woff,.woff2"
        multiple
        style="display:none"
        @change="onFileSelect"
      />
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <path d="M12 5v14M5 12h14"/>
      </svg>
      <span v-if="overallUploading">上传中...</span>
    </label>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { api } from '@/api'
import type { FontItem } from '@/types'

const fonts = ref<FontItem[]>([])
const dragging = ref(false)
const fileInput = ref<HTMLInputElement>()
const editingId = ref<number | null>(null)
const editingName = ref('')
const editInputRefs = ref(new Map<number, HTMLInputElement>())
const overallUploading = ref(false)

function setEditRef(el: HTMLInputElement | null, id: number) {
  if (el) editInputRefs.value.set(id, el)
  else editInputRefs.value.delete(id)
}

interface UploadItem {
  id: number
  file: File
  status: 'uploading' | 'done' | 'error'
  error?: string
}
const uploadQueue = ref<UploadItem[]>([])
let queueCounter = 0

let dragCounter = 0

async function load() {
  const res = await api.listFonts()
  if (res.code === 0) fonts.value = res.data
}

function onDragEnter(e: DragEvent) {
  e.preventDefault()
  dragCounter++
  dragging.value = true
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
}

function onDragLeave(e: DragEvent) {
  e.preventDefault()
  dragCounter--
  if (dragCounter <= 0) {
    dragCounter = 0
    dragging.value = false
  }
}

function onDrop(e: DragEvent) {
  e.preventDefault()
  dragCounter = 0
  dragging.value = false
  const files = e.dataTransfer?.files
  if (files && files.length > 0) {
    uploadFiles(Array.from(files))
  }
}

function onFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    uploadFiles(Array.from(input.files))
    input.value = ''
  }
}

async function uploadFiles(files: File[]) {
  const valid = files.filter(f => {
    const ext = '.' + f.name.split('.').pop()?.toLowerCase()
    return ['.ttf', '.otf', '.woff', '.woff2'].includes(ext)
  })
  if (valid.length === 0) return

  overallUploading.value = true

  for (const file of valid) {
    const item: UploadItem = { id: queueCounter++, file, status: 'uploading' }
    uploadQueue.value.push(item)
    try {
      const result = await api.uploadFont(file)
      if (result.code !== 0) {
        item.status = 'error'
        item.error = result.message || '上传失败'
      } else {
        item.status = 'done'
      }
    } catch (err: any) {
      item.status = 'error'
      item.error = err.message || '上传失败'
    }
  }

  overallUploading.value = false
  await load()

  setTimeout(() => {
    uploadQueue.value = uploadQueue.value.filter(i => i.status === 'uploading')
  }, 3000)
}

function startEdit(f: FontItem) {
  editingId.value = f.id
  editingName.value = f.display_name || f.filename
  nextTick(() => {
    editInputRefs.value.get(f.id)?.focus()
  })
}

async function saveName(id: number) {
  if (!editingName.value.trim()) { editingId.value = null; return }
  try {
    await api.updateFont(id, { display_name: editingName.value.trim() })
  } catch {
    // 静默处理——编辑模式仍然关闭，下次加载会恢复原先的名称
  }
  editingId.value = null
  await load()
}

async function handleDelete(id: number) {
  if (!confirm('确定要删除该字体吗？')) return
  await api.deleteFont(id)
  await load()
}

onMounted(load)
</script>

<style scoped>
.fonts-page {
  min-height: 100vh;
  padding: 24px;
  position: relative;
}

.page-header {
  display: flex;
  align-items: baseline;
  gap: 12px;
  margin-bottom: 24px;
}
.page-header h1 { margin: 0; font-size: 22px; }
.font-count { font-size: 13px; color: var(--text-secondary); }

.drop-overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
}
.drop-overlay-inner {
  background: var(--bg-card, #fff);
  border: 3px dashed var(--primary, #3b82f6);
  border-radius: 16px;
  padding: 60px 80px;
  text-align: center;
  color: var(--primary, #3b82f6);
}
.drop-overlay-inner svg { margin-bottom: 16px; }
.drop-overlay-inner p { font-size: 18px; font-weight: 600; margin: 0 0 8px; }
.drop-overlay-inner span { font-size: 13px; color: var(--text-secondary, #94a3b8); }

.overlay-fade-enter-active,
.overlay-fade-leave-active { transition: opacity 0.2s ease; }
.overlay-fade-enter-from,
.overlay-fade-leave-to { opacity: 0; }

.upload-queue {
  margin-bottom: 20px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.upload-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 14px;
  background: var(--bg-card, #fff);
  border: 1px solid var(--border, #e5e7eb);
  border-radius: 8px;
  font-size: 13px;
}
.upload-filename { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 300px; }
.upload-status { flex-shrink: 0; margin-left: 12px; }
.upload-status.uploading { color: var(--primary, #3b82f6); }
.upload-status.done { color: #22c55e; }
.upload-status.error { color: #ef4444; }

.font-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}
.font-card {
  background: var(--bg-card, #fff);
  border: 1px solid var(--border, #e5e7eb);
  border-radius: 10px;
  overflow: hidden;
  transition: box-shadow 0.2s;
}
.font-card:hover { box-shadow: 0 2px 12px rgba(0,0,0,0.08); }
.font-preview {
  padding: 20px;
  background: var(--bg, #f8fafc);
  font-size: 18px;
  text-align: center;
  border-bottom: 1px solid var(--border, #e5e7eb);
}
.font-info { padding: 12px 16px; }
.font-name { display: flex; align-items: center; gap: 6px; margin-bottom: 4px; }
.name-display { font-weight: 600; font-size: 14px; cursor: pointer; }
.name-display:hover { color: var(--primary, #3b82f6); }
.edit-input {
  flex: 1;
  padding: 2px 6px;
  border: 1px solid var(--primary, #3b82f6);
  border-radius: 4px;
  font-size: 13px;
}
.font-file { font-size: 12px; color: var(--text-secondary, #94a3b8); margin-bottom: 8px; }
.btn-icon {
  background: none;
  border: none;
  cursor: pointer;
  padding: 2px;
  color: var(--text-secondary, #94a3b8);
  transition: color 0.15s;
}
.btn-icon:hover { color: var(--primary, #3b82f6); }
.btn-sm { padding: 4px 8px; font-size: 12px; }
.btn-danger {
  background: #fee2e2;
  color: #dc2626;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}
.btn-danger:hover { background: #fecaca; }
.empty-state {
  grid-column: 1 / -1;
  text-align: center;
  padding: 60px 20px;
  color: var(--text-secondary, #94a3b8);
}
.empty-state p { margin: 12px 0 4px; font-size: 16px; }
.empty-state span { font-size: 13px; }

.fab-add {
  position: fixed;
  bottom: 32px;
  right: 32px;
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--primary, #3b82f6);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 4px 16px rgba(59,130,246,0.4);
  transition: all 0.2s;
  z-index: 50;
}
.fab-add:hover { transform: scale(1.08); box-shadow: 0 6px 20px rgba(59,130,246,0.5); }
.fab-add.uploading { opacity: 0.6; pointer-events: none; }
.fab-add span {
  position: absolute;
  bottom: -24px;
  right: 0;
  font-size: 11px;
  color: var(--text-secondary, #94a3b8);
  white-space: nowrap;
}
</style>
