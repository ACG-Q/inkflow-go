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

  function logout() {
    token.value = ''
    username.value = ''
    localStorage.removeItem('token')
  }

  return { token, username, isAuthenticated, login, logout }
})
