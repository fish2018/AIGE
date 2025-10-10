<template>
  <div class="playground">
    <div class="playground-header">
      <h2>AI 操练场</h2>
      <p class="header-desc">选择提供商和模型进行对话测试</p>
    </div>

    <div class="playground-container">
      <div class="chat-panel">
        <div class="config-bar">
          <el-select 
            v-model="selectedProviderId" 
            placeholder="选择提供商" 
            @change="onProviderChange"
            style="width: 200px"
            size="small"
          >
            <el-option
              v-for="provider in enabledProviders"
              :key="provider.id"
              :label="provider.name"
              :value="provider.id"
            />
          </el-select>

          <el-select 
            v-model="selectedModelId" 
            placeholder="选择模型"
            :disabled="!selectedProviderId"
            style="width: 250px"
            size="small"
          >
            <el-option
              v-for="model in availableModels"
              :key="model.id"
              :label="model.name"
              :value="model.model_id"
            />
          </el-select>

          <div style="display: flex; align-items: center; gap: 8px;">
            <span style="font-size: 14px; color: #666;">流式输出</span>
            <el-switch v-model="streamEnabled" size="small" />
          </div>

          <div style="flex: 1;"></div>

          <el-button size="small" @click="clearMessages" :disabled="messages.length === 0">
            清空对话
          </el-button>
        </div>

        <div class="messages-container" ref="messagesContainer">
          <div 
            v-for="(message, index) in messages" 
            :key="index"
            :class="['message', `message-${message.role}`]"
          >
            <div class="message-role">
              {{ message.role === 'user' ? '用户' : 'AI' }}
            </div>
            <div class="message-content" v-html="renderMarkdown(message.content)"></div>
          </div>
          
          <div v-if="isTyping" class="message message-assistant">
            <div class="message-role">AI</div>
            <div class="message-content typing-indicator">
              <span></span>
              <span></span>
              <span></span>
            </div>
          </div>
        </div>

        <div class="input-container">
          <el-input
            v-model="userInput"
            type="textarea"
            :rows="3"
            placeholder="输入消息... (Ctrl+Enter 发送)"
            @keydown="handleKeydown"
            :disabled="isLoading || !selectedModelId"
          />
          <div class="input-actions">
            <span class="input-hint">Ctrl + Enter 发送</span>
            <el-button 
              type="primary" 
              @click="sendMessage"
              :loading="isLoading"
              :disabled="!userInput.trim() || !selectedModelId"
            >
              发送
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { useAdminStore } from '@/stores/admin'
import { AIService, type Message } from '@/services/aiService'
import type { Provider, Model } from '@/types'
import { marked } from 'marked'

const adminStore = useAdminStore()

const providers = ref<Provider[]>([])
const models = ref<Model[]>([])
const selectedProviderId = ref<number | null>(null)
const selectedModelId = ref<string>('')
const streamEnabled = ref(true)
const messages = ref<Message[]>([])
const userInput = ref('')
const isLoading = ref(false)
const isTyping = ref(false)
const messagesContainer = ref<HTMLElement>()

const enabledProviders = computed(() => {
  return providers.value.filter(p => p.enabled)
})

const availableModels = computed(() => {
  if (!selectedProviderId.value) return []
  return models.value.filter(m => 
    m.provider_id === selectedProviderId.value && m.enabled
  )
})

const renderMarkdown = (content: string): string => {
  try {
    const result = marked(content, {
      breaks: true,
      gfm: true
    })
    return typeof result === 'string' ? result : String(result)
  } catch (error) {
    return content.replace(/\n/g, '<br>')
  }
}

const onProviderChange = () => {
  selectedModelId.value = ''
  loadModels()
}

const loadProviders = async () => {
  try {
    providers.value = await adminStore.getProviders()
  } catch (error: any) {
    ElMessage.error('加载提供商失败: ' + error.message)
  }
}

const loadModels = async () => {
  if (!selectedProviderId.value) return
  
  try {
    models.value = await adminStore.getModels(selectedProviderId.value)
  } catch (error: any) {
    ElMessage.error('加载模型失败: ' + error.message)
  }
}

const scrollToBottom = () => {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.ctrlKey && event.key === 'Enter') {
    event.preventDefault()
    sendMessage()
  }
}

const sendMessage = async () => {
  if (!userInput.value.trim() || !selectedModelId.value || !selectedProviderId.value) {
    return
  }

  const userMessage: Message = {
    role: 'user',
    content: userInput.value.trim()
  }

  messages.value.push(userMessage)
  userInput.value = ''
  scrollToBottom()

  console.log('[Playground] 开始发送消息')

  try {
    if (streamEnabled.value) {
      console.log('[Playground] 使用流式模式')
      isTyping.value = true
      scrollToBottom()

      let streamingContent = ''
      let aiMessageIndex = -1

      await AIService.chatStream(
        {
          provider_id: selectedProviderId.value,
          model_id: selectedModelId.value,
          messages: messages.value,
          stream: true
        },
        (content) => {
          console.log('[Playground] onChunk 被调用, content:', JSON.stringify(content), 'aiMessageIndex:', aiMessageIndex)
          
          if (aiMessageIndex === -1) {
            console.log('[Playground] 创建新的AI消息')
            isTyping.value = false
            messages.value.push({
              role: 'assistant',
              content: ''
            })
            aiMessageIndex = messages.value.length - 1
            console.log('[Playground] AI消息索引:', aiMessageIndex)
          }
          
          streamingContent += content
          console.log('[Playground] 累积内容长度:', streamingContent.length, '当前内容:', JSON.stringify(streamingContent.substring(0, 50)))
          messages.value[aiMessageIndex].content = streamingContent
          console.log('[Playground] 更新消息后, messages长度:', messages.value.length, '消息内容长度:', messages.value[aiMessageIndex].content.length)
          scrollToBottom()
        },
        (error) => {
          console.error('[Playground] onError 被调用:', error)
          isTyping.value = false
          isLoading.value = false
          ElMessage.error('对话失败: ' + error.message)
        },
        () => {
          console.log('[Playground] onComplete 被调用, 最终内容长度:', streamingContent.length)
          isTyping.value = false
          isLoading.value = false
          scrollToBottom()
        }
      )
    } else {
      isLoading.value = true
      
      const response = await AIService.chat({
        provider_id: selectedProviderId.value,
        model_id: selectedModelId.value,
        messages: messages.value,
        stream: false
      })

      messages.value.push({
        role: 'assistant',
        content: response.content
      })
      
      isLoading.value = false
      scrollToBottom()
    }
  } catch (error: any) {
    isLoading.value = false
    isTyping.value = false
    ElMessage.error('发送消息失败: ' + error.message)
  }
}

const clearMessages = () => {
  messages.value = []
}

onMounted(() => {
  loadProviders()
})
</script>

<style scoped>
.playground {
  padding: 20px;
}

.playground-header {
  margin-bottom: 24px;
}

.playground-header h2 {
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
}

.header-desc {
  margin: 0;
  color: #666;
  font-size: 14px;
}

.playground-container {
  height: calc(100vh - 200px);
}

.config-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
}

.chat-panel {
  display: flex;
  flex-direction: column;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  overflow: hidden;
  height: 100%;
}

.messages-container {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.message {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.message-role {
  font-size: 12px;
  font-weight: 600;
  color: #666;
}

.message-content {
  padding: 12px 16px;
  border-radius: 8px;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.6;
}

.message-content :deep(p) {
  margin: 0 0 8px 0;
}

.message-content :deep(p:last-child) {
  margin-bottom: 0;
}

.message-content :deep(code) {
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
}

.message-content :deep(pre) {
  background: #282c34;
  color: #abb2bf;
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 8px 0;
}

.message-content :deep(pre code) {
  background: none;
  padding: 0;
  color: inherit;
}

.message-user .message-role {
  color: #409eff;
  text-align: right;
}

.message-user .message-content {
  background: #409eff;
  color: white;
  align-self: flex-end;
  max-width: 70%;
}

.message-user .message-content :deep(code) {
  background: rgba(255, 255, 255, 0.2);
  color: white;
}

.message-assistant .message-role {
  color: #67c23a;
}

.message-assistant .message-content {
  background: #f5f7fa;
  color: #333;
  align-self: flex-start;
  max-width: 80%;
}

.typing-indicator {
  display: flex;
  gap: 4px;
  padding: 12px 16px !important;
}

.typing-indicator span {
  width: 8px;
  height: 8px;
  background: #409eff;
  border-radius: 50%;
  animation: bounce 1.4s infinite ease-in-out both;
}

.typing-indicator span:nth-child(1) {
  animation-delay: -0.32s;
}

.typing-indicator span:nth-child(2) {
  animation-delay: -0.16s;
}

@keyframes bounce {
  0%, 80%, 100% {
    transform: scale(0);
  }
  40% {
    transform: scale(1);
  }
}

.input-container {
  padding: 20px;
  border-top: 1px solid #eee;
}

.input-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12px;
}

.input-hint {
  font-size: 12px;
  color: #999;
}
</style>
