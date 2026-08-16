import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { ArrowRight } from 'lucide-react'
import { AuthLayout } from '@/components/auth-layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { authApi } from '@/lib/api'

interface AuthConfig {
  register_enabled: boolean
  register_email_verify: boolean
  password_min_length: number
}

export default function Register() {
  const navigate = useNavigate()
  const [config, setConfig] = useState<AuthConfig>({
    register_enabled: true,
    register_email_verify: false,
    password_min_length: 6
  })
  const [username, setUsername] = useState('')
  const [account, setAccount] = useState('')
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [sending, setSending] = useState(false)
  const [countdown, setCountdown] = useState(0)
  const timerRef = useRef<number | null>(null)

  useEffect(() => {
    authApi
      .config()
      .then((cfg) =>
        setConfig({
          register_enabled: cfg.register_enabled !== false,
          register_email_verify: cfg.register_email_verify === true,
          password_min_length: (cfg.password_min_length as number) || 6
        })
      )
      .catch(() => {})
    return () => {
      if (timerRef.current) window.clearInterval(timerRef.current)
    }
  }, [])

  const isEmail = account.includes('@')

  const sendCode = async () => {
    if (!account.trim()) {
      toast.warning('请先填写邮箱')
      return
    }
    setSending(true)
    try {
      await authApi.sendCode({ scope: 'register', account: account.trim() })
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
      setSending(false)
    }
  }

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!username.trim()) return toast.warning('请输入用户名')
    if (!account.trim()) return toast.warning('请输入邮箱或手机号')
    if (isEmail && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(account.trim())) return toast.warning('邮箱格式不正确')
    if (!isEmail && !/^1\d{10}$/.test(account.trim())) return toast.warning('手机号格式不正确')
    if (password.length < config.password_min_length)
      return toast.warning(`密码长度不能少于 ${config.password_min_length} 位`)
    if (config.register_email_verify && isEmail && !code) return toast.warning('请输入邮箱验证码')

    setLoading(true)
    try {
      await authApi.register({
        username: username.trim(),
        email: isEmail ? account.trim() : '',
        phone: isEmail ? '' : account.trim(),
        password,
        code
      })
      toast.success('注册成功，请登录')
      navigate('/login')
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <AuthLayout>
      <div className="mb-10">
        <h2 className="text-3xl font-light tracking-tighter">创建账号</h2>
        <p className="mt-2 text-sm text-muted-foreground">注册后即可创建应用并接入第三方登录</p>
      </div>

      <form onSubmit={onSubmit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="username">用户名</Label>
          <Input id="username" value={username} onChange={(e) => setUsername(e.target.value)} placeholder="登录用户名" autoFocus />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="account">邮箱 / 手机号</Label>
          <Input
            id="account"
            value={account}
            onChange={(e) => {
              setAccount(e.target.value)
              if (!e.target.value.includes('@')) setCode('')
            }}
            placeholder="邮箱（或手机号）"
          />
        </div>
        {config.register_email_verify && isEmail && (
          <div className="space-y-1.5">
            <Label htmlFor="code">邮箱验证码</Label>
            <div className="flex gap-2">
              <Input id="code" value={code} onChange={(e) => setCode(e.target.value)} placeholder="验证码" />
              <Button type="button" variant="outline" className="shrink-0" disabled={sending || countdown > 0} onClick={sendCode}>
                {countdown > 0 ? `${countdown}s` : '发送验证码'}
              </Button>
            </div>
          </div>
        )}
        <div className="space-y-1.5">
          <Label htmlFor="password">密码</Label>
          <Input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder={`不少于 ${config.password_min_length} 位`}
            autoComplete="new-password"
          />
        </div>
        <Button type="submit" size="lg" className="w-full" disabled={loading}>
          {loading ? '注册中…' : '注册'}
          {!loading && <ArrowRight className="h-4 w-4" />}
        </Button>
      </form>

      <p className="mt-8 text-center text-sm text-muted-foreground">
        已有账号？{' '}
        <Link to="/login" className="font-medium text-foreground transition-colors duration-150 hover:text-muted-foreground">
          去登录
        </Link>
      </p>
    </AuthLayout>
  )
}
