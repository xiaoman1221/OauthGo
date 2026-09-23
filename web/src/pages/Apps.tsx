import { useCallback, useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { Plus } from 'lucide-react'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { StatusBadge } from '@/components/status-badge'
import { Checkbox } from '@/components/ui/checkbox'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ConfirmDialog } from '@/components/ui/confirm'
import { CodeBlock } from '@/components/code-block'
import { appsApi, providersApi, type App, type Provider } from '@/lib/api'
import { useIsAdmin } from '@/store/user'

const TYPE_LABELS: Record<string, string> = {
  wechat: '微信',
  wechat_miniprogram: '微信小程序',
  qq: 'QQ',
  weibo: '微博',
  gitee: 'Gitee',
  douyin: '抖音',
  baidu: '百度',
  alipay: '支付宝',
  dingtalk: '钉钉',
  wecom: '企业微信',
  lark: '飞书',
  infoflow: '如流',
  google: 'Google',
  github: 'GitHub',
  microsoft: 'Microsoft',
  apple: 'Apple',
  discord: 'Discord',
  facebook: 'Facebook',
  linkedin: 'LinkedIn',
  oauth2: '通用OAuth2/OIDC'
}

const MODE_LABELS: Record<string, { label: string; variant: 'default' | 'secondary' | 'outline' }> = {
  compat: { label: '兼容模式', variant: 'default' },
  rainbow: { label: '仅彩虹协议', variant: 'secondary' },
  rest: { label: '仅REST接口', variant: 'outline' },
  oauth2: { label: '仅OAuth2/OIDC', variant: 'outline' }
}

interface AppForm {
  id?: number
  name: string
  platform: string
  mode: string
  types: string[]
  appid: string
  app_key: string
  domains: string
  redirect_uris: string
  sample_params: string
  enable_refresh: boolean
  status: number
  regenerate_key: boolean
}

const emptyForm: AppForm = {
  name: '',
  platform: 'web',
  mode: 'compat',
  types: [],
  appid: '',
  app_key: '',
  domains: '',
  redirect_uris: '',
  sample_params: '',
  enable_refresh: false,
  status: 1,
  regenerate_key: false
}

// parseSampleParams / formatSampleParams 接入示例参数在表单里按「每行 key=value」编辑，存储为 JSON 对象
const parseSampleParams = (text: string): Record<string, string> => {
  const out: Record<string, string> = {}
  text.split(/\n/).forEach((line) => {
    const i = line.indexOf('=')
    if (i <= 0) return
    const k = line.slice(0, i).trim()
    if (k) out[k] = line.slice(i + 1).trim()
  })
  return out
}

const formatSampleParams = (params?: Record<string, string>): string =>
  Object.entries(params || {})
    .map(([k, v]) => `${k}=${v}`)
    .join('\n')

export default function Apps() {
  const isAdmin = useIsAdmin()
  const [apps, setApps] = useState<App[]>([])
  const [providers, setProviders] = useState<Provider[]>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const [form, setForm] = useState<AppForm>(emptyForm)
  const [saving, setSaving] = useState(false)
  const [regenerating, setRegenerating] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<App | null>(null)
  const [docsApp, setDocsApp] = useState<App | null>(null)
  const [docsTab, setDocsTab] = useState('rainbow')

  const loadProviders = useCallback(async () => {
    try {
      const raw = isAdmin ? await providersApi.list() : await providersApi.public()
      const list = Array.isArray(raw)
        ? raw.map((r) => ({ id: 0, ...r, enabled: true, main_site: true, sort: 0, client_id: '', client_secret: '', config: '', callback_url: '' }))
        : (raw as { list: Provider[] }).list
      setProviders(list || [])
    } catch {
      // 忽略
    }
  }, [isAdmin])

  const load = useCallback(async () => {
    try {
      const data = await appsApi.list()
      setApps(data.list)
    } catch (err) {
      toast.error((err as Error).message)
    }
  }, [])

  useEffect(() => {
    load()
    loadProviders()
  }, [load, loadProviders])

  const providerOptions = useMemo(() => providers.map((p) => ({ name: p.name, display_name: p.display_name })), [providers])

  const allTypesSelected = providerOptions.length > 0 && providerOptions.every((p) => form.types.includes(p.name))
  const allTypesIndeterminate = form.types.length > 0 && !allTypesSelected

  const openCreate = () => {
    setForm(emptyForm)
    setDialogOpen(true)
  }

  const openEdit = (row: App) => {
    if (providers.length === 0) loadProviders()
    setForm({
      id: row.id,
      name: row.name,
      platform: row.platform,
      mode: row.mode,
      types: row.types || [],
      appid: row.appid,
      app_key: row.app_key,
      domains: row.domains,
      redirect_uris: (row.redirect_uris || []).join('\n'),
      sample_params: formatSampleParams(row.sample_params),
      enable_refresh: !!row.enable_refresh,
      status: row.status,
      regenerate_key: false
    })
    setDialogOpen(true)
  }

  const onSave = async () => {
    if (!form.name.trim()) return toast.warning('请输入应用名称')
    setSaving(true)
    try {
      const payload = {
        name: form.name.trim(),
        platform: form.platform,
        mode: form.mode,
        types: form.types,
        domains: form.domains,
        redirect_uris: form.redirect_uris
          .split(/\n/)
          .map((s) => s.trim())
          .filter(Boolean),
        sample_params: parseSampleParams(form.sample_params),
        enable_refresh: form.enable_refresh,
        status: form.status,
        regenerate_key: form.regenerate_key
      }
      if (form.id) {
        const res = await appsApi.update(form.id, payload)
        setForm((f) => ({ ...f, app_key: res.app_key, regenerate_key: false }))
      } else {
        await appsApi.create(payload)
      }
      toast.success('保存成功')
      setDialogOpen(false)
      load()
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setSaving(false)
    }
  }

  const onRegenerate = async () => {
    if (!form.id) return
    setRegenerating(true)
    try {
      const res = await appsApi.update(form.id, {
        name: form.name,
        platform: form.platform,
        mode: form.mode,
        types: form.types,
        domains: form.domains,
        redirect_uris: form.redirect_uris
          .split(/\n/)
          .map((s) => s.trim())
          .filter(Boolean),
        sample_params: parseSampleParams(form.sample_params),
        enable_refresh: form.enable_refresh,
        status: form.status,
        regenerate_key: true
      })
      setForm((f) => ({ ...f, app_key: res.app_key, regenerate_key: false }))
      toast.success('AppKey 已重新生成')
      load()
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setRegenerating(false)
    }
  }

  const onDelete = async () => {
    if (!deleteTarget) return
    try {
      await appsApi.remove(deleteTarget.id)
      toast.success('删除成功')
      setDeleteTarget(null)
      load()
    } catch (err) {
      toast.error((err as Error).message)
    }
  }

  const rowById = (id?: number) => apps.find((a) => a.id === id)

  const copy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      toast.success('已复制')
    } catch {
      toast.warning('复制失败，请手动复制')
    }
  }

  const openDocs = (row: App) => {
    setDocsApp(row)
    setDocsTab(row.mode === 'rest' ? 'rest' : row.mode === 'oauth2' ? 'oauth2' : 'rainbow')
  }

  const baseUrl = window.location.origin
  const docs = useMemo(() => {
    if (!docsApp) return null
    const type = (docsApp.types && docsApp.types[0]) || 'qq'
    const domain = (docsApp.domains || '').split(/\n/).map((s) => s.trim()).filter(Boolean)[0] || 'example.com'
    const callback = `https://${domain}/oauth/callback`
    const oauth2Callback = ((docsApp.redirect_uris || [])[0]) || callback
    // 示例地址里的接入方参数取自「接入示例参数」（应用管理中配置），未配置时回落到 defaults
    const withSamples = (defaults: Record<string, string>) =>
      Object.entries({ ...defaults, ...(docsApp.sample_params || {}) })
        .map(([k, v]) => `${k}=${encodeURIComponent(v)}`)
        .join('&')
    return {
      type,
      callback,
      rainbowLogin: `${baseUrl}/api/connect.php?act=login&appid=${docsApp.appid}&appkey=${docsApp.app_key}&type=${type}&redirect_uri=${encodeURIComponent(callback)}`,
      rainbowReturn: `${callback}?type=${type}&code=520DD95263C1CFEA0870FBB66E******&sign=xxxxxxxx`,
      rainbowCallback: `${baseUrl}/api/connect.php?act=callback&appid=${docsApp.appid}&appkey=${docsApp.app_key}&type=${type}&code=520DD95263C1CFEA0870FBB66E******`,
      rainbowQuery: `${baseUrl}/api/connect.php?act=query&appid=${docsApp.appid}&appkey=${docsApp.app_key}&type=${type}&social_uid=AD3F5033279C8187CBCBB29235D5F827`,
      restLogin: JSON.stringify(
        { appid: docsApp.appid, appkey: docsApp.app_key, type, redirect_uri: callback },
        null,
        2
      ),
      restReturn: `${callback}?type=${type}&code=520DD95263C1CFEA0870FBB66E******&sign=xxxxxxxx`,
      restUserinfo: JSON.stringify(
        { appid: docsApp.appid, code: '520DD95263C1CFEA0870FBB66E******', type, sign: 'md5(appid&code&type&key)' },
        null,
        2
      ),
      restQuery: JSON.stringify(
        { appid: docsApp.appid, type, social_uid: 'AD3F5033279C8187CBCBB29235D5F827', sign: 'md5(...)' },
        null,
        2
      ),
      oauth2Callback,
      oauth2Authorize: `${baseUrl}/authorize?response_type=code&client_id=${docsApp.appid}&redirect_uri=${encodeURIComponent(oauth2Callback)}&scope=openid%20profile&${withSamples({ state: 'YOUR_STATE' })}`,
      oauth2Return: `${oauth2Callback}?code=520DD95263C1CFEA0870FBB66E******&${withSamples({ state: 'YOUR_STATE' })}`,
      oauth2Token: `curl -X POST ${baseUrl}/token \\
  -d "grant_type=authorization_code" \\
  -d "code=520DD95263C1CFEA0870FBB66E******" \\
  -d "client_id=${docsApp.appid}" \\
  -d "client_secret=${docsApp.app_key}" \\
  -d "redirect_uri=${encodeURIComponent(oauth2Callback)}"`,
      oauth2Userinfo: `curl ${baseUrl}/userinfo \\
  -H "Authorization: Bearer ACCESS_TOKEN"`,
      oauth2Discovery: docsApp.oidc_discovery_url || `${baseUrl}/.well-known/openid-configuration`
    }
  }, [docsApp, baseUrl])

  return (
    <div>
      <PageHeader
        title="应用管理"
        description="应用即接入本站点的目标站点。创建后生成 AppID 与 AppKey，供彩虹协议与 REST 接口调用。"
        actions={
          <Button onClick={openCreate}>
            <Plus className="h-4 w-4" /> 新建应用
          </Button>
        }
      />

      <div className="overflow-hidden rounded-md border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-14">ID</TableHead>
              <TableHead>名称</TableHead>
              <TableHead className="w-20">平台</TableHead>
              <TableHead className="w-28">模式</TableHead>
              <TableHead>AppID</TableHead>
              <TableHead>AppKey</TableHead>
              <TableHead>登录类型</TableHead>
              <TableHead className="w-16">状态</TableHead>
              <TableHead className="w-44 text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {apps.length === 0 && (
              <TableRow>
                <TableCell colSpan={9} className="h-24 text-center text-sm text-muted-foreground">
                  暂无应用，点击右上角「新建应用」开始接入。
                </TableCell>
              </TableRow>
            )}
            {apps.map((app) => (
              <TableRow key={app.id}>
                <TableCell className="text-muted-foreground">{app.id}</TableCell>
                <TableCell className="font-medium">{app.name}</TableCell>
                <TableCell className="text-muted-foreground">{app.platform || '-'}</TableCell>
                <TableCell>
                  <Badge variant={MODE_LABELS[app.mode]?.variant || 'outline'}>{MODE_LABELS[app.mode]?.label || app.mode}</Badge>
                </TableCell>
                <TableCell>
                  <button onClick={() => copy(app.appid)} className="font-mono text-xs text-muted-foreground transition-colors duration-150 hover:text-foreground" title="点击复制">
                    {app.appid}
                  </button>
                </TableCell>
                <TableCell>
                  <button onClick={() => copy(app.app_key)} className="font-mono text-xs text-muted-foreground transition-colors duration-150 hover:text-foreground" title="点击复制">
                    {app.app_key.slice(0, 12)}…
                  </button>
                </TableCell>
                <TableCell>
                  <div className="flex flex-wrap gap-1">
                    {(app.types || []).slice(0, 4).map((t) => (
                      <Badge key={t} variant="secondary" className="font-normal">
                        {TYPE_LABELS[t] || t}
                      </Badge>
                    ))}
                    {(app.types || []).length > 4 && (
                      <Badge variant="outline" className="font-normal">
                        +{(app.types || []).length - 4}
                      </Badge>
                    )}
                    {(app.types || []).length === 0 && <span className="text-xs text-muted-foreground">未选择</span>}
                  </div>
                </TableCell>
                <TableCell>
                  <StatusBadge status={app.status === 1 ? 'success' : 'muted'}>{app.status === 1 ? '启用' : '禁用'}</StatusBadge>
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-1">
                    <Button variant="ghost" size="sm" onClick={() => openDocs(app)}>
                      接入文档
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => openEdit(app)}>
                      编辑
                    </Button>
                    <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => setDeleteTarget(app)}>
                      删除
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {/* 创建/编辑弹窗 */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{form.id ? '编辑应用' : '新建应用'}</DialogTitle>
            <DialogDescription>目标站点的基础信息与回调白名单配置。</DialogDescription>
          </DialogHeader>

          <div className="grid gap-5">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label>名称 *</Label>
                <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="目标站点名称" />
              </div>
              <div className="space-y-1.5">
                <Label>平台</Label>
                <Select value={form.platform} onValueChange={(v) => setForm({ ...form, platform: v })}>
                  <SelectTrigger>
                    <SelectValue placeholder="选择平台" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="web">Web</SelectItem>
                    <SelectItem value="ios">iOS</SelectItem>
                    <SelectItem value="android">Android</SelectItem>
                    <SelectItem value="pc">PC</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="space-y-2">
              <Label>接入模式</Label>
              <div className="flex flex-wrap gap-2">
                {(['compat', 'rainbow', 'rest', 'oauth2'] as const).map((m) => (
                  <button
                    key={m}
                    type="button"
                    onClick={() => setForm({ ...form, mode: m })}
                    className={`rounded-md border px-3 py-1.5 text-sm transition-colors duration-150 ${
                      form.mode === m
                        ? 'border-foreground bg-foreground text-background'
                        : 'border-border text-muted-foreground hover:border-foreground/40 hover:text-foreground'
                    }`}
                  >
                    {MODE_LABELS[m].label}
                  </button>
                ))}
              </div>
              <p className="text-xs text-muted-foreground">
                兼容模式同时开放彩虹聚合登录协议、REST 风格接口与标准 OAuth2/OIDC 授权服务器协议。
              </p>
            </div>

            <div className="space-y-2">
              <Label>登录类型</Label>
              <label className="flex items-center gap-2 text-sm text-muted-foreground">
                <input
                  type="checkbox"
                  className="h-4 w-4 accent-foreground"
                  checked={allTypesSelected}
                  ref={(el) => {
                    if (el) el.indeterminate = allTypesIndeterminate
                  }}
                  onChange={(e) =>
                    setForm({ ...form, types: e.target.checked ? providerOptions.map((p) => p.name) : [] })
                  }
                />
                全选
              </label>
              <div className="flex flex-wrap gap-x-5 gap-y-2 rounded-md border border-border p-3">
                {providerOptions.length === 0 && <span className="text-sm text-muted-foreground">暂无可选登录渠道</span>}
                {providerOptions.map((p) => (
                  <label key={p.name} className="flex cursor-pointer items-center gap-2 text-sm">
                    <Checkbox
                      checked={form.types.includes(p.name)}
                      onCheckedChange={(checked) =>
                        setForm({
                          ...form,
                          types: checked
                            ? [...form.types, p.name]
                            : form.types.filter((t) => t !== p.name)
                        })
                      }
                    />
                    {p.display_name}
                  </label>
                ))}
              </div>
              <p className="text-xs text-muted-foreground">该目标站点向自己用户开放的第三方登录方式。</p>
            </div>

            {form.id ? (
              <div className="space-y-2">
                <Separator />
                <div className="space-y-3">
                  <div className="space-y-1.5">
                    <Label>AppID</Label>
                    <div className="flex gap-2">
                      <Input value={form.appid} readOnly className="font-mono text-xs" />
                      <Button variant="outline" onClick={() => copy(form.appid)}>
                        复制
                      </Button>
                    </div>
                  </div>
                  <div className="space-y-1.5">
                    <Label>AppKey</Label>
                    <div className="flex gap-2">
                      <Input value={form.app_key} readOnly className="font-mono text-xs" />
                      <Button variant="outline" onClick={() => copy(form.app_key)}>
                        复制
                      </Button>
                      <Button variant="outline" onClick={onRegenerate} disabled={regenerating}>
                        {regenerating ? '生成中…' : '重新生成'}
                      </Button>
                    </div>
                    <p className="text-xs text-muted-foreground">重新生成后旧 AppKey 立即失效，请同步更新目标站点配置。</p>
                  </div>
                </div>
              </div>
            ) : (
              <Alert variant="info">
                <AlertTitle>凭证信息</AlertTitle>
                <AlertDescription>保存后系统将自动生成 AppID 与 AppKey，自动生成且不可自定义。</AlertDescription>
              </Alert>
            )}

            <div className="space-y-2">
              <Separator />
              <div className="space-y-1.5">
                <Label>回调白名单域名</Label>
                <Textarea
                  value={form.domains}
                  onChange={(e) => setForm({ ...form, domains: e.target.value })}
                  rows={4}
                  placeholder={'example.com\nwww.example.com'}
                />
                <p className="text-xs leading-relaxed text-muted-foreground">
                  每个域名一行（可含子域名）。redirect_uri 的域名等于白名单域名或其子域名时允许回跳，区分子域名（彩虹/REST 协议使用）。
                </p>
              </div>

              <div className="space-y-1.5">
                <Label>OAuth2/OIDC 回调地址</Label>
                <Textarea
                  value={form.redirect_uris}
                  onChange={(e) => setForm({ ...form, redirect_uris: e.target.value })}
                  rows={3}
                  placeholder={'https://example.com/oauth/callback'}
                />
                <p className="text-xs leading-relaxed text-muted-foreground">
                  每行一个完整回调地址（按 scheme+host+path 匹配，忽略 query，故接入方在回跳地址上追加动态参数也能通过）。接入标准 OAuth2 / OIDC（authorization code flow）时必填。
                </p>
              </div>

              <div className="space-y-1.5">
                <Label>接入示例参数</Label>
                <Textarea
                  value={form.sample_params}
                  onChange={(e) => setForm({ ...form, sample_params: e.target.value })}
                  rows={3}
                  placeholder={'state=YOUR_STATE\napp_id=FT20260923XWAIGM'}
                />
                <p className="text-xs leading-relaxed text-muted-foreground">
                  每行一个 <code>key=value</code>。接入方自带的参数样例（如 WPS SSO 的 app_id / oauth_state / state），仅用于生成下方接入文档中的示例地址，不参与任何校验。留空则用默认示例值。
                </p>
              </div>

              <div className="space-y-1.5">
                <Label>OIDC Discovery URL（自动发现）</Label>
                <div className="flex gap-2">
                  <Input
                    value={form.id ? (rowById(form.id)?.oidc_discovery_url || '') : ''}
                    readOnly
                    className="font-mono text-xs"
                    placeholder="保存后自动生成"
                  />
                  <Button
                    variant="outline"
                    onClick={() => rowById(form.id)?.oidc_discovery_url && copy(rowById(form.id)!.oidc_discovery_url)}
                  >
                    复制
                  </Button>
                </div>
                <p className="text-xs leading-relaxed text-muted-foreground">
                  供 oidc-client / Keycloak 等 SDK 配置（issuer 地址即本站 HOST）。
                </p>
              </div>

              <div className="flex items-center justify-between">
                <Label>签发刷新令牌</Label>
                <div className="flex items-center gap-2">
                  <span className="text-sm text-muted-foreground">
                    {form.enable_refresh ? '允许 refresh_token 轮换' : '不签发 refresh_token'}
                  </span>
                  <Switch
                    checked={form.enable_refresh}
                    onCheckedChange={(v) => setForm({ ...form, enable_refresh: v })}
                  />
                </div>
              </div>
            </div>

            <div className="flex items-center justify-between">
              <Label>状态</Label>
              <div className="flex items-center gap-2">
                <span className="text-sm text-muted-foreground">{form.status === 1 ? '启用' : '禁用'}</span>
                <Switch checked={form.status === 1} onCheckedChange={(v) => setForm({ ...form, status: v ? 1 : 0 })} />
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={onSave} disabled={saving}>
              {saving ? '保存中…' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 接入文档弹窗 */}
      <Dialog open={!!docsApp} onOpenChange={(open) => !open && setDocsApp(null)}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>接入文档 · {docsApp?.name}</DialogTitle>
            <DialogDescription>以下示例中的 appid / appkey 请在应用中复制替换，签名规则见「签名规则」。</DialogDescription>
          </DialogHeader>
          {docs && (
            <div className="mb-3 flex flex-wrap items-center gap-2">
              <span className="text-sm text-muted-foreground">已开通登录方式：</span>
              {(docsApp?.types || []).map((t) => (
                <Badge key={t} variant="secondary" className="font-normal">
                  {TYPE_LABELS[t] || t}
                </Badge>
              ))}
              {(docsApp?.types || []).length === 0 && <span className="text-sm text-muted-foreground">未选择或未配置</span>}
            </div>
          )}
          <Tabs value={docsTab} onValueChange={setDocsTab}>
            <TabsList>
              <TabsTrigger value="rainbow">彩虹聚合协议</TabsTrigger>
              <TabsTrigger value="rest">REST 接口</TabsTrigger>
              <TabsTrigger value="oauth2">OAuth2/OIDC</TabsTrigger>
              <TabsTrigger value="sign">签名规则</TabsTrigger>
            </TabsList>
            {docs && (
              <>
                <TabsContent value="rainbow" className="space-y-4">
                  <div className="space-y-1">
                    <p className="text-sm font-medium">1. 获取跳转登录地址</p>
                    <CodeBlock code={docs.rainbowLogin} />
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-medium">2. 用户登录后回跳（GET）</p>
                    <CodeBlock code={docs.rainbowReturn} />
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-medium">3. 用 code 换取用户信息</p>
                    <CodeBlock code={docs.rainbowCallback} />
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-medium">4. 按第三方 UID 查询用户（可选）</p>
                    <CodeBlock code={docs.rainbowQuery} />
                  </div>
                </TabsContent>
                <TabsContent value="rest" className="space-y-4">
                  <div className="space-y-1">
                    <p className="text-sm font-medium">1. 获取跳转登录地址</p>
                    <CodeBlock code={`POST ${baseUrl}/api/v1/oauth/login
Content-Type: application/json

${docs.restLogin}`} />
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-medium">2. 用户登录后回跳（GET）</p>
                    <CodeBlock code={docs.restReturn} />
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-medium">3. 用 code 换取用户信息（服务端签名）</p>
                    <CodeBlock code={`POST ${baseUrl}/api/v1/oauth/userinfo
Content-Type: application/json

${docs.restUserinfo}`} />
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-medium">4. 按第三方 UID 查询用户（可选）</p>
                    <CodeBlock code={`POST ${baseUrl}/api/v1/oauth/query
Content-Type: application/json

${docs.restQuery}`} />
                  </div>
                </TabsContent>
                <TabsContent value="oauth2" className="space-y-4">
                  <div className="space-y-1">
                    <p className="text-sm font-medium">0. OIDC Discovery（可选，主流 SDK 自动发现）</p>
                    <CodeBlock code={docs.oauth2Discovery} />
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-medium">1. 发起授权（浏览器跳转，appid 即 client_id、appkey 即 client_secret）</p>
                    <CodeBlock code={docs.oauth2Authorize} />
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-medium">2. 用户授权后 302 回跳（GET）</p>
                    <CodeBlock code={docs.oauth2Return} />
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-medium">3. 用 code 换取令牌（access_token / refresh_token / id_token）</p>
                    <CodeBlock code={docs.oauth2Token} />
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-medium">4. 获取用户信息（Bearer access_token）</p>
                    <CodeBlock code={docs.oauth2Userinfo} />
                  </div>
                  <p className="text-sm text-muted-foreground">
                    支持 PKCE（code_challenge + code_challenge_method=S256）、refresh_token 轮换与 RS256 签名的 id_token（公钥见 /jwks）。兼容旧前缀 /api/oauth2/* 别名。
                  </p>
                </TabsContent>
                <TabsContent value="sign" className="space-y-3">
                  <p className="text-sm text-muted-foreground">REST 接口的 userinfo / query 使用服务端签名校验：</p>
                  <CodeBlock
                    code={`1. 除 sign 外的所有参数按 key 升序排列
2. 拼接为 k1=v1&k2=v2
3. 末尾拼接 &key=AppKey
4. sign = md5(上述字符串)

示例（AppKey 为 abc123）：
sign = md5("appid=xxxx&code=yyyy&type=qq&key=abc123")`}
                  />
                  <p className="text-sm text-muted-foreground">彩虹协议则直接传 appid + appkey 参数鉴权。</p>
                </TabsContent>
              </>
            )}
          </Tabs>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={`确定删除应用「${deleteTarget?.name}」吗？`}
        description="删除后该应用的 AppID / AppKey 将立即失效，目标站点将无法继续使用本平台登录。"
        confirmText="删除"
        destructive
        onConfirm={onDelete}
      />
    </div>
  )
}
