<template>
  <div class="admin-container">
    <el-container style="height: 100vh">
      <!-- 左侧栏 -->
      <el-aside width="250px" class="admin-sidebar">
        <div class="logo">
          <h3>管理后台</h3>
        </div>
        
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
            <div class="breadcrumb">
              <el-breadcrumb separator="/">
                <el-breadcrumb-item>管理后台</el-breadcrumb-item>
                <el-breadcrumb-item>{{ getCurrentPageTitle() }}</el-breadcrumb-item>
              </el-breadcrumb>
            </div>
            <div class="user-info">
              <el-dropdown>
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
            </div>
          </div>
        </el-header>

        <!-- 主内容区 -->
        <el-main class="admin-main">
          <!-- 概览页面 -->
          <div v-if="activeMenu === 'overview'" class="overview-content">
            <el-row :gutter="20">
              <el-col :span="6">
                <el-card class="stats-card">
                  <div class="stats-item">
                    <div class="stats-number">{{ totalUsers }}</div>
                    <div class="stats-label">总用户数</div>
                  </div>
                </el-card>
              </el-col>
              <el-col :span="6">
                <el-card class="stats-card">
                  <div class="stats-item">
                    <div class="stats-number">{{ adminUsers }}</div>
                    <div class="stats-label">管理员数</div>
                  </div>
                </el-card>
              </el-col>
              <el-col :span="6">
                <el-card class="stats-card">
                  <div class="stats-item">
                    <div class="stats-number">{{ totalChats }}</div>
                    <div class="stats-label">对话总数</div>
                  </div>
                </el-card>
              </el-col>
              <el-col :span="6">
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

              <el-table :data="users" style="width: 100%" v-loading="loading">
                <el-table-column prop="id" label="ID" width="80" />
                <el-table-column prop="username" label="用户名" />
                <el-table-column prop="email" label="邮箱" />
                <el-table-column prop="is_admin" label="管理员" width="100">
                  <template #default="scope">
                    <el-tag :type="scope.row.is_admin ? 'success' : 'info'">
                      {{ scope.row.is_admin ? '是' : '否' }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="300">
                  <template #default="scope">
                    <el-button
                      size="small"
                      @click="showPasswordDialog(scope.row)"
                    >
                      修改密码
                    </el-button>
                    <el-button
                      size="small"
                      type="warning"
                      @click="toggleAdmin(scope.row)"
                      :disabled="scope.row.id === authStore.user?.id"
                    >
                      {{ scope.row.is_admin ? '取消管理员' : '设为管理员' }}
                    </el-button>
                    <el-button
                      size="small"
                      type="danger"
                      @click="deleteUser(scope.row)"
                      :disabled="scope.row.id === authStore.user?.id"
                    >
                      删除
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
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

          <!-- 系统设置页面 -->
          <div v-if="activeMenu === 'system'">
            <el-card>
              <template #header>
                <span>系统设置</span>
              </template>
              <el-form label-width="120px">
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
                <el-form-item>
                  <el-button type="primary" @click="saveSystemSettings">保存设置</el-button>
                </el-form-item>
              </el-form>
            </el-card>
          </div>
        </el-main>
      </el-container>
    </el-container>

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
  Connection
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
})

const handleMenuSelect = (index: string) => {
  activeMenu.value = index
  if (index === 'users') {
    loadUsers()
  }
}

const getCurrentPageTitle = () => {
  const titles: Record<string, string> = {
    overview: '概览',
    users: '用户管理',
    providers: '提供商管理',
    playground: '操练场',
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
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.1);
  margin-bottom: 20px;
}

.logo h3 {
  color: #fff;
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.admin-menu {
  border: none;
  height: calc(100vh - 84px);
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
</style>