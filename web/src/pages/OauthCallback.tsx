import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { authApi } from '@/lib/api'
import { useUserStore } from '@/store/user'

export default function OauthCallback() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const setToken = useUserStore((s) => s.setToken)
  const setUserInfo = useUserStore((s) => s.setUserInfo)
  const fetchUser = useUserStore((s) => s.fetchUser)
  const [status, setStatus] = useState('第三方登录中，请稍候…')

  useEffect(() => {
    const code = params.get('code')
    const token = params.get('token')
    if (!code && !token) {
      setStatus('未获取到登录凭据')
      toast.error('登录失败：未获取到登录凭据')
      setTimeout(() => navigate('/login', { replace: true }), 1200)
      return
    }

    // 自动适配回调参数：
    // - code：一次性登录码，调用 /auth/code-exchange 兑换 JWT（推荐，JWT 不进入 URL/浏览器历史）
    // - token：旧流程直接携带 JWT（保留兼容）
    const login = code
      ? authApi.exchangeLoginCode(code).then((data) => {
          setToken(data.token)
          if (data.user) setUserInfo(data.user)
        })
      : Promise.resolve().then(() => {
          setToken(token!)
          return fetchUser()
        })

    login
      .then(() => {
        toast.success('登录成功')
        navigate('/', { replace: true })
      })
      .catch((err) => {
        setStatus('登录失败')
        toast.error((err as Error).message || '登录失败')
        setTimeout(() => navigate('/login', { replace: true }), 1200)
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background">
      <div className="flex h-10 w-10 items-center justify-center rounded bg-foreground text-sm font-semibold text-background">
        O
      </div>
      <div className="mt-6 flex items-center gap-2 text-sm text-muted-foreground">
        <span className="h-4 w-4 animate-spin rounded-full border border-muted-foreground/30 border-t-muted-foreground" />
        {status}
      </div>
    </div>
  )
}
