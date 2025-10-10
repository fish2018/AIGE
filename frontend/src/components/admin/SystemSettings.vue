<template>
  <div class="system-settings">
    <div class="page-header">
      <h3>系统设置</h3>
    </div>

    <el-card>
      <div class="settings-content">
        <el-tabs v-model="activeTab">
          <el-tab-pane label="游戏配置" name="game">
            <el-form label-width="140px">
              <el-form-item label="游戏使用的模型">
                <el-select 
                  v-model="gameModelId" 
                  placeholder="请选择游戏使用的AI模型"
                  @change="onGameModelChange"
                  style="width: 100%"
                >
                  <el-option
                    v-for="model in availableModels"
                    :key="model.id"
                    :label="`${model.provider?.name} - ${model.name} (${model.model_id})`"
                    :value="model.id"
                    :disabled="!model.enabled || !model.provider?.enabled"
                  >
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                      <span>{{ model.provider?.name }} - {{ model.name }}</span>
                      <el-tag v-if="!model.enabled || !model.provider?.enabled" type="info" size="small">已禁用</el-tag>
                    </div>
                  </el-option>
                </el-select>
                <div style="margin-top: 8px; color: #666; font-size: 12px;">
                  当前游戏引擎将使用此模型进行AI叙事生成。请确保选择的模型已启用且测试通过。
                </div>
              </el-form-item>
              
              <el-form-item v-if="currentGameModel" label="当前模型信息">
                <el-descriptions :column="1" border size="small">
                  <el-descriptions-item label="提供商">{{ currentGameModel.provider?.name }}</el-descriptions-item>
                  <el-descriptions-item label="模型ID">{{ currentGameModel.model_id }}</el-descriptions-item>
                  <el-descriptions-item label="API类型">{{ currentGameModel.api_type || currentGameModel.provider?.type }}</el-descriptions-item>
                  <el-descriptions-item label="测试状态">
                    <el-tag v-if="currentGameModel.test_status === 'success'" type="success" size="small">通过</el-tag>
                    <el-tag v-else-if="currentGameModel.test_status === 'failed'" type="danger" size="small">失败</el-tag>
                    <el-tag v-else type="info" size="small">未测试</el-tag>
                  </el-descriptions-item>
                </el-descriptions>
              </el-form-item>
              
              <el-form-item label="配置操作">
                <el-button type="primary" @click="reloadGameConfig" :loading="reloading">
                  <span v-if="!reloading">立即应用配置</span>
                  <span v-else>重载中...</span>
                </el-button>
                <div style="margin-top: 8px; color: #666; font-size: 12px;">
                  点击此按钮可立即应用游戏模型配置，无需重启后端服务
                </div>
              </el-form-item>
            </el-form>
          </el-tab-pane>
          
          <el-tab-pane label="基本设置" name="basic">
            <el-form label-width="120px">
              <el-form-item label="系统名称">
                <el-input v-model="settings.systemName" />
              </el-form-item>
              <el-form-item label="系统描述">
                <el-input v-model="settings.systemDescription" type="textarea" :rows="3" />
              </el-form-item>
              <el-form-item label="管理员邮箱">
                <el-input v-model="settings.adminEmail" />
              </el-form-item>
            </el-form>
          </el-tab-pane>
          
          <el-tab-pane label="安全设置" name="security">
            <el-form label-width="120px">
              <el-form-item label="密码强度">
                <el-select v-model="settings.passwordStrength">
                  <el-option label="简单（6位以上）" value="simple" />
                  <el-option label="中等（8位+数字+字母）" value="medium" />
                  <el-option label="强（8位+数字+字母+特殊字符）" value="strong" />
                </el-select>
              </el-form-item>
              <el-form-item label="登录限制">
                <el-switch v-model="settings.loginRestriction" />
                <span style="margin-left: 8px; color: #666; font-size: 12px;">
                  启用后，失败5次将锁定账户30分钟
                </span>
              </el-form-item>
            </el-form>
          </el-tab-pane>
          
          <el-tab-pane label="数据库" name="database">
            <el-descriptions title="数据库信息" :column="1">
              <el-descriptions-item label="数据库类型">SQLite</el-descriptions-item>
              <el-descriptions-item label="数据库文件">chat.db</el-descriptions-item>
              <el-descriptions-item label="文件大小">{{ dbSize }}</el-descriptions-item>
              <el-descriptions-item label="用户表">{{ tableStats.users }} 条记录</el-descriptions-item>
              <el-descriptions-item label="对话表">{{ tableStats.messages }} 条记录</el-descriptions-item>
            </el-descriptions>
            
            <el-divider />
            
            <div class="database-actions">
              <el-button type="primary" @click="backupDatabase">
                备份数据库
              </el-button>
              <el-button type="warning" @click="cleanOldData">
                清理旧数据
              </el-button>
            </div>
          </el-tab-pane>
        </el-tabs>
        
        <el-divider />
        
        <div class="settings-actions">
          <el-button type="primary" @click="saveSettings">
            保存设置
          </el-button>
          <el-button @click="resetSettings">
            重置
          </el-button>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const activeTab = ref('game')

// 游戏配置
const gameModelId = ref<number | null>(null)
const availableModels = ref<any[]>([])
const reloading = ref(false)
const currentGameModel = computed(() => {
  if (!gameModelId.value) return null
  return availableModels.value.find(m => m.id === gameModelId.value)
})

const settings = ref({
  systemName: 'AI对话系统',
  systemDescription: '基于Vue3和Go Gin的AI对话系统',
  adminEmail: 'admin@example.com',
  passwordStrength: 'medium',
  loginRestriction: true
})

const dbSize = ref('2.5 MB')
const tableStats = ref({
  users: 5,
  messages: 128
})

// 加载所有模型
async function loadModels() {
  try {
    const response = await fetch('/api/admin/models', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    
    if (response.ok) {
      const data = await response.json()
      availableModels.value = data.models || []
    }
  } catch (error) {
    console.error('加载模型列表失败:', error)
  }
}

// 加载系统配置
async function loadSystemConfig() {
  try {
    const response = await fetch('/api/admin/config', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    
    if (response.ok) {
      const configs = await response.json()
      if (configs.game_model_id) {
        gameModelId.value = parseInt(configs.game_model_id)
      }
    }
  } catch (error) {
    console.error('加载系统配置失败:', error)
  }
}

// 游戏模型变更时保存
async function onGameModelChange() {
  try {
    const response = await fetch('/api/admin/config', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({
        key: 'game_model_id',
        value: gameModelId.value?.toString() || ''
      })
    })
    
    if (response.ok) {
      ElMessage.success('游戏模型配置已保存，点击"立即应用配置"按钮生效')
    } else {
      ElMessage.error('保存配置失败')
    }
  } catch (error) {
    console.error('保存配置失败:', error)
    ElMessage.error('网络错误')
  }
}

// 重新加载游戏配置
async function reloadGameConfig() {
  try {
    reloading.value = true
    const response = await fetch('/api/admin/game/reload-config', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    
    if (response.ok) {
      ElMessage.success('游戏配置已重新加载并生效')
    } else {
      ElMessage.error('重新加载配置失败')
    }
  } catch (error) {
    console.error('重新加载配置失败:', error)
    ElMessage.error('网络错误')
  } finally {
    reloading.value = false
  }
}

const saveSettings = () => {
  ElMessage.success('设置保存成功')
}

const resetSettings = () => {
  settings.value = {
    systemName: 'AI对话系统',
    systemDescription: '基于Vue3和Go Gin的AI对话系统',
    adminEmail: 'admin@example.com',
    passwordStrength: 'medium',
    loginRestriction: true
  }
  ElMessage.info('设置已重置')
}

const backupDatabase = async () => {
  try {
    await ElMessageBox.confirm('确定要备份数据库吗？', '确认备份', {
      type: 'info'
    })
    ElMessage.success('数据库备份成功')
  } catch (error) {
    // 用户取消
  }
}

const cleanOldData = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要清理30天前的对话数据吗？此操作不可恢复。',
      '确认清理',
      { type: 'warning' }
    )
    ElMessage.success('旧数据清理完成')
  } catch (error) {
    // 用户取消
  }
}

onMounted(() => {
  loadModels()
  loadSystemConfig()
})
</script>

<style scoped>
.system-settings {
  background: white;
  border-radius: 8px;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h3 {
  margin: 0;
  color: #333;
  font-size: 18px;
  font-weight: 600;
}

.settings-content {
  padding: 20px;
}

.database-actions {
  display: flex;
  gap: 12px;
}

.settings-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}
</style>