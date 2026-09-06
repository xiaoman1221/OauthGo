// WebAuthn / Passkey 前端辅助：把浏览器 Credential 序列化为后端可校验的 JSON
import { http } from '@/lib/api'

export interface PasskeyOption {
  options: unknown
  session_id: string
}

function bufToB64(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf)
  let s = ''
  for (let i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i])
  return btoa(s).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

// base64url 字符串 → Uint8Array（浏览器 WebAuthn 要求 challenge / id 为 BufferSource）
function b64ToBuf(b64: string): Uint8Array {
  let s = b64.replace(/-/g, '+').replace(/_/g, '/')
  while (s.length % 4 !== 0) s += '='
  const bin = atob(s)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return bytes
}

type AnyRecord = Record<string, any>

// 注册选项：challenge / user.id / excludeCredentials[].id 转字节
export function prepareCreationOptions(options: AnyRecord): PublicKeyCredentialCreationOptions {
  const o = JSON.parse(JSON.stringify(options)) as AnyRecord
  if (o.challenge) o.challenge = b64ToBuf(o.challenge)
  if (o.user && o.user.id) o.user.id = b64ToBuf(o.user.id)
  if (Array.isArray(o.excludeCredentials)) {
    o.excludeCredentials = o.excludeCredentials.map((c: AnyRecord) => ({ ...c, id: b64ToBuf(c.id as string) }))
  }
  return o as PublicKeyCredentialCreationOptions
}

// 断言选项：challenge / allowCredentials[].id 转字节
export function prepareRequestOptions(options: AnyRecord): PublicKeyCredentialRequestOptions {
  const o = JSON.parse(JSON.stringify(options)) as AnyRecord
  if (o.challenge) o.challenge = b64ToBuf(o.challenge)
  if (Array.isArray(o.allowCredentials)) {
    o.allowCredentials = o.allowCredentials.map((c: AnyRecord) => ({ ...c, id: b64ToBuf(c.id as string) }))
  }
  return o as PublicKeyCredentialRequestOptions
}

export interface PasskeyResponseBody {
  id: string
  rawId: string
  type: string
  response: Record<string, unknown>
  transports?: string[]
}

// 登录断言：PublicKeyCredential → 后端期望的 JSON
export function serializeAssertion(cred: PublicKeyCredential): PasskeyResponseBody {
  const r = cred.response as AuthenticatorAssertionResponse
  const body: PasskeyResponseBody = {
    id: cred.id,
    rawId: bufToB64(cred.rawId),
    type: cred.type,
    response: {
      clientDataJSON: bufToB64(r.clientDataJSON),
      authenticatorData: bufToB64(r.authenticatorData),
      signature: bufToB64(r.signature)
    }
  }
  if (r.userHandle) {
    ;(body.response as Record<string, unknown>).userHandle = bufToB64(r.userHandle)
  }
  return body
}

// 注册创建：PublicKeyCredential → 后端期望的 JSON（含 attestationObject）
export function serializeAttestation(cred: PublicKeyCredential): PasskeyResponseBody {
  const r = cred.response as AuthenticatorAttestationResponse
  const body: PasskeyResponseBody = {
    id: cred.id,
    rawId: bufToB64(cred.rawId),
    type: cred.type,
    response: {
      clientDataJSON: bufToB64(r.clientDataJSON),
      attestationObject: bufToB64(r.attestationObject)
    }
  }
  const withTransports = r as AuthenticatorAttestationResponse & { getTransports?: () => string[] }
  if (typeof withTransports.getTransports === 'function') {
    body.transports = withTransports.getTransports()
  }
  return body
}

export interface PasskeyLoginResult {
  token: string
  user: Record<string, unknown>
}

export async function passkeyLogin(username: string): Promise<void> {
  const begin = await http.get<{ code: number; data: PasskeyOption }>('/auth/passkey/login/begin', {
    params: { username }
  })
  const opt = begin.data.data
  if (!opt || !opt.options) throw new Error('获取 Passkey 登录挑战失败')

  const cred = (await navigator.credentials.get({
    publicKey: prepareRequestOptions(opt.options as AnyRecord)
  })) as PublicKeyCredential | null
  if (!cred) throw new Error('已取消 Passkey 认证')

  const body = serializeAssertion(cred)
  const fin = await http.post<{ code: number; data: PasskeyLoginResult }>(
    `/auth/passkey/login/finish?session_id=${encodeURIComponent(opt.session_id)}`,
    body
  )
  const data = fin.data.data
  if (!data || !data.token) throw new Error('Passkey 登录未返回令牌')
  localStorage.setItem('token', data.token)
  window.location.href = '/oauth-callback?token=' + encodeURIComponent(data.token)
}

// 完整注册流程（调用方需已登录）
export async function startPasskeyRegistration(name: string): Promise<void> {
  const res = await http.post<{ code: number; data: PasskeyOption }>('/auth/passkey/register/begin', {})
  const opt = res.data.data
  if (!opt || !opt.options) throw new Error('获取注册挑战失败')
  const cred = (await navigator.credentials.create({
    publicKey: prepareCreationOptions(opt.options as AnyRecord)
  })) as PublicKeyCredential | null
  if (!cred) throw new Error('已取消 Passkey 创建')
  const body = serializeAttestation(cred)
  const q = `session_id=${encodeURIComponent(opt.session_id)}&name=${encodeURIComponent(name || 'Passkey')}`
  await http.post(`/auth/passkey/register/finish?${q}`, body)
}

export function isWebAuthnSupported(): boolean {
  return typeof window !== 'undefined' && !!window.PublicKeyCredential
}
