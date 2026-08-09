<template>
  <el-card>
    <template #header>
      <div class="card-header">
        <span>第三方登录渠道</span>
        <span class="tip">配置登录渠道，登录页会自动展示已启用且「应用于主站」的渠道</span>
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
      <el-table-column label="回调地址" min-width="200">
        <template #default="{ row }">
          <span v-if="noCallback(row.name)" class="callback">前端 code 登录，无需回调</span>
          <span v-else class="callback">{{ row.callback_url }}</span>
        </template>
      </el-table-column>
      <el-table-column label="主站" width="70">
        <template #default="{ row }">
          <el-tag size="small" :type="row.main_site ? 'success' : 'info'">
            {{ row.main_site ? '启用' : '停用' }}
          </el-tag>
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

    <el-dialog v-model="dialogVisible" :title="`配置「${form.display_name}」渠道`" width="600px">
      <el-alert type="info" :closable="false" class="tips">
        <div class="tips-body">
          <span class="tips-text">{{ schema.tips }}</span>
          <el-link v-if="schema.registerUrl" type="primary" :href="schema.registerUrl" target="_blank">
            {{ schema.registerLabel || '接入地址' }} →
          </el-link>
        </div>
      </el-alert>

      <el-form :model="form" label-width="130px" class="channel-form">
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
          <span class="switch-tip">开启后该渠道可发起登录</span>
        </el-form-item>
        <el-form-item label="应用于主站登录">
          <el-switch v-model="form.main_site" />
          <span class="switch-tip">开启后展示在主站登录页</span>
        </el-form-item>

        <el-form-item :label="schema.idLabel">
          <el-input v-model="form.client_id" :placeholder="schema.idPlaceholder" />
        </el-form-item>
        <el-form-item :label="schema.secretLabel">
          <el-input v-model="form.client_secret" type="password" show-password
            :placeholder="form.client_secret ? '留空则不修改' : schema.secretPlaceholder" />
        </el-form-item>

        <el-form-item label="回调地址">
          <el-input :model-value="callbackUrl" readonly>
            <template #append>
              <el-button :data-clipboard-text="callbackUrl" @click="copyCallback">复制</el-button>
            </template>
          </el-input>
          <div class="field-tip">由系统根据 HOST 自动拼接，不支持自定义</div>
        </el-form-item>

        <template v-if="schema.configFields.length">
          <el-divider content-position="left">{{ schema.divider }}</el-divider>
          <el-form-item v-for="f in schema.configFields" :key="f.key" :label="f.label">
            <el-select v-if="f.type === 'select'" v-model="config[f.key]">
              <el-option v-for="o in f.options" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
            <el-input v-else-if="f.type === 'textarea'" v-model="config[f.key]" type="textarea"
              :rows="f.rows || 5" :placeholder="f.placeholder" />
            <el-input v-else v-model="config[f.key]" :placeholder="f.placeholder" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button :loading="testing" @click="onTest">测试渠道</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { listProviders, updateProvider, testProvider } from '../api/modules'
import { channelSchema } from '../utils/providerDocs'

const providers = ref([])
const dialogVisible = ref(false)
const saving = ref(false)
const testing = ref(false)
const form = ref({})
const config = ref({})

const schema = computed(() => channelSchema(form.value.name))

const callbackUrl = computed(() => {
  if (!form.value.name) return ''
  return form.value.callback_url || `${window.location.origin}/api/oauth/${form.value.name}/callback`
})

onMounted(load)

function noCallback(name) {
  return name === 'wechat_miniprogram'
}

async function load() {
  const data = await listProviders()
  providers.value = data.list
}

function openDialog(row) {
  form.value = {
    ...row,
    client_secret: row.client_secret || '',
    main_site: !!row.main_site
  }
  try {
    config.value = row.config ? JSON.parse(row.config) : {}
  } catch (e) {
    config.value = {}
  }
  dialogVisible.value = true
}

function copyCallback() {
  navigator.clipboard?.writeText(callbackUrl.value)
  ElMessage.success('回调地址已复制')
}

async function onTest() {
  testing.value = true
  try {
    const payload = {
      client_id: form.value.client_id,
      config: JSON.stringify(config.value)
    }
    if (form.value.client_secret) {
      payload.client_secret = form.value.client_secret
    }
    const data = await testProvider(form.value.name, payload)
    ElMessage.success(data.message || '配置有效')
  } finally {
    testing.value = false
  }
}

async function onSave() {
  saving.value = true
  try {
    const payload = {
      client_id: form.value.client_id,
      enabled: !!form.value.enabled,
      main_site: !!form.value.main_site,
      config: JSON.stringify(config.value)
    }
    if (form.value.client_secret) {
      payload.client_secret = form.value.client_secret
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
  word-break: break-all;
}
.tips {
  margin-bottom: 16px;
}
.tips-body {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.tips-text {
  flex: 1;
  line-height: 1.6;
}
.channel-form .switch-tip {
  margin-left: 10px;
  font-size: 12px;
  color: #909399;
}
.field-tip {
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
}
</style>
