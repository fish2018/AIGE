<template>
  <div class="admin-container">
    <el-container style="height: 100vh">
      <!-- PC端左侧栏 -->
      <el-aside width="250px" class="admin-sidebar pc-sidebar">
        <el-menu
          :default-active="activeMenu"
          class="admin-menu"
          background-color="#001529"
          text-color="#ffffff"
          active-text-color="#1890ff"
          @select="handleMenuSelect"
        >
          <el-menu-item index="overview">
            <el-icon><House /></el-icon>
            <span>概览</span>
          </el-menu-item>
          
          <el-menu-item index="users">
            <el-icon><UserIcon /></el-icon>
            <span>用户管理</span>
          </el-menu-item>

          <el-menu-item index="providers">
            <el-icon><Connection /></el-icon>
            <span>提供商管理</span>
          </el-menu-item>

          <el-menu-item index="playground">
            <el-icon><ChatDotRound /></el-icon>
            <span>操练场</span>
          </el-menu-item>

          <el-menu-item index="oauth">
            <el-icon><Key /></el-icon>
            <span>OAuth配置</span>
          </el-menu-item>
          
          <el-menu-item index="system">
            <el-icon><Setting /></el-icon>
            <span>系统设置</span>
          </el-menu-item>
        </el-menu>
      </el-aside>

      <el-container>
        <!-- 顶部导航 -->
        <el-header class="admin-header">
          <div class="header-content">
            <!-- 移动端菜单按钮 -->
            <div class="mobile-menu-btn">
              <el-button 
                type="text" 
                @click="toggleMobileMenu"
                class="menu-toggle"
                size="large"
              >
                <el-icon size="20"><Menu /></el-icon>
              </el-button>
            </div>
            
            <div class="breadcrumb">
              <el-breadcrumb separator="/" class="pc-breadcrumb">
                <el-breadcrumb-item>管理后台</el-breadcrumb-item>
                <el-breadcrumb-item>{{ getCurrentPageTitle() }}</el-breadcrumb-item>
              </el-breadcrumb>
              <!-- 移动端简化标题 -->
              <h3 class="mobile-title">{{ getCurrentPageTitle() }}</h3>
            </div>
            
            <div class="user-info">
              <!-- PC端用户信息 -->
              <el-dropdown class="pc-user-dropdown">
                <span class="el-dropdown-link">
                  <el-icon><UserFilled /></el-icon>
                  {{ authStore.user?.username }}
                  <el-icon class="el-icon--right"><arrow-down /></el-icon>
                </span>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="goToChat">
                      <el-icon><ChatDotRound /></el-icon>
                      返回主页
                    </el-dropdown-item>
                    <el-dropdown-item divided @click="logout">
                      <el-icon><SwitchButton /></el-icon>
                      退出登录
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
              
              <!-- 移动端用户菜单 -->
              <el-dropdown class="mobile-user-dropdown">
                <el-button type="text" size="large">
                  <el-icon size="20"><UserFilled /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="goToChat">
                      <el-icon><ChatDotRound /></el-icon>
                      返回主页
                    </el-dropdown-item>
                    <el-dropdown-item divided @click="logout">
                      <el-icon><SwitchButton /></el-icon>
                      退出登录
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </div>
        </el-header>

        <!-- 主内容区 -->
        <el-main class="admin-main">
          <!-- 概览页面 -->
          <div v-if="activeMenu === 'overview'" class="overview-content">
            <el-row :gutter="20">
              <el-col :xs="12" :sm="12" :md="6" :lg="6" :xl="6">
                <el-card class="stats-card">
                  <div class="stats-item">
                    <div class="stats-number">{{ totalUsers }}</div>
                    <div class="stats-label">总用户数</div>
                  </div>
                </el-card>
              </el-col>
              <el-col :xs="12" :sm="12" :md="6" :lg="6" :xl="6">
                <el-card class="stats-card">
                  <div class="stats-item">
                    <div class="stats-number">{{ adminUsers }}</div>
                    <div class="stats-label">管理员数</div>
                  </div>
                </el-card>
              </el-col>
              <el-col :xs="12" :sm="12" :md="6" :lg="6" :xl="6">
                <el-card class="stats-card">
                  <div class="stats-item">
                    <div class="stats-number">{{ totalChats }}</div>
                    <div class="stats-label">对话总数</div>
                  </div>
                </el-card>
              </el-col>
              <el-col :xs="12" :sm="12" :md="6" :lg="6" :xl="6">
                <el-card class="stats-card">
                  <div class="stats-item">
                    <div class="stats-number">Online</div>
                    <div class="stats-label">系统状态</div>
                  </div>
                </el-card>
              </el-col>
            </el-row>
          </div>

          <!-- 用户管理页面 -->
          <div v-if="activeMenu === 'users'">
            <el-card>
              <template #header>
                <div class="card-header">
                  <span>用户管理</span>
                  <el-button type="primary" @click="refreshUsers">
                    <el-icon><Refresh /></el-icon>
                    刷新
                  </el-button>
                </div>
              </template>

              <!-- PC端表格显示 -->
              <div class="table-container pc-table">
                <el-table :data="users" style="width: 100%" v-loading="loading" class="responsive-table">
                  <el-table-column prop="id" label="ID" width="80" />
                  <el-table-column prop="username" label="用户名" min-width="120" />
                  <el-table-column prop="email" label="邮箱" min-width="180" />
                  <el-table-column prop="is_admin" label="管理员" width="100">
                    <template #default="scope">
                      <el-tag :type="scope.row.is_admin ? 'success' : 'info'" size="small">
                        {{ scope.row.is_admin ? '是' : '否' }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" min-width="200">
                    <template #default="scope">
                      <div class="action-buttons">
                        <el-button
                          size="small"
                          @click="showPasswordDialog(scope.row)"
                        >
                          密码
                        </el-button>
                        <el-button
                          size="small"
                          type="warning"
                          @click="toggleAdmin(scope.row)"
                          :disabled="scope.row.id === authStore.user?.id"
                        >
                          {{ scope.row.is_admin ? '取消' : '设置' }}
                        </el-button>
                        <el-button
                          size="small"
                          type="danger"
                          @click="deleteUser(scope.row)"
                          :disabled="scope.row.id === authStore.user?.id"
                        >
                          删除
                        </el-button>
                      </div>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
              
              <!-- 移动端卡片列表显示 -->
              <div class="mobile-user-list" v-loading="loading">
                <div class="mobile-user-card" v-for="user in users" :key="user.id">
                  <div class="user-info">
                    <div class="user-header">
                      <span class="username">{{ user.username }}</span>
                      <el-tag :type="user.is_admin ? 'success' : 'info'" size="small">
                        {{ user.is_admin ? '管理员' : '普通用户' }}
                      </el-tag>
                    </div>
                    <div class="user-detail">
                      <div class="detail-item">
                        <span class="detail-label">ID:</span>
                        <span class="detail-value">{{ user.id }}</span>
                      </div>
                      <div class="detail-item">
                        <span class="detail-label">邮箱:</span>
                        <span class="detail-value">{{ user.email || '未设置' }}</span>
                      </div>
                    </div>
                  </div>
                  <div class="user-actions">
                    <el-button
                      size="small"
                      @click="showPasswordDialog(user)"
                      class="mobile-action-btn"
                    >
                      <el-icon><Key /></el-icon>
                    </el-button>
                    <el-button
                      size="small"
                      type="warning"
                      @click="toggleAdmin(user)"
                      :disabled="user.id === authStore.user?.id"
                      class="mobile-action-btn"
                    >
                      <el-icon><UserFilled /></el-icon>
                    </el-button>
                    <el-button
                      size="small"
                      type="danger"
                      @click="deleteUser(user)"
                      :disabled="user.id === authStore.user?.id"
                      class="mobile-action-btn"
                    >
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                </div>
              </div>
            </el-card>
          </div>

          <!-- 提供商管理页面 -->
          <div v-if="activeMenu === 'providers'">
            <ProviderManagement />
          </div>

          <!-- 操练场页面 -->
          <div v-if="activeMenu === 'playground'">
            <Playground />
          </div>

          <!-- OAuth配置页面 -->
          <div v-if="activeMenu === 'oauth'">
            <el-card>
              <template #header>
                <div class="card-header">
                  <span>Linux.Do OAuth 配置</span>
                  <el-button type="primary" @click="saveOAuthConfig" :loading="oauthSaving">
                    <el-icon><Check /></el-icon>
                    保存配置
                  </el-button>
                </div>
              </template>

              <el-form :model="oauthConfig" label-width="150px" class="oauth-form">
                <el-form-item label="启用状态">
                  <el-switch v-model="oauthConfig.enabled" />
                  <span style="margin-left: 10px; color: #909399;">
                    {{ oauthConfig.enabled ? '已启用' : '已禁用' }}
                  </span>
                </el-form-item>

                <el-divider />

                <el-form-item label="Client ID" required>
                  <el-input 
                    v-model="oauthConfig.client_id" 
                    placeholder="请输入 Linux.Do OAuth Client ID"
                  />
                </el-form-item>

                <el-form-item label="Client Secret" required>
                  <el-input 
                    v-model="oauthConfig.client_secret" 
                    placeholder="请输入 Linux.Do OAuth Client Secret"
                  />
                </el-form-item>

                <el-form-item label="Redirect URL">
                  <el-input 
                    v-model="oauthConfig.redirect_url" 
                    placeholder="http://localhost:3000/auth/callback/linux-do"
                  />
                  <div style="color: #909399; font-size: 12px; margin-top: 5px;">
                    需要在 Linux.Do OAuth 应用中配置此回调地址
                  </div>
                </el-form-item>

                <el-divider>高级配置</el-divider>

                <el-form-item label="Authorization URL">
                  <el-input 
                    v-model="oauthConfig.auth_url" 
                    placeholder="https://connect.linux.do/oauth2/authorize"
                  />
                </el-form-item>

                <el-form-item label="Token URL">
                  <el-input 
                    v-model="oauthConfig.token_url" 
                    placeholder="https://connect.linux.do/oauth2/token"
                  />
                </el-form-item>

                <el-form-item label="User Info URL">
                  <el-input 
                    v-model="oauthConfig.user_info_url" 
                    placeholder="https://connect.linux.do/api/user"
                  />
                </el-form-item>

                <el-alert 
                  title="配置说明" 
                  type="info" 
                  :closable="false"
                  style="margin-top: 20px;"
                >
                  <p>1. 前往 Linux.Do 开发者中心创建 OAuth 应用</p>
                  <p>2. 获取 Client ID 和 Client Secret</p>
                  <p>3. 配置回调地址为上方的 Redirect URL</p>
                  <p>4. 启用 OAuth 功能后，用户即可使用 Linux.Do 账号登录</p>
                </el-alert>
              </el-form>
            </el-card>
          </div>

          <!-- 系统设置页面 -->
          <div v-if="activeMenu === 'system'">
            <el-card>
              <template #header>
                <span>系统设置</span>
              </template>
              <el-form label-width="120px" class="system-form">
                <el-form-item label="系统名称">
                  <el-input v-model="systemSettings.siteName" placeholder="AI Chat System" />
                </el-form-item>
                <el-form-item label="系统描述">
                  <el-input 
                    type="textarea" 
                    v-model="systemSettings.siteDescription" 
                    placeholder="一个基于AI的智能对话系统"
                    :rows="3"
                  />
                </el-form-item>
                <el-form-item label="允许注册">
                  <el-switch v-model="systemSettings.allowRegister" />
                </el-form-item>
                <el-form-item class="form-actions">
                  <el-button type="primary" @click="saveSystemSettings" class="save-btn">保存设置</el-button>
                </el-form-item>
              </el-form>
            </el-card>
          </div>
        </el-main>
      </el-container>
    </el-container>

    <!-- 移动端侧边栏抽屉 -->
    <el-drawer
      v-model="mobileMenuVisible"
      direction="ltr"
      size="280px"
      :with-header="false"
      class="mobile-menu-drawer"
    >
      <div class="mobile-sidebar">
        <el-menu
          :default-active="activeMenu"
          class="mobile-admin-menu"
          background-color="#001529"
          text-color="#ffffff"
          active-text-color="#1890ff"
          @select="handleMobileMenuSelect"
        >
          <el-menu-item index="overview">
            <el-icon><House /></el-icon>
            <span>概览</span>
          </el-menu-item>
          
          <el-menu-item index="users">
            <el-icon><UserIcon /></el-icon>
            <span>用户管理</span>
          </el-menu-item>

          <el-menu-item index="providers">
            <el-icon><Connection /></el-icon>
            <span>提供商管理</span>
          </el-menu-item>

          <el-menu-item index="playground">
            <el-icon><ChatDotRound /></el-icon>
            <span>操练场</span>
          </el-menu-item>

          <el-menu-item index="oauth">
            <el-icon><Key /></el-icon>
            <span>OAuth配置</span>
          </el-menu-item>
          
          <el-menu-item index="system">
            <el-icon><Setting /></el-icon>
            <span>系统设置</span>
          </el-menu-item>
        </el-menu>
      </div>
    </el-drawer>

    <!-- 修改密码对话框 -->
    <el-dialog
      v-model="passwordDialogVisible"
      title="修改用户密码"
      width="400px"
    >
      <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef">
        <el-form-item label="用户名">
          <el-input :value="selectedUser?.username" disabled />
        </el-form-item>
        <el-form-item label="新密码" prop="newPassword">
          <el-input
            v-model="passwordForm.newPassword"
            type="password"
            placeholder="请输入新密码"
            show-password
          />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input
            v-model="passwordForm.confirmPassword"
            type="password"
            placeholder="请确认新密码"
            show-password
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="passwordDialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            @click="updatePassword"
            :loading="passwordLoading"
          >
            确认
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useAdminStore } from '@/stores/admin'
import type { User } from '@/types'
import type { FormInstance } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  House,
  User as UserIcon,
  Setting,
  UserFilled,
  ArrowDown,
  ChatDotRound,
  SwitchButton,
  Refresh,
  Connection,
  Menu,
  Key,
  Delete,
  Check
} from '@element-plus/icons-vue'
import ProviderManagement from '@/components/admin/ProviderManagement.vue'
import Playground from '@/components/admin/Playground.vue'

const router = useRouter()
const authStore = useAuthStore()
const adminStore = useAdminStore()

const loading = ref(false)
const users = ref<User[]>([])
const passwordDialogVisible = ref(false)
const passwordLoading = ref(false)
const selectedUser = ref<User | null>(null)
const passwordFormRef = ref<FormInstance>()
const activeMenu = ref('overview')
const oauthSaving = ref(false)

// 移动端状态
const mobileMenuVisible = ref(false)

// 统计数据
const totalUsers = computed(() => users.value.length)
const adminUsers = computed(() => users.value.filter(u => u.is_admin).length)
const totalChats = ref(0) // 这里可以从API获取

// 系统设置
const systemSettings = reactive({
  siteName: 'AI Game Engine',
  siteDescription: '一个基于AI的文字冒险游戏引擎',
  allowRegister: true
})

const passwordForm = reactive({
  newPassword: '',
  confirmPassword: ''
})

const validateConfirmPassword = (rule: any, value: any, callback: any) => {
  if (value !== passwordForm.newPassword) {
    callback(new Error('两次输入密码不一致'))
  } else {
    callback()
  }
}

const passwordRules = reactive({
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于 6 位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
})

onMounted(() => {
  if (!authStore.isAdmin()) {
    router.push('/chat')
    return
  }
  loadUsers()
  loadOAuthConfig()
})

const handleMenuSelect = (index: string) => {
  activeMenu.value = index
  if (index === 'users') {
    loadUsers()
  } else if (index === 'oauth') {
    loadOAuthConfig()
  }
}

// 移动端菜单相关方法
const toggleMobileMenu = () => {
  mobileMenuVisible.value = !mobileMenuVisible.value
}

const handleMobileMenuSelect = (index: string) => {
  activeMenu.value = index
  mobileMenuVisible.value = false
  if (index === 'users') {
    loadUsers()
  } else if (index === 'oauth') {
    loadOAuthConfig()
  }
}

const oauthConfig = reactive({
  client_id: '',
  client_secret: '',
  redirect_url: `${window.location.origin}/auth/callback/linux-do`,
  auth_url: 'https://connect.linux.do/oauth2/authorize',
  token_url: 'https://connect.linux.do/oauth2/token',
  user_info_url: 'https://connect.linux.do/api/user',
  enabled: false
})

const loadOAuthConfig = async () => {
  try {
    const data = await adminStore.getOAuthConfig()
    Object.assign(oauthConfig, data)
  } catch (error) {
    console.error('加载 OAuth 配置失败:', error)
  }
}

const saveOAuthConfig = async () => {
  try {
    oauthSaving.value = true
    await adminStore.saveOAuthConfig(oauthConfig)
    ElMessage.success('OAuth 配置保存成功')
  } catch (error) {
    console.error('保存 OAuth 配置失败:', error)
  } finally {
    oauthSaving.value = false
  }
}

const getCurrentPageTitle = () => {
  const titles: Record<string, string> = {
    overview: '概览',
    users: '用户管理',
    providers: '提供商管理',
    playground: '操练场',
    oauth: 'OAuth配置',
    system: '系统设置'
  }
  return titles[activeMenu.value] || '概览'
}

const loadUsers = async () => {
  loading.value = true
  try {
    users.value = await adminStore.getUsers()
  } catch (error) {
    console.error('获取用户列表失败:', error)
  } finally {
    loading.value = false
  }
}

const refreshUsers = () => {
  loadUsers()
}

const showPasswordDialog = (user: User) => {
  selectedUser.value = user
  passwordForm.newPassword = ''
  passwordForm.confirmPassword = ''
  passwordDialogVisible.value = true
}

const updatePassword = async () => {
  if (!passwordFormRef.value || !selectedUser.value) return

  try {
    const valid = await passwordFormRef.value.validate()
    if (!valid) return

    passwordLoading.value = true
    
    await adminStore.updateUserPassword(selectedUser.value.id, passwordForm.newPassword)
    
    ElMessage.success('密码修改成功')
    passwordDialogVisible.value = false
  } catch (error) {
    console.error('密码修改失败:', error)
  } finally {
    passwordLoading.value = false
  }
}

const toggleAdmin = async (user: User) => {
  try {
    await ElMessageBox.confirm(
      `确定要${user.is_admin ? '取消' : '设置'}用户 "${user.username}" 的管理员权限吗？`,
      '确认操作',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    await adminStore.toggleUserAdmin(user.id)
    ElMessage.success('权限修改成功')
    await loadUsers()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('权限修改失败:', error)
    }
  }
}

const deleteUser = async (user: User) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除用户 "${user.username}" 吗？此操作不可恢复！`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    await adminStore.deleteUser(user.id)
    ElMessage.success('用户删除成功')
    await loadUsers()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('用户删除失败:', error)
    }
  }
}

const logout = () => {
  authStore.logout()
}

const goToChat = () => {
  router.push('/chat')
}

const saveSystemSettings = () => {
  ElMessage.success('系统设置已保存')
}
</script>

<style scoped>
.admin-container {
  height: 100vh;
  background: #f0f2f5;
}

.admin-sidebar {
  background: #001529;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}


.admin-menu {
  border: none;
  height: 100vh;
  padding-top: 20px;
}

.admin-menu .el-menu-item {
  height: 56px;
  line-height: 56px;
}

.admin-menu .el-menu-item:hover {
  background-color: rgba(24, 144, 255, 0.1);
}

.admin-header {
  background: #fff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  padding: 0 24px;
  border-bottom: 1px solid #e8e8e8;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 100%;
}

.breadcrumb {
  font-size: 16px;
}

.user-info .el-dropdown-link {
  cursor: pointer;
  color: #666;
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-info .el-dropdown-link:hover {
  color: #1890ff;
}

.admin-main {
  background: #f0f2f5;
  padding: 24px;
}

.overview-content {
  margin-bottom: 24px;
}

.stats-card {
  text-align: center;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.stats-item {
  padding: 20px 0;
}

.stats-number {
  font-size: 32px;
  font-weight: bold;
  color: #1890ff;
  margin-bottom: 8px;
}

.stats-label {
  font-size: 14px;
  color: #666;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
}

.dialog-footer {
  text-align: right;
}

:deep(.el-card) {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

:deep(.el-table) {
  border-radius: 8px;
}

:deep(.el-breadcrumb__inner) {
  font-weight: normal;
}

:deep(.el-breadcrumb__inner.is-link) {
  color: #1890ff;
}

/* ============ 管理后台移动端适配 (1024px以下) ============ */
@media (max-width: 1024px) {
  /* 隐藏PC端元素 */
  .pc-sidebar {
    display: none !important;
  }
  
  .pc-breadcrumb {
    display: none;
  }
  
  .pc-user-dropdown {
    display: none;
  }
  
  .pc-table {
    display: none;
  }
  
  /* 显示移动端元素 */
  .mobile-menu-btn {
    display: block;
  }
  
  .mobile-title {
    display: block;
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: #333;
  }
  
  .mobile-user-dropdown {
    display: block;
  }
  
  .mobile-user-list {
    display: block;
  }
  
  /* 修复容器布局 */
  .admin-container {
    overflow-x: hidden;
  }
  
  /* 头部布局调整 */
  .admin-header {
    padding: 0 16px;
    height: 56px;
    position: sticky;
    top: 0;
    z-index: 100;
  }
  
  .header-content {
    height: 100%;
  }
  
  .breadcrumb {
    flex: 1;
    text-align: center;
  }
  
  /* 主内容区调整 */
  .admin-main {
    padding: 16px;
    min-width: 0;
    overflow-x: hidden;
  }
  
  /* 概览卡片调整 */
  .overview-content :deep(.el-row) {
    margin: 0 !important;
  }
  
  .overview-content :deep(.el-col) {
    margin-bottom: 16px;
    padding: 0 8px !important;
  }
  
  /* 移动端用户卡片 */
  .mobile-user-card {
    background: white;
    border-radius: 8px;
    border: 1px solid #e0e0e0;
    margin-bottom: 12px;
    padding: 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }
  
  .user-info {
    flex: 1;
    min-width: 0;
  }
  
  .user-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }
  
  .username {
    font-size: 16px;
    font-weight: 600;
    color: #333;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  
  .user-detail {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .detail-item {
    display: flex;
    font-size: 14px;
  }
  
  .detail-label {
    color: #666;
    width: 40px;
    flex-shrink: 0;
  }
  
  .detail-value {
    color: #333;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  
  .user-actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-left: 16px;
    flex-shrink: 0;
  }
  
  .mobile-action-btn {
    width: 36px;
    height: 36px;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  
  /* 表单移动端优化 */
  .system-form {
    width: 100%;
  }
  
  .system-form :deep(.el-form-item__label) {
    font-size: 14px;
    width: 100px !important;
  }
  
  .system-form :deep(.el-input__wrapper) {
    padding: 10px 12px;
  }
  
  .form-actions {
    text-align: center;
  }
  
  .save-btn {
    width: 100%;
    height: 44px;
    font-size: 16px;
  }
  
  /* 卡片优化 */
  :deep(.el-card) {
    margin-bottom: 16px;
    width: 100%;
    box-sizing: border-box;
  }
  
  :deep(.el-card__header) {
    padding: 16px;
    font-size: 16px;
  }
  
  :deep(.el-card__body) {
    padding: 16px;
    overflow-x: hidden;
  }
  
  /* 对话框优化 */
  :deep(.el-dialog) {
    width: 90% !important;
    margin: 5vh auto;
  }
  
  :deep(.el-dialog__header) {
    padding: 16px 16px 0;
  }
  
  :deep(.el-dialog__body) {
    padding: 16px;
  }
  
  :deep(.el-dialog__footer) {
    padding: 0 16px 16px;
  }
}

/* PC端样式 (1025px以上) */
@media (min-width: 1025px) {
  /* 隐藏移动端元素 */
  .mobile-menu-btn {
    display: none;
  }
  
  .mobile-title {
    display: none;
  }
  
  .mobile-user-dropdown {
    display: none;
  }
  
  .mobile-user-list {
    display: none;
  }
  
  /* 显示PC端元素 */
  .pc-sidebar {
    display: flex !important;
  }
  
  .pc-breadcrumb {
    display: block;
  }
  
  .pc-user-dropdown {
    display: block;
  }
  
  .pc-table {
    display: block;
  }
}

/* 移动端抽屉样式 */
.mobile-menu-drawer {
  z-index: 2000;
}

.mobile-menu-drawer :deep(.el-drawer__body) {
  padding: 0;
  background: #001529;
  overflow: hidden;
}

.mobile-sidebar {
  height: 100%;
  background: #001529;
}


.mobile-admin-menu {
  border: none;
  height: 100vh;
  padding-top: 20px;
}

.mobile-admin-menu .el-menu-item {
  height: 56px;
  line-height: 56px;
}

.mobile-admin-menu .el-menu-item:hover {
  background-color: rgba(24, 144, 255, 0.1);
}

/* 小屏幕进一步优化 (480px以下) */
@media (max-width: 480px) {
  .admin-header {
    padding: 0 12px;
    height: 48px;
  }
  
  .mobile-title {
    font-size: 16px;
  }
  
  .admin-main {
    padding: 12px;
  }
  
  .overview-content :deep(.el-col) {
    padding: 0 4px !important;
  }
  
  .stats-number {
    font-size: 24px !important;
  }
  
  .stats-item {
    padding: 16px 0 !important;
  }
  
  /* 移动端用户卡片小屏优化 */
  .mobile-user-card {
    padding: 12px;
    flex-direction: column;
    align-items: flex-start;
  }
  
  .user-actions {
    flex-direction: row;
    margin-left: 0;
    margin-top: 12px;
    width: 100%;
    justify-content: space-around;
  }
  
  .mobile-action-btn {
    width: 40px;
    height: 40px;
  }
  
  .username {
    font-size: 15px;
  }
  
  .detail-item {
    font-size: 13px;
  }
  
  .system-form :deep(.el-form-item__label) {
    width: 80px !important;
    font-size: 13px;
  }
  
  :deep(.el-card__header) {
    padding: 12px;
    font-size: 15px;
  }
  
  :deep(.el-card__body) {
    padding: 12px;
  }
  
  :deep(.el-dialog) {
    width: 95% !important;
    margin: 2vh auto;
  }
}
</style>