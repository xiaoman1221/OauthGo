import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { ExternalLink } from 'lucide-react'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { StatusBadge } from '@/components/status-badge'
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
import { providersApi, type Provider } from '@/lib/api'
import { channelSchemaTyped, type ProviderSchema } from '@/lib/providerDocs'

export default function Providers() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const [form, setForm] = useState<Provider | null>(null)
  const [config, setConfig] = useState<Record<string, unknown>>({})
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)

  const load = async () => {
    try {
      const data = await providersApi.list()
      setProviders(data.list)
    } catch (err) {
      toast.error((err as Error).message)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const schema: ProviderSchema | null = useMemo(() => (form ? channelSchemaTyped(form.name) : null), [form])

  const openDialog = (row: Provider) => {
    setForm({ ...row, main_site: !!row.main_site, enabled: !!row.enabled })
    try {
      const parsed = row.config ? (JSON.parse(row.config) as Record<string, unknown>) : {}
      parsed.use_proxy = !!parsed.use_proxy
      setConfig(parsed)
    } catch {
      setConfig({ use_proxy: false })
    }
    setDialogOpen(true)
  }

  const copy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      toast.success('已复制')
    } catch {
      toast.warning('复制失败，请手动复制')
    }
  }

  const onTest = async () => {
    if (!form) return
    setTesting(true)
    try {
      const payload: Record<string, unknown> = {
        client_id: form.client_id,
        config: JSON.stringify(config)
      }
      if (form.client_secret) payload.client_secret = form.client_secret
      const data = await providersApi.test(form.name, payload)
      toast.success(data.message || '配置有效')
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setTesting(false)
    }
  }

  const onSave = async () => {
    if (!form) return
    setSaving(true)
    try {
      const payload: Record<string, unknown> = {
        client_id: form.client_id,
        enabled: form.enabled,
        main_site: form.main_site,
        config: JSON.stringify(config)
      }
      if (form.client_secret) payload.client_secret = form.client_secret
      await providersApi.update(form.name, payload)
      toast.success('保存成功')
      setDialogOpen(false)
      load()
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div>
      <PageHeader
        title="登录渠道"
        description="配置第三方登录渠道。启用且「应用于主站」的渠道会展示在主站登录页。"
      />

      <div className="overflow-hidden rounded-md border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-14">ID</TableHead>
              <TableHead>渠道</TableHead>
              <TableHead>ClientID / AppID</TableHead>
              <TableHead>回调地址</TableHead>
              <TableHead className="w-20">主站</TableHead>
              <TableHead className="w-20">状态</TableHead>
              <TableHead className="w-20 text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {providers.map((p) => (
              <TableRow key={p.id}>
                <TableCell className="text-muted-foreground">{p.id}</TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{p.display_name}</span>
                    <Badge variant={p.category === 'enterprise' ? 'warning' : 'secondary'} className="font-normal">
                      {p.category === 'enterprise' ? '企业' : '社交'}
                    </Badge>
                    <span className="text-xs text-muted-foreground">{p.name}</span>
                  </div>
                </TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">{p.client_id || '-'}</TableCell>
                <TableCell className="max-w-[220px] truncate text-xs text-muted-foreground">
                  {p.name === 'wechat_miniprogram' ? '前端 code 登录，无需回调' : p.callback_url}
                </TableCell>
                <TableCell>
                  <StatusBadge status={p.main_site ? 'success' : 'muted'}>{p.main_site ? '启用' : '停用'}</StatusBadge>
                </TableCell>
                <TableCell>
                  <StatusBadge status={p.enabled ? 'success' : 'muted'}>{p.enabled ? '启用' : '禁用'}</StatusBadge>
                </TableCell>
                <TableCell className="text-right">
                  <Button variant="ghost" size="sm" onClick={() => openDialog(p)}>
                    配置
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>配置「{form?.display_name}」渠道</DialogTitle>
            <DialogDescription>{schema?.tips}</DialogDescription>
          </DialogHeader>

          {schema && form && (
            <div className="grid gap-5">
              {schema.registerUrl && (
                <a
                  href={schema.registerUrl}
                  target="_blank"
                  rel="noreferrer"
                  className="inline-flex items-center gap-1.5 text-sm text-foreground underline-offset-4 hover:underline"
                >
                  {schema.registerLabel || '接入地址'} <ExternalLink className="h-3.5 w-3.5" />
                </a>
              )}

              <div className="flex items-center justify-between rounded-md border border-border px-4 py-3">
                <span className="text-sm">启用渠道</span>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground">开启后该渠道可发起登录</span>
                  <Switch checked={form.enabled} onCheckedChange={(v) => setForm({ ...form, enabled: v })} />
                </div>
              </div>
              <div className="flex items-center justify-between rounded-md border border-border px-4 py-3">
                <span className="text-sm">应用于主站登录</span>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground">开启后展示在主站登录页</span>
                  <Switch checked={form.main_site} onCheckedChange={(v) => setForm({ ...form, main_site: v })} />
                </div>
              </div>

              <div className="space-y-1.5">
                <Label>{schema.idLabel}</Label>
                <Input value={form.client_id} onChange={(e) => setForm({ ...form, client_id: e.target.value })} placeholder={schema.idPlaceholder} />
              </div>
              <div className="space-y-1.5">
                <Label>{schema.secretLabel}</Label>
                <Input
                  type="password"
                  value={form.client_secret}
                  onChange={(e) => setForm({ ...form, client_secret: e.target.value })}
                  placeholder={form.client_secret ? '留空则不修改' : schema.secretPlaceholder}
                />
              </div>

              <div className="space-y-1.5">
                <Label>回调地址</Label>
                <div className="flex gap-2">
                  <Input value={form.callback_url} readOnly className="font-mono text-xs" />
                  <Button variant="outline" onClick={() => copy(form.callback_url)}>
                    复制
                  </Button>
                </div>
                <p className="text-xs text-muted-foreground">由系统根据 HOST 自动拼接，不支持自定义。</p>
              </div>

              {schema.configFields.length > 0 && (
                <div className="space-y-3">
                  <Separator />
                  <p className="text-sm font-medium">{schema.divider || '扩展配置'}</p>
                  {schema.configFields.map((f) => (
                    <div key={f.key} className="space-y-1.5">
                      <Label>{f.label}</Label>
                      {f.type === 'select' ? (
                        <Select
                          value={String(config[f.key] || '')}
                          onValueChange={(v) => setConfig({ ...config, [f.key]: v })}
                        >
                          <SelectTrigger>
                            <SelectValue placeholder="请选择" />
                          </SelectTrigger>
                          <SelectContent>
                            {(f.options || []).map((o) => (
                              <SelectItem key={o.value} value={o.value}>
                                {o.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      ) : f.type === 'textarea' ? (
                        <Textarea
                          value={String(config[f.key] || '')}
                          onChange={(e) => setConfig({ ...config, [f.key]: e.target.value })}
                          rows={f.rows || 5}
                          placeholder={f.placeholder}
                        />
                      ) : (
                        <Input
                          value={String(config[f.key] || '')}
                          onChange={(e) => setConfig({ ...config, [f.key]: e.target.value })}
                          placeholder={f.placeholder}
                        />
                      )}
                    </div>
                  ))}
                </div>
              )}

              {schema.supportProxy && (
                <div className="space-y-2">
                  <Separator />
                  <div className="flex items-center justify-between rounded-md border border-border px-4 py-3">
                    <span className="text-sm">使用代理</span>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-muted-foreground">境外渠道建议开启（系统设置配置 SOCKS5）</span>
                      <Switch
                        checked={!!config.use_proxy}
                        onCheckedChange={(v) => setConfig({ ...config, use_proxy: v })}
                      />
                    </div>
                  </div>
                </div>
              )}

              {schema.notes.length > 0 && (
                <Alert variant="warning">
                  <AlertTitle>注意事项</AlertTitle>
                  <AlertDescription>
                    <ul className="list-disc space-y-1 pl-4">
                      {schema.notes.map((n, i) => (
                        <li key={i}>{n}</li>
                      ))}
                    </ul>
                  </AlertDescription>
                </Alert>
              )}
            </div>
          )}

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button variant="outline" onClick={onTest} disabled={testing}>
              {testing ? '测试中…' : '测试渠道'}
            </Button>
            <Button onClick={onSave} disabled={saving}>
              {saving ? '保存中…' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
