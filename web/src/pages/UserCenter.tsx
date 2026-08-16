import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { StatusBadge } from '@/components/status-badge'
import { UserAvatar } from '@/components/user-avatar'
import { Separator } from '@/components/ui/separator'
import { ConfirmDialog } from '@/components/ui/confirm'
import { authApi, type Binding } from '@/lib/api'
import { useUserStore } from '@/store/user'
import { cn } from '@/lib/utils'

type Tab = 'profile' | 'password' | 'bindings'

const TABS: { key: Tab; label: string }[] = [
  { key: 'profile', label: '基本资料' },
  { key: 'password', label: '修改密码' },
  { key: 'bindings', label: '账号绑定' }
]

export default function UserCenter() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const userInfo = useUserStore((s) => s.userInfo)
  const setUserInfo = useUserStore((s) => s.setUserInfo)
  const fetchUser = useUserStore((s) => s.fetchUser)

  const [tab, setTab] = useState<Tab>('profile')
  const [bindings, setBindings] = useState<Binding[]>([])

  // profile form
  const [nickname, setNickname] = useState('')
  const [avatar, setAvatar] = useState('')
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [emailCode, setEmailCode] = useState('')
  const [phoneCode, setPhoneCode] = useState('')
  const [origEmail, setOrigEmail] = useState('')
  const [origPhone, setOrigPhone] = useState('')
  const [saving, setSaving] = useState(false)
  const [sending, setSending] = useState<'email' | 'phone' | null>(null)
  const [countdown, setCountdown] = useState(0)
  const timerRef = useRef<number | null>(null)

  // password form
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  // bindings
  const [bindingLoading, setBindingLoading] = useState('')
  const [unbindTarget, setUnbindTarget] = useState<Binding | null>(null)

  useEffect(() => {
    applyUser()
    loadBindings()
    const bind = params.get('bind')
    if (bind === 'success') {
      toast.success('第三方账号绑定成功')
      navigate('/user-center', { replace: true })
    } else if (bind === 'fail') {
      toast.error(params.get('msg') || '绑定失败')
      navigate('/user-center', { replace: true })
    }
    return () => {
      if (timerRef.current) window.clearInterval(timerRef.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function applyUser() {
    const u = userInfo
    setNickname(u?.nickname || '')
    setAvatar(u?.avatar || '')
    setUsername(u?.username || '')
    setEmail(u?.email || '')
    setPhone(u?.phone || '')
    setOrigEmail(u?.email || '')
    setOrigPhone(u?.phone || '')
  }

  async function loadBindings() {
    try {
      setBindings(await authApi.bindings())
    } catch {
      setBindings([])
    }
  }

  const emailChanged = email.trim() !== origEmail
  const phoneChanged = phone.trim() !== origPhone

  const onSaveProfile = async () => {
    setSaving(true)
    try {
      const user = await authApi.updateProfile({
        nickname,
        avatar,
        username,
        email,
        email_code: emailChanged ? emailCode : '',
        phone,
        phone_code: phoneChanged ? phoneCode : ''
      })
      setUserInfo(user)
      setOrigEmail(email.trim())
      setOrigPhone(phone.trim())
      setEmailCode('')
      setPhoneCode('')
      toast.success('保存成功')
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setSaving(false)
    }
  }

  const onSendCode = async (field: 'email' | 'phone') => {
    const account = (field === 'email' ? email : phone).trim()
    if (!account) {
      toast.warning(field === 'email' ? '请先填写邮箱' : '请先填写手机号')
      return
    }
    setSending(field)
    try {
      await authApi.sendCode({ scope: 'bind', account })
      toast.success('验证码已发送')
      setCountdown(60)
      timerRef.current = window.setInterval(() => {
        setCountdown((c) => {
          if (c <= 1 && timerRef.current) window.clearInterval(timerRef.current)
          return c - 1
        })
      }, 1000)
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setSending(null)
    }
  }

  const onChangePassword = async () => {
    if (newPassword.length < 6) return toast.warning('密码长度不能少于 6 位')
    if (newPassword !== confirmPassword) return toast.warning('两次输入的密码不一致')
    setSaving(true)
    try {
      await authApi.changePassword({
        old_password: userInfo?.password_set ? oldPassword : undefined,
        new_password: newPassword
      })
      toast.success('密码修改成功')
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
      await fetchUser()
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setSaving(false)
    }
  }

  const onBind = async (p: Binding) => {
    setBindingLoading(p.name)
    try {
      const data = await authApi.bindLogin(p.name)
      window.location.href = data.url
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setBindingLoading('')
    }
  }

  const onUnbind = async () => {
    if (!unbindTarget) return
    try {
      await authApi.unbindLogin(unbindTarget.name)
      toast.success('解绑成功')
      setUnbindTarget(null)
      loadBindings()
    } catch (err) {
      toast.error((err as Error).message)
    }
  }

  return (
    <div>
      <PageHeader title="用户中心" description="管理个人资料、登录密码与第三方账号绑定。" />

      <div className="grid grid-cols-1 gap-10 lg:grid-cols-4">
        {/* 左侧导航 */}
        <aside className="lg:sticky lg:top-20 lg:self-start">
          <div className="mb-6 flex items-center gap-3">
            <UserAvatar avatar={userInfo?.avatar} email={userInfo?.email} username={userInfo?.nickname || userInfo?.username} size="lg" />
            <div>
              <div className="font-medium">{userInfo?.nickname || userInfo?.username}</div>
              <div className="text-xs text-muted-foreground">{userInfo?.email || userInfo?.username}</div>
            </div>
          </div>
          <nav className="flex gap-1 lg:flex-col">
            {TABS.map((t) => (
              <button
                key={t.key}
                onClick={() => setTab(t.key)}
                className={cn(
                  'rounded-md px-3 py-2 text-left text-sm transition-colors duration-150',
                  tab === t.key ? 'bg-accent font-medium text-foreground' : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground'
                )}
              >
                {t.label}
              </button>
            ))}
          </nav>
        </aside>

        {/* 内容 */}
        <div className="lg:col-span-3">
          {tab === 'profile' && (
            <div className="max-w-xl space-y-5">
              <div className="flex items-center gap-4">
                <UserAvatar avatar={avatar} email={email} username={username} size="lg" />
                <div className="text-xs text-muted-foreground">头像支持：QQ 头像（QQ 邮箱）或 Gravatar（镜像地址可在系统设置中配置）</div>
              </div>
              <Field label="头像地址">
                <Input value={avatar} onChange={(e) => setAvatar(e.target.value)} placeholder="http(s) 图片地址，留空则自动使用 QQ 头像 / Gravatar" />
              </Field>
              <Field label="昵称">
                <Input value={nickname} onChange={(e) => setNickname(e.target.value)} placeholder="昵称" />
              </Field>
              <Field label="用户名">
                <Input value={username} onChange={(e) => setUsername(e.target.value)} placeholder="登录用户名" />
              </Field>
              {emailChanged && (
                <Field label="邮箱验证码">
                  <div className="flex gap-2">
                    <Input value={emailCode} onChange={(e) => setEmailCode(e.target.value)} placeholder="验证码" />
                    <Button variant="outline" className="shrink-0" disabled={sending === 'email' || countdown > 0} onClick={() => onSendCode('email')}>
                      {countdown > 0 ? `${countdown}s` : '发送验证码'}
                    </Button>
                  </div>
                </Field>
              )}
              <Field label="邮箱">
                <Input value={email} onChange={(e) => setEmail(e.target.value)} placeholder="绑定邮箱" />
              </Field>
              {phoneChanged && (
                <Field label="手机验证码">
                  <div className="flex gap-2">
                    <Input value={phoneCode} onChange={(e) => setPhoneCode(e.target.value)} placeholder="验证码" />
                    <Button variant="outline" className="shrink-0" disabled={sending === 'phone' || countdown > 0} onClick={() => onSendCode('phone')}>
                      {countdown > 0 ? `${countdown}s` : '发送验证码'}
                    </Button>
                  </div>
                </Field>
              )}
              <Field label="手机号">
                <Input value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="绑定手机号" />
              </Field>
              <div className="pt-2">
                <Button onClick={onSaveProfile} disabled={saving}>
                  {saving ? '保存中…' : '保存'}
                </Button>
              </div>
            </div>
          )}

          {tab === 'password' && (
            <div className="max-w-md space-y-5">
              {userInfo?.password_set && (
                <Field label="原密码">
                  <Input type="password" value={oldPassword} onChange={(e) => setOldPassword(e.target.value)} placeholder="请输入原密码" />
                </Field>
              )}
              <Field label="新密码">
                <Input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} placeholder="不少于 6 位" />
              </Field>
              <Field label="确认密码">
                <Input type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} placeholder="请再次输入新密码" />
              </Field>
              <div className="pt-2">
                <Button onClick={onChangePassword} disabled={saving}>
                  {saving ? '修改中…' : '修改密码'}
                </Button>
              </div>
              {!userInfo?.password_set && (
                <p className="text-xs text-muted-foreground">当前账号未设置密码（第三方注册），可直接设置。</p>
              )}
            </div>
          )}

          {tab === 'bindings' && (
            <div className="max-w-xl">
              <p className="mb-4 text-sm text-muted-foreground">绑定后可使用第三方渠道直接登录本平台。</p>
              {bindings.length === 0 && <p className="text-sm text-muted-foreground">暂无可绑定的登录渠道。</p>}
              <div className="space-y-2">
                {bindings.map((p) => (
                  <div key={p.name} className="flex items-center justify-between rounded-md border border-border px-4 py-3">
                    <div className="flex items-center gap-3">
                      {p.bound && p.avatar ? (
                        <UserAvatar avatar={p.avatar} email={email} username={p.nickname} size="sm" />
                      ) : (
                        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-muted text-xs font-medium">
                          {p.display_name.slice(0, 1)}
                        </span>
                      )}
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium">{p.display_name}</span>
                          <StatusBadge status={p.bound ? 'success' : 'muted'}>{p.bound ? '已绑定' : '未绑定'}</StatusBadge>
                        </div>
                        {p.bound && p.nickname && <div className="text-xs text-muted-foreground">{p.nickname}</div>}
                      </div>
                    </div>
                    {p.bound ? (
                      <Button variant="outline" size="sm" className="text-destructive" onClick={() => setUnbindTarget(p)}>
                        解绑
                      </Button>
                    ) : (
                      <Button size="sm" disabled={bindingLoading === p.name} onClick={() => onBind(p)}>
                        {bindingLoading === p.name ? '跳转中…' : '绑定'}
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      <Separator className="mt-12" />
      <p className="mt-4 text-xs text-muted-foreground">OauthGo · 统一授权管理平台</p>

      <ConfirmDialog
        open={!!unbindTarget}
        onOpenChange={(open) => !open && setUnbindTarget(null)}
        title={`确定解绑「${unbindTarget?.display_name}」吗？`}
        description="解绑后无法通过该渠道登录。若当前账号未设置密码且为最后一个绑定渠道，将无法解绑。"
        confirmText="解绑"
        destructive
        onConfirm={onUnbind}
      />
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}
