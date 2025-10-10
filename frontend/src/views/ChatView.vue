<template>
  <div class="chat-container">
    <el-container>
      <el-header class="chat-header">
        <div class="header-content">
          <h2>AI 对话助手</h2>
          <div class="user-info">
            <span>欢迎，{{ authStore.user?.username }}</span>
            <el-dropdown>
              <el-button type="text" class="user-menu">
                <el-icon><UserFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item v-if="authStore.isAdmin()" @click="goToAdmin">
                    管理后台
                  </el-dropdown-item>
                  <el-dropdown-item @click="logout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </el-header>

      <el-main class="chat-main">
        <div class="messages-container" ref="messagesContainer">
          <div
            v-for="msg in messages"
            :key="msg.id"
            class="message-group"
          >
            <div class="user-message">
              <div class="message-content user">
                <div class="message-text">{{ msg.message }}</div>
              </div>
            </div>
            <div class="ai-message">
              <div class="message-content ai">
                <div class="message-text">{{ msg.response }}</div>
              </div>
            </div>
          </div>
          
          <div v-if="loading" class="message-group">
            <div class="ai-message">
              <div class="message-content ai">
                <div class="message-text">
                  <el-icon class="is-loading"><Loading /></el-icon>
                  AI正在思考中...
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-main>

      <el-footer class="chat-footer">
        <div class="input-container">
          <el-input
            v-model="currentMessage"
            placeholder="输入您的问题..."
            @keyup.enter="sendMessage"
            :disabled="loading"
            size="large"
          >
            <template #append>
              <el-button
                type="primary"
                @click="sendMessage"
                :loading="loading"
                :disabled="!currentMessage.trim()"
              >
                发送
              </el-button>
            </template>
          </el-input>
        </div>
      </el-footer>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useChatStore } from '@/stores/chat'
import { UserFilled, Loading } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const router = useRouter()
const authStore = useAuthStore()
const chatStore = useChatStore()

const currentMessage = ref('')
const loading = ref(false)
const messagesContainer = ref<HTMLElement>()

const messages = ref(chatStore.messages)

onMounted(async () => {
  try {
    await chatStore.loadChatHistory()
    messages.value = chatStore.messages
    await nextTick()
    scrollToBottom()
  } catch (error) {
    console.error('加载聊天记录失败:', error)
  }
})

const sendMessage = async () => {
  if (!currentMessage.value.trim() || loading.value) return

  const messageText = currentMessage.value.trim()
  currentMessage.value = ''
  loading.value = true

  try {
    await chatStore.sendMessage(messageText)
    messages.value = chatStore.messages
    await nextTick()
    scrollToBottom()
  } catch (error) {
    console.error('发送消息失败:', error)
    ElMessage.error('发送消息失败，请重试')
  } finally {
    loading.value = false
  }
}

const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

const logout = () => {
  authStore.logout()
}

const goToAdmin = () => {
  router.push('/admin')
}
</script>

<style scoped>
.chat-container {
  height: 100vh;
  background: #f5f5f5;
}

.chat-header {
  background: #fff;
  border-bottom: 1px solid #e0e0e0;
  padding: 0 20px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 100%;
}

.header-content h2 {
  margin: 0;
  color: #333;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.user-menu {
  color: #666;
  font-size: 18px;
}

.chat-main {
  padding: 0;
  overflow: hidden;
}

.messages-container {
  height: 100%;
  overflow-y: auto;
  padding: 20px;
}

.message-group {
  margin-bottom: 20px;
}

.user-message {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 10px;
}

.ai-message {
  display: flex;
  justify-content: flex-start;
}

.message-content {
  max-width: 70%;
  padding: 12px 16px;
  border-radius: 18px;
  word-wrap: break-word;
}

.message-content.user {
  background: #007bff;
  color: white;
}

.message-content.ai {
  background: white;
  color: #333;
  border: 1px solid #e0e0e0;
}

.message-text {
  line-height: 1.4;
}

.chat-footer {
  background: #fff;
  border-top: 1px solid #e0e0e0;
  padding: 20px;
}

.input-container {
  max-width: 800px;
  margin: 0 auto;
}

.is-loading {
  animation: rotating 2s linear infinite;
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>