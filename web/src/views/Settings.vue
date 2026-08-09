<template>
  <el-card>
    <template #header>
      <div class="card-header">
        <span>系统设置</span>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </div>
    </template>

    <el-tabs v-model="activeTab">
      <el-tab-pane v-for="g in groups" :key="g.name" :label="g.label" :name="g.name">
        <el-form :model="form" label-width="220px" style="max-width: 720px">
          <el-form-item v-for="item in g.items" :key="item.key" :label="item.description || item.key">
            <el-switch
              v-if="isSwitch(item)"
              v-model="form[item.key]"
              active-value="1"
              inactive-value="0"
            />
            <el-input
              v-else-if="item.sensitive"
              v-model="form[item.key]"
              type="password"
              show-password
            />
            <el-input v-else-if="g.name === 'template'" v-model="form[item.key]" type="textarea" :rows="4" />
            <el-input v-else v-model="form[item.key]" />
          </el-form-item>
        </el-form>
      </el-tab-pane>
    </el-tabs>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSettings, updateSettings } from '../api/modules'

const BOOL_KEYS = ['register_enabled', 'register_email_verify', 'smtp_enabled', 'smtp_tls']

const activeTab = ref('site')
const groups = ref([])
const form = ref({})
const saving = ref(false)

const GROUP_LABELS = {
  site: '站点设置',
  security: '安全设置',
  smtp: 'SMTP 邮件',
  sms: '短信设置',
  template: '邮件模板'
}

function isSwitch(item) {
  return BOOL_KEYS.includes(item.key)
}

onMounted(async () => {
  const data = await getSettings()
  groups.value = Object.keys(GROUP_LABELS).map((name) => ({
    name,
    label: GROUP_LABELS[name],
    items: (data.groups && data.groups[name]) || []
  }))
  Object.values(data.groups || {}).forEach((items) => {
    items.forEach((item) => {
      form.value[item.key] = item.value
    })
  })
})

async function onSave() {
  saving.value = true
  try {
    const payload = []
    groups.value.forEach((g) => {
      g.items.forEach((item) => {
        payload.push({ key: item.key, value: form.value[item.key] || '', description: item.description })
      })
    })
    await updateSettings(payload)
    ElMessage.success('保存成功')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
