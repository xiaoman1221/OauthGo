<template>
  <el-card>
    <div class="toolbar">
      <el-button type="primary" @click="openDialog()">
        <el-icon><Plus /></el-icon> 新增应用
      </el-button>
      <div class="spacer" />
      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="本平台为核心目标站点提供第三方登录服务，应用即接入本站点的目标站点。"
      />
    </div>

    <el-table :data="apps" border stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" min-width="130" />
      <el-table-column prop="platform" label="平台" width="90" />
      <el-table-column label="模式" width="110">
        <template #default="{ row }">
          <el-tag :type="modeType(row.mode)">{{ modeLabel(row.mode) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="AppID" min-width="150">
        <template #default="{ row }">
          <span class="mono" @click="onCopy(row.appid)">{{ row.appid }}</span>
        </template>
      </el-table-column>
      <el-table-column label="AppKey" min-width="200">
        <template #default="{ row }">
          <span class="mono" @click="onCopy(row.app_key)">{{ row.app_key }}</span>
        </template>
      </el-table-column>
      <el-table-column label="登录类型" min-width="180">
        <template #default="{ row }">
          <el-tag
            v-for="t in row.types"
            :key="t"
            size="small"
            class="type-tag"
          >{{ typeLabel(t) }}</el-tag>
          <span v-if="row.types.length === 0" class="muted">未选择</span>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'">
            {{ row.status === 1 ? '启用' : '禁用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openDocs(row)">接入文档</el-button>
          <el-button size="small" @click="openDialog(row)">编辑</el-button>
          <el-button size="small" type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑应用' : '新增应用'" width="620px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="目标站点名称" />
        </el-form-item>
        <el-form-item label="平台">
          <el-select v-model="form.platform" placeholder="请选择平台">
            <el-option label="Web" value="web" />
            <el-option label="iOS" value="ios" />
            <el-option label="Android" value="android" />
            <el-option label="PC" value="pc" />
          </el-select>
        </el-form-item>
        <el-form-item label="接入模式">
          <el-radio-group v-model="form.mode">
            <el-radio value="compat">兼容模式</el-radio>
            <el-radio value="rainbow">仅彩虹协议</el-radio>
            <el-radio value="rest">仅REST接口</el-radio>
          </el-radio-group>
          <div class="hint">兼容模式同时开放彩虹聚合登录协议与 REST 风格接口</div>
        </el-form-item>
        <el-form-item label="登录类型">
          <div class="types-box">
            <el-checkbox
              :model-value="allTypesSelected"
              :indeterminate="allTypesIndeterminate"
              @change="toggleAllTypes"
            >全选</el-checkbox>
            <el-checkbox-group v-model="form.types" class="types-group">
              <el-checkbox v-for="p in providerOptions" :key="p.name" :value="p.name">
                {{ p.display_name }}
              </el-checkbox>
            </el-checkbox-group>
          </div>
          <div class="hint">该目标站点向自己用户开放的第三方登录方式</div>
        </el-form-item>

        <el-divider content-position="left">凭证信息</el-divider>
        <el-alert
          v-if="!form.id"
          type="info"
          :closable="false"
          show-icon
          title="保存后系统将自动生成 AppID 与 AppKey，自动生成且不可自定义。"
          class="cred-alert"
        />
        <template v-else>
          <el-form-item label="AppID">
            <el-input v-model="form.appid" readonly>
              <template #append>
                <el-button @click="onCopy(form.appid)">复制</el-button>
              </template>
            </el-input>
            <div class="hint">自动生成，用于接口调用标识，不可自定义</div>
          </el-form-item>
          <el-form-item label="AppKey">
            <div class="appkey-box">
              <el-input v-model="form.app_key" readonly>
                <template #append>
                  <el-button @click="onCopy(form.app_key)">复制</el-button>
                </template>
              </el-input>
              <el-button type="warning" @click="onRegenerate">重新生成</el-button>
            </div>
            <div class="hint">彩虹协议与签名校验使用的密钥，请妥善保管</div>
          </el-form-item>
        </template>

        <el-divider content-position="left">回调白名单</el-divider>
        <el-form-item label="回调域名">
          <el-input
            v-model="form.domains"
            type="textarea"
            :rows="4"
            placeholder="example.com&#10;www.example.com"
          />
          <div class="hint">
            每个域名一行，仅填写域名（可含子域名）。redirect_uri 的域名等于白名单域名或其子域名时允许回跳，区分子域名
          </div>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="docsVisible" :title="`接入文档 - ${docsApp.name}`" width="760px">
      <el-alert
        type="warning"
        :closable="false"
        show-icon
        title="以下示例中的 appid / appkey 请在应用中复制替换，签名规则见文末。"
        class="doc-alert"
      />
      <el-tabs v-model="docsTab">
        <el-tab-pane label="彩虹聚合协议" name="rainbow">
          <div class="doc-block">
            <p>1. 获取跳转登录地址</p>
            <pre>{{ docsRainbowLogin }}</pre>
            <p>2. 用户登录后回跳（GET）</p>
            <pre>{{ docsRainbowReturn }}</pre>
            <p>3. 用 code 换取用户信息</p>
            <pre>{{ docsRainbowCallback }}</pre>
            <p>4. 按第三方 UID 查询用户（可选）</p>
            <pre>{{ docsRainbowQuery }}</pre>
          </div>
        </el-tab-pane>
        <el-tab-pane label="REST 接口" name="rest">
          <div class="doc-block">
            <p>1. 获取跳转登录地址</p>
            <pre>POST {{ baseUrl }}/api/v1/oauth/login
Content-Type: application/json

{{ docsRestLogin }}</pre>
            <p>2. 用户登录后回跳（GET）</p>
            <pre>{{ docsRestReturn }}</pre>
            <p>3. 用 code 换取用户信息（服务端签名）</p>
            <pre>POST {{ baseUrl }}/api/v1/oauth/userinfo
Content-Type: application/json

{{ docsRestUserinfo }}</pre>
            <p>4. 按第三方 UID 查询用户（可选）</p>
            <pre>POST {{ baseUrl }}/api/v1/oauth/query

{{ docsRestQuery }}</pre>
          </div>
        </el-tab-pane>
        <el-tab-pane label="签名规则" name="sign">
          <div class="doc-block">
            <p>REST 接口的 userinfo / query 使用服务端签名校验：</p>
            <pre>1. 除 sign 外的所有参数按 key 升序排列
2. 拼接为 k1=v1&amp;k2=v2
3. 末尾拼接 &amp;key=AppKey
4. sign = md5(上述字符串)

示例（AppKey 为 abc123）：
sign = md5("appid=xxxx&amp;code=yyyy&amp;type=qq&amp;key=abc123")</pre>
            <p>彩虹协议则直接传 appid + appkey 参数鉴权。</p>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { listApps, createApp, updateApp, deleteApp, listProviders } from '../api/modules'

const apps = ref([])
const providers = ref([])
const dialogVisible = ref(false)
const docsVisible = ref(false)
const docsTab = ref('rainbow')
const docsApp = ref({})
const saving = ref(false)
const form = ref({})

const baseUrl = window.location.origin

const providerOptions = computed(() =>
  providers.value.map((p) => ({ name: p.name, display_name: p.display_name }))
)

const allTypesSelected = computed(() => {
  const names = providerOptions.value.map((p) => p.name)
  const types = form.value.types || []
  return names.length > 0 && names.every((n) => types.includes(n))
})
const allTypesIndeterminate = computed(() => {
  const types = form.value.types || []
  return types.length > 0 && !allTypesSelected.value
})
function toggleAllTypes(checked) {
  form.value.types = checked ? providerOptions.value.map((p) => p.name) : []
}

const TYPE_LABELS = {
  wechat: '微信', wechat_miniprogram: '微信小程序', qq: 'QQ', weibo: '微博',
  gitee: 'Gitee', douyin: '抖音', baidu: '百度', alipay: '支付宝',
  dingtalk: '钉钉', wecom: '企业微信', lark: '飞书', infoflow: '如流'
}

function typeLabel(name) {
  return TYPE_LABELS[name] || name
}
function modeLabel(mode) {
  return { compat: '兼容模式', rainbow: '仅彩虹协议', rest: '仅REST接口' }[mode] || mode
}
function modeType(mode) {
  return { compat: 'success', rainbow: 'warning', rest: 'primary' }[mode] || 'info'
}

onMounted(async () => {
  load()
  try {
    providers.value = (await listProviders()).list
  } catch (e) {
    // 忽略
  }
})

async function load() {
  const data = await listApps()
  apps.value = data.list
}

function openDialog(row) {
  form.value = row
    ? { ...row, regenerate_key: false }
    : {
        name: '', platform: 'web', mode: 'compat', types: [],
        appid: '', app_key: '', domains: '', status: 1
      }
  dialogVisible.value = true
}

async function onSave() {
  saving.value = true
  try {
    if (form.value.id) {
      await updateApp(form.value.id, form.value)
    } else {
      await createApp(form.value)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    load()
  } finally {
    saving.value = false
  }
}

async function onDelete(row) {
  await ElMessageBox.confirm(`确定删除应用「${row.name}」吗？`, '提示', { type: 'warning' })
  await deleteApp(row.id)
  ElMessage.success('删除成功')
  load()
}

async function onCopy(text) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.warning('复制失败，请手动复制')
  }
}

async function onRegenerate() {
  await ElMessageBox.confirm(
    '重新生成后旧 AppKey 立即失效，请同步更新目标站点配置，确定继续吗？',
    '提示',
    { type: 'warning' }
  )
  form.value.regenerate_key = true
}

function openDocs(row) {
  docsApp.value = row
  docsTab.value = row.mode === 'rest' ? 'rest' : 'rainbow'
  docsVisible.value = true
}

const typeExample = () => (docsApp.value.types && docsApp.value.types[0]) || 'qq'

const exampleDomain = () => {
  const list = (docsApp.value.domains || '')
    .split(/\n/)
    .map((s) => s.trim())
    .filter(Boolean)
  return list[0] || 'example.com'
}
const exampleCallback = () => `https://${exampleDomain()}/oauth/callback`

const docsRainbowLogin = computed(() =>
  `${baseUrl}/api/connect.php?act=login&appid=${docsApp.value.appid}&appkey=${docsApp.value.app_key}&type=${typeExample()}&redirect_uri=${encodeURIComponent(exampleCallback())}`
)
const docsRainbowReturn = computed(() =>
  `${exampleCallback()}?type=${typeExample()}&code=520DD95263C1CFEA0870FBB66E******&sign=xxxxxxxx`
)
const docsRainbowCallback = computed(() =>
  `${baseUrl}/api/connect.php?act=callback&appid=${docsApp.value.appid}&appkey=${docsApp.value.app_key}&type=${typeExample()}&code=520DD95263C1CFEA0870FBB66E******`
)
const docsRainbowQuery = computed(() =>
  `${baseUrl}/api/connect.php?act=query&appid=${docsApp.value.appid}&appkey=${docsApp.value.app_key}&type=${typeExample()}&social_uid=AD3F5033279C8187CBCBB29235D5F827`
)
const docsRestLogin = computed(() =>
  JSON.stringify(
    { appid: docsApp.value.appid, appkey: docsApp.value.app_key, type: typeExample(), redirect_uri: exampleCallback() },
    null, 2
  )
)
const docsRestReturn = computed(() =>
  `${exampleCallback()}?type=${typeExample()}&code=520DD95263C1CFEA0870FBB66E******&sign=xxxxxxxx`
)
const docsRestUserinfo = computed(() =>
  JSON.stringify(
    { appid: docsApp.value.appid, code: '520DD95263C1CFEA0870FBB66E******', type: typeExample(), sign: 'md5(appid&code&type&key)' },
    null, 2
  )
)
const docsRestQuery = computed(() =>
  JSON.stringify(
    { appid: docsApp.value.appid, type: typeExample(), social_uid: 'AD3F5033279C8187CBCBB29235D5F827', sign: 'md5(...)' },
    null, 2
  )
)
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}
.spacer {
  flex: 1;
}
.mono {
  font-family: monospace;
  cursor: pointer;
}
.type-tag {
  margin-right: 4px;
}
.types-box {
  width: 100%;
}
.types-group {
  margin-top: 8px;
}
.cred-alert {
  margin-bottom: 8px;
}
.appkey-box {
  display: flex;
  width: 100%;
  gap: 8px;
}
.appkey-box .el-input {
  flex: 1;
}
.muted {
  color: #999;
}
.hint {
  font-size: 12px;
  color: #999;
  line-height: 1.6;
}
.doc-alert {
  margin-bottom: 12px;
}
.doc-block pre {
  background: #f5f7fa;
  padding: 10px;
  border-radius: 4px;
  overflow-x: auto;
  font-size: 12px;
  line-height: 1.6;
}
</style>
