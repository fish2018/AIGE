<template>
  <div class="user-management">
    <div class="page-header">
      <h3>用户管理</h3>
      <el-button type="primary" @click="refreshUsers">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <el-card>
      <el-table :data="users" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户名" />
        <el-table-column prop="email" label="邮箱" />
        <el-table-column prop="is_admin" label="管理员" width="100">
          <template #default="{ row }">
            <el-tag :type="row.is_admin ? 'success' : 'info'">
              {{ row.is_admin ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button 
              type="primary" 
              size="small" 
              @click="openPasswordDialog(row)"
            >
              修改密码
            </el-button>
            <el-button 
              v-if="row.id !== authStore.user?.id"
              type="danger" 
              size="small" 
              @click="deleteUser(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 修改密码对话框 -->
    <el-dialog
      v-model="passwordDialogVisible"
      title="修改用户密码"
      width="400px"
    >
      <el-form 
        ref="passwordFormRef"
        :model="passwordForm"
        :rules="passwordRules"
        label-width="100px"
      >
        <el-form-item label="用户名">
          <el-input :value="currentUser?.username" readonly />
        </el-form-item>
        <el-form-item label="新密码" prop="newPassword">
          <el-input
            v-model="passwordForm.newPassword"
            type="password"
            show-password
            placeholder="请输入新密码"
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="passwordDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="updating" @click="updatePassword">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import api from '@/utils/api'
import type { User } from '@/types'

const authStore = useAuthStore()

const users = ref<User[]>([])
const loading = ref(false)
const updating = ref(false)
const passwordDialogVisible = ref(false)
const currentUser = ref<User | null>(null)
const passwordFormRef = ref<FormInstance>()

const passwordForm = ref({
  newPassword: ''
})

const passwordRules: FormRules = {
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, max: 20, message: '密码长度在 6 到 20 个字符', trigger: 'blur' }
  ]
}

const loadUsers = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/users')
    users.value = response.users || []
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '加载用户列表失败')
  } finally {
    loading.value = false
  }
}

const refreshUsers = () => {
  loadUsers()
}

const openPasswordDialog = (user: User) => {
  currentUser.value = user
  passwordForm.value.newPassword = ''
  passwordDialogVisible.value = true
}

const updatePassword = async () => {
  if (!passwordFormRef.value || !currentUser.value) return
  
  const valid = await passwordFormRef.value.validate().catch(() => false)
  if (!valid) return

  updating.value = true
  try {
    await api.put(`/admin/users/${currentUser.value.id}/password`, {
      new_password: passwordForm.value.newPassword
    })
    ElMessage.success('密码修改成功')
    passwordDialogVisible.value = false
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '密码修改失败')
  } finally {
    updating.value = false
  }
}

const deleteUser = async (user: User) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除用户 "${user.username}" 吗？此操作不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    
    await api.delete(`/admin/users/${user.id}`)
    ElMessage.success('用户删除成功')
    loadUsers()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || '删除用户失败')
    }
  }
}

onMounted(() => {
  loadUsers()
})
</script>

<style scoped>
.user-management {
  background: white;
  border-radius: 8px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h3 {
  margin: 0;
  color: #333;
  font-size: 18px;
  font-weight: 600;
}
</style>