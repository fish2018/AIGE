<template>
  <div class="welcome-page">
    <el-card>
      <div class="welcome-content">
        <h2>欢迎来到管理后台</h2>
        <p>选择左侧菜单开始管理系统功能</p>
        
        <div class="quick-stats">
          <el-row :gutter="20">
            <el-col :span="8">
              <el-statistic title="注册用户" :value="stats.userCount" />
            </el-col>
            <el-col :span="8">
              <el-statistic title="今日对话" :value="stats.todayChats" />
            </el-col>
            <el-col :span="8">
              <el-statistic title="系统运行" :value="stats.uptime" suffix="天" />
            </el-col>
          </el-row>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const stats = ref({
  userCount: 0,
  todayChats: 0,
  uptime: 1
})

onMounted(() => {
  // 这里可以加载实际的统计数据
  stats.value = {
    userCount: 5,
    todayChats: 12,
    uptime: 1
  }
})
</script>

<style scoped>
.welcome-page {
  max-width: 800px;
  margin: 0 auto;
}

.welcome-content {
  text-align: center;
  padding: 40px;
}

.welcome-content h2 {
  color: #333;
  margin-bottom: 16px;
  font-size: 24px;
  font-weight: 600;
}

.welcome-content p {
  color: #666;
  margin-bottom: 40px;
  font-size: 16px;
}

.quick-stats {
  margin-top: 40px;
}
</style>