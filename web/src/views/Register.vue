<template>
  <div class="auth-page">
    <el-card class="auth-card">
      <h2 class="title">注册账号</h2>
      <el-form ref="formRef" :model="form" :rules="rules" size="large">
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="用户名" clearable>
            <template #prefix><el-icon><User /></el-icon></template>
          </el-input>
        </el-form-item>
        <el-form-item prop="account">
          <el-input v-model="form.account" placeholder="邮箱（或手机号）" clearable @change="accountChange">
            <template #prefix><el-icon><Message /></el-icon></template>
          </el-input>
        </el-form-item>
        <el-form-item v-if="config.register_email_verify && isEmail" prop="code">
          <el-input v-model="form.code" placeholder="邮箱验证码">
            <template #append>
              <el-button :loading="sending" :disabled="countdown > 0" @click="onSendCode">
                {{ countdown > 0 ? countdown + 's' : '发送验证码' }}
              </el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="密码" show-password>
            <template #prefix><el-icon><Lock /></el-icon></template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" class="submit" :loading="loading" @click="onRegister">
            注册
          </el-button>
        </el-form-item>
      </el-form>
      <div class="links">
        <el-link type="primary" @click="router.push('/login')">已有账号？去登录</el-link>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Message } from '@element-plus/icons-vue'
import { register, sendCode } from '../api/auth'

const router = useRouter()
const formRef = ref()
const loading = ref(false)
const sending = ref(false)
const countdown = ref(0)
const config = ref({ register_email_verify: false })
const timer = ref(null)

const form = reactive({ username: '', account: '', code: '', password: '' })

const isEmail = computed(() => /@/.test(form.account))
const minLen = computed(() => config.value.password_min_length || 6)

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  account: [
    { required: true, message: '请输入邮箱或手机号', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (!value) return callback()
        if (/@/.test(value)) {
          /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)
            ? callback()
            : callback(new Error('邮箱格式不正确'))
        } else {
          /^1\d{10}$/.test(value)
            ? callback()
            : callback(new Error('手机号格式不正确'))
        }
      },
      trigger: 'blur'
    }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: minLen.value, message: `密码长度不能少于 ${minLen.value} 位`, trigger: 'blur' }
  ]
}

function accountChange() {
  if (!isEmail.value) form.code = ''
}

onMounted(async () => {
  try {
    const res = await fetch('/api/auth/config')
    config.value = (await res.json()).data
  } catch (e) {
    // 忽略
  }
})

async function onSendCode() {
  if (!form.account) {
    ElMessage.warning('请先填写邮箱')
    return
  }
  sending.value = true
  try {
    await sendCode({ scope: 'register', account: form.account })
    ElMessage.success('验证码已发送')
    countdown.value = 60
    timer.value = setInterval(() => {
      countdown.value -= 1
      if (countdown.value <= 0) clearInterval(timer.value)
    }, 1000)
  } finally {
    sending.value = false
  }
}

async function onRegister() {
  await formRef.value.validate()
  loading.value = true
  try {
    await register({
      username: form.username,
      email: isEmail.value ? form.account : '',
      phone: isEmail.value ? '' : form.account,
      password: form.password,
      code: form.code
    })
    ElMessage.success('注册成功，请登录')
    router.push('/login')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1f4037 0%, #99f2c8 100%);
}
.auth-card {
  width: 380px;
  border-radius: 8px;
}
.title {
  text-align: center;
  margin-bottom: 24px;
  color: #333;
}
.submit {
  width: 100%;
}
.links {
  text-align: center;
}
</style>
