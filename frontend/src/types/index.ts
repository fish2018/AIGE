export interface User {
  id: number
  username: string
  email: string
  is_admin: boolean
}

export interface LoginRequest {
  username: string
  password: string
}

export interface RegisterRequest {
  username: string
  password: string
  email: string
}

export interface AuthResponse {
  token: string
  user: User
}

export interface ChatMessage {
  id: number
  message: string
  response: string
}

export interface ApiResponse<T = any> {
  data?: T
  message?: string
  error?: string
}

export interface Provider {
  id: number
  name: string
  type: 'openai' | 'anthropic' | 'google' | 'custom'
  api_key: string
  base_url?: string
  enabled: boolean
  allow_custom_url?: boolean
  models?: Model[]
  created_at?: string
  updated_at?: string
}

export interface Model {
  id: number
  model_id: string
  name: string
  provider_id: number
  provider?: Provider
  enabled: boolean
  api_type?: string
  capabilities?: string
  last_tested?: string
  test_status?: 'untested' | 'testing' | 'success' | 'failed'
  created_at?: string
  updated_at?: string
}