# feedsystem_video_go 精确到文件的复现指南

> 本指南详细描述如何将 `feedsystem_video_go` 项目在本地或容器环境中完整复现，精确到每个关键文件的路径与作用。

---

## 一、项目结构总览（精确到文件）

```
feedsystem_video_go/
├── backend/                          # Go 后端工程
│   ├── cmd/
│   │   ├── main.go                   # API 服务入口：加载配置 -> 连接 DB/Redis/MQ -> 注册路由 -> 启动 HTTP 8080
│   │   └── worker/
│   │       └── main.go               # Worker 进程入口：连接 DB/Redis/MQ -> 声明拓扑 -> 启动 4 类消费者
│   ├── configs/
│   │   ├── config.yaml               # 本机原生环境配置（MySQL localhost:3306）
│   │   ├── config.compose-local.yaml # 本地开发对接 Compose 依赖（MySQL localhost:3307）
│   │   └── config.docker.yaml        # 容器内运行配置（host 为服务名：mysql/redis/rabbitmq）
│   ├── internal/
│   │   ├── account/
│   │   │   ├── entity.go             # Account 模型（GORM struct）与请求/响应 DTO
│   │   │   ├── handler.go            # HTTP Handler：注册/登录/改名/改密/登出/查询
│   │   │   ├── repo.go               # 数据访问层：Account CRUD
│   │   │   └── service.go            # 业务逻辑层：bcrypt 哈希、JWT 签发、Token 缓存/失效
│   │   ├── auth/
│   │   │   └── jwt.go                # JWT 工具：Token 生成与解析（依赖 golang-jwt/jwt/v5）
│   │   ├── config/
│   │   │   └── loadconfig.go         # YAML 配置加载 + 文件缺失时回退默认本地配置
│   │   ├── db/
│   │   │   └── db.go                 # GORM 连接 MySQL + AutoMigrate 五张核心表
│   │   ├── feed/
│   │   │   ├── entity.go             # Feed 响应结构体与请求 DTO
│   │   │   ├── handler.go            # Feed Handler：最新流/点赞排序/热榜/关注流
│   │   │   ├── repo.go               # Feed 数据层：复合游标分页、ZSET 热榜查询
│   │   │   └── service.go            # Feed 业务层：缓存策略、快照分页、降级逻辑
│   │   ├── http/
│   │   │   └── router.go             # 唯一依赖注入中心：初始化所有 Repo/Service/Handler/MQ/Middleware 并注册路由
│   │   ├── middleware/
│   │   │   ├── jwt/
│   │   │   │   └── jwt.go            # JWTAuth（强制鉴权）+ SoftJWTAuth（软鉴权，允许匿名）
│   │   │   ├── rabbitmq/
│   │   │   │   ├── rabbitMQ.go       # RabbitMQ 连接封装
│   │   │   │   ├── likeMQ.go         # 点赞事件发布端
│   │   │   │   ├── commentMQ.go      # 评论事件发布端
│   │   │   │   ├── socialMQ.go       # 关注事件发布端
│   │   │   │   ├── popularityMQ.go   # 热度更新事件发布端
│   │   │   │   └── timelineMQ.go     # 时间线/发件箱事件发布端
│   │   │   ├── ratelimit/
│   │   │   │   └── ratelimit.go      # 基于 Redis 的滑动窗口限流中间件
│   │   │   └── redis/
│   │   │       ├── redis.go          # Redis 客户端初始化
│   │   │       ├── cache.go          # 通用缓存读写工具
│   │   │       ├── zset.go           # ZSET 热榜操作（滑动窗口、快照合并）
│   │   │       └── redis_test.go     # miniredis 单元测试（限流计数器 TTL）
│   │   ├── observability/
│   │   │   ├── pprof.go              # pprof 调试服务器
│   │   │   └── pprof_test.go         # pprof 可用性测试
│   │   ├── social/
│   │   │   ├── entity.go             # Social 模型与 DTO
│   │   │   ├── handler.go            # 关注/取关/粉丝列表/关注列表 Handler
│   │   │   ├── repo.go               # Social 数据层
│   │   │   └── service.go            # Social 业务层（MQ 发布 + 降级直写）
│   │   ├── video/
│   │   │   ├── video_entity.go       # Video 模型与 DTO
│   │   │   ├── video_handler.go      # 视频发布/列表/详情/上传 Handler
│   │   │   ├── video_repo.go         # Video 数据层
│   │   │   ├── video_service.go      # 视频业务层（三级缓存：L1 本地 3s -> L2 Redis 5m -> L3 MySQL）
│   │   │   ├── popularity_cache.go   # 热度本地缓存
│   │   │   ├── like_entity.go        # Like 模型
│   │   │   ├── like_handler.go       # 点赞/取消点赞/是否点赞/我点赞的视频列表 Handler
│   │   │   ├── like_repo.go          # Like 数据层
│   │   │   ├── like_service.go       # 点赞业务层（MQ 发布 + 降级）
│   │   │   ├── comment_entity.go     # Comment 模型
│   │   │   ├── comment_handler.go    # 评论发布/删除/列表 Handler
│   │   │   ├── comment_repo.go       # Comment 数据层
│   │   │   └── comment_service.go    # 评论业务层（MQ 发布 + 降级）
│   │   └── worker/
│   │       ├── likeworker.go         # Like 事件消费者：异步写 Like 记录、更新计数
│   │       ├── commentworker.go      # Comment 事件消费者：异步写评论、更新计数
│   │       ├── socialworker.go       # Social 事件消费者：异步写关注关系
│   │       ├── popularityworker.go   # Popularity 事件消费者：更新 Redis ZSET 热榜
│   │       └── outboxworker.go       # 发件箱轮询与消费者：时间线更新
│   ├── go.mod                        # Go 模块定义（Go 1.24.5，依赖 Gin/GORM/JWT/RabbitMQ/Redis 等）
│   ├── go.sum                        # Go 依赖校验和
│   ├── Dockerfile                    # 多阶段构建：build -> base -> api / worker 两个 target
│   └── .dockerignore                 # Docker 构建忽略规则
├── frontend/                         # Vue 3 前端工程
│   ├── src/
│   │   ├── api/
│   │   │   ├── client.ts             # 通用 HTTP 请求层（fetch 封装、错误处理）
│   │   │   ├── account.ts            # 账号模块 API 调用
│   │   │   ├── video.ts              # 视频模块 API 调用
│   │   │   ├── like.ts               # 点赞模块 API 调用
│   │   │   ├── comment.ts            # 评论模块 API 调用
│   │   │   ├── social.ts             # 关注模块 API 调用
│   │   │   ├── feed.ts               # Feed 流 API 调用
│   │   │   ├── types.ts              # API 类型定义
│   │   │   └── normalize.ts          # 数据规范化工具
│   │   ├── components/
│   │   │   ├── AppShell.vue          # 应用外壳布局
│   │   │   ├── FeedVideoCard.vue     # Feed 视频卡片组件
│   │   │   ├── HelloWorld.vue        # 示例组件
│   │   │   ├── JsonBox.vue           # JSON 展示组件
│   │   │   ├── Toaster.vue           # 消息提示组件
│   │   │   └── UserAvatar.vue        # 用户头像组件
│   │   ├── router/
│   │   │   └── index.ts              # Vue Router 配置（history 模式）
│   │   ├── stores/
│   │   │   ├── auth.ts               # Pinia 认证状态管理
│   │   │   ├── social.ts             # Pinia 关注状态管理
│   │   │   └── toast.ts              # Pinia 消息提示状态管理
│   │   ├── views/
│   │   │   ├── HomeView.vue          # 首页
│   │   │   ├── FeedView.vue          # Feed 流页面
│   │   │   ├── HotView.vue           # 热榜页面
│   │   │   ├── VideoView.vue         # 视频发布页面
│   │   │   ├── VideoDetailView.vue   # 视频详情页面
│   │   │   ├── AccountView.vue       # 账号/登录页面
│   │   │   ├── RegisterView.vue      # 注册页面
│   │   │   ├── UserProfileView.vue   # 用户资料页面
│   │   │   ├── SettingsView.vue      # 设置页面
│   │   │   └── ChangePasswordView.vue # 修改密码页面
│   │   ├── main.ts                   # 前端入口：创建 Vue 应用、挂载 Pinia/Router
│   │   ├── style.css                 # 全局样式
│   │   └── utils/
│   │       └── jwt.ts                # 前端 JWT 解析工具
│   ├── index.html                    # HTML 入口模板
│   ├── vite.config.ts                # Vite 配置（含 /api 代理到 127.0.0.1:8080）
│   ├── tsconfig.json                 # TypeScript 根配置
│   ├── tsconfig.app.json             # App 层 TS 配置
│   ├── tsconfig.node.json            # Node 层 TS 配置
│   ├── package.json                  # npm 依赖（Vue 3.5 + Pinia + Vue Router + Vite）
│   ├── package-lock.json             # npm 锁文件
│   ├── nginx.conf                    # Nginx 配置（SPA fallback + /api 反向代理 + /static 转发）
│   ├── Dockerfile                    # 前端多阶段构建：node build -> nginx serve
│   └── .dockerignore / .gitignore    # 忽略规则
├── docker-compose.yml                # 全量编排：mysql / redis / rabbitmq / backend / worker / frontend
├── start.sh                          # 本地开发一键启动脚本（支持环境变量控制各进程）
├── test/
│   └── postman.json                  # Postman 接口集合（预置变量与批量接口）
├── picture/                          # 架构图与表设计图（11 张 PNG）
├── README.md                         # 项目简介与快速启动
├── feedsystem_video_go项目设计.md    # 详细设计文档（模块设计、表结构、流程图、接口清单）
└── AGENTS.md                         # AI Agent 开发指南
```

---

## 二、环境准备

### 2.1 必备软件

| 软件             | 版本要求          | 用途     | 验证命令                     |
| -------------- | ------------- | ------ | ------------------------ |
| Go             | >= 1.24.5     | 编译后端   | `go version`             |
| Node.js        | >= 20（推荐 22+） | 前端构建   | `node -v`                |
| npm            | >= 10         | 前端依赖管理 | `npm -v`                 |
| Docker         | >= 24.0       | 容器化部署  | `docker --version`       |
| Docker Compose | >= 2.20       | 依赖编排   | `docker compose version` |

> 注：若仅使用 **方式一（Docker Compose 全量启动）**，则只需要 Docker + Docker Compose，无需安装 Go 和 Node.js。

### 2.2 端口占用检查

复现前请确保以下端口未被占用：

| 端口    | 服务                   | 说明             |
| ----- | -------------------- | -------------- |
| 3307  | MySQL（Compose 映射）    | 宿主机访问数据库的端口    |
| 6379  | Redis                | 缓存服务           |
| 5672  | RabbitMQ AMQP        | 消息队列协议端口       |
| 15672 | RabbitMQ Management  | 管理台 Web 端口     |
| 8080  | Backend API          | HTTP API 服务    |
| 5173  | Frontend（Compose 映射） | 前端页面访问端口       |
| 6060  | pprof（API）           | Go 性能分析（仅本地开发） |
| 6061  | pprof（Worker）        | Go 性能分析（仅本地开发） |

检查命令示例：

```bash
# Linux/macOS
sudo lsof -i :8080
sudo lsof -i :3307

# 或
sudo netstat -tlnp | grep -E '8080|3307|6379|5173'
```

---

## 三、方式一：Docker Compose 一键启动（推荐，最简单）

此方式通过 `docker-compose.yml` 一键启动全部 6 个服务：MySQL、Redis、RabbitMQ、Backend API、Worker、Frontend。

### 3.1 确认关键文件存在

```bash
# 在项目根目录执行
ls -la docker-compose.yml
ls -la backend/Dockerfile
ls -la frontend/Dockerfile
ls -la backend/configs/config.docker.yaml
```

### 3.2 执行启动

```bash
# 在项目根目录
docker compose up -d --build
```

**构建过程说明（精确到文件）：**

1. **MySQL 服务**（`docker-compose.yml` 第 4-23 行）
   
   - 镜像：`mysql:8.0`
   - 环境变量：`MYSQL_ROOT_PASSWORD=123456`，`MYSQL_DATABASE=feedsystem`
   - 端口映射：`3307:3306`
   - 数据卷：`mysql_data`（Docker Volume，持久化到 `/var/lib/mysql`）
   - 字符集：`utf8mb4`（`--character-set-server=utf8mb4`）

2. **Redis 服务**（`docker-compose.yml` 第 25-37 行）
   
   - 镜像：`redis:7-alpine`
   - 密码：`123456`（`--requirepass 123456`）
   - 端口映射：`6379:6379`
   - 数据卷：`redis_data`

3. **RabbitMQ 服务**（`docker-compose.yml` 第 39-54 行）
   
   - 镜像：`rabbitmq:3-management`
   - 账号：`admin` / `password123`
   - 端口映射：`5672:5672`（AMQP）、`15672:15672`（Management UI）

4. **Backend 服务**（`docker-compose.yml` 第 56-73 行）
   
   - 构建上下文：项目根目录（`.`）
   - Dockerfile：`backend/Dockerfile`
   - Target：`api`（多阶段构建的 api 阶段）
   - 端口映射：`8080:8080`
   - **关键挂载**：`backend/configs/config.docker.yaml:/app/configs/config.yaml:ro`
     - 容器内运行时使用 `config.docker.yaml`，其中 host 为服务名（`mysql`、`redis`、`rabbitmq`）
   - 数据卷：`backend_uploads:/app/.run/uploads`（视频/封面上传文件持久化）
   - 健康检查依赖：等待 mysql、redis、rabbitmq 均 healthy 后才启动

5. **Worker 服务**（`docker-compose.yml` 第 75-89 行）
   
   - 构建上下文：项目根目录（`.`）
   - Dockerfile：`backend/Dockerfile`
   - Target：`worker`（多阶段构建的 worker 阶段）
   - **关键挂载**：同样挂载 `config.docker.yaml`
   - 无端口暴露（纯后台消费进程）

6. **Frontend 服务**（`docker-compose.yml` 第 91-99 行）
   
   - 构建上下文：项目根目录（`.`）
   - Dockerfile：`frontend/Dockerfile`
   - 端口映射：`5173:80`
   - 依赖：等待 backend 启动

### 3.3 多阶段构建详解（`backend/Dockerfile`）

```dockerfile
# Stage 1: build
FROM golang:1.24.5 AS build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./    # 先拷贝依赖文件，利用缓存层
RUN go mod download
COPY backend/ ./
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd
RUN go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# Stage 2: base
FROM alpine:3.21 AS base
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -s /sbin/nologin app
WORKDIR /app
COPY --from=build /src/backend/configs ./configs
RUN mkdir -p ./.run/uploads && chown -R app:app /app
USER app

# Stage 3a: api
FROM base AS api
COPY --from=build /out/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]

# Stage 3b: worker
FROM base AS worker
COPY --from=build /out/worker /app/worker
ENTRYPOINT ["/app/worker"]
```

### 3.4 前端构建详解（`frontend/Dockerfile`）

```dockerfile
# Stage 1: build
FROM node:24-alpine AS build
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build          # 输出到 /src/dist

# Stage 2: serve
FROM nginx:1.27-alpine
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### 3.5 Nginx 配置详解（`frontend/nginx.conf`）

```nginx
server {
  listen 80;
  server_name _;
  client_max_body_size 300m;           # 允许大文件上传（视频）
  root /usr/share/nginx/html;
  index index.html;

  location / {
    try_files $uri $uri/ /index.html;  # SPA fallback（Vue Router history 模式）
  }

  location /api/ {
    proxy_pass http://backend:8080/;   # 反向代理到 backend 服务（Docker 网络内 DNS）
    proxy_http_version 1.1;
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }

  location /static/ {
    proxy_pass http://backend:8080/static/;  # 转发静态文件请求到后端
    proxy_buffering off;
  }
}
```

### 3.6 验证启动成功

```bash
# 查看所有服务状态
docker compose ps

# 预期输出：mysql / redis / rabbitmq / backend / worker / frontend 均为 Up

# 查看日志
docker compose logs -f backend
docker compose logs -f worker
```

### 3.7 访问端点

| 端点           | URL                    | 说明                         |
| ------------ | ---------------------- | -------------------------- |
| 前端页面         | http://localhost:5173  | Vue SPA                    |
| 后端 API       | http://localhost:8080  | Gin HTTP 服务                |
| RabbitMQ 管理台 | http://localhost:15672 | 账号 `admin` / `password123` |

### 3.8 停止服务

```bash
docker compose down              # 停止并移除容器
docker compose down -v           # 停止并移除容器 + 数据卷（会清空数据库）
```

---

## 四、方式二：本地非容器化开发（适合调试后端/前端代码）

此方式将依赖（MySQL、Redis、RabbitMQ）通过 Docker Compose 拉起，但后端 API、Worker、前端以本地进程运行，便于断点调试。

### 4.1 启动基础设施依赖

```bash
# 在项目根目录，仅启动 mysql + redis + rabbitmq
docker compose up -d mysql redis rabbitmq
```

**端口映射结果：**

- MySQL：`localhost:3307`（容器内 3306 映射到宿主机 3307）
- Redis：`localhost:6379`
- RabbitMQ AMQP：`localhost:5672`
- RabbitMQ Management：`localhost:15672`

### 4.2 确认后端配置文件

本地开发使用 **`backend/configs/config.compose-local.yaml`**：

```yaml
server:
  port: 8080

database:
  host: localhost
  port: 3307          # 对应 Compose 映射的宿主机端口
  user: root
  password: 123456
  dbname: feedsystem

redis:
  host: localhost
  port: 6379
  password: 123456
  db: 0

rabbitmq:
  host: localhost
  port: 5672
  username: admin
  password: password123

observability:
  pprof:
    enabled: true
    api_addr: localhost:6060
    worker_addr: localhost:6061
```

> 对比文件：
> 
> - `config.yaml`：MySQL 端口为 `3306`（本机原生安装）
> - `config.docker.yaml`：host 为服务名（`mysql`、`redis`、`rabbitmq`），pprof 禁用

### 4.3 启动后端 API

```bash
cd backend
CONFIG_PATH=configs/config.compose-local.yaml go run ./cmd
```

**启动流程（精确到文件）：**

1. **`cmd/main.go`** 执行：
   
   - 读取环境变量 `CONFIG_PATH`（若未设置则默认 `configs/config.yaml`）
   - 调用 `internal/config/loadconfig.go` 的 `LoadLocalDev()` 加载 YAML
   - 调用 `internal/db/db.go` 的 `NewDB()` 连接 MySQL
   - 调用 `db.AutoMigrate()` 自动迁移表结构（`Account`、`Video`、`Like`、`Comment`、`Social`、`OutboxMsg`）
   - 调用 `internal/middleware/redis/redis.go` 的 `NewFromEnv()` 连接 Redis（可选，失败则降级）
   - 调用 `internal/middleware/rabbitmq/rabbitMQ.go` 的 `NewRabbitMQ()` 连接 MQ（可选，失败则降级）
   - 调用 `internal/observability/pprof.go` 启动 pprof 调试服务器（`localhost:6060`）
   - 调用 `internal/http/router.go` 的 `SetRouter()` 注册所有路由
   - 启动 HTTP 服务监听 `:8080`

2. **路由注册详情（`internal/http/router.go`）：**
   
   - `/account/register` — `account/handler.go:CreateAccount`
   - `/account/login` — `account/handler.go:Login`
   - `/account/changePassword` — `account/handler.go:ChangePassword`
   - `/account/findByID` — `account/handler.go:FindByID`
   - `/account/findByUsername` — `account/handler.go:FindByUsername`
   - `/account/logout` — JWTAuth 保护 — `account/handler.go:Logout`
   - `/account/rename` — JWTAuth 保护 — `account/handler.go:Rename`
   - `/video/listByAuthorID` — `video/video_handler.go:ListByAuthorID`
   - `/video/getDetail` — `video/video_handler.go:GetDetail`
   - `/video/uploadVideo` — JWTAuth — `video/video_handler.go:UploadVideo`
   - `/video/uploadCover` — JWTAuth — `video/video_handler.go:UploadCover`
   - `/video/publish` — JWTAuth — `video/video_handler.go:PublishVideo`
   - `/like/like` / `/like/unlike` / `/like/isLiked` / `/like/listMyLikedVideos` — `video/like_handler.go`
   - `/comment/listAll` — `video/comment_handler.go:GetAllComments`
   - `/comment/publish` / `/comment/delete` — JWTAuth — `video/comment_handler.go`
   - `/social/follow` / `/social/unfollow` / `/social/getAllFollowers` / `/social/getAllVloggers` — JWTAuth — `social/handler.go`
   - `/feed/listLatest` / `/feed/listLikesCount` / `/feed/listByPopularity` — SoftJWTAuth — `feed/handler.go`
   - `/feed/listByFollowing` — JWTAuth — `feed/handler.go`
   - 静态文件：`/static` — `gin.Static("/static", "./.run/uploads")`

### 4.4 启动 Worker

> **重要：Worker 是异步落库/更新热榜的必要进程。若只启动 API 而不启动 Worker，点赞/评论/关注等操作的事件将堆积在 MQ 中，无法异步消费。**

```bash
cd backend
CONFIG_PATH=configs/config.compose-local.yaml go run ./cmd/worker
```

**启动流程（精确到文件 `cmd/worker/main.go`）：**

1. 加载配置（同 API 进程）
2. 连接 MySQL（`internal/db/db.go`）
3. 连接 Redis（`internal/middleware/redis/redis.go`）— 用于 PopularityWorker
4. 连接 RabbitMQ（直接使用 `amqp091-go`）
5. 声明 Exchange + Queue + Binding：
   - `social.events` / `social.events` / `social.*`
   - `like.events` / `like.events` / `like.*`
   - `comment.events` / `comment.events` / `comment.*`
   - `video.popularity.events` / `video.popularity.events` / `video.popularity.*`
6. 初始化 4 类 Worker：
   - `internal/worker/socialworker.go` — `SocialWorker`
   - `internal/worker/likeworker.go` — `LikeWorker`
   - `internal/worker/commentworker.go` — `CommentWorker`
   - `internal/worker/popularityworker.go` — `PopularityWorker`（仅 Redis 可用时）
7. 启动 pprof（`localhost:6061`）
8. 并发消费各队列

### 4.5 启动前端开发服务器

```bash
cd frontend
npm install      # 首次执行，安装 node_modules
npm run dev      # 启动 Vite 开发服务器
```

**代理配置（`frontend/vite.config.ts`）：**

```typescript
export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',  // 代理到本地后端 API
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
```

> 前端默认地址：http://localhost:5173（Vite 默认端口，实际以终端输出为准）

---

## 五、方式三：start.sh 脚本一键启动

项目根目录的 **`start.sh`** 脚本支持通过环境变量控制启动内容，适合本地快速启动全链路。

### 5.1 脚本逻辑说明（精确到文件 `start.sh`）

```bash
#!/usr/bin/env bash
set -euo pipefail

# 环境变量控制开关（默认均为 1，表示启动）
START_REDIS=1          # 尝试启动本地 redis-server
START_RABBITMQ=1       # 通过 docker compose 拉起 RabbitMQ
START_BACKEND=1        # 启动后端 API
START_WORKER=1         # 启动 Worker
START_FRONTEND=1       # 启动前端

# 若同时启动 compose 依赖，自动使用 compose-local 配置
CONFIG_PATH=configs/config.compose-local.yaml

# 自动处理 Ctrl+C 信号，清理所有子进程
```

### 5.2 常用启动命令

```bash
# 启动全部（backend + worker + frontend + rabbitmq + redis）
./start.sh

# 仅启动后端 + Worker（不启动前端）
START_FRONTEND=0 ./start.sh

# 仅启动后端（不启动 Worker 和前端）
START_WORKER=0 START_FRONTEND=0 ./start.sh

# 不通过脚本启动 Redis（使用自己已有的 Redis）
START_REDIS=0 ./start.sh

# 不通过脚本启动 RabbitMQ（使用自己已有的 RabbitMQ）
START_RABBITMQ=0 ./start.sh

# 指定自定义配置文件
CONFIG_PATH=configs/config.yaml ./start.sh

# 前端使用 preview 模式（生产构建预览）
FRONTEND_SCRIPT=preview ./start.sh
```

### 5.3 脚本启动的进程关系

```
start.sh (父进程)
├── docker compose up -d rabbitmq   # 若 START_RABBITMQ=1
├── redis-server                     # 若 START_REDIS=1 且本地无 Redis
├── go run ./cmd (Backend API)       # 若 START_BACKEND=1
├── go run ./cmd/worker (Worker)     # 若 START_WORKER=1
└── npm run dev (Frontend)           # 若 START_FRONTEND=1
```

> 按 `Ctrl+C` 时，`start.sh` 的 `trap cleanup` 会优雅地终止所有子进程。

---

## 六、配置文件完整对比

| 配置项           | `configs/config.yaml` | `configs/config.compose-local.yaml` | `configs/config.docker.yaml` |
| ------------- | --------------------- | ----------------------------------- | ---------------------------- |
| 适用场景          | 本机原生 MySQL            | 本地开发对接 Compose                      | 容器内运行                        |
| MySQL host    | `localhost`           | `localhost`                         | `mysql`                      |
| MySQL port    | `3306`                | `3307`                              | `3306`                       |
| Redis host    | `localhost`           | `localhost`                         | `redis`                      |
| RabbitMQ host | `localhost`           | `localhost`                         | `rabbitmq`                   |
| pprof enabled | `true`                | `true`                              | `false`                      |
| pprof API     | `localhost:6060`      | `localhost:6060`                    | —                            |
| pprof Worker  | `localhost:6061`      | `localhost:6061`                    | —                            |

> `internal/config/loadconfig.go` 中的 `LoadLocalDev()` 在配置文件不存在时会自动回退到硬编码的默认本地配置（等效于 `config.yaml`）。

---

## 七、数据库初始化与表结构

### 7.1 自动迁移

项目启动时，**`internal/db/db.go`** 的 `AutoMigrate()` 会自动创建以下表：

```go
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &account.Account{},      // 用户表
        &video.Video{},          // 视频表
        &video.Like{},           // 点赞表
        &video.Comment{},        // 评论表
        &social.Social{},        // 关注表
        &video.OutboxMsg{},      // 发件箱表
    )
}
```

> 无需手动执行 SQL，首次启动 API 进程即自动建表。

### 7.2 手动连接数据库验证

```bash
# 使用 Compose 启动的 MySQL
mysql -h 127.0.0.1 -P 3307 -u root -p123456 -D feedsystem

# 查看表
SHOW TABLES;

# 预期输出：accounts, videos, likes, comments, socials, outbox_msgs
```

---

## 八、接口测试（Postman）

### 8.1 导入集合

1. 打开 Postman
2. 点击 **Import** -> **File** -> 选择 `test/postman.json`
3. 集合名称为 `feedsystem_video_go`

### 8.2 预置变量

集合已预置以下变量，通常无需修改即可直接调用：

- `baseUrl`：`http://localhost:8080`
- `token`：（登录后由测试脚本自动写入）

### 8.3 推荐测试顺序

1. `POST /account/register` — 注册账号
2. `POST /account/login` — 登录（成功后 `token` 变量自动更新）
3. `POST /account/findByUsername` — 查询账号信息
4. `POST /video/publish` — 发布视频
5. `POST /feed/listLatest` — 浏览最新 Feed
6. `POST /like/like` — 点赞
7. `POST /comment/publish` — 评论
8. `POST /social/follow` — 关注
9. `POST /feed/listByFollowing` — 查看关注流

---

## 九、测试与验证

### 9.1 后端单元测试

```bash
cd backend

# 运行全部测试
go test ./...

# 运行特定测试（带 Redis 的测试使用 miniredis，无需外部 Redis）
go test -v ./internal/middleware/redis/
go test -v ./internal/observability/
```

**测试文件清单：**

- `backend/internal/middleware/redis/redis_test.go` — miniredis 测试限流计数器 TTL
- `backend/internal/observability/pprof_test.go` — pprof 服务器基本可用性

### 9.2 手动 curl 验证 API

```bash
# 1. 注册
curl -X POST http://localhost:8080/account/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testpass123"}'

# 2. 登录
curl -X POST http://localhost:8080/account/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testpass123"}'

# 3. 获取最新 Feed（匿名访问，无需 token）
curl -X POST http://localhost:8080/feed/listLatest \
  -H "Content-Type: application/json" \
  -d '{"limit":10,"latest_time":0}'
```

---

## 十、常见问题排查

### 10.1 端口冲突

**现象**：`docker compose up` 报错 `bind: address already in use`

**排查**：

```bash
# 查找占用端口的进程
sudo lsof -i :8080
sudo lsof -i :3307
sudo lsof -i :6379
sudo lsof -i :5173

# 终止进程
kill -9 <PID>
```

### 10.2 MySQL 连接失败

**现象**：后端日志 `Failed to connect database`

**排查**：

```bash
# 1. 检查 MySQL 容器是否 healthy
docker compose ps

# 2. 检查端口映射是否正确
docker compose exec mysql mysql -uroot -p123456 -e "SHOW DATABASES;"

# 3. 检查配置文件端口是否对应
# 若使用 Compose 依赖，确认 CONFIG_PATH=configs/config.compose-local.yaml（端口 3307）
```

### 10.3 Redis 连接失败（降级不影响核心功能）

**现象**：日志 `Redis not available (cache disabled)`

**说明**：Redis 为可选依赖，连接失败时自动降级走 MySQL，不影响核心业务。但热榜和缓存功能会受限。

**排查**：

```bash
docker compose ps redis
docker compose logs redis
redis-cli -h 127.0.0.1 -p 6379 -a 123456 ping
```

### 10.4 RabbitMQ 连接失败（降级不影响核心功能）

**现象**：日志 `RabbitMQ config error (disabled)`

**说明**：RabbitMQ 为可选依赖，接口层发布失败时会降级为直写 MySQL/Redis。

**排查**：

```bash
docker compose ps rabbitmq
docker compose logs rabbitmq
# 管理台验证：http://localhost:15672
```

### 10.5 前端无法访问后端 API

**现象**：浏览器报错 `Failed to fetch` 或 CORS 错误

**排查**：

```bash
# 1. 确认后端是否在 8080 端口运行
curl http://localhost:8080/account/findByUsername \
  -H "Content-Type: application/json" -d '{"username":"test"}'

# 2. 检查 Vite 代理配置（frontend/vite.config.ts）
# 确保 target 为 'http://127.0.0.1:8080'

# 3. 若前端也在容器中，确保通过 Nginx 反向代理（frontend/nginx.conf）
# 检查 location /api/ 的 proxy_pass 是否正确指向 backend:8080
```

### 10.6 Worker 未启动导致数据不一致

**现象**：点赞/评论后页面刷新数据未更新

**原因**：API 进程仅将事件发布到 MQ，实际落库由 Worker 消费完成。若 Worker 未启动，事件堆积在队列中。

**解决**：

```bash
# Docker Compose 方式
docker compose ps worker
docker compose logs -f worker

# 本地方式
cd backend
CONFIG_PATH=configs/config.compose-local.yaml go run ./cmd/worker
```

### 10.7 上传文件过大

**现象**：上传视频/封面时报 `413 Payload Too Large`

**解决**：Nginx 已配置 `client_max_body_size 300m`（`frontend/nginx.conf` 第 6 行）。若使用纯本地后端调试，Gin 默认限制为 32MB，需检查前端是否直接请求后端（非经 Nginx）。

---

## 十一、生产部署注意事项

1. **JWT Secret**：`backend/internal/auth/jwt.go` 中的签名为硬编码示例（`my-secret-key`），生产环境必须通过环境变量注入。
2. **文件存储**：当前上传文件保存在 `backend/.run/uploads/`（本地磁盘），生产环境应替换为 OSS/S3/MinIO。
3. **数据库密码**：配置文件中密码为明文示例，生产应使用环境变量或密钥管理服务。
4. **pprof**：生产环境应禁用 pprof（`config.docker.yaml` 已设置 `enabled: false`）。
5. **Nginx / CORS**：生产由 Nginx 统一反向代理，避免直接暴露后端端口。

---

## 十二、快速参考命令卡

```bash
# === Docker Compose 全量启动 ===
docker compose up -d --build
docker compose down -v

# === 仅启动依赖 ===
docker compose up -d mysql redis rabbitmq

# === 本地后端 API ===
cd backend
CONFIG_PATH=configs/config.compose-local.yaml go run ./cmd

# === 本地 Worker ===
cd backend
CONFIG_PATH=configs/config.compose-local.yaml go run ./cmd/worker

# === 本地前端 ===
cd frontend
npm install
npm run dev

# === 脚本一键启动 ===
./start.sh
START_FRONTEND=0 ./start.sh

# === 测试 ===
cd backend && go test ./...

# === 查看日志 ===
docker compose logs -f backend
docker compose logs -f worker

# === 连接 MySQL ===
mysql -h 127.0.0.1 -P 3307 -u root -p123456 -D feedsystem

# === 连接 Redis ===
redis-cli -h 127.0.0.1 -p 6379 -a 123456
```

---

> 本指南基于项目文件实际内容编写，覆盖 `docker-compose.yml`、`backend/Dockerfile`、`frontend/Dockerfile`、三个配置文件、`cmd/main.go`、`cmd/worker/main.go`、`internal/http/router.go` 等全部关键文件。如项目代码有变更，请以实际文件为准。
