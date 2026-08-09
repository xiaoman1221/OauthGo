# OauthGo

统一授权管理 - 使用Go语言编写的集成各类第三方登录于一体的综合平台,兼容彩虹聚合登录。

## 功能特性

- **第三方登录聚合** - 对标彩虹聚合登录，为其他站点提供第三方登录服务，兼容彩虹协议与自研 REST 接口
- **应用管理** - 目标站点注册应用，自动生成 AppID/AppKey，配置支持登录类型（QQ/微信/支付宝/微博/百度/抖音/钉钉/Gitee/企业微信等）与回调域名白名单（区分子域名）
- **登录管理** - 登录记录的增删改查、批量操作、CSV导入导出
- **到期通知** - 多渠道域名到期提醒（邮件、Webhook等）
- **用户系统** - 多用户支持、角色权限管理
- **用户中心** - 个人资料（昵称/头像/用户名/邮箱/手机号）修改、修改密码、绑定/解绑第三方登录
- **系统设置** - 系统参数配置

## 技术栈

**后端**
- Go 1.26 + Gin
- GORM + SQLite
- JWT 认证

**前端**
- Vue 3 + Vite
- Element Plus
- Pinia 状态管理
- Vue Router

## 快速开始

### Docker 部署（推荐）

```bash
docker compose up -d --build
```

访问 http://localhost:8080

> 数据默认保存在 Docker 命名卷 `oauthgo-data`（挂载到容器 `/app/data`）中。
> 建议在部署前设置环境变量 `JWT_KEY`（持久化随机密钥）与 `HOST`（站点对外地址，
> 用于第三方登录回调跳转），示例：

```bash
JWT_KEY=your-strong-random-secret HOST=https://oauth.example.com docker compose up -d
```

#### 中国大陆网络环境构建

Dockerfile 默认已使用大陆可直连的镜像源（`goproxy.cn`、`sum.golang.google.cn`、
`npmmirror`），无需额外配置。海外构建需切回官方源：

```bash
NPM_REGISTRY=https://registry.npmjs.org \
GOPROXY=https://proxy.golang.org,direct \
GOSUMDB=sum.golang.org \
docker compose build
```

若无法从 Docker Hub 拉取基础镜像，可覆盖基础镜像（阿里云镜像仓库）或为 Docker
守护进程配置 registry-mirror：

```bash
BASE_NODE_IMAGE=registry.cn-hangzhou.aliyuncs.com/library/node:20-alpine \
BASE_GO_IMAGE=registry.cn-hangzhou.aliyuncs.com/library/golang:1.26-alpine \
BASE_RUNTIME_IMAGE=registry.cn-hangzhou.aliyuncs.com/library/alpine:3.20 \
docker compose build
```

### 手动部署

**环境要求**

- Go 1.26+
- Node.js 18+
- npm

**配置**

复制 `.env.example` 为 `.env` 并修改配置：

```bash
cp .env.example .env
```

| 变量 | 说明 | 默认值  |
|------|------|---------|
| `PORT` | 服务端口 | 8080    |
| `GIN_MODE` | Gin模式（debug/release） | debug   |
| `DB_PATH` | 数据库文件路径 | data.db |
| `JWT_KEY` | JWT密钥 | -       |
**构建运行**

```bash
# 一键构建（前端+后端）
make
# 或
bash build.sh

# 运行
./domain-manager
```

### 默认账号

首次运行会自动创建管理员账号：

- 用户名：`admin`
- 密码：`123456`

> 请登录后立即修改默认密码。

## API 接口

| 模块 | 路径前缀 | 说明 |
|------|----------|------|
| 彩虹兼容 | `/connect.php` | 彩虹聚合登录协议（login/callback/query），兼容彩虹官方调用方式（根路径、GET/POST），`/api/connect.php` 为等价别名 |
| REST 接口 | `/api/v1/oauth` | 自研登录接口（login/userinfo/query），MD5 签名校验 |
| 登录渠道 | `/api/oauth` | 各第三方渠道登录/回调（应用会话跳转） |
| 应用管理 | `/api/apps` | 目标站点应用（自动生成凭证、模式、支持类型、回调域名白名单） |
| 登录记录 | `/api/logins` | 登录记录的增删改查、导入导出 |
| 认证 | `/api/auth` | 平台注册、登录、找回密码、用户中心（资料/密码/第三方绑定） |
| 渠道配置 | `/api/providers` | 第三方登录渠道凭据配置 |
| 通知 | `/api/notifications` | 通知渠道与日志管理 |
| 设置 | `/api/settings` | 系统设置、用户管理 |
| 接口文档 | `/docs` | 接入文档（彩虹协议 + REST 接口），`/docs/openapi.yaml` 为 OpenAPI 规范，`/docs/swagger` 为 Swagger UI 在线调试 |

### 目标站点接入（兼容彩虹协议）

1. 平台创建应用，获取 `appid`、`appkey`，并配置支持类型与回调域名白名单
2. 跳转登录（接口支持 GET/POST，参数可放查询串或表单）：

```text
GET /connect.php?act=login&appid={appid}&appkey={appkey}&type=gitee&redirect_uri={redirect_uri}
```

返回 `{code:0,msg:"succ",type,url,qrcode:""}`，将用户引导至 `url` 完成授权。

3. 授权成功后平台回调 `redirect_uri?state={state}&code={code}&sign={sign}`，目标站点用 `code` 换取用户信息：

```text
GET /connect.php?act=callback&appid={appid}&appkey={appkey}&type=gitee&code={code}
```

返回 `{code:0,msg,type,access_token,social_uid,faceimg,nickname,location,gender,ip}`。

`code` 一次性有效，也可通过 `act=query&social_uid={social_uid}` 随时查询。

### REST 接口签名规则

除 `sign` 外的参数按 key 升序拼接为 `k1=v1&k2=v2...`，末尾追加 `&key={appkey}`，整体取 MD5 作为 `sign`。`userinfo`/`query` 仅凭签名鉴权，`login` 需携带 `appid`+`appkey`。

## 项目结构

```
OauthGo/
├── config/          # 配置加载
├── database/        # 数据库初始化与迁移
├── docs/            # 接口文档资源（index.html / openapi.yaml / swagger.html）
├── handlers/        # 请求处理器
├── middleware/       # 中间件（JWT认证）
├── models/          # 数据模型
├── router/          # 路由定义
├── services/        # 业务逻辑
├── utils/           # 工具函数
├── web/             # Vue前端
│   ├── src/
│   └── dist/        # 前端构建产物
├── .dockerignore
├── .env.example     # 环境变量模板
├── Dockerfile       # 多阶段构建（前端 + 后端）
├── docker-compose.yml
├── go.mod
└── main.go
```

## License

MIT
