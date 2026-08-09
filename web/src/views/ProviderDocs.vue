<template>
  <div class="provider-docs">
    <el-card class="intro-card">
      <template #header>
        <div class="card-header">
          <span>第三方渠道接入文档</span>
          <span class="tip">统一授权管理平台支持的第三方登录渠道接入说明</span>
        </div>
      </template>
      <p>
        OauthGo 聚合了国内主流第三方登录渠道。接入流程为：在对应开放平台创建应用 → 获取凭据 →
        在「登录渠道」中配置 → 将回调地址填入开放平台。配置完成后，开启「应用于主站登录」即可在主站登录页展示。
      </p>
      <p class="muted">
        每个渠道的当前凭据（AppID / AppKey）与接入步骤、回调配置说明如下，点击「接入地址」可直接跳转到对应开放平台的注册/创建页面。
      </p>
    </el-card>

    <div class="docs-grid">
      <el-card v-for="c in channelList" :key="c.name" class="channel-card">
        <template #header>
          <div class="card-header">
            <div class="channel-title">
              <span class="channel-name">{{ c.displayName }}</span>
              <el-tag size="small" :type="c.category === 'enterprise' ? 'warning' : 'primary'">
                {{ c.category === 'enterprise' ? '企业' : '社交' }}
              </el-tag>
              <span class="channel-key">{{ c.name }}</span>
            </div>
            <el-link v-if="c.registerUrl" type="primary" :href="c.registerUrl" target="_blank">
              {{ c.registerLabel }} →
            </el-link>
          </div>
        </template>

        <p class="tips">{{ c.tips }}</p>

        <el-divider content-position="left">当前配置</el-divider>
        <div class="credential">
          <div class="credential-row">
            <span class="credential-label">{{ c.idLabel }}</span>
            <code class="credential-value" :class="{ empty: !credOf(c).client_id }">
              {{ credOf(c).client_id || '未配置' }}
            </code>
            <el-button v-if="credOf(c).client_id" size="small" text type="primary"
              @click="copy(credOf(c).client_id)">复制</el-button>
          </div>
          <div class="credential-row">
            <span class="credential-label">{{ c.secretLabel }}</span>
            <code class="credential-value" :class="{ empty: !credOf(c).client_secret }">
              {{ credOf(c).client_secret || '未配置' }}
            </code>
            <el-button v-if="credOf(c).client_secret" size="small" text type="primary"
              @click="copy(credOf(c).client_secret)">复制</el-button>
          </div>
        </div>

        <el-divider content-position="left">所需信息</el-divider>
        <div class="fields">
          <el-tag v-for="f in c.fields" :key="f" size="small" class="field-tag">{{ f }}</el-tag>
        </div>

        <el-divider content-position="left">接入步骤</el-divider>
        <ol class="steps">
          <li v-for="(s, i) in c.steps" :key="i">{{ s }}</li>
        </ol>

        <el-divider content-position="left">回调地址配置</el-divider>
        <el-alert :title="c.callbackNote" type="info" :closable="false" class="callback-note" />

        <template v-if="c.notes.length">
          <el-divider content-position="left">注意事项</el-divider>
          <ul class="notes">
            <li v-for="(n, i) in c.notes" :key="i">{{ n }}</li>
          </ul>
        </template>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { listProviders } from '../api/modules'
import { channelList } from '../utils/providerDocs'

const creds = ref({})

onMounted(async () => {
  try {
    const data = await listProviders()
    for (const p of data.list) {
      creds.value[p.name] = { client_id: p.client_id || '', client_secret: p.client_secret || '' }
    }
  } catch (e) {
    // 非管理员或接口失败时忽略，展示「未配置」
  }
})

function credOf(c) {
  return creds.value[c.name] || { client_id: '', client_secret: '' }
}

function copy(text) {
  navigator.clipboard?.writeText(text)
  ElMessage.success('已复制')
}
</script>

<style scoped>
.intro-card {
  margin-bottom: 16px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.tip {
  font-size: 12px;
  color: #909399;
}
.intro-card p {
  margin: 0 0 8px;
  color: #606266;
}
.intro-card p.muted {
  color: #909399;
  font-size: 13px;
}
.docs-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(440px, 1fr));
  gap: 16px;
  align-items: stretch;
}
.channel-card {
  display: flex;
  flex-direction: column;
}
.channel-card :deep(.el-card__body) {
  display: flex;
  flex-direction: column;
  flex: 1;
}
.channel-title {
  display: flex;
  align-items: center;
  gap: 8px;
}
.channel-name {
  font-weight: 600;
}
.channel-key {
  font-size: 12px;
  color: #909399;
}
.tips {
  margin: 0 0 4px;
  color: #606266;
  font-size: 13px;
  line-height: 1.6;
}
.credential {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.credential-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.credential-label {
  width: 110px;
  flex-shrink: 0;
  font-size: 13px;
  color: #909399;
}
.credential-value {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: 13px;
  color: #409eff;
  background: #f0f2f5;
  padding: 4px 8px;
  border-radius: 4px;
}
.credential-value.empty {
  color: #909399;
  font-style: italic;
}
.fields {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.field-tag {
  background: #ecf5ff;
  border-color: #d9ecff;
  color: #409eff;
}
.steps {
  margin: 0;
  padding-left: 20px;
  color: #606266;
  font-size: 13px;
  line-height: 1.9;
}
.callback-note {
  font-size: 13px;
}
.notes {
  margin: 0;
  padding-left: 20px;
  color: #e6a23c;
  font-size: 13px;
  line-height: 1.9;
}
</style>
