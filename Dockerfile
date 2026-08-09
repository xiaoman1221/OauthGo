# ===== 构建参数（中国大陆网络支持） =====
# 默认已使用中国大陆可直连的镜像源（goproxy.cn / npmmirror / sum.golang.google.cn），
# 海外环境或需要官方源时可通过 --build-arg 覆盖，例如：
#   docker build \
#     --build-arg GOPROXY=https://proxy.golang.org,direct \
#     --build-arg GOSUMDB=sum.golang.org \
#     --build-arg NPM_REGISTRY=https://registry.npmjs.org \
#     .
# 使用 docker compose 时，可通过同名环境变量覆盖（见 docker-compose.yml build.args）。
# 若 Docker Hub 基础镜像拉取缓慢/失败，可覆盖 BASE_*_IMAGE 为国内镜像仓库
# （如 registry.cn-hangzhou.aliyuncs.com/library/node:20-alpine），或为 Docker
# 守护进程配置 registry-mirror。
ARG BASE_NODE_IMAGE=node:20-alpine
ARG BASE_GO_IMAGE=golang:1.26-alpine
ARG BASE_RUNTIME_IMAGE=alpine:3.20
ARG NPM_REGISTRY=https://registry.npmmirror.com
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn

# ===== Stage 1: 构建前端 =====
FROM ${BASE_NODE_IMAGE} AS web-builder
WORKDIR /build
ARG NPM_REGISTRY
COPY web/package.json web/package-lock.json ./
RUN npm ci --registry="$NPM_REGISTRY"
COPY web/ ./
RUN npm run build

# ===== Stage 2: 编译后端 =====
FROM ${BASE_GO_IMAGE} AS go-builder
WORKDIR /build
ARG GOPROXY
ARG GOSUMDB
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY="$GOPROXY" \
    GOSUMDB="$GOSUMDB"
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 使用前端构建产物覆盖（.dockerignore 已排除本地 dist）
COPY --from=web-builder /build/dist ./web/dist
RUN go build -trimpath -ldflags="-s -w" -o oauthgo .

# ===== Stage 3: 运行 =====
FROM ${BASE_RUNTIME_IMAGE}
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /build/oauthgo .
COPY --from=go-builder /build/web/dist ./web/dist
ENV GIN_MODE=release \
    DB_PATH=/app/data/data.db
EXPOSE 8080
VOLUME /app/data
CMD ["/app/oauthgo"]
