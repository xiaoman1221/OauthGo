# OauthGo

使用 Go 编写的第三方登录聚合平台（兼容彩虹聚合登录），同时可作为标准 **OAuth2 / OIDC 授权服务器**与 **IDP / CAS**，为下游站点提供统一登录与单点授权。

## 功能特性

- **三种接入协议** - 彩虹聚合登录兼容（`/connect.php`）、自研 REST 接口（MD5 签名）、标准 OAuth2/OIDC 授权服务器（authorization code + PKCE + RS256 id_token + Discovery/JWKS + revoke/introspect）
- **20 个登录渠道** - QQ、微信、支付宝、微博、百度、抖音、钉钉、企业微信、飞书、Gitee、GitHub、Google、Microsoft、Apple、Discord、Facebook、LinkedIn 等；另含「通用 OAuth2/OIDC」渠道（Discovery / 手动端点 / claims 映射），可接入任意外部身份源
- **平台账号授权（IDP / CAS）** - OAuth2 授权页支持使用平台账号（密码或 Passkey）直接授权，实现单点登录
- **Passkey / WebAuthn** - 无密码登录与通行密钥管理（登录页直登、用户中心注册/删除）
- **应用管理** - AppID/AppKey 自动生成；rainbow / rest / oauth2 / compat 四种模式；渠道白名单、回调域名白名单（区分子域名）、OAuth2 精确回调白名单、refresh_token 签发开关；普通用户应用配额
- **用户系统** - 注册（可邮箱验证）、用户名/邮箱/手机号登录、验证码找回密码、管理员/普通用户角色、用户中心（资料/改密/第三方绑定）
- **登录记录** - 分页查询、批量删除、CSV 导出（防公式注入），普通用户仅可见自己应用的记录
- **系统设置** - SMTP 邮件与阿里云/腾讯云/短信宝验证码、邮件模板、SOCKS5 代理（境外渠道）、头像源（QQ / Gravatar 镜像）、登录页背景等

## 技术栈

**后端**：Go 1.26 + Gin、GORM + SQLite（单文件零依赖部署）、go-webauthn、JWT（控制台 HS256 / OIDC id_token RS256）

**前端**：React 18 + TypeScript + Vite、Shadcn UI + Tailwind CSS、Zustand、React Router

## 快速开始

### Docker 部署（推荐）

```bash
docker compose up -d   # 使用官方镜像 xiaoman1221/oauthgo:latest
```

访问 http://localhost:8080，数据持久化在宿主 `./data` 目录（挂载到容器 `/app/data`）。

建议部署前设置 `HOST`（站点对外地址，决定第三方回调与 OIDC issuer）与 `JWT_KEY`
（控制台登录密钥；**留空时首次启动自动生成随机密钥并持久化到数据库**，跨实例共享登录态才需显式配置）：

```bash
JWT_KEY=your-strong-random-secret HOST=https://oauth.example.com docker compose up -d
```

本地构建镜像（可选）：

```bash
docker build -t xiaoman1221/oauthgo:latest .
docker compose up -d
```

<details>
<summary>中国大陆网络环境构建</summary>

Dockerfile 默认使用大陆可直连源（`goproxy.cn`、`npmmirror`），无需额外配置。海外构建切回官方源：

```bash
docker build \
  --build-arg GOPROXY=https://proxy.golang.org,direct \
  --build-arg GOSUMDB=sum.golang.org \
  --build-arg NPM_REGISTRY=https://registry.npmjs.org \
  -t xiaoman1221/oauthgo:latest .
```

无法拉取 Docker Hub 基础镜像时，用 `--build-arg BASE_NODE_IMAGE=...` / `BASE_GO_IMAGE=...` / `BASE_RUNTIME_IMAGE=...` 覆盖为阿里云镜像仓库地址。

</details>

### 手动部署

环境要求：Go 1.26+、Node.js 18+、npm。

```bash
cp .env.example .env   # 按需修改配置
make                   # 一键构建前端 + 后端（等价于 bash build.sh），产物 bin/oauthgo
./bin/oauthgo
```

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PORT` | 服务端口 | `8080` |
| `HOST` | 站点对外地址（第三方回调、OIDC issuer、Passkey RP 均由它派生） | `http://localhost:8080` |
| `JWT_KEY` | 控制台 JWT 密钥；**留空则首次启动自动生成随机密钥并持久化到数据库**，跨实例共享登录态时需显式配置 | 自动生成 |
| `GIN_MODE` | Gin 模式（debug / release） | `debug` |
| `DB_PATH` | SQLite 数据库文件路径 | `data.db` |
| `TRUSTED_PROXIES` | 可信代理 CIDR（逗号分隔）；反向代理部署时配置，默认不信任任何代理（防伪造 IP） | 空 |

### 默认账号

首次运行自动创建管理员：`admin` / `123456`，**请登录后立即修改密码**。

## API 概览

| 模块 | 路径 | 说明 |
|------|------|------|
| 彩虹协议 | `/connect.php`（`/api/connect.php` 别名） | `act=login/callback/query`，GET/POST，兼容彩虹官方客户端 |
| REST 接口 | `/api/v1/oauth` | `login` / `userinfo` / `query`，MD5 签名鉴权 |
| OAuth2/OIDC | `/authorize` `/token` `/userinfo` `/jwks` `/revoke` `/introspect` `/.well-known/openid-configuration` | 标准授权服务器，`/api/oauth2/*` 为等价别名 |
| 第三方登录 | `/api/oauth` | 渠道发起/回调、主站登录、公开渠道列表 |
| 认证 | `/api/auth` | 注册、登录、验证码、找回密码、用户中心、第三方绑定、一次性登录码兑换 |
| Passkey | `/api/auth/passkey` | WebAuthn 注册 / 登录 / 管理 |
| 应用管理 | `/api/apps` | 应用 CRUD（需登录） |
| 登录记录 | `/api/logins` | 查询 / 删除 / 批量删除 / CSV 导出（需登录） |
| 用户管理 | `/api/users` | 仅管理员 |
| 渠道配置 | `/api/providers` | 渠道凭据配置与测试，仅管理员 |
| 系统设置 | `/api/settings` | 读取（需登录）/ 写入（仅管理员） |
| 接口文档 | `/docs` | 接入文档、`openapi.yaml`、Swagger UI（`/docs/swagger`） |

## 目标站点接入

### 彩虹协议（兼容）

1. 平台创建应用，获取 `appid` / `appkey`，配置支持的登录类型与回调域名白名单
2. 跳转登录（GET/POST 均可，参数可放查询串或表单）：

```text
GET /connect.php?act=login&appid={appid}&appkey={appkey}&type=gitee&redirect_uri={redirect_uri}
```

返回 `{code:0, msg:"succ", url, ...}`，引导用户访问 `url` 完成授权。

3. 授权后平台回跳 `redirect_uri?type={type}&code={code}&sign={sign}`（`sign` 覆盖 `type`+`code`，可服务端二次校验），用 `code` 换取用户信息：

```text
GET /connect.php?act=callback&appid={appid}&appkey={appkey}&type=gitee&code={code}
```

返回 `{code:0, social_uid, access_token, nickname, faceimg, gender, location, ip}`。`code` 一次性有效；也可用 `act=query&social_uid={social_uid}` 随时查询最近登录。

### REST 接口签名规则

除 `sign` 外的参数按 key 升序拼接为 `k1=v1&k2=v2...`，末尾追加 `&key={appkey}`，整体取 MD5 作为 `sign`。`userinfo` / `query` 仅凭签名鉴权，`login` 需携带 `appid` + `appkey`。

### OAuth2 / OIDC（授权服务器）

`appid` 即 `client_id`、`appkey` 即 `client_secret`；应用模式需为 `oauth2` / `compat`，并配置「OAuth2/OIDC 回调地址」白名单。任意 OIDC/OAuth2 客户端（oidc-client、Keycloak 等）可按 Discovery 自动接入：

```text
授权   GET  {HOST}/authorize?response_type=code&client_id={appid}&redirect_uri=...&scope=openid%20profile&state=...
回调   {redirect_uri}?code=...&state=...
令牌   POST {HOST}/token    (grant_type=authorization_code / refresh_token)
用户   GET  {HOST}/userinfo (Authorization: Bearer <access_token>)
发现   GET  {HOST}/.well-known/openid-configuration
公钥   GET  {HOST}/jwks     (id_token RS256 验签, kid=oauthgo-rsa-1)
```

- 回调白名单按 `scheme://host/path` 匹配、忽略 query，可接入在回跳地址上追加动态参数（如 `oauth_state`）的 SSO 平台；`client_id` / `redirect_uri` 兼容 `appid`、`redirect_url` 等非标准命名，POST `/authorize` 表单体同样有效
- 支持 PKCE（S256/plain）、refresh_token 轮换、nonce 透传写入 id_token、一次性授权码
- scope 含 `openid` 时签发 RS256 id_token；`profile` / `email` / `phone` scope 控制用户信息字段
- `/authorize` 未指定 `type` 时返回授权页：可选择第三方渠道，也可使用平台账号（密码或 Passkey）登录后直接授权
- 应用管理可配置「接入示例参数」（`key=value` 形式，如接入方自带的 `state` / `app_id` / `oauth_state`），仅用于生成应用接入文档里的示例地址，不参与任何校验；不配置则用默认示例值

### 通用 OAuth2/OIDC 登录渠道（作为客户端）

在「登录渠道」中启用 `oauth2` 渠道，填入外部身份源的 Discovery URL（或手动端点）+ Client ID/Secret + scope，即可使用外部 OIDC 身份源登录，并可作为登录类型透传给目标站点。

## 项目结构

```
OauthGo/
├── main.go
├── config/        # 环境变量配置
├── router/        # 路由定义
├── handlers/      # 请求处理器（含测试）
├── services/      # 业务逻辑
├── providers/     # 第三方登录渠道适配器（20 个）
├── models/        # 数据模型
├── middleware/    # JWT 认证等中间件
├── database/      # 数据库初始化与迁移
├── utils/         # 工具函数
├── docs/          # 内嵌接口文档（openapi.yaml / Swagger UI）
└── web/           # React 前端（Vite，构建产物 dist/ 由后端托管）
```

## License

MIT
