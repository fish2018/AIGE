<template>
  <div class="game-container">
    <!-- 游戏选择界面 -->
    <div v-if="!currentGame" class="game-selection">
      <div class="selection-header">
        <h1>选择你的冒险</h1>
        <p>在不同的世界中展开独一无二的旅程</p>
      </div>

      <div class="game-list">
        <div 
          v-for="mod in availableMods" 
          :key="mod.game_id"
          class="game-card"
          @click="selectGame(mod.game_id)"
        >
          <div class="game-card-icon">🎮</div>
          <h3>{{ mod.name }}</h3>
          <p>{{ mod.description }}</p>
          <div class="game-card-footer">
            <span class="version">v{{ mod.version }}</span>
            <span class="author">{{ mod.author }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 游戏进行界面 -->
    <div v-else class="game-play">
      <!-- 游戏头部 -->
      <header class="game-header">
        <div class="game-title">
          <h2>{{ currentModInfo?.name }}</h2>
          <span class="mod-id">{{ currentGame }}</span>
        </div>
        <div class="game-actions">
          <span class="opportunities">
            剩余机缘: <strong>{{ sessionState?.opportunities_remaining ?? 10 }}</strong>
          </span>
          <button @click="saveGame" class="btn-save" :disabled="isSaving">
            {{ isSaving ? '存档中...' : '💾 手动存档' }}
          </button>
          <button @click="showRestartConfirm" class="btn-restart" title="清空所有存档，重新开始">
            🔄 重启机缘
          </button>
          <button @click="switchGame" class="btn-secondary">切换游戏</button>
          <button @click="logout" class="btn-danger">退出</button>
        </div>
      </header>

      <div class="game-content">
        <!-- 左侧状态面板 -->
        <aside class="status-panel">
          <div class="panel-header">
            <h3>角色状态</h3>
          </div>
          <div class="panel-content">
            <div v-if="filteredCurrentLife" class="character-status">
              <div v-for="(value, key) in filteredCurrentLife" :key="key" class="status-item">
                <div class="status-key">{{ formatKey(key) }}</div>
                <div class="status-value" v-html="formatValue(value)"></div>
              </div>
            </div>
            <div v-else class="no-character">
              <p>尚未开始冒险</p>
            </div>
          </div>
        </aside>

        <!-- 右侧游戏主内容 -->
        <main class="game-main">
          <!-- 叙事窗口 -->
          <div class="narrative-window" ref="narrativeWindow">
            <div 
              v-for="(text, index) in displayHistory" 
              :key="index"
              :class="['narrative-block', getBlockClass(text)]"
              v-html="renderMarkdown(text)"
            ></div>
          </div>

          <!-- 输入区域 -->
          <footer class="input-area">
            <button 
              v-if="!sessionState?.is_in_trial && !sessionState?.daily_success_achieved" 
              @click="startTrial"
              :disabled="!wsReady || isProcessing || isRolling || (sessionState?.opportunities_remaining ?? 0) <= 0"
              class="btn-start"
            >
              {{ getStartButtonText() }}
            </button>

            <div v-else-if="sessionState?.is_in_trial" class="action-input-row">
              <input 
                v-model="userInput"
                type="text"
                placeholder="汝欲何为..."
                @keydown.enter="sendAction"
                :disabled="isProcessing || isRolling"
                class="action-input"
              />
              <button 
                @click="sendAction"
                :disabled="isProcessing || isRolling || !userInput.trim()"
                class="btn-primary"
              >
                {{ isProcessing ? '处理中...' : isRolling ? '判定中...' : '行动' }}
              </button>
            </div>

            <div v-else-if="sessionState?.daily_success_achieved" class="success-message">
              <p>🎉 今日功德圆满！明日再来。</p>
            </div>
          </footer>
        </main>
      </div>
    </div>

    <!-- 判定动画遮罩 -->
    <div v-if="showRollAnimation" class="roll-overlay">
      <div class="roll-panel">
        <div class="dice-cup">🎲</div>
        <div class="roll-details">
          <div class="roll-type">{{ rollEvent?.description || '天道裁决' }}</div>
          <div v-if="rollEvent?.target" class="roll-target">目标值: ≤{{ rollEvent.target }}</div>
        </div>
        <div v-if="rollEvent?.result !== undefined" class="roll-result-display">
          <div class="roll-outcome" :class="`outcome-${rollEvent.success ? '成功' : '失败'}`">
            {{ rollEvent.success ? '成功' : '失败' }}
          </div>
          <div class="roll-value">{{ rollEvent.result }}</div>
        </div>
      </div>
    </div>

    <!-- Loading遮罩 -->
    <div v-if="isLoading" class="loading-overlay">
      <div class="loading-spinner">
        <div class="spinner"></div>
        <div class="loading-text">{{ loadingText }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'
import { marked } from 'marked'

const router = useRouter()
const authStore = useAuthStore()

// 状态管理
const availableMods = ref<any[]>([])
const currentGame = ref<string>('')
const currentModInfo = ref<any>(null)
const gameState = ref<any>(null)
const displayHistory = computed(() => gameState.value?.display_history || [])

// 辅助computed属性，便于访问嵌套的state
const sessionState = computed(() => gameState.value?.state || gameState.value || {})

// 过滤current_life中的空值属性 - 仅支持新数据结构
const filteredCurrentLife = computed(() => {
  const currentLife = sessionState.value?.current_life
  if (!currentLife || typeof currentLife !== 'object') {
    return null
  }
  
  const filtered: Record<string, any> = {}
  //console.log('[filteredCurrentLife] 处理新数据结构，current_life:', currentLife)
  
  // 遍历current_life的所有顶层字段（属性、位置、故事事件、目标体系等）
  for (const [key, value] of Object.entries(currentLife)) {
    // 只保留有值的属性
    if (value !== null && value !== undefined && value !== '' && value !== 0) {
      // 如果是数组，检查是否为空
      if (Array.isArray(value) && value.length === 0) {
        continue
      }
      // 如果是对象，检查是否为空对象
      if (typeof value === 'object' && !Array.isArray(value) && Object.keys(value).length === 0) {
        continue
      }
      filtered[key] = value
    }
  }
  
  //console.log('[filteredCurrentLife] 过滤后的数据:', filtered)
  return Object.keys(filtered).length > 0 ? filtered : null
})
const userInput = ref('')
const isProcessing = computed(() => sessionState.value?.is_processing || false)
const isLoading = ref(false)
const loadingText = ref('加载中...')
const showRollAnimation = ref(false)
const rollEvent = ref<any>(null)
const isRolling = ref(false) // 判定期间禁用输入
const isSaving = ref(false) // 保存中状态

// WebSocket
let ws: WebSocket | null = null
const wsReady = ref(false) // 追踪WebSocket连接状态
const shouldReconnect = ref(true) // 控制是否应该重连
const narrativeWindow = ref<HTMLElement>()

// 获取可用的游戏mod列表
async function loadAvailableMods() {
  try {
    isLoading.value = true
    const response = await fetch('/api/game/mods', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      availableMods.value = await response.json()
    } else {
      ElMessage.error('加载游戏列表失败')
    }
  } catch (error) {
    console.error('加载mod列表失败:', error)
    ElMessage.error('网络错误')
  } finally {
    isLoading.value = false
  }
}

// 选择游戏
async function selectGame(modId: string) {
  shouldReconnect.value = true // 重新启用重连
  currentGame.value = modId
  currentModInfo.value = availableMods.value.find(m => m.game_id === modId)
  await initializeGame()
}

// 初始化游戏
async function initializeGame() {
  try {
    isLoading.value = true
    loadingText.value = '正在初始化游戏...'
    
    const response = await fetch('/api/game/init', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({ mod_id: currentGame.value })
    })

    if (response.ok) {
      loadingText.value = '正在加载游戏数据...'
      const data = await response.json()
      //console.log('[GameView] 初始化响应:', data)
      gameState.value = data.state || data
      //console.log('[GameView] gameState设置为:', gameState.value)
      loadingText.value = '正在建立实时连接...'
      connectWebSocket()
    } else {
      const error = await response.text()
      console.error('[GameView] 初始化失败:', error)
      ElMessage.error('初始化游戏失败: ' + error)
    }
  } catch (error) {
    console.error('初始化游戏失败:', error)
    ElMessage.error('网络错误')
  } finally {
    isLoading.value = false
  }
}

// WebSocket连接
function connectWebSocket() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  // WebSocket不支持自定义header，需要在URL中传递token
  const wsUrl = `${protocol}//${window.location.host}/api/game/ws?mod_id=${currentGame.value}&token=${authStore.token}`
  
  //console.log('[GameView] 正在连接WebSocket:', wsUrl.replace(authStore.token || '', 'TOKEN'))
  
  ws = new WebSocket(wsUrl)
  
  ws.onopen = () => {
    //console.log('[GameView] ✅ WebSocket已连接')
    wsReady.value = true
    isLoading.value = false // 连接成功后隐藏加载动画
    ElMessage.success('连接成功')
  }
  
  ws.onmessage = (event) => {
    //console.log('[GameView] 收到WebSocket消息:', event.data)
    try {
      const message = JSON.parse(event.data)
      handleWebSocketMessage(message)
    } catch (error) {
      console.error('[GameView] 解析WebSocket消息失败:', error)
    }
  }
  
  ws.onerror = (error) => {
    console.error('[GameView] ❌ WebSocket错误:', error)
    wsReady.value = false
    ElMessage.error('连接错误')
  }
  
  ws.onclose = (event) => {
    //console.log('[GameView] WebSocket已断开, code:', event.code, 'reason:', event.reason)
    wsReady.value = false
    // 自动重连 - 仅在需要重连且有当前游戏时
    if (shouldReconnect.value && currentGame.value && authStore.token) {
      setTimeout(() => {
        //console.log('[GameView] 尝试重新连接WebSocket...')
        connectWebSocket()
      }, 3000)
    }
  }
}

// 流式narrative缓冲区
const streamingNarrative = ref('')
const isStreaming = ref(false)
const pendingRollResult = ref<string | null>(null)
const secondStageNarrative = ref('')
const isSecondStageStreaming = ref(false)

// 处理WebSocket消息
function handleWebSocketMessage(message: any) {
  // 收到任何消息都隐藏加载动画
  isLoading.value = false
  
  switch (message.type) {
    case 'narrative_chunk':
      // 流式narrative开始
      if (!isStreaming.value) {
        isStreaming.value = true
        streamingNarrative.value = ''
        // 在display_history中添加一个占位项
        if (gameState.value && gameState.value.display_history) {
          gameState.value.display_history = [
            ...gameState.value.display_history,
            streamingNarrative.value
          ]
        }
      }
      // 累积内容
      streamingNarrative.value += message.data.content
      
      // 过滤掉```json标记及之后的内容（第一阶段过滤）
      let firstStageContent = streamingNarrative.value
      const jsonMarkIndex1 = firstStageContent.indexOf('```json')
      if (jsonMarkIndex1 >= 0) {
        firstStageContent = firstStageContent.substring(0, jsonMarkIndex1).trim()
      }
      const jsonStartIndex1 = firstStageContent.indexOf('```')
      if (jsonStartIndex1 >= 0) {
        firstStageContent = firstStageContent.substring(0, jsonStartIndex1).trim()
      }
      
      // 检测判定结果
      const rollResultMatch = firstStageContent.match(/【判定结果：(成功|失败)】/)
      if (rollResultMatch && !pendingRollResult.value) {
        // 暂停显示，先显示判定动画
        pendingRollResult.value = firstStageContent
        showDiceRollAnimation(rollResultMatch[1] === '成功')
        return
      }
      
      // 更新最后一项（使用过滤后的内容）
      if (gameState.value && gameState.value.display_history) {
        const lastIndex = gameState.value.display_history.length - 1
        gameState.value.display_history = [
          ...gameState.value.display_history.slice(0, lastIndex),
          firstStageContent
        ]
      }
      nextTick(() => scrollToBottom())
      break
    case 'full_state':
      // 流式结束，接收完整状态
      isStreaming.value = false
      isSecondStageStreaming.value = false
      isRolling.value = false // 结束判定，恢复输入
      
      // 保留前端的display_history（包含用户消息和流式内容）
      const frontendHistory = gameState.value?.display_history || []
      
      // 更新游戏状态，但保留前端的display_history
      gameState.value = {
        ...message.data,
        display_history: frontendHistory
      }
      
      streamingNarrative.value = ''
      secondStageNarrative.value = ''
      pendingRollResult.value = null
      nextTick(() => scrollToBottom())
      break
    case 'roll_event':
      isRolling.value = true // 开始判定，禁用输入
      showDiceRollAnimation(message.data.success, message.data)
      break
    case 'roll_result':
      // 判定结果作为单独消息显示
      if (gameState.value && gameState.value.display_history) {
        gameState.value.display_history = [
          ...gameState.value.display_history,
          message.data.content
        ]
        nextTick(() => scrollToBottom())
      }
      break
    case 'second_stage_narrative':
      // 第二阶段叙事流式累积（类似narrative_chunk）
      if (!isSecondStageStreaming.value) {
        isSecondStageStreaming.value = true
        secondStageNarrative.value = ''
        // 在display_history中添加一个占位项
        if (gameState.value && gameState.value.display_history) {
          gameState.value.display_history = [
            ...gameState.value.display_history,
            secondStageNarrative.value
          ]
        }
      }
      
      // 累积内容
      secondStageNarrative.value += message.data.content
      
      // 过滤掉```json标记及之后的内容
      let secondStageContent = secondStageNarrative.value
      const jsonMarkIndex2 = secondStageContent.indexOf('```json')
      if (jsonMarkIndex2 >= 0) {
        secondStageContent = secondStageContent.substring(0, jsonMarkIndex2).trim()
      }
      const jsonStartIndex2 = secondStageContent.indexOf('```')
      if (jsonStartIndex2 >= 0) {
        secondStageContent = secondStageContent.substring(0, jsonStartIndex2).trim()
      }
      
      // 更新最后一项
      if (gameState.value && gameState.value.display_history) {
        const lastIndex = gameState.value.display_history.length - 1
        gameState.value.display_history = [
          ...gameState.value.display_history.slice(0, lastIndex),
          secondStageContent
        ]
        nextTick(() => scrollToBottom())
      }
      break
    case 'error':
      isStreaming.value = false
      isSecondStageStreaming.value = false
      streamingNarrative.value = ''
      secondStageNarrative.value = ''
      pendingRollResult.value = null
      ElMessage.error(message.detail || '发生错误')
      break
  }
}

// 显示骰子判定动画
function showDiceRollAnimation(success: boolean, rollData?: any) {
  showRollAnimation.value = true
  
  if (rollData) {
    rollEvent.value = {
      type: rollData.type || '天道裁决',
      target: rollData.target,
      description: rollData.description || '判定中...',
      result: rollData.result,
      success: success,
      outcome: rollData.outcome
    }
  } else {
    rollEvent.value = {
      description: '判定中...',
      success: success
    }
  }
  
  // 2秒后显示结果
  setTimeout(() => {
    if (rollEvent.value) {
      rollEvent.value.description = rollData?.description || (success ? '判定成功！' : '判定失败！')
    }
  }, 1500)
  
  // 3秒后隐藏动画
  setTimeout(() => {
    showRollAnimation.value = false
    rollEvent.value = null
    
    // 恢复显示pending的narrative
    if (pendingRollResult.value && gameState.value) {
      const lastIndex = gameState.value.display_history.length - 1
      gameState.value.display_history = [
        ...gameState.value.display_history.slice(0, lastIndex),
        pendingRollResult.value
      ]
      nextTick(() => scrollToBottom())
    }
    pendingRollResult.value = null
  }, 3000)
}

// 显示判定动画
function showRollEvent(event: any) {
  rollEvent.value = event
  showRollAnimation.value = true
  
  setTimeout(() => {
    rollEvent.value = { ...event, result: event.result }
  }, 1000)
  
  setTimeout(() => {
    showRollAnimation.value = false
    rollEvent.value = null
  }, 3500)
}

// 开始试炼
async function startTrial() {
  //console.log('[GameView] startTrial 被调用，发送start_trial消息')
  
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    // 理论上不应该到这里，因为按钮已经disabled
    console.error('[GameView] WebSocket未连接')
    ElMessage.error('连接未就绪，请稍候重试')
    return
  }
  
  // 立即设置为处理状态，禁用按钮
  if (gameState.value && gameState.value.state) {
    gameState.value.state.is_processing = true
  }
  
  // 立即显示开始试炼的消息
  if (gameState.value && gameState.value.display_history) {
    gameState.value.display_history = [
      ...gameState.value.display_history,
      '> 开始试炼'
    ]
    nextTick(() => scrollToBottom())
  }
  
  // 只在开始试炼时显示短暂加载，AI开始响应后立即隐藏
  isLoading.value = true
  loadingText.value = '正在开启试炼...'
  
  ws.send(JSON.stringify({ action: 'start_trial' }))
}

// 发送行动
function sendAction() {
  if (!userInput.value.trim()) return
  
  if (ws && ws.readyState === WebSocket.OPEN) {
    const action = userInput.value.trim()
    
    // 立即设置为处理状态，禁用输入
    if (gameState.value && gameState.value.state) {
      gameState.value.state.is_processing = true
    }
    
    // 立即显示用户消息到对话框
    if (gameState.value && gameState.value.display_history) {
      gameState.value.display_history = [
        ...gameState.value.display_history,
        `> ${action}`
      ]
      nextTick(() => scrollToBottom())
    }
    
    // 发送消息到后端
    ws.send(JSON.stringify({ action: action }))
    userInput.value = ''
  }
}

// 手动保存游戏
async function saveGame() {
  if (!currentGame.value) return
  
  isSaving.value = true
  try {
    const response = await fetch('/api/game/save', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authStore.token}`
      },
      body: JSON.stringify({ mod_id: currentGame.value })
    })

    if (response.ok) {
      ElMessage.success('存档成功')
    } else {
      const error = await response.text()
      console.error('[GameView] 存档失败:', error)
      ElMessage.error('存档失败')
    }
  } catch (error) {
    console.error('[GameView] 存档异常:', error)
    ElMessage.error('存档异常')
  } finally {
    isSaving.value = false
  }
}

// 显示重启确认对话框
function showRestartConfirm() {
  if (confirm('⚠️ 确定要重启机缘吗？\n\n这将会：\n• 清空所有存档数据\n• 重置机缘次数为10次\n• 无法恢复已删除的数据\n\n确认继续？')) {
    restartOpportunities()
  }
}

// 重启机缘
async function restartOpportunities() {
  try {
    isLoading.value = true
    loadingText.value = '正在重启机缘...'
    
    //console.log('[GameView] 重启机缘 - token:', authStore.token ? 'exists' : 'missing')
    
    const response = await fetch('/api/game/restart-opportunities', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json'
      }
    })

    if (response.ok) {
      const data = await response.json()
      //console.log('[GameView] 重启机缘成功:', data)
      
      // 停止WebSocket连接
      shouldReconnect.value = false
      if (ws) {
        ws.close()
        ws = null
      }
      wsReady.value = false
      
      // 清空当前游戏状态
      currentGame.value = ''
      gameState.value = null
      
      // 重新加载可用模组
      await loadAvailableMods()
      
      ElMessage.success(`机缘已重启！删除了 ${data.deleted_saves} 条存档`)
    } else {
      const error = await response.text()
      console.error('[GameView] 重启机缘失败 - 状态码:', response.status)
      console.error('[GameView] 重启机缘失败 - 响应:', error)
      console.error('[GameView] 重启机缘失败 - token长度:', authStore.token?.length || 0)
      
      if (response.status === 401) {
        ElMessage.error('认证失败，请重新登录')
        // 如果是认证失败，可能需要重新登录
        authStore.logout()
        router.push('/login')
      } else {
        ElMessage.error('重启机缘失败: ' + error)
      }
    }
  } catch (error) {
    console.error('重启机缘失败:', error)
    ElMessage.error('网络错误')
  } finally {
    isLoading.value = false
  }
}

// 切换游戏
async function switchGame() {
  // 停止自动重连
  shouldReconnect.value = false
  
  // 先保存当前游戏
  if (currentGame.value) {
    await saveGame()
  }
  
  if (ws) {
    ws.close()
    ws = null
  }
  wsReady.value = false
  currentGame.value = ''
  gameState.value = null
}

// 退出
async function logout() {
  // 停止自动重连
  shouldReconnect.value = false
  
  // 退出前保存游戏
  if (currentGame.value) {
    await saveGame()
  }
  
  if (ws) {
    ws.close()
    ws = null
  }
  authStore.logout()
  router.push('/login')
}

// Markdown渲染
function renderMarkdown(text: string): string {
  try {
    // 先将转义的\n替换为真正的换行符
    let processedText = text.replace(/\\n/g, '\n')
    
    // 检测并保护ASCII艺术块
    processedText = protectAsciiArt(processedText)
    
    return marked.parse(processedText, { breaks: true, gfm: true }) as string
  } catch (error) {
    return text.replace(/\\n/g, '<br>').replace(/\n/g, '<br>')
  }
}

// 保护ASCII艺术格式
function protectAsciiArt(text: string): string {
  // 检测包含ASCII艺术的行模式
  const lines = text.split('\n')
  const result: string[] = []
  let inAsciiBlock = false
  let asciiBlock: string[] = []
  
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    
    // 检测ASCII艺术特征：包含多个连续空格和特殊字符的行
    const isAsciiLine = /[\s]{4,}.*[│┼↑↓←→─]|[\s]{4,}.*[├┤┬┴]|^\s*[│┼↑↓←→─├┤┬┴]/.test(line) ||
                       /^\s*[│┼↑↓←→─├┤┬┴].*[\s]{2,}/.test(line) ||
                       /^\s{8,}.*[↑↓←→]/.test(line) ||
                       line.includes('───') || line.includes('│') || line.includes('┼')
    
    if (isAsciiLine && !inAsciiBlock) {
      // 开始ASCII块
      inAsciiBlock = true
      asciiBlock = [line]
    } else if (isAsciiLine && inAsciiBlock) {
      // 继续ASCII块
      asciiBlock.push(line)
    } else if (!isAsciiLine && inAsciiBlock) {
      // 结束ASCII块
      inAsciiBlock = false
      // 将ASCII块包装在pre标签中
      result.push('\n```\n' + asciiBlock.join('\n') + '\n```\n')
      asciiBlock = []
      result.push(line)
    } else {
      // 普通文本行
      result.push(line)
    }
  }
  
  // 处理结尾的ASCII块
  if (inAsciiBlock && asciiBlock.length > 0) {
    result.push('\n```\n' + asciiBlock.join('\n') + '\n```\n')
  }
  
  return result.join('\n')
}

// 获取消息块样式类
function getBlockClass(text: string): string {
  if (text.startsWith('> ')) return 'user-message'
  if (text.startsWith('【判定结果：')) return 'roll-result-message'
  if (text.startsWith('【')) return 'system-message'
  return 'narrative-message'
}

// 格式化键名 - 通用实现，根据内容类型自动选择图标
function formatKey(key: string): string {
  // 通用规则：根据 key 的语义自动选择合适的图标
  const iconRules: Array<{ pattern: RegExp, icon: string }> = [
    { pattern: /^(姓名|名称|角色)$/i, icon: '🧙‍♂️' },
    { pattern: /^(修为|经验|exp)$/i, icon: '⚡' },
    { pattern: /^(境界|等级|level|阶位)$/i, icon: '🏔️' },
    { pattern: /^(生命|血量|hp|health)$/i, icon: '❤️' },
    { pattern: /^(寿元|寿命|年龄|age)$/i, icon: '⏳' },
    { pattern: /^(灵石|金币|货币|money|gold)$/i, icon: '💎' },
    { pattern: /^(位置|地点|location)$/i, icon: '📍' },
    { pattern: /^(出身|背景|origin)$/i, icon: '🏡' },
    { pattern: /^(天赋|才能|talent)$/i, icon: '⭐' },
    { pattern: /^(灵根|资质|根骨)$/i, icon: '🌿' },
    { pattern: /^(属性|状态|stats)$/i, icon: '📊' },
    { pattern: /^(功法|技能|skill)$/i, icon: '📚' },
    { pattern: /^(物品|道具|背包|inventory)$/i, icon: '🎒' },
    { pattern: /^(状态|效果|buff|debuff)$/i, icon: '💫' },
    { pattern: /^(事件|历史|故事|history)$/i, icon: '📜' },
    { pattern: /^(目标|任务|quest)$/i, icon: '🎯' },
    { pattern: /^(关系|人际|社交)$/i, icon: '👥' },
    { pattern: /^(药园|农场|种植)$/i, icon: '🌱' },
    { pattern: /^(悟性|智力|intelligence)$/i, icon: '🧠' },
    { pattern: /^(气运|运气|luck)$/i, icon: '🍀' },
    { pattern: /^(声望|名望|reputation)$/i, icon: '⭐' }
  ]
  
  // 尝试匹配规则
  for (const rule of iconRules) {
    if (rule.pattern.test(key)) {
      return `${rule.icon} ${key}`
    }
  }
  
  // 默认图标
  return `📋 ${key}`
}

// 格式化值
function formatValue(value: any): string {
  if (value === null || value === undefined) {
    return '<span class="empty-value">无</span>'
  }
  
  if (typeof value === 'string') {
    return value
  }
  
  if (typeof value === 'number') {
    return value.toString()
  }
  
  if (Array.isArray(value)) {
    if (value.length === 0) {
      return '<span class="empty-value">无</span>'
    }
    
    return value.map(item => {
      if (typeof item === 'string') {
        return `<div class="array-item">• ${item}</div>`
      } else if (typeof item === 'object' && item !== null) {
        if (item.名称 && item.数量) {
          return `<div class="item-entry">• ${item.名称} × ${item.数量}${item.说明 ? `<br><span class="item-desc">${item.说明}</span>` : ''}</div>`
        } else {
          return `<div class="array-item">• ${formatObjectInline(item)}</div>`
        }
      }
      return `<div class="array-item">• ${item}</div>`
    }).join('')
  }
  
  if (typeof value === 'object') {
    // 特殊处理"目标体系"对象
    if (value.人生目标 || value.立即目标 || value.阶段目标) {
      let html = '<div class="goal-system-container">'
      
      // 人生目标
      if (value.人生目标) {
        html += '<div class="goal-section goal-life">'
        html += '<div class="goal-section-title">🌟 人生目标</div>'
        html += '<div class="goal-section-content">'
        if (value.人生目标.描述) html += `<div class="goal-field"><span class="field-label">描述：</span>${value.人生目标.描述}</div>`
        if (value.人生目标.核心动机) html += `<div class="goal-field"><span class="field-label">核心动机：</span>${value.人生目标.核心动机}</div>`
        if (value.人生目标.最终愿景) html += `<div class="goal-field"><span class="field-label">最终愿景：</span>${value.人生目标.最终愿景}</div>`
        if (value.人生目标.宿命纠葛 && Array.isArray(value.人生目标.宿命纠葛) && value.人生目标.宿命纠葛.length > 0) {
          html += `<div class="goal-field"><span class="field-label">宿命纠葛：</span><div class="field-list">${value.人生目标.宿命纠葛.map((item: string) => `<div class="list-item">• ${item}</div>`).join('')}</div></div>`
        }
        html += '</div></div>'
      }
      
      // 立即目标
      if (value.立即目标) {
        html += '<div class="goal-section goal-immediate">'
        html += '<div class="goal-section-title">⚡ 立即目标</div>'
        html += '<div class="goal-section-content">'
        if (value.立即目标.描述) html += `<div class="goal-field"><span class="field-label">描述：</span>${value.立即目标.描述}</div>`
        if (value.立即目标.紧迫程度) html += `<div class="goal-field"><span class="field-label">紧迫程度：</span><span class="urgency-badge urgency-${value.立即目标.紧迫程度}">${value.立即目标.紧迫程度}</span></div>`
        if (value.立即目标.完成条件) html += `<div class="goal-field"><span class="field-label">完成条件：</span>${value.立即目标.完成条件}</div>`
        if (value.立即目标.奖励预期) html += `<div class="goal-field"><span class="field-label">奖励预期：</span>${value.立即目标.奖励预期}</div>`
        html += '</div></div>'
      }
      
      // 阶段目标
      if (value.阶段目标) {
        html += '<div class="goal-section goal-stage">'
        html += '<div class="goal-section-title">📈 阶段目标</div>'
        html += '<div class="goal-section-content">'
        if (value.阶段目标.描述) html += `<div class="goal-field"><span class="field-label">描述：</span>${value.阶段目标.描述}</div>`
        if (value.阶段目标.关键节点 && Array.isArray(value.阶段目标.关键节点) && value.阶段目标.关键节点.length > 0) {
          html += `<div class="goal-field"><span class="field-label">关键节点：</span><div class="field-list">${value.阶段目标.关键节点.map((item: string) => `<div class="list-item">• ${item}</div>`).join('')}</div></div>`
        }
        if (value.阶段目标.障碍分析) html += `<div class="goal-field"><span class="field-label">障碍分析：</span>${value.阶段目标.障碍分析}</div>`
        if (value.阶段目标.解决路径) html += `<div class="goal-field"><span class="field-label">解决路径：</span>${value.阶段目标.解决路径}</div>`
        html += '</div></div>'
      }
      
      html += '</div>'
      return html
    }
    
    // 特殊处理"属性"对象（包含角色基础属性）
    if (value.姓名 || value.境界 || value.修为 || value.灵根) {
      return formatAttributesObject(value)
    }
    
    // 通用对象处理：递归展开显示所有键值对
    return formatObjectExpanded(value)
  }
  
  return String(value)
}

// 格式化"属性"对象
function formatAttributesObject(value: any): string {
  let html = '<div class="attributes-expanded">'
  const attributeOrder = ['姓名', '出身', '灵根', '境界', '寿元', '生命值', '修为', '灵石', '根骨', '悟性', '气运', '初始天赋', '功法', '神通', '物品', '状态效果', '关系网', '声望', '特殊标记', '灵兽', '道侣', '洞府', '药园']
  
  // 按顺序显示属性
  attributeOrder.forEach(key => {
    if (value[key] !== undefined && value[key] !== null && value[key] !== '' && 
        !(Array.isArray(value[key]) && value[key].length === 0) &&
        !(typeof value[key] === 'object' && !Array.isArray(value[key]) && Object.keys(value[key]).length === 0)) {
      html += `<div class="attr-sub-item">
        <div class="attr-sub-key">${formatKey(key)}</div>
        <div class="attr-sub-value">${formatValue(value[key])}</div>
      </div>`
    }
  })
  
  // 显示其他未列出的属性
  Object.keys(value).forEach(key => {
    if (!attributeOrder.includes(key) && value[key] !== undefined && value[key] !== null && value[key] !== '' &&
        !(Array.isArray(value[key]) && value[key].length === 0) &&
        !(typeof value[key] === 'object' && !Array.isArray(value[key]) && Object.keys(value[key]).length === 0)) {
      html += `<div class="attr-sub-item">
        <div class="attr-sub-key">${formatKey(key)}</div>
        <div class="attr-sub-value">${formatValue(value[key])}</div>
      </div>`
    }
  })
  
  html += '</div>'
  return html
}

// 通用对象展开格式化 - 递归处理所有嵌套结构
function formatObjectExpanded(obj: any, depth: number = 0): string {
  const entries = Object.entries(obj).filter(([key, value]) => {
    // 过滤空值
    if (value === null || value === undefined || value === '') return false
    if (Array.isArray(value) && value.length === 0) return false
    if (typeof value === 'object' && !Array.isArray(value) && Object.keys(value).length === 0) return false
    return true
  })
  
  if (entries.length === 0) return '<span class="empty-value">无</span>'
  
  let html = '<div class="object-expanded">'
  
  entries.forEach(([key, value]) => {
    html += '<div class="object-item">'
    html += `<div class="object-key">${formatKey(key)}</div>`
    html += '<div class="object-value">'
    
    // 递归处理值
    if (value === null || value === undefined) {
      html += '<span class="empty-value">无</span>'
    } else if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
      html += String(value)
    } else if (Array.isArray(value)) {
      html += formatArrayValue(value, depth + 1)
    } else if (typeof value === 'object') {
      // 嵌套对象继续展开
      html += formatObjectExpanded(value, depth + 1)
    } else {
      html += String(value)
    }
    
    html += '</div>'
    html += '</div>'
  })
  
  html += '</div>'
  return html
}

// 格式化数组值
function formatArrayValue(arr: any[], depth: number = 0): string {
  if (arr.length === 0) return '<span class="empty-value">无</span>'
  
  let html = '<div class="array-container">'
  
  arr.forEach(item => {
    html += '<div class="array-item-wrapper">'
    
    if (typeof item === 'string' || typeof item === 'number' || typeof item === 'boolean') {
      html += `<div class="array-item-simple">• ${item}</div>`
    } else if (typeof item === 'object' && item !== null) {
      // 对象项：展开显示
      html += '<div class="array-item-object">'
      html += formatObjectInline(item)
      html += '</div>'
    } else {
      html += `<div class="array-item-simple">• ${item}</div>`
    }
    
    html += '</div>'
  })
  
  html += '</div>'
  return html
}

// 内联格式化对象（用于数组中的对象）
function formatObjectInline(obj: any): string {
  const entries = Object.entries(obj)
  if (entries.length === 0) return '<span class="empty-value">无</span>'
  
  let html = '<div class="inline-object">'
  
  entries.forEach(([key, value]) => {
    if (value === null || value === undefined || value === '') return
    
    html += '<div class="inline-item">'
    html += `<span class="inline-key">${key}:</span> `
    
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
      html += `<span class="inline-value">${value}</span>`
    } else if (Array.isArray(value)) {
      html += `<span class="inline-value">[${value.length}项]</span>`
    } else if (typeof value === 'object') {
      html += `<span class="inline-value">{对象}</span>`
    } else {
      html += `<span class="inline-value">${value}</span>`
    }
    
    html += '</div>'
  })
  
  html += '</div>'
  return html
}

// 获取开始按钮文本
function getStartButtonText(): string {
  if (!wsReady.value) return '连接中...'
  const opps = sessionState.value?.opportunities_remaining ?? 10
  //console.log('[GameView] getStartButtonText - opportunities_remaining:', opps)
  if (opps <= 0) return '机缘已尽'
  if (opps === 10) return '开始第一次试炼'
  return '开启下一次试炼'
}

// 滚动到底部
function scrollToBottom() {
  if (narrativeWindow.value) {
    narrativeWindow.value.scrollTop = narrativeWindow.value.scrollHeight
  }
}

// 生命周期
onMounted(() => {
  loadAvailableMods()
})

onUnmounted(() => {
  // 停止自动重连
  shouldReconnect.value = false
  
  if (ws) {
    ws.close()
    ws = null
  }
  wsReady.value = false
})
</script>

<style scoped>
.game-container {
  width: 100%;
  height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  overflow: hidden;
}

/* 游戏选择界面 */
.game-selection {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 2rem;
}

.selection-header {
  text-align: center;
  margin-bottom: 3rem;
  color: white;
}

.selection-header h1 {
  font-size: 3rem;
  margin-bottom: 1rem;
  text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.3);
}

.selection-header p {
  font-size: 1.2rem;
  opacity: 0.9;
}

.game-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 2rem;
  max-width: 1200px;
  width: 100%;
}

.game-card {
  background: rgba(255, 255, 255, 0.95);
  border-radius: 12px;
  padding: 2rem;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}

.game-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 12px rgba(0, 0, 0, 0.2);
}

.game-card-icon {
  font-size: 3rem;
  text-align: center;
  margin-bottom: 1rem;
}

.game-card h3 {
  font-size: 1.5rem;
  margin-bottom: 0.5rem;
  color: #333;
}

.game-card p {
  color: #666;
  margin-bottom: 1rem;
  min-height: 3rem;
}

.game-card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 1rem;
  border-top: 1px solid #eee;
  font-size: 0.9rem;
  color: #999;
}

/* 游戏进行界面 */
.game-play {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: white;
}

.game-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 2rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.game-title h2 {
  margin: 0;
  font-size: 1.5rem;
}

.mod-id {
  font-size: 0.9rem;
  opacity: 0.8;
}

.game-actions {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.opportunities {
  font-size: 1rem;
}

.opportunities strong {
  font-size: 1.2rem;
  color: #ffd700;
}

.game-content {
  flex: 1;
  display: grid;
  grid-template-columns: 300px 1fr;
  gap: 2px;
  background: #e0e0e0;
  overflow: hidden;
}

/* 状态面板 */
.status-panel {
  background: #f8f9fa;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-header {
  padding: 1rem;
  background: #667eea;
  color: white;
}

.panel-header h3 {
  margin: 0;
  font-size: 1.2rem;
}

.panel-content {
  flex: 1;
  padding: 1rem;
  overflow-y: auto;
}

.character-status {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.status-item {
  margin-bottom: 0.75rem;
  padding: 0.75rem;
  background: white;
  border-radius: 8px;
  border-left: 3px solid #667eea;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  font-size: 0.9rem;
}

.status-key {
  font-weight: bold;
  color: #667eea;
  margin-bottom: 0.5rem;
  font-size: 1rem;
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.status-value {
  color: #333;
  line-height: 1.4;
}

/* 特殊格式化样式 */
.empty-value {
  color: #999;
  font-style: italic;
}

/* 通用对象展开样式 */
.object-expanded {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.object-item {
  padding: 0.5rem;
  background: rgba(102, 126, 234, 0.05);
  border-radius: 6px;
  border-left: 2px solid rgba(102, 126, 234, 0.3);
}

.object-key {
  font-weight: 600;
  color: #667eea;
  margin-bottom: 0.3rem;
  font-size: 0.85rem;
}

.object-value {
  color: #555;
  font-size: 0.85rem;
  line-height: 1.4;
}

/* 属性展开样式 */
.attributes-expanded {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.attr-sub-item {
  padding: 0.5rem;
  background: rgba(102, 126, 234, 0.05);
  border-radius: 6px;
  border-left: 2px solid rgba(102, 126, 234, 0.3);
}

.attr-sub-key {
  font-weight: 600;
  color: #667eea;
  margin-bottom: 0.3rem;
  font-size: 0.85rem;
}

.attr-sub-value {
  color: #555;
  font-size: 0.85rem;
  line-height: 1.4;
}

/* 目标体系样式 */
.goal-system-container {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.goal-section {
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
}

.goal-section-title {
  font-weight: bold;
  font-size: 0.95rem;
  padding: 0.6rem 0.75rem;
  color: white;
}

.goal-section-content {
  padding: 0.75rem;
  background: white;
}

.goal-life .goal-section-title {
  background: linear-gradient(135deg, #ff6b6b 0%, #ee5a6f 100%);
}

.goal-immediate .goal-section-title {
  background: linear-gradient(135deg, #ffa726 0%, #fb8c00 100%);
}

.goal-stage .goal-section-title {
  background: linear-gradient(135deg, #4caf50 0%, #388e3c 100%);
}

.goal-field {
  margin-bottom: 0.5rem;
  font-size: 0.85rem;
  line-height: 1.5;
}

.goal-field:last-child {
  margin-bottom: 0;
}

.field-label {
  font-weight: 600;
  color: #666;
  margin-right: 0.25rem;
}

.field-list {
  margin-top: 0.3rem;
  padding-left: 0.5rem;
}

.list-item {
  color: #555;
  padding: 0.15rem 0;
}

.urgency-badge {
  display: inline-block;
  padding: 0.15rem 0.5rem;
  border-radius: 4px;
  font-weight: bold;
  font-size: 0.8rem;
}

.urgency-高 {
  background: #ffebee;
  color: #c62828;
  border: 1px solid #ffcdd2;
}

.urgency-中 {
  background: #fff3e0;
  color: #f57c00;
  border: 1px solid #ffcc80;
}

.urgency-低 {
  background: #e8f5e9;
  color: #2e7d32;
  border: 1px solid #c8e6c9;
}

/* 数组容器样式 */
.array-container {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.array-item-wrapper {
  padding: 0.25rem 0;
}

.array-item-simple {
  color: #555;
  padding: 0.2rem 0;
  border-bottom: 1px solid #f0f0f0;
}

.array-item-simple:last-child {
  border-bottom: none;
}

.array-item-object {
  background: rgba(255, 255, 255, 0.5);
  padding: 0.4rem;
  border-radius: 4px;
  border: 1px solid rgba(102, 126, 234, 0.15);
}

/* 内联对象样式（用于数组中的对象） */
.inline-object {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.inline-item {
  display: flex;
  gap: 0.5rem;
  font-size: 0.85rem;
}

.inline-key {
  font-weight: 500;
  color: #667eea;
  min-width: 60px;
}

.inline-value {
  color: #555;
  flex: 1;
}

.no-character {
  text-align: center;
  color: #999;
  padding: 2rem;
}

/* 游戏主内容 */
.game-main {
  background: white;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.narrative-window {
  flex: 1;
  padding: 2rem;
  overflow-y: auto;
  scroll-behavior: smooth;
}

.narrative-block {
  margin-bottom: 1.5rem;
  padding: 1rem;
  border-radius: 8px;
  line-height: 1.6;
}

/* ASCII艺术保护样式 */
.narrative-block pre {
  background: rgba(0, 0, 0, 0.05);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 4px;
  padding: 1rem;
  font-family: 'Courier New', Monaco, 'Lucida Console', monospace;
  font-size: 0.9rem;
  line-height: 1.2;
  white-space: pre;
  overflow-x: auto;
  color: #2c3e50;
}

.narrative-block code {
  font-family: 'Courier New', Monaco, 'Lucida Console', monospace;
  font-size: 0.9rem;
  white-space: pre;
}

.narrative-message {
  background: #f8f9fa;
}

.user-message {
  background: #e3f2fd;
  border-left: 4px solid #2196f3;
}

.system-message {
  background: #fff3e0;
  border-left: 4px solid #ff9800;
  font-weight: bold;
}

.roll-result-message {
  background: #f3e5f5;
  border-left: 4px solid #9c27b0;
  font-weight: bold;
  text-align: center;
  font-size: 1.1rem;
}

/* 输入区域 */
.input-area {
  padding: 1.5rem;
  background: #f8f9fa;
  border-top: 2px solid #e0e0e0;
}

.action-input-row {
  display: flex;
  gap: 1rem;
}

.action-input {
  flex: 1;
  padding: 0.75rem 1rem;
  border: 2px solid #ddd;
  border-radius: 8px;
  font-size: 1rem;
  transition: border-color 0.3s;
}

.action-input:focus {
  outline: none;
  border-color: #667eea;
}

.btn-start {
  width: 100%;
  padding: 1rem;
  font-size: 1.2rem;
  font-weight: bold;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-start:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.btn-start:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  padding: 0.75rem 2rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: bold;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-save {
  padding: 0.5rem 1rem;
  background: rgba(76, 175, 80, 0.9);
  color: white;
  border: 1px solid rgba(76, 175, 80, 1);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.3s;
  font-weight: 500;
}

.btn-save:hover:not(:disabled) {
  background: rgba(76, 175, 80, 1);
  box-shadow: 0 2px 8px rgba(76, 175, 80, 0.3);
}

.btn-save:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-restart {
  padding: 0.5rem 1rem;
  background: rgba(255, 152, 0, 0.9);
  color: white;
  border: 1px solid rgba(255, 152, 0, 1);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.3s;
  font-weight: 500;
}

.btn-restart:hover {
  background: rgba(255, 152, 0, 1);
  box-shadow: 0 2px 8px rgba(255, 152, 0, 0.3);
  transform: translateY(-1px);
}

.btn-secondary {
  padding: 0.5rem 1rem;
  background: rgba(255, 255, 255, 0.2);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-secondary:hover {
  background: rgba(255, 255, 255, 0.3);
}

.btn-danger {
  padding: 0.5rem 1rem;
  background: rgba(244, 67, 54, 0.8);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-danger:hover {
  background: rgba(244, 67, 54, 1);
}

.success-message {
  text-align: center;
  padding: 2rem;
  font-size: 1.5rem;
  color: #4caf50;
}

/* 判定动画 */
.roll-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.3s;
}

.roll-panel {
  background: white;
  padding: 3rem;
  border-radius: 16px;
  text-align: center;
  min-width: 300px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

.dice-cup {
  font-size: 5rem;
  animation: diceRoll 1s cubic-bezier(.36,.07,.19,.97) infinite;
  margin-bottom: 1rem;
}

.roll-details {
  margin: 1.5rem 0;
}

.roll-type {
  font-size: 1.4rem;
  font-weight: bold;
  color: #2c3e50;
  margin-bottom: 0.5rem;
}

.roll-target {
  font-size: 1.1rem;
  color: #7f8c8d;
  margin-bottom: 1rem;
}

.roll-result-display {
  margin-top: 2rem;
}

.roll-outcome {
  font-size: 2rem;
  font-weight: bold;
  margin-bottom: 1rem;
}

.outcome-成功 {
  color: #27ae60;
  text-shadow: 0 0 10px rgba(39, 174, 96, 0.3);
}

.outcome-失败 {
  color: #e74c3c;
  text-shadow: 0 0 10px rgba(231, 76, 60, 0.3);
}

.roll-value {
  font-size: 2.5rem;
  font-weight: bold;
  color: #34495e;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

@keyframes diceRoll {
  0% { transform: rotate(0deg) scale(1); }
  25% { transform: rotate(90deg) scale(1.1); }
  50% { transform: rotate(180deg) scale(1); }
  75% { transform: rotate(270deg) scale(1.1); }
  100% { transform: rotate(360deg) scale(1); }
}

.roll-info {
  margin: 1.5rem 0;
}

.roll-type {
  font-size: 1.5rem;
  font-weight: bold;
  color: #667eea;
  margin-bottom: 0.5rem;
}

.roll-target {
  font-size: 1rem;
  color: #666;
}

.roll-result {
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  border-top: 2px solid #eee;
}

.roll-outcome {
  font-size: 2rem;
  font-weight: bold;
  margin-bottom: 0.5rem;
}

.outcome-成功 { color: #4caf50; }
.outcome-大成功 { color: #ffd700; }
.outcome-失败 { color: #f44336; }
.outcome-大失败 { color: #b71c1c; }

.roll-value {
  font-size: 1.2rem;
  color: #666;
}

/* Loading遮罩 */
.loading-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  backdrop-filter: blur(5px);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.loading-spinner {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1.5rem;
}

.spinner {
  width: 60px;
  height: 60px;
  border: 4px solid rgba(255, 255, 255, 0.3);
  border-top: 4px solid #d4af37;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.loading-text {
  color: white;
  font-size: 1.2rem;
  text-align: center;
  text-shadow: 0 0 10px rgba(212, 175, 55, 0.5);
  font-weight: 500;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading-overlay p {
  margin-top: 1rem;
  color: white;
  font-size: 1.2rem;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}
</style>
