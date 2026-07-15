<template>
  <nav class="nav">
    <div class="nav-inner">
      <router-link to="/" class="nav-brand">InkFlow</router-link>
      <div class="nav-links">
        <router-link to="/" class="nav-link">首页</router-link>
        <router-link v-if="isLoggedIn" to="/admin" class="nav-link">管理</router-link>
        <router-link v-if="!isLoggedIn" to="/admin/login" class="nav-link">登录</router-link>
        <span v-else class="nav-link logout-btn" @click="handleLogout">退出</span>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()
const isLoggedIn = computed(() => auth.isAuthenticated)

function handleLogout() {
  auth.logout()
  router.push('/')
}
</script>

<style scoped>
.nav {
  background: var(--bg-card);
  border-bottom: 1px solid var(--border);
}
.nav-inner {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 24px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.nav-brand {
  font-size: 20px;
  font-weight: 700;
  color: var(--primary);
}
.nav-links {
  display: flex;
  gap: 16px;
  align-items: center;
}
.nav-link {
  color: var(--text-secondary);
  font-size: 14px;
}
.nav-link:hover { color: var(--text); }
.logout-btn { cursor: pointer; }
</style>
