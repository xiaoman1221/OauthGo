<template>
  <div class="auth-page">
    <el-card class="auth-card">
      <h2 class="title">找回密码</h2>
      <el-form ref="formRef" :model="form" :rules="rules" size="large">
        <el-form-item prop="account">
          <el-input v-model="form.account" placeholder="注册邮箱或手机号" clearable>
            <template #prefix><el-icon><Message /></el-icon></template>
          </el-input>
        </el-form-item>
        <el-form-item prop="code">
          <el-input v-model="form.code" placeholder="验证码">
            <template #append>
              <el-button :loading="sending" :disabled="countdown > 0" @click="onSendCode">
                {{ countdown > 0 ? countdown + 's' : '发送验证码' }}
              </el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="新密码" show-password>
            <template #prefix><el-icon><Lock /></el-icon></template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" class="submit" :loading="loading" @click="onReset">
            重置密码
          </el-button>
        </el-form-item>
      </el-form>
      <div class="links">
        <el-link type="primary" @click="router.push('/login')">返回登录</el-link>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Message, Lock } from '@element-plus/icons-vue'
import { sendCode, forgotPassword } from '../api/auth'

const router = useRouter()
const formRef = ref()
const loading = ref(false)
const sending = ref(false)
const countdown = ref(0)
const minLen = ref(6)
const timer = ref(null)

const form = reactive({ account: '', code: '', password: '' })

const rules = {
  account: [{ required: true, message: '请输入注册邮箱或手机号', trigger: 'blur' }],
  code: [{ required: true, message: '请输入验证码', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: minLen.value, message: `密码长度不能少于 ${minLen.value} 位`, trigger: 'blur' }
  ]
}

onMounted(async () => {
  try {
    const res = await fetch('/api/auth/config')
    const data = (await res.json()).data
    minLen.value = data.password_min_length || 6
  } catch (e) {
    // 忽略
  }
})

async function onSendCode() {
  if (!form.account) {
    ElMessage.warning('请先填写注册账号')
    return
  }
  sending.value = true
  try {
    await sendCode({ scope: 'reset', account: form.account })
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

async function onReset() {
  await formRef.value.validate()
  loading.value = true
  try {
    await forgotPassword({
      account: form.account,
      code: form.code,
      password: form.password
    })
    ElMessage.success('密码重置成功，请登录')
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
