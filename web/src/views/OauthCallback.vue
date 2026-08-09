<template>
  <div class="callback-page">
    <el-card class="callback-card">
      <el-icon class="loading-icon" :size="40" color="#409eff"><Loading /></el-icon>
      <div class="text">第三方登录中，请稍候...</div>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { useUserStore } from '../stores/user'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

onMounted(async () => {
  const token = route.query.token
  if (!token) {
    ElMessage.error('登录失败：未获取到令牌')
    router.replace('/login')
    return
  }
  userStore.token = token
  localStorage.setItem('token', token)
  await userStore.fetchUser()
  ElMessage.success('登录成功')
  router.replace('/')
})
</script>

<style scoped>
.callback-page {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f0f2f5;
}
.callback-card {
  width: 320px;
  text-align: center;
  padding: 24px 0;
}
.loading-icon {
  margin-bottom: 16px;
}
.text {
  color: #909399;
}
</style>
