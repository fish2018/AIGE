import { defineStore } from 'pinia'
import type { User, Provider, Model } from '@/types'
import api from '@/utils/api'

export const useAdminStore = defineStore('admin', () => {
  const getUsers = async (): Promise<User[]> => {
    const response = await api.get<{ users: User[] }>('/admin/users')
    return response.users
  }

  const getUser = async (userId: number): Promise<User> => {
    const response = await api.get<User>(`/admin/users/${userId}`)
    return response
  }

  const updateUserPassword = async (userId: number, newPassword: string): Promise<void> => {
    await api.put(`/admin/users/${userId}/password`, { new_password: newPassword })
  }

  const deleteUser = async (userId: number): Promise<void> => {
    await api.delete(`/admin/users/${userId}`)
  }

  const toggleUserAdmin = async (userId: number): Promise<void> => {
    await api.put(`/admin/users/${userId}/toggle-admin`)
  }

  const getProviders = async (): Promise<Provider[]> => {
    const response = await api.get<{ providers: Provider[] }>('/admin/providers')
    return response.providers
  }

  const getProvider = async (providerId: number): Promise<Provider> => {
    const response = await api.get<Provider>(`/admin/providers/${providerId}`)
    return response
  }

  const createProvider = async (provider: Partial<Provider>): Promise<Provider> => {
    const response = await api.post<Provider>('/admin/providers', provider)
    return response
  }

  const updateProvider = async (providerId: number, updates: Partial<Provider>): Promise<Provider> => {
    const response = await api.put<Provider>(`/admin/providers/${providerId}`, updates)
    return response
  }

  const deleteProvider = async (providerId: number): Promise<void> => {
    await api.delete(`/admin/providers/${providerId}`)
  }

  const toggleProvider = async (providerId: number): Promise<Provider> => {
    const response = await api.put<Provider>(`/admin/providers/${providerId}/toggle`)
    return response
  }

  const getAvailableModels = async (providerId: number, apiType?: string): Promise<string[]> => {
    const url = apiType 
      ? `/admin/providers/${providerId}/models/available?api_type=${apiType}`
      : `/admin/providers/${providerId}/models/available`
    const response = await api.get<{ models: string[] }>(url)
    return response.models
  }

  const testConnection = async (providerId: number, modelId?: string): Promise<any> => {
    const url = modelId
      ? `/admin/providers/${providerId}/test?model_id=${modelId}`
      : `/admin/providers/${providerId}/test`
    const response = await api.get<any>(url)
    return response
  }

  const getModels = async (providerId?: number): Promise<Model[]> => {
    const url = providerId 
      ? `/admin/models?provider_id=${providerId}`
      : '/admin/models'
    const response = await api.get<{ models: Model[] }>(url)
    return response.models
  }

  const getModel = async (modelId: number): Promise<Model> => {
    const response = await api.get<Model>(`/admin/models/${modelId}`)
    return response
  }

  const createModel = async (model: Partial<Model>): Promise<Model> => {
    const response = await api.post<Model>('/admin/models', model)
    return response
  }

  const updateModel = async (modelId: number, updates: Partial<Model>): Promise<Model> => {
    const response = await api.put<Model>(`/admin/models/${modelId}`, updates)
    return response
  }

  const deleteModel = async (modelId: number): Promise<void> => {
    await api.delete(`/admin/models/${modelId}`)
  }

  const toggleModel = async (modelId: number): Promise<Model> => {
    const response = await api.put<Model>(`/admin/models/${modelId}/toggle`)
    return response
  }

  const testModel = async (modelId: number): Promise<any> => {
    const response = await api.post<any>(`/admin/models/${modelId}/test`)
    return response
  }

  const updateModelCapabilities = async (modelId: number, capabilities: string): Promise<Model> => {
    const response = await api.put<Model>(`/admin/models/${modelId}/capabilities`, { capabilities })
    return response
  }

  const getOAuthConfig = async (): Promise<any> => {
    const response = await api.get<any>('/admin/oauth/config')
    return response
  }

  const saveOAuthConfig = async (config: any): Promise<void> => {
    await api.post('/admin/oauth/config', config)
  }

  return {
    getUsers,
    getUser,
    updateUserPassword,
    deleteUser,
    toggleUserAdmin,
    getProviders,
    getProvider,
    createProvider,
    updateProvider,
    deleteProvider,
    toggleProvider,
    getAvailableModels,
    testConnection,
    getModels,
    getModel,
    createModel,
    updateModel,
    deleteModel,
    toggleModel,
    testModel,
    updateModelCapabilities,
    getOAuthConfig,
    saveOAuthConfig,
  }
})