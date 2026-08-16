import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { settingsApi, type SettingDef } from '@/lib/api'
import { useAvatarStore } from '@/store/avatar'

const GROUP_LABELS: Record<string, string> = {
  site: '站点设置',
  security: '安全设置',
  avatar: '头像设置',
  smtp: 'SMTP 邮件',
  sms: '短信设置',
  proxy: '代理设置',
  template: '邮件模板'
}

const BOOL_KEYS = ['register_enabled', 'register_email_verify', 'smtp_enabled', 'smtp_tls', 'proxy_enabled', 'gravatar_mirror_enabled']

const SMS_PROVIDERS = [
  { value: 'none', label: '未启用' },
  { value: 'aliyun', label: '阿里云短信' },
  { value: 'tencent', label: '腾讯云短信' },
  { value: 'smsbao', label: '短信宝' }
]

const SMS_PROVIDER_FIELDS: Record<string, string[]> = {
  none: [],
  aliyun: ['sms_access_key_id', 'sms_access_key_secret', 'sms_region_id', 'sms_sign_name', 'sms_aliyun_template_code'],
  tencent: ['sms_access_key_id', 'sms_access_key_secret', 'sms_region_id', 'sms_sign_name', 'sms_tencent_sdk_app_id', 'sms_tencent_template_id'],
  smsbao: ['smsbao_username', 'smsbao_password', 'sms_sign_name']
}

export default function Settings() {
  const [groups, setGroups] = useState<Record<string, SettingDef[]>>({})
  const [values, setValues] = useState<Record<string, string>>({})
  const [activeTab, setActiveTab] = useState('site')
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testEmail, setTestEmail] = useState('')
  const [testPhone, setTestPhone] = useState('')

  useEffect(() => {
    settingsApi
      .list()
      .then((data) => {
        setGroups(data.groups || {})
        const v: Record<string, string> = {}
        Object.values(data.groups || {}).forEach((items) => {
          items.forEach((item) => {
            v[item.key] = item.value
          })
        })
        setValues(v)
      })
      .catch((err) => toast.error((err as Error).message))
  }, [])

  const set = (key: string, value: string) => setValues((v) => ({ ...v, [key]: value }))

  const smsProvider = values.sms_provider || 'none'

  const visibleItems = useMemo(() => {
    const items = groups[activeTab] || []
    if (activeTab !== 'sms') return items
    const visible = SMS_PROVIDER_FIELDS[smsProvider] || []
    return items.filter((item) => item.key === 'sms_provider' || visible.includes(item.key))
  }, [groups, activeTab, smsProvider])

  const isSwitch = (key: string) => BOOL_KEYS.includes(key)

  const onSave = async () => {
    setSaving(true)
    try {
      const payload = Object.values(groups)
        .flat()
        .filter((item) => values[item.key] !== '********')
        .map((item) => ({ key: item.key, value: values[item.key] || '' }))
      await settingsApi.update(payload)
      // 头像设置可能已变更，刷新头像配置（侧边栏/列表头像即时生效）
      await useAvatarStore.getState().load().catch(() => {})
      toast.success('保存成功')
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setSaving(false)
    }
  }

  const onTestSMTP = async () => {
    if (!testEmail.trim()) return toast.warning('请先输入收件邮箱')
    setTesting(true)
    try {
      await settingsApi.testSMTP(testEmail.trim())
      toast.success('测试邮件已发送')
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setTesting(false)
    }
  }

  const onTestSMS = async () => {
    if (!testPhone.trim()) return toast.warning('请先输入收件手机号')
    setTesting(true)
    try {
      await settingsApi.testSMS(testPhone.trim())
      toast.success('测试短信已发送')
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setTesting(false)
    }
  }

  const bgMode = values.login_bg_mode || 'color'

  return (
    <div>
      <PageHeader
        title="系统设置"
        description="站点信息、安全策略、邮件短信与代理配置。"
        actions={
          <Button onClick={onSave} disabled={saving}>
            {saving ? '保存中…' : '保存'}
          </Button>
        }
      />

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="flex-wrap h-auto">
          {Object.keys(GROUP_LABELS).map((g) => (
            <TabsTrigger key={g} value={g}>
              {GROUP_LABELS[g]}
            </TabsTrigger>
          ))}
        </TabsList>

        <div className="mt-6 max-w-2xl space-y-5">
          <TabsContent value="site" className="mt-0 space-y-5">
            {groups.site?.map((item) => (
              <SettingRow key={item.key} label={item.description} hint={item.key}>
                {item.key === 'site_desc' ? (
                  <Textarea rows={3} value={values[item.key] || ''} onChange={(e) => set(item.key, e.target.value)} />
                ) : isSwitch(item.key) ? (
                  <div className="flex items-center gap-2">
                    <Switch checked={(values[item.key] || '0') === '1'} onCheckedChange={(v) => set(item.key, v ? '1' : '0')} />
                    <span className="text-sm text-muted-foreground">{values[item.key] === '1' ? '开启' : '关闭'}</span>
                  </div>
                ) : (
                  <Input value={values[item.key] || ''} onChange={(e) => set(item.key, e.target.value)} />
                )}
              </SettingRow>
            ))}

            <div className="space-y-2 border-t border-border pt-5">
              <Label>登录页背景</Label>
              <div className="flex flex-wrap items-end gap-3">
                <Select value={bgMode} onValueChange={(v) => set('login_bg_mode', v)}>
                  <SelectTrigger className="w-44">
                    <SelectValue placeholder="选择背景类型" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="color">纯色</SelectItem>
                    <SelectItem value="image">图片 URL</SelectItem>
                    <SelectItem value="bing">Bing 每日一图</SelectItem>
                  </SelectContent>
                </Select>
                {bgMode === 'image' && (
                  <Input
                    value={values.login_bg_image_url || ''}
                    onChange={(e) => set('login_bg_image_url', e.target.value)}
                    placeholder="图片 URL"
                    className="w-64"
                  />
                )}
                {bgMode === 'color' && (
                  <Input
                    type="color"
                    value={values.login_bg_color || '#1f4037'}
                    onChange={(e) => set('login_bg_color', e.target.value)}
                    className="h-9 w-24 p-1"
                  />
                )}
                {bgMode === 'bing' && <span className="text-sm text-muted-foreground">使用 Bing 每日一图作为登录页背景</span>}
              </div>
              <div
                className="mt-3 flex h-20 w-60 items-center justify-center rounded-md border border-border"
                style={
                  bgMode === 'color'
                    ? { background: values.login_bg_color || '#1f4037' }
                    : bgMode === 'image'
                      ? { backgroundImage: `url(${values.login_bg_image_url})`, backgroundSize: 'cover', backgroundPosition: 'center' }
                      : { backgroundImage: 'url(/api/site/bing-daily)', backgroundSize: 'cover', backgroundPosition: 'center' }
                }
              >
                <span className="rounded bg-black/35 px-2 py-1 text-xs text-white">预览</span>
              </div>
            </div>
          </TabsContent>

          {(['avatar', 'security', 'smtp', 'sms', 'proxy', 'template'] as const).map((g) => (
            <TabsContent key={g} value={g} className="mt-0 space-y-5">
              {visibleItems.map((item) => (
                <SettingRow key={item.key} label={item.description} hint={item.key}>
                  {item.key === 'avatar_source' ? (
                    <Select value={values[item.key] || 'auto'} onValueChange={(v) => set(item.key, v)}>
                      <SelectTrigger className="w-72">
                        <SelectValue placeholder="选择头像来源" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="auto">自动（QQ 邮箱用 QQ 头像，其余用 Gravatar）</SelectItem>
                        <SelectItem value="qq">仅 QQ 邮箱使用 QQ 头像</SelectItem>
                        <SelectItem value="gravatar">全部使用 Gravatar</SelectItem>
                      </SelectContent>
                    </Select>
                  ) : item.key === 'sms_provider' ? (
                    <Select value={values[item.key] || 'none'} onValueChange={(v) => set(item.key, v)}>
                      <SelectTrigger className="w-52">
                        <SelectValue placeholder="选择短信服务商" />
                      </SelectTrigger>
                      <SelectContent>
                        {SMS_PROVIDERS.map((p) => (
                          <SelectItem key={p.value} value={p.value}>
                            {p.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  ) : isSwitch(item.key) ? (
                    <div className="flex items-center gap-2">
                      <Switch checked={(values[item.key] || '0') === '1'} onCheckedChange={(v) => set(item.key, v ? '1' : '0')} />
                      <span className="text-sm text-muted-foreground">{values[item.key] === '1' ? '开启' : '关闭'}</span>
                    </div>
                  ) : item.sensitive ? (
                    <Input type="password" value={values[item.key] === '********' ? '' : values[item.key] || ''} onChange={(e) => set(item.key, e.target.value)} placeholder="已保存（留空则不修改）" />
                  ) : g === 'template' ? (
                    <Textarea rows={4} value={values[item.key] || ''} onChange={(e) => set(item.key, e.target.value)} />
                  ) : (
                    <Input value={values[item.key] || ''} onChange={(e) => set(item.key, e.target.value)} />
                  )}
                </SettingRow>
              ))}

              {g === 'smtp' && (
                <div className="flex items-end gap-2 border-t border-border pt-5">
                  <div className="space-y-1.5">
                    <Label>发信测试</Label>
                    <Input value={testEmail} onChange={(e) => setTestEmail(e.target.value)} placeholder="输入收件邮箱" className="w-72" />
                  </div>
                  <Button variant="outline" disabled={testing} onClick={onTestSMTP}>
                    {testing ? '发送中…' : '发送测试邮件'}
                  </Button>
                </div>
              )}
              {g === 'sms' && smsProvider !== 'none' && (
                <div className="flex items-end gap-2 border-t border-border pt-5">
                  <div className="space-y-1.5">
                    <Label>发送测试</Label>
                    <Input value={testPhone} onChange={(e) => setTestPhone(e.target.value)} placeholder="输入收件手机号" className="w-72" />
                  </div>
                  <Button variant="outline" disabled={testing} onClick={onTestSMS}>
                    {testing ? '发送中…' : '发送测试短信'}
                  </Button>
                </div>
              )}
            </TabsContent>
          ))}
        </div>
      </Tabs>
    </div>
  )
}

function SettingRow({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="grid gap-1.5">
      <Label>{label}</Label>
      {children}
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  )
}
