import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { ArrowRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { ThemeToggle } from '@/components/theme-toggle'
import { providersApi } from '@/lib/api'
import { useUserStore } from '@/store/user'

interface PublicProvider {
  name: string
  display_name: string
  category: string
}

export default function Login() {
  const navigate = useNavigate()
  const token = useUserStore((s) => s.token)
  const login = useUserStore((s) => s.login)

  const [open, setOpen] = useState(false)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [providers, setProviders] = useState<PublicProvider[]>([])
  const [loading, setLoading] = useState(false)

  // 已登录用户直接进入控制台
  useEffect(() => {
    if (token) {
      navigate('/dashboard', { replace: true })
    }
  }, [token, navigate])

  useEffect(() => {
    providersApi
      .public()
      .then((list) => setProviders(list || []))
      .catch(() => {})
  }, [])

  const openLogin = () => {
    setUsername('')
    setPassword('')
    setOpen(true)
  }

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!username.trim() || !password) {
      toast.warning('请输入账号与密码')
      return
    }
    setLoading(true)
    try {
      await login(username.trim(), password)
      toast.success('登录成功')
      setOpen(false)
      navigate('/dashboard')
    } catch (err) {
      toast.error((err as Error).message)
    } finally {
      setLoading(false)
    }
  }

  const onProvider = (p: PublicProvider) => {
    window.location.href = `/api/oauth/${p.name}/login`
  }

  return (
    <div className="relative flex min-h-screen flex-col bg-background">
      <header className="flex items-center justify-between px-6 py-6 md:px-10">
        <div className="flex items-center gap-2.5">
          <span className="flex h-7 w-7 items-center justify-center rounded bg-foreground text-xs font-semibold text-background">
            O
          </span>
          <span className="text-base font-medium tracking-tight">OauthGo</span>
        </div>
        <ThemeToggle />
      </header>

      <main className="flex flex-1 flex-col items-center justify-center px-6 pb-28 text-center">
        <div className="max-w-2xl animate-fade-in">
          <h1 className="text-5xl font-light leading-[1.05] tracking-tighter text-foreground sm:text-6xl">
            统一授权，<br className="sm:hidden" />一行接入。
          </h1>
          <p className="mx-auto mt-6 max-w-md text-[15px] leading-relaxed text-muted-foreground">
            聚合微信、QQ、GitHub、Google 等二十余个第三方登录渠道，为你的目标站点提供彩虹协议与 REST 两种接入方式。
          </p>
          <div className="mt-10">
            <Button type="button" size="lg" className="h-11 px-8" onClick={openLogin}>
              进入控制台
              <ArrowRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </main>

      <footer className="pb-6 text-center text-xs text-muted-foreground">
        <Link to="/docs" target="_blank" className="transition-colors duration-150 hover:text-foreground">
          接口文档
        </Link>
      </footer>

      {/* 登录弹窗：中心渐变出现 */}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-sm gap-5">
          <DialogHeader>
            <DialogTitle>登录控制台</DialogTitle>
            <DialogDescription>使用用户名、邮箱或手机号登录</DialogDescription>
          </DialogHeader>

          <form onSubmit={onSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="login-username">账号</Label>
              <Input
                id="login-username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="用户名 / 邮箱 / 手机号"
                autoComplete="username"
                autoFocus
              />
            </div>
            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <Label htmlFor="login-password">密码</Label>
                <Link
                  to="/forgot-password"
                  className="text-xs text-muted-foreground transition-colors duration-150 hover:text-foreground"
                >
                  忘记密码？
                </Link>
              </div>
              <Input
                id="login-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="请输入密码"
                autoComplete="current-password"
              />
            </div>
            <Button type="submit" size="lg" className="w-full" disabled={loading}>
              {loading ? '登录中…' : '登录'}
              {!loading && <ArrowRight className="h-4 w-4" />}
            </Button>
          </form>

          {providers.length > 0 && (
            <>
              <div className="flex items-center gap-4">
                <Separator className="flex-1" />
                <span className="text-xs text-muted-foreground">其他登录方式</span>
                <Separator className="flex-1" />
              </div>
              <div className="flex flex-wrap gap-2">
                {providers.map((p) => (
                  <Button key={p.name} type="button" variant="outline" size="sm" onClick={() => onProvider(p)}>
                    {p.display_name}
                  </Button>
                ))}
              </div>
            </>
          )}

          <DialogFooter className="sm:justify-center">
            <p className="text-sm text-muted-foreground">
              还没有账号？{' '}
              <Link to="/register" className="font-medium text-foreground transition-colors duration-150 hover:text-muted-foreground">
                注册
              </Link>
            </p>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
