<template>
  <div>
    <el-row :gutter="16">
      <el-col :span="6" v-for="card in visibleCards" :key="card.title">
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

  <el-row :gutter="16" style="margin-top:16px">
    <el-col :span="12">
      <el-card>
        <div style="font-weight:600;margin-bottom:8px">应用平台分布</div>
        <div v-if="platformCounts.length === 0" class="muted">暂无数据</div>
        <div v-else>
          <div v-for="p in platformCounts" :key="p.k" style="display:flex;align-items:center;margin-bottom:6px">
            <div style="width:100px;color:#909399">{{ p.k }}</div>
            <div style="flex:1;height:10px;background:#f5f7fa;border-radius:4px;margin:0 8px;overflow:hidden">
              <div :style="{ width: platformTotal ? (p.v / platformTotal * 100) + '%' : '0%', background: '#409eff', height: '100%' }"></div>
            </div>
            <div style="width:40px;text-align:right">{{ p.v }}</div>
          </div>
        </div>
      </el-card>
    </el-col>
    <el-col :span="12">
      <el-card>
        <div style="font-weight:600;margin-bottom:8px">最近登录状态分布（采样）</div>
        <div v-if="loginStatusCounts.length === 0" class="muted">暂无数据</div>
        <div v-else>
          <div v-for="s in loginStatusCounts" :key="s.k" style="display:flex;align-items:center;margin-bottom:6px">
            <div style="width:100px;color:#909399">{{ s.k === '1' ? '成功' : s.k === '2' ? '失败' : s.k }}</div>
            <div style="flex:1;height:10px;background:#f5f7fa;border-radius:4px;margin:0 8px;overflow:hidden">
              <div :style="{ width: loginTotal ? (s.v / loginTotal * 100) + '%' : '0%', background: '#67c23a', height: '100%' }"></div>
            </div>
            <div style="width:40px;text-align:right">{{ s.v }}</div>
          </div>
        </div>
      </el-card>
    </el-col>
  </el-row>

  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { Grid, List, User } from '@element-plus/icons-vue'
import { listApps, listLogins, listUsers } from '../api/modules'
import { useUserStore } from '../stores/user'

const user = useUserStore()

const cards = ref([
  { title: '应用数量', value: 0, icon: Grid, color: '#409eff' },
  { title: '登录记录', value: 0, icon: List, color: '#67c23a' },
  { title: '用户数量', value: 0, icon: User, color: '#f56c6c', adminOnly: true }
])

const visibleCards = computed(() => cards.value.filter((c) => !c.adminOnly || user.isAdmin))

const platformCounts = ref([])
const loginStatusCounts = ref([])

const platformTotal = computed(() => platformCounts.value.reduce((s, c) => s + (c.v || 0), 0))
const loginTotal = computed(() => loginStatusCounts.value.reduce((s, c) => s + (c.v || 0), 0))

onMounted(async () => {
  try {
    const [apps, logins] = await Promise.all([
      listApps(),
      listLogins({ page: 1, page_size: 50 })
    ])
    cards.value[0].value = apps.list.length
    cards.value[1].value = logins.total

    // compute simple charts
    const platformCountsMap = {}
    for (const a of apps.list) {
      const p = a.platform || 'unknown'
      platformCountsMap[p] = (platformCountsMap[p] || 0) + 1
    }
    platformCounts.value = Object.entries(platformCountsMap).map(([k, v]) => ({ k, v }))

    const loginStatusMap = {}
    for (const r of (logins.list || [])) {
      const s = r.status === undefined ? 'unknown' : String(r.status)
      loginStatusMap[s] = (loginStatusMap[s] || 0) + 1
    }
    loginStatusCounts.value = Object.entries(loginStatusMap).map(([k, v]) => ({ k, v }))

    // 仅管理员显示用户统计
    if (user.isAdmin) {
      try {
        const users = await listUsers({ page: 1, page_size: 1 })
        const idx = cards.value.findIndex((c) => c.adminOnly)
        if (idx !== -1) cards.value[idx].value = users.total
      } catch (e) {}
    }
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
