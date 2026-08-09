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
          <el-form-item
            v-for="item in groupItems(g)"
            :key="item.key"
            :label="item.description || item.key"
          >
            <el-select
              v-if="item.key === 'sms_provider'"
              v-model="form[item.key]"
              style="width: 240px"
            >
              <el-option
                v-for="p in SMS_PROVIDERS"
                :key="p.value"
                :label="p.label"
                :value="p.value"
              />
            </el-select>
            <el-switch
              v-else-if="isSwitch(item)"
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
          <el-form-item v-if="g.name === 'smtp'" label="发信测试">
            <div class="test-box">
              <el-input v-model="testEmail" placeholder="输入收件邮箱" style="width: 260px" />
              <el-button :loading="testing" @click="onTestSMTP">发送测试邮件</el-button>
            </div>
          </el-form-item>
          <el-form-item
            v-if="g.name === 'sms' && form.sms_provider && form.sms_provider !== 'none'"
            label="发送测试"
          >
            <div class="test-box">
              <el-input v-model="testPhone" placeholder="输入收件手机号" style="width: 260px" />
              <el-button :loading="testing" @click="onTestSMS">发送测试短信</el-button>
            </div>
          </el-form-item>
        </el-form>
      </el-tab-pane>
    </el-tabs>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSettings, updateSettings, testSMTP, testSMS } from '../api/modules'

const BOOL_KEYS = ['register_enabled', 'register_email_verify', 'smtp_enabled', 'smtp_tls']

const activeTab = ref('site')
const groups = ref([])
const form = ref({})
const saving = ref(false)
const testing = ref(false)
const testEmail = ref('')
const testPhone = ref('')

const GROUP_LABELS = {
  site: '站点设置',
  security: '安全设置',
  smtp: 'SMTP 邮件',
  sms: '短信设置',
  template: '邮件模板'
}

const SMS_PROVIDERS = [
  { value: 'none', label: '未启用' },
  { value: 'aliyun', label: '阿里云短信' },
  { value: 'tencent', label: '腾讯云短信' },
  { value: 'smsbao', label: '短信宝' }
]

// 各短信提供商需要展示的字段
const SMS_PROVIDER_FIELDS = {
  none: [],
  aliyun: [
    'sms_access_key_id',
    'sms_access_key_secret',
    'sms_region_id',
    'sms_sign_name',
    'sms_aliyun_template_code'
  ],
  tencent: [
    'sms_access_key_id',
    'sms_access_key_secret',
    'sms_region_id',
    'sms_sign_name',
    'sms_tencent_sdk_app_id',
    'sms_tencent_template_id'
  ],
  smsbao: [
    'smsbao_username',
    'smsbao_password',
    'sms_sign_name'
  ]
}

function isSwitch(item) {
  return BOOL_KEYS.includes(item.key)
}

// 短信组按当前选中的提供商过滤字段
function groupItems(g) {
  if (g.name !== 'sms') return g.items
  const provider = form.value.sms_provider || 'none'
  const visible = SMS_PROVIDER_FIELDS[provider] || []
  return g.items.filter((item) => item.key === 'sms_provider' || visible.includes(item.key))
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

async function onTestSMTP() {
  if (!testEmail.value) {
    ElMessage.warning('请先输入收件邮箱')
    return
  }
  testing.value = true
  try {
    await testSMTP(testEmail.value)
    ElMessage.success('测试邮件已发送')
  } finally {
    testing.value = false
  }
}

async function onTestSMS() {
  if (!testPhone.value) {
    ElMessage.warning('请先输入收件手机号')
    return
  }
  testing.value = true
  try {
    await testSMS(testPhone.value)
    ElMessage.success('测试短信已发送')
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.test-box {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
