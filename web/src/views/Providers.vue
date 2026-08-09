<template>
  <el-card>
    <template #header>
      <div class="card-header">
        <span>第三方登录渠道</span>
        <span class="tip">配置中国大陆主流登录渠道（对齐 Casdoor），登录页会自动展示已启用渠道</span>
      </div>
    </template>

    <el-table :data="providers" border stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column label="渠道" min-width="140">
        <template #default="{ row }">
          <div class="provider-name">
            <span class="name">{{ row.display_name }}</span>
            <el-tag size="small" :type="row.category === 'enterprise' ? 'warning' : 'primary'">
              {{ row.category === 'enterprise' ? '企业' : '社交' }}
            </el-tag>
          </div>
          <div class="provider-key">{{ row.name }}</div>
        </template>
      </el-table-column>
      <el-table-column prop="client_id" label="ClientID / AppID" min-width="160" show-overflow-tooltip />
      <el-table-column label="回调地址" min-width="180">
        <template #default="{ row }">
          <span class="callback">{{ row.redirect_url || '未设置（默认使用 HOST 拼接）' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openDialog(row)">配置</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="`配置「${form.display_name}」渠道`" width="560px">
      <el-form :model="form" label-width="130px">
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item label="ClientID">
          <el-input v-model="form.client_id" placeholder="AppID / ClientID / appid" />
        </el-form-item>
        <el-form-item label="ClientSecret">
          <el-input v-model="form.client_secret" type="password" show-password
            :placeholder="form.client_secret ? '留空则不修改' : 'AppSecret / ClientSecret'" />
        </el-form-item>
        <el-form-item label="回调地址">
          <el-input v-model="form.redirect_url"
            placeholder="留空则使用 HOST 自动拼接 /api/oauth/xxx/callback" />
        </el-form-item>

        <template v-if="form.name === 'wecom'">
          <el-divider content-position="left">企业微信扩展配置</el-divider>
          <el-form-item label="AgentId">
            <el-input v-model="config.agent_id" placeholder="企业自建应用 AgentId" />
          </el-form-item>
          <el-form-item label="CorpId">
            <el-input v-model="config.corp_id" placeholder="企业 ID（第三方应用时为服务商 corpid）" />
          </el-form-item>
          <el-form-item label="登录类型">
            <el-select v-model="config.login_type">
              <el-option label="企业自建 (CorpApp)" value="CorpApp" />
              <el-option label="第三方 (ServiceApp)" value="ServiceApp" />
            </el-select>
          </el-form-item>
        </template>

        <template v-else-if="form.name === 'alipay'">
          <el-divider content-position="left">支付宝扩展配置</el-divider>
          <el-form-item label="应用私钥">
            <el-input v-model="config.app_private_key" type="textarea" :rows="5"
              placeholder="RSA2 应用私钥（PKCS1 或 PKCS8 PEM 格式）" />
          </el-form-item>
        </template>

        <template v-else>
          <el-divider content-position="left">扩展配置（JSON）</el-divider>
          <el-form-item label="Config">
            <el-input v-model="configText" type="textarea" :rows="5"
              placeholder='{"key":"value"}' />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { listProviders, updateProvider } from '../api/modules'

const providers = ref([])
const dialogVisible = ref(false)
const saving = ref(false)
const form = ref({})
const config = ref({})
const configText = ref('')

onMounted(load)

async function load() {
  const data = await listProviders()
  providers.value = data.list
}

function openDialog(row) {
  form.value = { ...row }
  try {
    config.value = row.config ? JSON.parse(row.config) : {}
  } catch (e) {
    config.value = {}
  }
  configText.value = JSON.stringify(config.value, null, 2)
  dialogVisible.value = true
}

async function onSave() {
  saving.value = true
  try {
    const payload = {
      client_id: form.value.client_id,
      redirect_url: form.value.redirect_url,
      enabled: form.value.enabled
    }
    if (form.value.client_secret) {
      payload.client_secret = form.value.client_secret
    }
    payload.config = JSON.stringify(config.value)
    if (form.value.name !== 'wecom' && form.value.name !== 'alipay') {
      try {
        payload.config = configText.value ? JSON.stringify(JSON.parse(configText.value)) : ''
      } catch (e) {
        ElMessage.error('扩展配置必须是合法的 JSON')
        saving.value = false
        return
      }
    }
    await updateProvider(form.value.name, payload)
    ElMessage.success('保存成功')
    dialogVisible.value = false
    load()
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
.tip {
  font-size: 12px;
  color: #909399;
}
.provider-name {
  display: flex;
  align-items: center;
  gap: 8px;
}
.provider-name .name {
  font-weight: 600;
}
.provider-key {
  font-size: 12px;
  color: #909399;
}
.callback {
  font-size: 12px;
  color: #606266;
}
</style>
