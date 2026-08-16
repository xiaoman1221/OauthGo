import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useUserStore } from '@/store/user'

export default function OauthCallback() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const setToken = useUserStore((s) => s.setToken)
  const fetchUser = useUserStore((s) => s.fetchUser)
  const [status, setStatus] = useState('第三方登录中，请稍候…')

  useEffect(() => {
    const token = params.get('token')
    if (!token) {
      setStatus('未获取到登录令牌')
      toast.error('登录失败：未获取到令牌')
      setTimeout(() => navigate('/login', { replace: true }), 1200)
      return
    }
    setToken(token)
    fetchUser()
      .then(() => {
        toast.success('登录成功')
        navigate('/', { replace: true })
      })
      .catch(() => {
        setStatus('登录状态校验失败')
        toast.error('登录失败')
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
