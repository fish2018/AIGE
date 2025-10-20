import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { User, LoginRequest, RegisterRequest, AuthResponse } from '@/types'
import api from '@/utils/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const token = ref<string | null>(localStorage.getItem('token'))

  const login = async (loginData: LoginRequest): Promise<AuthResponse> => {
    const data = await api.post<AuthResponse>('/auth/login', loginData)
    
    user.value = data.user
    token.value = data.token
    localStorage.setItem('token', data.token)
    
    return data
  }

  const register = async (registerData: RegisterRequest): Promise<AuthResponse> => {
    const data = await api.post<AuthResponse>('/auth/register', registerData)
    
    user.value = data.user
    token.value = data.token
    localStorage.setItem('token', data.token)
    
    return data
  }

  const logout = () => {
    user.value = null
    token.value = null
    localStorage.removeItem('token')
    // 使用动态导入避免循环依赖
    import('@/router').then(({ default: router }) => {
      router.push('/login')
    })
  }

  const getProfile = async (): Promise<User> => {
    if (!token.value) {
      throw new Error('No token')
    }
    
    const userData = await api.get<User>('/profile')
    user.value = userData
    return userData
  }

  const isAuthenticated = () => {
    return !!token.value
  }

  const isAdmin = () => {
    return user.value?.is_admin || false
  }

  const loginWithLinuxDo = async (): Promise<string> => {
    const data = await api.get<{ auth_url: string }>('/auth/oauth/linux-do')
    return data.auth_url
  }

  const handleLinuxDoCallback = async (code: string, state: string): Promise<AuthResponse> => {
    const data = await api.get<AuthResponse>(`/auth/oauth/linux-do/callback?code=${code}&state=${state}`)
    
    user.value = data.user
    token.value = data.token
    localStorage.setItem('token', data.token)
    
    return data
  }

  return {
    user,
    token,
    login,
    register,
    logout,
    getProfile,
    isAuthenticated,
    isAdmin,
    loginWithLinuxDo,
    handleLinuxDoCallback,
  }
})