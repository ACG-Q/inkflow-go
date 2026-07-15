<template>
  <div class="editor-page">
    <div class="editor-header">
      <div class="header-left">
        <router-link to="/admin" class="btn-ghost btn-back">返回</router-link>
        <input
          class="title-input"
          v-model="store.name"
          placeholder="模板名称"
          @input="store.dirty = true"
        />
      </div>
      <div class="header-right">
        <button class="btn-ghost" @click="showRules = !showRules">规则</button>
        <span v-if="store.dirty" class="unsaved-badge">未保存</span>
        <button class="btn-primary" @click="handleSave" :disabled="store.saving || !store.name">
          {{ store.saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
    <div class="editor-body">
      <EditorToolbar />
      <CanvasStage />
      <PropertiesPanel />
    </div>
    <RulesPanel :visible="showRules" @close="showRules = false" @add="openAddRule" @edit="openEditRule" />
    <RuleModal :visible="showRuleModal" :editing-id="editingRuleId" @close="closeRuleModal" @saved="closeRuleModal" />
    <AppToast />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import { useEditorStore } from '@/components/editor/editorStore'
import EditorToolbar from '@/components/editor/EditorToolbar.vue'
import CanvasStage from '@/components/editor/CanvasStage.vue'
import PropertiesPanel from '@/components/editor/PropertiesPanel.vue'
import RulesPanel from '@/components/editor/RulesPanel.vue'
import RuleModal from '@/components/editor/RuleModal.vue'
import AppToast from '@/components/AppToast.vue'

const route = useRoute()
const router = useRouter()
const store = useEditorStore()

const showRules = ref(false)
const showRuleModal = ref(false)
const editingRuleId = ref<string | null>(null)

onMounted(async () => {
  const id = route.query.id
  if (id) {
    await store.loadTemplate(Number(id))
  }
})

onBeforeRouteLeave((_to, _from, next) => {
  if (store.dirty) {
    if (!confirm('有未保存的修改，确定离开吗？')) {
      next(false)
      return
    }
  }
  next()
})

async function handleSave() {
  try {
    await store.save()
    if (!route.query.id && store.templateId) {
      router.replace({ query: { id: String(store.templateId) } })
    }
  } catch (e: any) {
    alert(e.message || '保存失败')
  }
}

function openAddRule() {
  editingRuleId.value = null
  showRuleModal.value = true
}

function openEditRule(id: string) {
  editingRuleId.value = id
  showRuleModal.value = true
}

function closeRuleModal() {
  showRuleModal.value = false
  editingRuleId.value = null
}
</script>

<style scoped>
.editor-page {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 56px);
}
.editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: var(--bg-card);
  border-bottom: 1px solid var(--border);
  gap: 16px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}
.btn-back {
  font-size: 13px;
  padding: 6px 12px;
}
.title-input {
  font-size: 18px;
  font-weight: 600;
  border: none;
  outline: none;
  background: transparent;
  color: var(--text);
  flex: 1;
  max-width: 400px;
  padding: 4px 0;
}
.title-input:focus {
  border-bottom: 2px solid var(--primary);
  margin-bottom: -2px;
}
.unsaved-badge {
  font-size: 12px;
  color: #d97706;
  background: #fef3c7;
  padding: 2px 8px;
  border-radius: 4px;
}
.editor-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}
.btn-ghost {
  background: none;
  border: 1px solid var(--border);
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text);
}
.btn-ghost:hover { background: var(--bg); }
</style>
