import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowRight } from 'lucide-react'
import { PageHeader } from '@/components/page-header'
import { Skeleton } from '@/components/ui/skeleton'
import { appsApi, loginsApi, usersApi } from '@/lib/api'
import { useIsAdmin, useUserStore } from '@/store/user'

interface Stat {
  label: string
  value: number | null
  to?: string
}

export default function Dashboard() {
  const isAdmin = useIsAdmin()
  const userInfo = useUserStore((s) => s.userInfo)
  const [stats, setStats] = useState<Stat[]>([
    { label: '应用数量', value: null, to: '/apps' },
    { label: '登录记录', value: null, to: '/logins' }
  ])
  const [platformCounts, setPlatformCounts] = useState<{ k: string; v: number }[]>([])
  const [loginStatusCounts, setLoginStatusCounts] = useState<{ k: string; v: number }[]>([])

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const [apps, logins] = await Promise.all([
          appsApi.list(),
          loginsApi.list({ page: 1, page_size: 50 })
        ])
        if (cancelled) return

        const platformMap: Record<string, number> = {}
        for (const a of apps.list) {
          const p = a.platform || 'unknown'
          platformMap[p] = (platformMap[p] || 0) + 1
        }
        const statusMap: Record<string, number> = {}
        for (const r of logins.list || []) {
          const s = r.status === undefined ? 'unknown' : String(r.status)
          statusMap[s] = (statusMap[s] || 0) + 1
        }
        setStats([
          { label: '应用数量', value: apps.list.length, to: '/apps' },
          { label: '登录记录', value: logins.total, to: '/logins' }
        ])
        setPlatformCounts(Object.entries(platformMap).map(([k, v]) => ({ k, v })))
        setLoginStatusCounts(Object.entries(statusMap).map(([k, v]) => ({ k, v })))
      } catch {
        // 统计失败不阻塞页面
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (!isAdmin) return
    usersApi
      .list({ page: 1, page_size: 1 })
      .then((res) => setStats((s) => [...s, { label: '用户数量', value: res.total, to: '/users' }]))
      .catch(() => {})
  }, [isAdmin])

  const platformTotal = platformCounts.reduce((s, c) => s + c.v, 0)
  const loginTotal = loginStatusCounts.reduce((s, c) => s + c.v, 0)

  const statusLabel = (k: string) => (k === '1' ? '成功' : k === '0' ? '失败' : k)

  return (
    <div>
      <PageHeader
        title="仪表盘"
        description={`欢迎回来，${userInfo?.nickname || userInfo?.username || ''}。这里是 OauthGo 的运行概览。`}
      />

      {/* 指标区：无卡片、细线分隔、超大数字 */}
      <section className="grid grid-cols-1 divide-y divide-border border-y border-border sm:grid-cols-3 sm:divide-x sm:divide-y-0">
        {stats.map((s) =>
          s.value === null ? (
            <div key={s.label} className="px-1 py-6">
              <Skeleton className="h-9 w-20" />
              <div className="mt-2 h-4 w-16">
                <Skeleton className="h-4 w-16" />
              </div>
            </div>
          ) : (
            <Link key={s.label} to={s.to || '#'} className="group px-1 py-6 transition-colors duration-150 hover:bg-accent/40">
              <div className="text-4xl font-light tracking-tighter text-foreground">{s.value}</div>
              <div className="mt-2 flex items-center gap-1.5 text-sm text-muted-foreground">
                {s.label}
                {s.to && <ArrowRight className="h-3.5 w-3.5 opacity-0 transition-opacity duration-150 group-hover:opacity-100" />}
              </div>
            </Link>
          )
        )}
      </section>

      {/* 非对称双栏分布 */}
      <section className="mt-12 grid grid-cols-1 gap-12 lg:grid-cols-5">
        <div className="lg:col-span-2">
          <h3 className="text-sm font-medium uppercase tracking-wider text-muted-foreground">应用平台分布</h3>
          <div className="mt-5 space-y-3">
            {platformCounts.length === 0 && <p className="text-sm text-muted-foreground">暂无数据</p>}
            {platformCounts.map((p) => (
              <div key={p.k} className="flex items-center gap-3 text-sm">
                <span className="w-16 shrink-0 text-muted-foreground">{p.k}</span>
                <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                  <div
                    className="h-full rounded-full bg-foreground/80 transition-all duration-300 ease-out"
                    style={{ width: platformTotal ? `${(p.v / platformTotal) * 100}%` : '0%' }}
                  />
                </div>
                <span className="w-8 text-right tabular-nums">{p.v}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="lg:col-span-3">
          <h3 className="text-sm font-medium uppercase tracking-wider text-muted-foreground">最近登录状态分布</h3>
          <div className="mt-5 space-y-3">
            {loginStatusCounts.length === 0 && <p className="text-sm text-muted-foreground">暂无数据</p>}
            {loginStatusCounts.map((s) => (
              <div key={s.k} className="flex items-center gap-3 text-sm">
                <span className="w-16 shrink-0 text-muted-foreground">{statusLabel(s.k)}</span>
                <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                  <div
                    className={`h-full rounded-full transition-all duration-300 ease-out ${s.k === '1' ? 'bg-emerald-500' : s.k === '0' ? 'bg-red-500' : 'bg-foreground/70'}`}
                    style={{ width: loginTotal ? `${(s.v / loginTotal) * 100}%` : '0%' }}
                  />
                </div>
                <span className="w-8 text-right tabular-nums">{s.v}</span>
              </div>
            ))}
          </div>
        </div>
      </section>
    </div>
  )
}
