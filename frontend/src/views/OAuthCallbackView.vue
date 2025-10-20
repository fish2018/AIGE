<template>
  <div class="callback-container">
    <el-card class="callback-card">
      <div class="loading-content">
        <el-icon class="is-loading" :size="50">
          <Loading />
        </el-icon>
        <p>正在处理 Linux.Do 登录...</p>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

onMounted(async () => {
  const code = route.query.code as string
  const state = route.query.state as string

  if (!code || !state) {
    ElMessage.error('无效的回调参数')
    router.push('/login')
    return
  }

  try {
    await authStore.handleLinuxDoCallback(code, state)
    ElMessage.success('登录成功')
    
    if (authStore.isAdmin()) {
      router.push('/admin')
    } else {
      router.push('/game')
    }
  } catch (error: any) {
    console.error('OAuth 回调处理失败:', error)
    ElMessage.error('登录失败，请重试')
    router.push('/login')
  }
})
</script>

<style scoped>
.callback-container {
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.callback-card {
  width: 400px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
}

.loading-content {
  text-align: center;
  padding: 40px 20px;
}

.loading-content p {
  margin-top: 20px;
  font-size: 16px;
  color: #666;
}
</style>
