<template>
  <div>
    <el-card v-if="userStore.isAdmin">
      <template #header>
        <div class="card-header">
          <span>系统设置</span>
          <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane v-for="g in groups" :key="g.name" :label="g.label" :name="g.name">
          <el-form :model="form" label-width="220px" style="max-width: 720px">
            <template v-if="g.name === 'site'">
              <div class="section">
                <div class="section-title">站点信息</div>
                <el-form-item v-for="k in SITE_INFO_KEYS" :key="k" :label="getItemDesc('site', k) || k">
                  <el-input v-if="k !== 'site_desc'" v-model="form[k]" />
                  <el-input v-else type="textarea" :rows="3" v-model="form[k]" />
                </el-form-item>
              </div>

              <div class="section">
                <div class="section-title">用户策略</div>
                <el-form-item v-for="k in POLICY_KEYS" :key="k" :label="getItemDesc('site', k) || k">
                  <el-switch v-if="BOOL_KEYS.includes(k)" v-model="form[k]" active-value="1" inactive-value="0" />
                  <el-input v-else v-model="form[k]" />
                </el-form-item>
              </div>

              <div class="section">
                <div class="section-title">外观 / 登录页</div>
                <el-form-item label="登录页背景">
                  <el-select v-model="form.login_bg_mode" placeholder="选择背景类型" style="width: 220px">
                    <el-option label="纯色" value="color" />
                    <el-option label="图片 URL" value="image" />
                    <el-option label="Bing 每日一图" value="bing" />
                  </el-select>
                  <div style="margin-top:8px">
                    <el-input v-if="form.login_bg_mode === 'image'" v-model="form.login_bg_image_url" placeholder="图片 URL" />
                    <el-input v-else-if="form.login_bg_mode === 'color'" v-model="form.login_bg_color" type="color" style="width:120px" />
                    <div v-else style="color:#909399; margin-top:6px">使用 Bing 每日一图作为登录页背景</div>
                  </div>
                  <div style="margin-top:10px">
                    <div style="width:240px;height:80px;border:1px solid #eaeaea;border-radius:4px;overflow:hidden;display:flex;align-items:center;justify-content:center" :style="previewStyle">
                      <span style="color:#fff;background:rgba(0,0,0,0.35);padding:4px 8px;border-radius:4px">预览</span>
                    </div>
                  </div>
                </el-form-item>

                <el-form-item label="站点 Logo (登录页)" v-if="getItemDesc('site','site_logo') !== undefined">
                  <el-input v-model="form.site_logo" placeholder="Logo 地址" />
                </el-form-item>
              </div>
            </template>

            <template v-else>
              <template v-for="item in groupItems(g)" :key="item.key">
                <el-form-item :label="item.description || item.key">

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

              </template>
            </template>

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

    <el-empty v-else description="无权限访问">
      <template #footer>
        <el-button type="primary" @click="$router.push('/')">返回仪表盘</el-button>
      </template>
    </el-empty>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { getSettings, updateSettings, testSMTP, testSMS } from '../api/modules'
import { useUserStore } from '../stores/user'

const userStore = useUserStore()

const BOOL_KEYS = ['register_enabled', 'register_email_verify', 'smtp_enabled', 'smtp_tls', 'proxy_enabled']

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
  proxy: '代理设置',
  template: '邮件模板'
}

const SITE_INFO_KEYS = ['site_name', 'site_url', 'site_logo', 'site_desc']
const POLICY_KEYS = ['register_enabled', 'register_email_verify', 'default_role', 'user_max_apps']

function getItemDesc(groupName, key) {
  const g = groups.value.find((x) => x.name === groupName)
  if (!g) return ''
  const it = (g.items || []).find((i) => i.key === key)
  return it ? it.description : ''
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

  // 初始化登录页背景设置默认值（若服务器未提供）
  if (!form.value.login_bg_mode) form.value.login_bg_mode = 'color'
  if (!form.value.login_bg_color) form.value.login_bg_color = '#1f4037'
  if (!form.value.login_bg_image_url) form.value.login_bg_image_url = ''
})

const previewStyle = computed(() => {
  if (form.value.login_bg_mode === 'color') {
    return { background: form.value.login_bg_color }
  }
  if (form.value.login_bg_mode === 'image') {
    return { backgroundImage: `url(${form.value.login_bg_image_url})`, backgroundSize: 'cover', backgroundPosition: 'center' }
  }
  if (form.value.login_bg_mode === 'bing') {
    return { backgroundImage: `url(/api/site/bing-daily)`, backgroundSize: 'cover', backgroundPosition: 'center' }
  }
  return {}
})

async function onSave() {
  saving.value = true
  try {
    const payload = []
    // include dynamic group items
    groups.value.forEach((g) => {
      g.items.forEach((item) => {
        payload.push({ key: item.key, value: form.value[item.key] || '', description: item.description })
      })
    })
    // include login background settings explicitly
    payload.push({ key: 'login_bg_mode', value: form.value.login_bg_mode || 'color', description: '登录页背景类型 (color|image|bing)' })
    payload.push({ key: 'login_bg_color', value: form.value.login_bg_color || '', description: '登录页背景纯色（hex）' })
    payload.push({ key: 'login_bg_image_url', value: form.value.login_bg_image_url || '', description: '登录页背景图片 URL' })

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
.section{margin-bottom:18px;padding:12px;border:1px solid #f3f6f8;border-radius:6px;background:#fff}
.section-title{font-weight:600;margin-bottom:8px}
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
