import { useEffect, useMemo, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import {
  BookOpenText,
  LayoutDashboard,
  Link2,
  ListOrdered,
  LogOut,
  Menu,
  Settings,
  User as UserIcon,
  Users
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { ThemeToggle } from '@/components/theme-toggle'
import { UserAvatar } from '@/components/user-avatar'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { useIsAdmin, useUserStore } from '@/store/user'
import { useAvatarStore } from '@/store/avatar'

const TITLES: Record<string, string> = {
  '/dashboard': '仪表盘',
  '/apps': '应用管理',
  '/logins': '登录记录',
  '/providers': '登录渠道',
  '/settings': '系统设置',
  '/users': '用户管理',
  '/user-center': '用户中心',
  '/docs/providers': '第三方渠道接入',
  '/docs/service': '本站服务文档'
}

function NavItem({ to, icon: Icon, label, end }: { to: string; icon: React.ElementType; label: string; end?: boolean }) {
  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) =>
        cn(
          'flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm transition-colors duration-150',
          isActive
            ? 'bg-accent font-medium text-foreground'
            : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground'
        )
      }
    >
      <Icon className="h-4 w-4 shrink-0" />
      <span>{label}</span>
    </NavLink>
  )
}

export function AppShell() {
  const location = useLocation()
  const navigate = useNavigate()
  const isAdmin = useIsAdmin()
  const userInfo = useUserStore((s) => s.userInfo)
  const logout = useUserStore((s) => s.logout)
  const fetchUser = useUserStore((s) => s.fetchUser)
  const loadAvatarSettings = useAvatarStore((s) => s.load)
  const avatarLoaded = useAvatarStore((s) => s.loaded)
  const [mobileNavOpen, setMobileNavOpen] = useState(false)

  useEffect(() => {
    if (!userInfo) {
      fetchUser().catch(() => {})
    }
    if (!avatarLoaded) {
      loadAvatarSettings()
    }
  }, [userInfo, fetchUser, avatarLoaded, loadAvatarSettings])

  const title = TITLES[location.pathname] || '统一授权管理'

  const mobileNavItems = useMemo(
    () =>
      [
        { to: '/dashboard', label: '仪表盘', icon: LayoutDashboard },
        { to: '/apps', label: '应用管理', icon: Link2 },
        { to: '/logins', label: '登录记录', icon: ListOrdered },
        ...(isAdmin ? [{ to: '/providers', label: '登录渠道', icon: Link2 }] : []),
        ...(isAdmin ? [{ to: '/settings', label: '系统设置', icon: Settings }] : []),
        ...(isAdmin ? [{ to: '/users', label: '用户管理', icon: Users }] : []),
        { to: '/user-center', label: '用户中心', icon: UserIcon },
        { to: '/docs/service', label: '服务文档', icon: BookOpenText },
        ...(isAdmin ? [{ to: '/docs/providers', label: '渠道接入文档', icon: BookOpenText }] : [])
      ] as const,
    [isAdmin]
  )

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <div className="min-h-screen bg-background">
      {/* 侧边栏 */}
      <aside className="fixed inset-y-0 left-0 z-30 hidden w-56 flex-col border-r border-border bg-background md:flex">
        <div className="flex h-14 items-center border-b border-border px-5">
          <NavLink to="/dashboard" className="flex items-center gap-2">
            <span className="flex h-6 w-6 items-center justify-center rounded bg-foreground text-[11px] font-semibold text-background">
              O
            </span>
            <span className="text-[15px] font-medium tracking-tight">OauthGo</span>
          </NavLink>
        </div>
        <nav className="flex-1 space-y-0.5 overflow-y-auto px-3 py-4">
          <NavItem to="/dashboard" icon={LayoutDashboard} label="仪表盘" />
          <NavItem to="/apps" icon={Link2} label="应用管理" />
          <NavItem to="/logins" icon={ListOrdered} label="登录记录" />
          {isAdmin && <NavItem to="/providers" icon={Link2} label="登录渠道" />}
          {isAdmin && <NavItem to="/settings" icon={Settings} label="系统设置" />}
          {isAdmin && <NavItem to="/users" icon={Users} label="用户管理" />}
          <div className="my-3 border-t border-border" />
          <NavItem to="/user-center" icon={UserIcon} label="用户中心" />
          <NavItem to="/docs/service" icon={BookOpenText} label="服务文档" />
          {isAdmin && <NavItem to="/docs/providers" icon={BookOpenText} label="渠道接入文档" />}
        </nav>
        <div className="border-t border-border p-3">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button className="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left transition-colors duration-150 hover:bg-accent">
                <UserAvatar avatar={userInfo?.avatar} email={userInfo?.email} username={userInfo?.nickname || userInfo?.username} size="sm" />
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-medium">{userInfo?.nickname || userInfo?.username || '未登录'}</div>
                  <div className="truncate text-xs text-muted-foreground">{userInfo?.role === 'admin' ? '管理员' : '普通用户'}</div>
                </div>
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-48">
              <DropdownMenuLabel>{userInfo?.email || userInfo?.username}</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => navigate('/user-center')}>
                <UserIcon /> 个人中心
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleLogout}>
                <LogOut /> 退出登录
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </aside>

      {/* 移动端顶栏 */}
      <header className="sticky top-0 z-20 flex h-14 items-center justify-between border-b border-border bg-background/95 px-4 backdrop-blur md:hidden">
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="icon" onClick={() => setMobileNavOpen(true)} aria-label="打开导航">
            <Menu className="h-5 w-5" />
          </Button>
          <NavLink to="/dashboard" className="flex items-center gap-2">
            <span className="flex h-6 w-6 items-center justify-center rounded bg-foreground text-[11px] font-semibold text-background">O</span>
            <span className="text-[15px] font-medium tracking-tight">OauthGo</span>
          </NavLink>
        </div>
        <div className="flex items-center gap-1">
          <ThemeToggle />
          <Button variant="ghost" size="icon" onClick={() => navigate('/user-center')}>
            <UserIcon className="h-4 w-4" />
          </Button>
        </div>
      </header>

      {/* 移动端导航弹窗 */}
      <Dialog open={mobileNavOpen} onOpenChange={setMobileNavOpen}>
        <DialogContent className="max-w-xs gap-4 sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>导航</DialogTitle>
            <DialogDescription>OauthGo 控制台</DialogDescription>
          </DialogHeader>
          <nav className="grid gap-1">
            {mobileNavItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                onClick={() => setMobileNavOpen(false)}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-2.5 rounded-md px-3 py-2.5 text-sm transition-colors duration-150',
                    isActive
                      ? 'bg-accent font-medium text-foreground'
                      : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground'
                  )
                }
              >
                <item.icon className="h-4 w-4 shrink-0" />
                <span>{item.label}</span>
              </NavLink>
            ))}
          </nav>
          <Button
            variant="outline"
            className="mt-1 w-full text-destructive"
            onClick={() => {
              setMobileNavOpen(false)
              handleLogout()
            }}
          >
            <LogOut className="h-4 w-4" /> 退出登录
          </Button>
        </DialogContent>
      </Dialog>

      {/* 主区域 */}
      <div className="md:pl-56">
        <header className="sticky top-0 z-20 hidden h-14 items-center justify-between border-b border-border bg-background/80 px-8 backdrop-blur md:flex">
          <h2 className="text-sm font-medium tracking-tight text-muted-foreground">{title}</h2>
          <div className="flex items-center gap-1">
            <ThemeToggle />
          </div>
        </header>
        <main className="mx-auto w-full max-w-6xl px-5 py-8 md:px-8 md:py-10">
          <div key={location.pathname} className="animate-fade-in">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}
