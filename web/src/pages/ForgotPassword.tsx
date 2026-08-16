import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { ArrowRight } from 'lucide-react'
import { AuthLayout } from '@/components/auth-layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { authApi } from '@/lib/api'

export default function ForgotPassword() {
  const navigate = useNavigate()
  const [account, setAccount] = useState('')
  const [code, setCode] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [minLen, setMinLen] = useState(6)
  const [loading, setLoading] = useState(false)
  const [sending, setSending] = useState(false)
  const [countdown, setCountdown] = useState(0)
  const timerRef = useRef<number | null>(null)

  useEffect(() => {
    authApi
      .config()
      .then((cfg) => setMinLen((cfg.password_min_length as number) || 6))
      .catch(() => {})
    return () => {
      if (timerRef.current) window.clearInterval(timerRef.current)
    }
  }, [])

  const sendCode = async () => {
    if (!account.trim()) return toast.warning('请先填写注册账号')
    setSending(true)
    try {
      await authApi.sendCode({ scope: 'reset', account: account.trim() })
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
    if (!account.trim()) return toast.warning('请输入注册邮箱或手机号')
    if (!code) return toast.warning('请输入验证码')
    if (password.length < minLen) return toast.warning(`密码长度不能少于 ${minLen} 位`)
    if (password !== confirm) return toast.warning('两次输入的密码不一致')

    setLoading(true)
    try {
      await authApi.forgot({ account: account.trim(), code, password })
      toast.success('密码重置成功，请登录')
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
        <h2 className="text-3xl font-light tracking-tighter">找回密码</h2>
        <p className="mt-2 text-sm text-muted-foreground">通过注册邮箱或手机号验证码重置密码</p>
      </div>

      <form onSubmit={onSubmit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="account">注册账号</Label>
          <Input id="account" value={account} onChange={(e) => setAccount(e.target.value)} placeholder="注册邮箱或手机号" autoFocus />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="code">验证码</Label>
          <div className="flex gap-2">
            <Input id="code" value={code} onChange={(e) => setCode(e.target.value)} placeholder="验证码" />
            <Button type="button" variant="outline" className="shrink-0" disabled={sending || countdown > 0} onClick={sendCode}>
              {countdown > 0 ? `${countdown}s` : '发送验证码'}
            </Button>
          </div>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="password">新密码</Label>
          <Input id="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder={`不少于 ${minLen} 位`} autoComplete="new-password" />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="confirm">确认密码</Label>
          <Input id="confirm" type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} placeholder="请再次输入新密码" autoComplete="new-password" />
        </div>
        <Button type="submit" size="lg" className="w-full" disabled={loading}>
          {loading ? '重置中…' : '重置密码'}
          {!loading && <ArrowRight className="h-4 w-4" />}
        </Button>
      </form>

      <p className="mt-8 text-center text-sm text-muted-foreground">
        <Link to="/login" className="font-medium text-foreground transition-colors duration-150 hover:text-muted-foreground">
          返回登录
        </Link>
      </p>
    </AuthLayout>
  )
}
