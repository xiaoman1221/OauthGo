<template>
  <div>
    <el-card>
      <template #header>
        <div class="card-header">
          <span>通知渠道</span>
          <el-button type="primary" size="small" @click="openDialog()">新增渠道</el-button>
        </div>
      </template>

      <el-table :data="channels" border stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column prop="type" label="类型" width="110">
          <template #default="{ row }">
            <el-tag :type="row.type === 'email' ? 'primary' : 'success'">
              {{ row.type === 'email' ? '邮件' : 'Webhook' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="enabled" label="启用" width="80">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'">
              {{ row.enabled ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="onTest(row)">测试</el-button>
            <el-button size="small" @click="openDialog(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="onDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card style="margin-top: 16px">
      <template #header>
        <span>通知日志</span>
      </template>

      <el-table :data="logs" border stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="channel_name" label="渠道" width="120" />
        <el-table-column prop="subject" label="主题" min-width="160" />
        <el-table-column prop="content" label="内容" min-width="200" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">
              {{ ['待发送', '成功', '失败'][row.status] || row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="error" label="错误信息" min-width="160" show-overflow-tooltip />
        <el-table-column prop="created_at" label="时间" width="170" />
      </el-table>

      <el-pagination
        class="pagination"
        layout="total, prev, pager, next"
        :total="logTotal"
        :page-size="pageSize"
        @current-change="loadLogs"
      />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑渠道' : '新增渠道'" width="520px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="form.type">
            <el-radio value="email">邮件</el-radio>
            <el-radio value="webhook">Webhook</el-radio>
          </el-radio-group>
        </el-form-item>

        <template v-if="form.type === 'email'">
          <el-form-item label="SMTP 主机">
            <el-input v-model="form.config.smtp_host" placeholder="smtp.example.com" />
          </el-form-item>
          <el-form-item label="端口">
            <el-input-number v-model="form.config.smtp_port" :min="1" :max="65535" />
          </el-form-item>
          <el-form-item label="账号">
            <el-input v-model="form.config.username" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="form.config.password" type="password" show-password />
          </el-form-item>
          <el-form-item label="发件人">
            <el-input v-model="form.config.from" />
          </el-form-item>
          <el-form-item label="收件人">
            <el-select v-model="form.config.to" multiple filterable allow-create>
            </el-select>
          </el-form-item>
        </template>

        <template v-if="form.type === 'webhook'">
          <el-form-item label="Webhook URL">
            <el-input v-model="form.config.url" placeholder="https://example.com/hook" />
          </el-form-item>
        </template>

        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listChannels, createChannel, updateChannel, deleteChannel, testChannel, listNotificationLogs
} from '../api/modules'

const channels = ref([])
const logs = ref([])
const logTotal = ref(0)
const pageSize = ref(20)
const dialogVisible = ref(false)
const saving = ref(false)
const form = ref({})

onMounted(() => {
  loadChannels()
  loadLogs(1)
})

function emptyForm() {
  return {
    name: '',
    type: 'webhook',
    enabled: true,
    config: { url: '', smtp_host: '', smtp_port: 25, username: '', password: '', from: '', to: [] }
  }
}

async function loadChannels() {
  const data = await listChannels()
  channels.value = data.list
}

async function loadLogs(page) {
  const data = await listNotificationLogs({ page, page_size: pageSize.value })
  logs.value = data.list
  logTotal.value = data.total
}

function openDialog(row) {
  form.value = row
    ? { ...row, config: row.config ? JSON.parse(row.config) : {} }
    : emptyForm()
  dialogVisible.value = true
}

async function onSave() {
  saving.value = true
  try {
    const payload = {
      name: form.value.name,
      type: form.value.type,
      enabled: form.value.enabled,
      config: JSON.stringify(form.value.config)
    }
    if (form.value.id) {
      await updateChannel(form.value.id, payload)
    } else {
      await createChannel(payload)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    loadChannels()
  } finally {
    saving.value = false
  }
}

async function onTest(row) {
  await testChannel(row.id)
  loadLogs(1)
}

async function onDelete(row) {
  await ElMessageBox.confirm(`确定删除渠道「${row.name}」吗？`, '提示', { type: 'warning' })
  await deleteChannel(row.id)
  ElMessage.success('删除成功')
  loadChannels()
}

function statusType(status) {
  return ['info', 'success', 'danger'][status] || 'info'
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.pagination {
  margin-top: 16px;
  justify-content: flex-end;
}
</style>
