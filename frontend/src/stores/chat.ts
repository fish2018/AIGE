import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ChatMessage } from '@/types'
import api from '@/utils/api'

export const useChatStore = defineStore('chat', () => {
  const messages = ref<ChatMessage[]>([])

  const sendMessage = async (message: string): Promise<ChatMessage> => {
    const chatMessage = await api.post<ChatMessage>('/chat', { message })
    
    messages.value.unshift(chatMessage)
    return chatMessage
  }

  const loadChatHistory = async (): Promise<ChatMessage[]> => {
    const data = await api.get<{ messages: ChatMessage[] }>('/chat/history')
    messages.value = data.messages
    return data.messages
  }

  const clearMessages = () => {
    messages.value = []
  }

  return {
    messages,
    sendMessage,
    loadChatHistory,
    clearMessages,
  }
})