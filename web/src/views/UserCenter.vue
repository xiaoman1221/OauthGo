<template>
  <el-card>
    <template #header>
      <div class="card-header">
        <span>用户中心</span>
      </div>
    </template>

    <el-tabs v-model="activeTab">
      <!-- 基本资料 -->
      <el-tab-pane label="基本资料" name="profile">
        <el-form :model="profile" label-width="100px" style="max-width: 560px">
          <el-form-item label="头像">
            <div class="avatar-row">
              <el-input v-model="profile.avatar" placeholder="头像图片地址（http/https）" clearable />
              <el-avatar v-if="profile.avatar" :src="profile.avatar" :size="40" class="avatar-preview" />
            </div>
          </el-form-item>
          <el-form-item label="昵称">
            <el-input v-model="profile.nickname" placeholder="请输入昵称" clearable />
          </el-form-item>
          <el-form-item label="用户名">
            <el-input v-model="profile.username" placeholder="登录用户名" clearable />
          </el-form-item>
          <el-form-item v-if="emailChanged" label="邮箱验证码">
            <div class="code-row">
              <el-input v-model="profile.email_code" placeholder="请输入验证码" clearable />
              <el-button :disabled="emailSending" @click="onSendCode('email')">
                {{ emailCountdown > 0 ? emailCountdown + 's' : '发送验证码' }}
              </el-button>
            </div>
          </el-form-item>
          <el-form-item label="邮箱">
            <el-input v-model="profile.email" placeholder="绑定邮箱" clearable />
          </el-form-item>
          <el-form-item v-if="phoneChanged" label="手机验证码">
            <div class="code-row">
              <el-input v-model="profile.phone_code" placeholder="请输入验证码" clearable />
              <el-button :disabled="phoneSending" @click="onSendCode('phone')">
                {{ phoneCountdown > 0 ? phoneCountdown + 's' : '发送验证码' }}
              </el-button>
            </div>
          </el-form-item>
          <el-form-item label="手机号">
            <el-input v-model="profile.phone" placeholder="绑定手机号" clearable />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="saving" @click="onSaveProfile">保存</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- 修改密码 -->
      <el-tab-pane label="修改密码" name="password">
        <el-form ref="pwdFormRef" :model="pwd" :rules="pwdRules" label-width="100px" style="max-width: 420px">
          <el-form-item v-if="userStore.userInfo?.password_set" label="原密码" prop="old_password">
            <el-input v-model="pwd.old_password" type="password" show-password placeholder="请输入原密码" />
          </el-form-item>
          <el-form-item label="新密码" prop="new_password">
            <el-input v-model="pwd.new_password" type="password" show-password placeholder="请输入新密码" />
          </el-form-item>
          <el-form-item label="确认密码" prop="confirm_password">
            <el-input v-model="pwd.confirm_password" type="password" show-password placeholder="请再次输入新密码" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="saving" @click="onChangePassword">修改密码</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- 账号绑定 -->
      <el-tab-pane label="账号绑定" name="bindings">
        <div class="binding-tip">
          绑定后可使用第三方渠道直接登录，仅展示系统设置中允许的登录渠道。
        </div>
        <div v-for="p in bindings" :key="p.name" class="binding-item">
          <div class="binding-info">
            <el-avatar v-if="p.bound && p.avatar" :src="p.avatar" :size="36" />
            <div class="binding-name">
              <span class="provider-name">{{ p.display_name }}</span>
              <el-tag v-if="p.bound" size="small" type="success">已绑定</el-tag>
              <el-tag v-else size="small" type="info">未绑定</el-tag>
              <span v-if="p.bound && p.nickname" class="account-nick">{{ p.nickname }}</span>
            </div>
          </div>
          <el-button v-if="p.bound" type="danger" plain size="small" @click="onUnbind(p)">
            解绑
          </el-button>
          <el-button v-else type="primary" size="small" :loading="bindingLoading === p.name" @click="onBind(p)">
            绑定
          </el-button>
        </div>
        <el-empty v-if="bindings.length === 0" description="暂无可绑定的登录渠道" />
      </el-tab-pane>
    </el-tabs>
  </el-card>
</template>

<script setup>
import { reactive, ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useUserStore } from '../stores/user'
import {
  updateProfile, changePassword, myBindings, bindLogin, unbindLogin, sendCode
} from '../api/auth'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const activeTab = ref('profile')
const saving = ref(false)
const bindingLoading = ref('')
const bindings = ref([])

const emailSending = ref(false)
const phoneSending = ref(false)

const emailCountdown = ref(0)
const phoneCountdown = ref(0)

const profile = reactive({
  nickname: '',
  avatar: '',
  username: '',
  email: '',
  email_code: '',
  phone: '',
  phone_code: ''
})
const original = reactive({ email: '', phone: '' })
const emailChanged = computed(() => profile.email.trim() !== original.email)
const phoneChanged = computed(() => profile.phone.trim() !== original.phone)

const pwdFormRef = ref()
const pwd = reactive({ old_password: '', new_password: '', confirm_password: '' })
const pwdRules = {
  new_password: [{ required: true, message: '请输入新密码', trigger: 'blur' }],
  confirm_password: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (_, value, callback) => {
        if (value !== pwd.new_password) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

onMounted(() => {
  applyUser()
  loadBindings()
  const bind = route.query.bind
  if (bind === 'success') {
    ElMessage.success('第三方账号绑定成功')
    router.replace({ query: {} })
  } else if (bind === 'fail') {
    ElMessage.error(route.query.msg || '绑定失败')
    router.replace({ query: {} })
  }
})

function applyUser() {
  const u = userStore.userInfo || {}
  profile.nickname = u.nickname || ''
  profile.avatar = u.avatar || ''
  profile.username = u.username || ''
  profile.email = u.email || ''
  profile.phone = u.phone || ''
  original.email = u.email || ''
  original.phone = u.phone || ''
}

async function loadBindings() {
  try {
    bindings.value = await myBindings()
  } catch (e) {
    bindings.value = []
  }
}

async function onSaveProfile() {
  saving.value = true
  try {
    const payload = { ...profile }
    payload.email_code = emailChanged.value ? profile.email_code : ''
    payload.phone_code = phoneChanged.value ? profile.phone_code : ''
    const user = await updateProfile(payload)
    userStore.userInfo = user
    original.email = profile.email.trim()
    original.phone = profile.phone.trim()
    profile.email_code = ''
    profile.phone_code = ''
    ElMessage.success('保存成功')
  } finally {
    saving.value = false
  }
}

async function onChangePassword() {
  await pwdFormRef.value.validate()
  saving.value = true
  try {
    await changePassword({
      old_password: pwd.old_password,
      new_password: pwd.new_password
    })
    ElMessage.success('密码修改成功')
    pwd.old_password = ''
    pwd.new_password = ''
    pwd.confirm_password = ''
    await userStore.fetchUser()
  } finally {
    saving.value = false
  }
}

function onSendCode(field) {
  const account = field === 'email' ? profile.email.trim() : profile.phone.trim()
  if (!account) {
    ElMessage.warning(field === 'email' ? '请先填写邮箱' : '请先填写手机号')
    return
  }
  const sendingRef = field === 'email' ? emailSending : phoneSending
  const countdownRef = field === 'email' ? emailCountdown : phoneCountdown
  sendingRef.value = true
  sendCode({ scope: 'bind', account })
    .then(() => {
      countdownRef.value = 60
      const timer = setInterval(() => {
        countdownRef.value -= 1
        if (countdownRef.value <= 0) {
          clearInterval(timer)
          sendingRef.value = false
        }
      }, 1000)
    })
    .catch(() => {
      sendingRef.value = false
    })
}

async function onBind(p) {
  bindingLoading.value = p.name
  try {
    const data = await bindLogin(p.name)
    window.location.href = data.url
  } finally {
    bindingLoading.value = ''
  }
}

async function onUnbind(p) {
  await ElMessageBox.confirm(
    `确定解绑「${p.display_name}」吗？解绑后无法通过该渠道登录。`,
    '提示',
    { type: 'warning' }
  )
  await unbindLogin(p.name)
  ElMessage.success('解绑成功')
  loadBindings()
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.avatar-row,
.code-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}
.avatar-preview {
  flex-shrink: 0;
}
.binding-tip {
  color: #909399;
  font-size: 13px;
  margin-bottom: 16px;
}
.binding-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  margin-bottom: 12px;
  max-width: 560px;
}
.binding-info {
  display: flex;
  align-items: center;
  gap: 12px;
}
.binding-name {
  display: flex;
  align-items: center;
  gap: 8px;
}
.provider-name {
  font-weight: 600;
}
.account-nick {
  color: #909399;
  font-size: 13px;
}
</style>
