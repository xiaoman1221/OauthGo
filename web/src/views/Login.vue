<template>
  <div class="login-page" :style="loginBgStyle">
    <el-card class="login-card">
      <h2 class="title">OauthGo 统一授权管理</h2>
      <el-form ref="formRef" :model="form" :rules="rules" size="large">
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="用户名" clearable>
            <template #prefix><el-icon><User /></el-icon></template>
          </el-input>
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="密码" show-password @keyup.enter="onLogin">
            <template #prefix><el-icon><Lock /></el-icon></template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" class="submit" :loading="loading" @click="onLogin">
            登录
          </el-button>
        </el-form-item>
      </el-form>

      <el-divider v-if="providers.length > 0">其他登录方式</el-divider>
      <div v-if="providers.length > 0" class="provider-list">
        <el-button
          v-for="p in providers"
          :key="p.name"
          class="provider-btn"
          plain
          @click="onProviderLogin(p)"
        >
          {{ p.display_name }}
        </el-button>
      </div>

      <div class="links">
        <el-link type="primary" underline="never" href="/docs" target="_blank">接口文档</el-link>
        <el-divider direction="vertical" />
        <el-link type="primary" underline="never" @click="onRegister">注册账号</el-link>
        <el-divider direction="vertical" />
        <el-link type="primary" underline="never" @click="router.push('/forgot-password')">
          忘记密码
        </el-link>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import { useUserStore } from '../stores/user'
import { publicProviders } from '../api/modules'

const router = useRouter()
const userStore = useUserStore()

const formRef = ref()
const loading = ref(false)
const providers = ref([])
const form = reactive({ username: '', password: '' })
const authConfig = ref({})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

onMounted(async () => {
  try {
    const [cfg, provs] = await Promise.all([
      fetch('/api/auth/config').then((r) => r.json()).then((r) => r.data).catch(() => ({})),
      publicProviders().catch(() => [])
    ])
    authConfig.value = cfg || {}
    providers.value = provs || []
  } catch (e) {
    // 忽略加载失败
  }
})

const loginBgStyle = computed(() => {
  const mode = authConfig.value.login_bg_mode || 'color'
  if (mode === 'color') {
    return { background: authConfig.value.login_bg_color || 'linear-gradient(135deg, #1f4037 0%, #99f2c8 100%)' }
  }
  if (mode === 'image') {
    const url = authConfig.value.login_bg_image_url || ''
    return { backgroundImage: url ? `url(${url})` : '', backgroundSize: 'cover', backgroundPosition: 'center' }
  }
  if (mode === 'bing') {
    return { backgroundImage: `url(/api/site/bing-daily)`, backgroundSize: 'cover', backgroundPosition: 'center' }
  }
  return {}
})

async function onLogin() {
  await formRef.value.validate()
  loading.value = true
  try {
    await userStore.login(form.username, form.password)
    ElMessage.success('登录成功')
    router.push('/')
  } finally {
    loading.value = false
  }
}

function onProviderLogin(p) {
  window.location.href = `/api/oauth/${p.name}/login`
}

async function onRegister() {
  try {
    const res = await fetch('/api/auth/config')
    const cfg = (await res.json()).data
    if (!cfg.register_enabled) {
      ElMessage.warning('暂未开放注册')
      return
    }
  } catch (e) {
    // 忽略
  }
  router.push('/register')
}
</script>

<style scoped>
.login-page {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1f4037 0%, #99f2c8 100%);
}
.login-card {
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
.provider-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
}
.provider-btn {
  flex: 1 1 auto;
  min-width: 96px;
  margin: 0 !important;
}
.links {
  margin-top: 16px;
  text-align: center;
}
</style>
