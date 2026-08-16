import { Routes, Route, Navigate } from 'react-router-dom'
import { AppShell } from '@/components/app-shell'
import { useIsLogin, useUserStore } from '@/store/user'
import Login from '@/pages/Login'
import Register from '@/pages/Register'
import ForgotPassword from '@/pages/ForgotPassword'
import OauthCallback from '@/pages/OauthCallback'
import Dashboard from '@/pages/Dashboard'
import Apps from '@/pages/Apps'
import Logins from '@/pages/Logins'
import Providers from '@/pages/Providers'
import ProviderDocs from '@/pages/ProviderDocs'
import ServiceDocs from '@/pages/ServiceDocs'
import UserCenter from '@/pages/UserCenter'
import Settings from '@/pages/Settings'
import Users from '@/pages/Users'
import { LoadingScreen } from '@/components/loading-screen'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const isLogin = useIsLogin()
  if (!isLogin) return <Navigate to="/login" replace />
  return <>{children}</>
}

function RequireAdmin({ children }: { children: React.ReactNode }) {
  const userInfo = useUserStore((s) => s.userInfo)
  // 页面刷新后 userInfo 尚未加载，等待 fetchUser 完成后校验，避免误跳转
  if (userInfo === null) return <LoadingScreen label="权限校验中…" />
  if (userInfo.role !== 'admin') return <Navigate to="/dashboard" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route path="/forgot-password" element={<ForgotPassword />} />
      <Route path="/oauth-callback" element={<OauthCallback />} />

      <Route
        element={
          <RequireAuth>
            <AppShell />
          </RequireAuth>
        }
      >
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/apps" element={<Apps />} />
        <Route path="/logins" element={<Logins />} />
        <Route path="/user-center" element={<UserCenter />} />
        <Route path="/docs/service" element={<ServiceDocs />} />
        <Route
          path="/providers"
          element={
            <RequireAdmin>
              <Providers />
            </RequireAdmin>
          }
        />
        <Route
          path="/settings"
          element={
            <RequireAdmin>
              <Settings />
            </RequireAdmin>
          }
        />
        <Route
          path="/users"
          element={
            <RequireAdmin>
              <Users />
            </RequireAdmin>
          }
        />
        <Route
          path="/docs/providers"
          element={
            <RequireAdmin>
              <ProviderDocs />
            </RequireAdmin>
          }
        />
      </Route>

      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  )
}
