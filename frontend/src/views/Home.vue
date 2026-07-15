<template>
  <div class="page">
    <div class="page-header">
      <h1>模板列表</h1>
    </div>
    <div class="template-grid">
      <div v-for="t in templates" :key="t.id" class="card template-card" :class="{ deleting: deletingId === t.id }">
        <div class="card-preview">
          <img v-if="t.bg_image" :src="`/static/${t.bg_image}`" alt="bg" loading="lazy" />
          <div v-else class="placeholder">📄 协议背景</div>
        </div>
        <div class="card-body">
          <h3>{{ t.name }}</h3>
          <p class="meta">更新于: {{ formatDate(t.created_at) }}</p>
          <div class="actions">
            <router-link :to="`/sign?template_id=${t.id}`" class="btn-primary btn-sm">前往签署</router-link>
            <router-link v-if="isLoggedIn" :to="`/admin/editor?id=${t.id}`" class="btn-ghost btn-sm">配置</router-link>
            <button v-if="isLoggedIn" class="btn-danger btn-sm" @click="handleDelete(t.id, t.name)">删除</button>
          </div>
        </div>
      </div>
      <div v-if="templates.length === 0" class="empty">
        <div>📭</div>
        <h2>暂无任何模板</h2>
        <p>点击管理后台"创建新模板"来开启您的第一个协议。</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api } from '@/api'
import { useAuthStore } from '@/stores/auth'
import type { TemplateListItem } from '@/types'

const templates = ref<TemplateListItem[]>([])
const auth = useAuthStore()
const isLoggedIn = computed(() => auth.isAuthenticated)
const deletingId = ref<number | null>(null)

async function load() {
  const res = await api.listTemplates()
  if (res.code === 0) templates.value = res.data.items
}

function formatDate(d: string) {
  return new Date(d).toLocaleDateString('zh-CN')
}

async function handleDelete(id: number, name: string) {
  if (!confirm(`⚠️ 警告\n\n您确定要删除模板《${name}》吗？\n删除后所有配置及关联的签署链接都将失效。`)) return
  await api.deleteTemplate(id)
  load()
}

onMounted(load)
</script>

<style scoped>
.template-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}
.template-card {
  padding: 0;
  overflow: hidden;
  transition: opacity 0.3s, transform 0.3s;
}
.template-card.deleting {
  opacity: 0;
  transform: scale(0.9);
}
.card-preview {
  height: 180px;
  background: #f8fafc;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-bottom: 1px solid var(--border);
}
.card-preview img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.placeholder { color: #94a3b8; font-size: 14px; }
.card-body { padding: 16px; }
.card-body h3 { font-size: 16px; margin: 0 0 4px; }
.meta { color: var(--text-secondary); font-size: 13px; margin: 0 0 12px; }
.actions { display: flex; gap: 8px; }
.btn-sm { padding: 6px 12px; font-size: 13px; }
.btn-ghost {
  background: none;
  border: 1px solid var(--border);
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text);
  text-decoration: none;
}
.btn-ghost:hover { background: var(--bg); }
.empty {
  grid-column: 1 / -1;
  text-align: center;
  padding: 48px;
  color: var(--text-secondary);
}
.empty h2 { margin: 8px 0 4px; }
.empty p { margin: 0; }
</style>
