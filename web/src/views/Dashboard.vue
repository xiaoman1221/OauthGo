<template>
  <div>
    <el-row :gutter="16">
      <el-col :span="6" v-for="card in cards" :key="card.title">
        <el-card>
          <div class="stat">
            <el-icon class="stat-icon" :color="card.color"><component :is="card.icon" /></el-icon>
            <div>
              <div class="stat-value">{{ card.value }}</div>
              <div class="stat-title">{{ card.title }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Grid, List, Bell, User } from '@element-plus/icons-vue'
import { listApps, listLogins, listChannels, listUsers } from '../api/modules'

const cards = ref([
  { title: '应用数量', value: 0, icon: Grid, color: '#409eff' },
  { title: '登录记录', value: 0, icon: List, color: '#67c23a' },
  { title: '通知渠道', value: 0, icon: Bell, color: '#e6a23c' },
  { title: '用户数量', value: 0, icon: User, color: '#f56c6c' }
])

onMounted(async () => {
  try {
    const [apps, logins, channels, users] = await Promise.all([
      listApps(),
      listLogins({ page: 1, page_size: 1 }),
      listChannels(),
      listUsers({ page: 1, page_size: 1 })
    ])
    cards.value[0].value = apps.list.length
    cards.value[1].value = logins.total
    cards.value[2].value = channels.list.length
    cards.value[3].value = users.total
  } catch (e) {
    // 忽略统计失败
  }
})
</script>

<style scoped>
.stat {
  display: flex;
  align-items: center;
  gap: 12px;
}
.stat-icon {
  font-size: 40px;
}
.stat-value {
  font-size: 24px;
  font-weight: 600;
}
.stat-title {
  color: #909399;
  font-size: 14px;
}
</style>
