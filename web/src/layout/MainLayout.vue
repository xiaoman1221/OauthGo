<template>
  <el-container class="layout">
    <el-aside width="220px" class="aside">
      <div class="logo">OauthGo</div>
      <el-menu
        :default-active="$route.path"
        router
        background-color="#001529"
        text-color="#a6adb4"
        active-text-color="#ffffff"
      >
        <el-menu-item index="/dashboard">
          <el-icon><DataBoard /></el-icon>
          <span>仪表盘</span>
        </el-menu-item>
        <el-menu-item index="/apps">
          <el-icon><Grid /></el-icon>
          <span>应用管理</span>
        </el-menu-item>
        <el-menu-item index="/logins">
          <el-icon><List /></el-icon>
          <span>登录记录</span>
        </el-menu-item>
        <el-menu-item index="/notifications">
          <el-icon><Bell /></el-icon>
          <span>到期通知</span>
        </el-menu-item>
        <el-menu-item v-if="userStore.isAdmin" index="/providers">
          <el-icon><Link /></el-icon>
          <span>登录渠道</span>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Setting /></el-icon>
          <span>系统设置</span>
        </el-menu-item>
        <el-menu-item v-if="userStore.isAdmin" index="/users">
          <el-icon><User /></el-icon>
          <span>用户管理</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="header">
        <div class="header-title">{{ $route.meta.title || '统一授权管理' }}</div>
        <el-dropdown @command="onCommand">
          <span class="user-name">
            {{ userStore.userInfo?.nickname || userStore.userInfo?.username || '未登录' }}
            <el-icon><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">个人中心</el-dropdown-item>
              <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>

      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import {
  DataBoard, Grid, List, Bell, Link, Setting, User, ArrowDown
} from '@element-plus/icons-vue'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()

onMounted(() => {
  if (userStore.isLogin && !userStore.userInfo) {
    userStore.fetchUser()
  }
})

async function onCommand(command) {
  if (command === 'profile') {
    router.push('/user-center')
    return
  }
  if (command === 'logout') {
    await ElMessageBox.confirm('确定退出登录吗？', '提示', { type: 'warning' })
    userStore.logout()
    router.push('/login')
  }
}
</script>

<style scoped>
.layout {
  height: 100%;
}
.aside {
  background: #001529;
}
.logo {
  height: 60px;
  line-height: 60px;
  text-align: center;
  color: #fff;
  font-size: 20px;
  font-weight: bold;
  letter-spacing: 2px;
}
.aside :deep(.el-menu) {
  border-right: none;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
}
.header-title {
  font-size: 16px;
  font-weight: 600;
}
.user-name {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
}
.main {
  background: #f0f2f5;
  padding: 16px;
}
</style>
