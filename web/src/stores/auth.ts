import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Admin } from '../types'
import { getMe } from '../api/auth'

export const useAuthStore = defineStore('auth', () => {
  const admin = ref<Admin | null>(null)
  const loading = ref(true)

  async function fetchMe() {
    loading.value = true
    const res = await getMe()
    if (res.ok && res.data) {
      admin.value = res.data as Admin
    } else {
      admin.value = null
    }
    loading.value = false
  }

  function setAdmin(a: Admin) {
    admin.value = a
  }

  function clear() {
    admin.value = null
  }

  return { admin, loading, fetchMe, setAdmin, clear }
})
