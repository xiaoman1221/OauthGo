<template>
  <el-card>
    <div class="toolbar">
      <el-input
        v-model="keyword"
        placeholder="搜索用户名 / 昵称 / IP"
        clearable
        style="width: 240px"
        @keyup.enter="load(1)"
        @clear="load(1)"
      />
      <el-button type="primary" @click="load(1)">查询</el-button>
      <el-button @click="reset">重置</el-button>
      <div class="spacer" />
      <el-button @click="onImport">
        <el-icon><Upload /></el-icon> 导入 CSV
      </el-button>
      <el-button type="primary" @click="onExport">
        <el-icon><Download /></el-icon> 导出 CSV
      </el-button>
      <el-button
        type="danger"
        :disabled="selected.length === 0"
        @click="onBatchDelete"
      >
        批量删除
      </el-button>
    </div>

    <el-table
      :data="list"
      border
      stripe
      @selection-change="(rows) => (selected = rows)"
    >
      <el-table-column type="selection" width="45" />
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column label="应用" min-width="120">
        <template #default="{ row }">
          {{ row.app_name || 'NULL' }}
        </template>
      </el-table-column>
      <el-table-column label="用户名" min-width="150">
        <template #default="{ row }">
          <div class="uid-cell">
            <span class="uid-label">{{ row.uid_label }}</span>
            <span class="uid-value">{{ row.uid_value || '-' }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="nickname" label="昵称" min-width="110" />
      <el-table-column prop="platform" label="平台" width="90" />
      <el-table-column prop="ip" label="IP" width="130" />
      <el-table-column prop="location" label="归属地" min-width="120" />
      <el-table-column prop="login_time" label="登录时间" width="160" />
      <el-table-column prop="status" label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'">
            {{ row.status === 1 ? '成功' : '失败' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="onDetail(row)">详情</el-button>
          <el-button size="small" type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="detailVisible" title="登录记录详情" width="600px">
      <el-descriptions v-if="current" :column="2" border>
        <el-descriptions-item label="ID">{{ current.id }}</el-descriptions-item>
        <el-descriptions-item label="应用">{{ current.app_name || 'NULL' }}</el-descriptions-item>
        <el-descriptions-item label="用户名">
          <span class="uid-label">{{ current.uid_label }}</span>
          <span class="uid-value">{{ current.uid_value || '-' }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="昵称">{{ current.nickname || '-' }}</el-descriptions-item>
        <el-descriptions-item label="平台">
          <el-tag size="small">{{ current.platform || '-' }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="current.status === 1 ? 'success' : 'danger'">
            {{ current.status === 1 ? '成功' : '失败' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="头像">
          <el-avatar
            v-if="current.avatar"
            :src="current.avatar"
            :size="48"
            shape="square"
            fit="cover"
          />
          <span v-else>-</span>
        </el-descriptions-item>
        <el-descriptions-item label="IP">{{ current.ip || '-' }}</el-descriptions-item>
        <el-descriptions-item label="归属地">{{ current.location || '-' }}</el-descriptions-item>
        <el-descriptions-item label="登录时间">{{ current.login_time || '-' }}</el-descriptions-item>
        <el-descriptions-item label="User-Agent" :span="2">
          {{ current.user_agent || '-' }}
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>

    <el-pagination
      class="pagination"
      layout="total, prev, pager, next"
      :total="total"
      :page-size="pageSize"
      :current-page="page"
      @current-change="load"
    />

    <input ref="fileInput" type="file" accept=".csv" hidden @change="onFileChange" />
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Upload, Download } from '@element-plus/icons-vue'
import {
  listLogins, deleteLogin, batchDeleteLogins, exportLogins, importLogins
} from '../api/modules'

const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const selected = ref([])
const fileInput = ref()
const detailVisible = ref(false)
const current = ref(null)

onMounted(() => load(1))

async function load(p) {
  page.value = p || 1
  const data = await listLogins({
    page: page.value,
    page_size: pageSize.value,
    keyword: keyword.value
  })
  list.value = data.list
  total.value = data.total
}

function reset() {
  keyword.value = ''
  load(1)
}

function onDetail(row) {
  current.value = row
  detailVisible.value = true
}

async function onDelete(row) {
  await ElMessageBox.confirm('确定删除该条记录吗？', '提示', { type: 'warning' })
  await deleteLogin(row.id)
  ElMessage.success('删除成功')
  load(page.value)
}

async function onBatchDelete() {
  await ElMessageBox.confirm(`确定删除选中的 ${selected.value.length} 条记录吗？`, '提示', {
    type: 'warning'
  })
  await batchDeleteLogins(selected.value.map((r) => r.id))
  ElMessage.success('批量删除成功')
  load(page.value)
}

async function onExport() {
  const blob = await exportLogins({ keyword: keyword.value })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'login_records.csv'
  a.click()
  URL.revokeObjectURL(url)
}

function onImport() {
  fileInput.value.click()
}

async function onFileChange(e) {
  const file = e.target.files[0]
  if (!file) return
  const formData = new FormData()
  formData.append('file', file)
  const data = await importLogins(formData)
  ElMessage.success(`成功导入 ${data.imported} 条记录`)
  e.target.value = ''
  load(1)
}
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}
.spacer {
  flex: 1;
}
.uid-cell {
  display: flex;
  flex-direction: column;
  line-height: 1.4;
}
.uid-label {
  font-size: 12px;
  color: #909399;
}
.uid-value {
  font-size: 13px;
  color: #303133;
  word-break: break-all;
}
.pagination {
  margin-top: 16px;
  justify-content: flex-end;
}
</style>
