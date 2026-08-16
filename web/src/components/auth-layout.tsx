import * as React from 'react'
import { Link } from 'react-router-dom'
import { ThemeToggle } from '@/components/theme-toggle'

// 认证页容器：纯主题色背景 + 居中表单（登录页为独立落地页，见 pages/Login.tsx）
export function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative flex min-h-screen flex-col bg-background">
      <header className="flex items-center justify-between px-6 py-6 md:px-10">
        <Link to="/login" className="flex items-center gap-2.5">
          <span className="flex h-7 w-7 items-center justify-center rounded bg-foreground text-xs font-semibold text-background">
            O
          </span>
          <span className="text-base font-medium tracking-tight">OauthGo</span>
        </Link>
        <ThemeToggle />
      </header>
      <main className="flex flex-1 items-center justify-center px-6 pb-24">
        <div className="w-full max-w-sm animate-fade-in">{children}</div>
      </main>
    </div>
  )
}
