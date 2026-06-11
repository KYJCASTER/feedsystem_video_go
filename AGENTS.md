# feedsystem_video_go — Agent Guide

> 本文件面向 AI Coding Agent。项目主要注释与文档语言为 **中文**。Agent 在修改代码、编写注释、生成文档时，默认使用中文。

---

## 项目概述

`feedsystem_video_go` 是一款基于 **Go + Vue 3** 的短视频 Feed 流系统，包含账号、视频、点赞、评论、关注（Social）与 Feed 流等核心功能。

**核心特性：**
- 后端使用 **Gin** 提供 HTTP API，**GORM** 操作 MySQL。
- **Redis** 用作缓存与热榜（ZSET），**RabbitMQ** 用于异步事件驱动（点赞/评论/关注/热度更新）。
- API 进程（`cmd`）与 Worker 进程（`cmd/worker`）可独立部署，支持拆分伸缩。
- 前端为 **Vue 3 + Vite + TypeScript + Pinia** 单页应用，Nginx 做反向代理。
- 提供 Docker Compose 一键启动全链路，也支持 `./start.sh` 本地非容器化开发。

---

## 技术栈

| 维度 | 技术/组件 | 说明 |
|------|----------|------|
| 后端语言 | Go 1.24.5 | 模块路径 `feedsystem_video_go` |
| Web 框架 | Gin v1.11 | HTTP 路由、参数绑定、中间件链 |
| ORM | GORM v1.31 + mysql driver | 模型定义、CRUD、启动时 AutoMigrate |
| 数据库 | MySQL 8.0 | 核心五张表：`Account / Video / Like / Comment / Social` |
| 缓存 | Redis 7 | Token 缓存、Feed 缓存、视频详情缓存、热榜 ZSET |
| 消息队列 | RabbitMQ 3 (Management) | Topic Exchange 事件总线 |
| 鉴权 | JWT (golang-jwt/jwt/v5) + bcrypt | JWT 签发与校验，token 同时落库 + 缓存 |
| 前端 | Vue 3.5 + Vite + TypeScript 5.9 + Pinia + Vue Router | SPA，Vite 代理 `/api` 到后端 |
| 容器化 | Docker / Docker Compose | 多阶段构建，API / Worker 分离镜像 |
| 调试 | Postman Collection | `test/postman.json` 预置变量与批量接口 |

---

## 项目结构

```
.
├── backend/                     # Go 后端
│   ├── cmd/
│   │   ├── main.go              # API 服务入口
│   │   └── worker/main.go       # Worker 进程入口（消费 MQ）
│   ├── configs/
│   │   ├── config.yaml          # 本机原生环境（MySQL 3306）
│   │   ├── config.compose-local.yaml  # 本地开发（MySQL 3307，对接 compose 依赖）
│   │   └── config.docker.yaml   # 容器内运行（host 为服务名）
│   └── internal/
│       ├── account/             # 用户模块：entity, handler, repo, service
│       ├── video/               # 视频模块：包含 video/like/comment 子功能
│       ├── social/              # 关注模块
│       ├── feed/                # Feed 流模块
│       ├── auth/                # JWT Token 生成
│       ├── middleware/
│       │   ├── jwt/             # JWTAuth（强制鉴权）+ SoftJWTAuth（软鉴权，允许匿名）
│       │   ├── rabbitmq/        # MQ 发布端封装（Like/Comment/Social/Popularity/Timeline）
│       │   ├── redis/           # Redis 客户端、缓存工具、ZSET 操作、限流
│       │   └── ratelimit/       # 基于 Redis 的限流中间件
│       ├── worker/              # MQ 消费者：Like/Comment/Social/Popularity/Outbox
│       ├── http/router.go       # Gin 路由注册（依赖注入中心）
│       ├── config/loadconfig.go # YAML 配置加载（支持文件缺失时回退默认值）
│       ├── db/db.go             # GORM 连接 + AutoMigrate
│       └── observability/       # pprof 调试服务器
├── frontend/                    # Vue 3 前端
│   ├── src/
│   │   ├── api/                 # 按模块封装的 API 调用（client.ts 为通用请求层）
│   │   ├── stores/              # Pinia：auth / social / toast
│   │   ├── views/               # 页面视图
│   │   ├── components/          # 可复用组件
│   │   └── router/              # Vue Router 配置
│   ├── vite.config.ts           # Vite 配置（含 /api 代理到 localhost:8080）
│   ├── Dockerfile               # 多阶段构建：node build -> nginx serve
│   └── nginx.conf               # Nginx 配置（SPA fallback + /api 反向代理）
├── docker-compose.yml           # 全量编排：mysql / redis / rabbitmq / backend / worker / frontend
├── start.sh                     # 本地开发一键启动脚本（支持环境变量开关各进程）
├── test/postman.json            # Postman 接口集合
└── picture/                     # 架构图与表设计图
```

---

## 构建与运行

### Docker Compose 一键启动（推荐）

```bash
docker compose up -d --build
```

访问：
- 前端：`http://localhost:5173`
- 后端 API：`http://localhost:8080`
- RabbitMQ 管理台：`http://localhost:15672`（admin / password123）

Compose 会启动 `mysql:8.0`（映射宿主机 3307）、`redis:7-alpine`（6379）、`rabbitmq:3-management`（5672/15672）、`backend`（API，8080）、`worker`、`frontend`（80，映射宿主机 5173）。

### 本地非容器化开发

1. 启动依赖（MySQL / Redis / RabbitMQ）：
```bash
docker compose up -d mysql redis rabbitmq
```

2. 启动后端 API（使用 `config.compose-local.yaml`，对应 MySQL 宿主机端口 3307）：
```bash
cd backend
CONFIG_PATH=configs/config.compose-local.yaml go run ./cmd
```

3. 启动 Worker（消费 MQ，异步落库/更新热榜）：
```bash
cd backend
CONFIG_PATH=configs/config.compose-local.yaml go run ./cmd/worker
```

4. 启动前端开发服务器：
```bash
cd frontend
npm install
npm run dev
```

前端默认通过 Vite 代理 `/api` 到 `http://127.0.0.1:8080`。

### `./start.sh` 脚本启动

脚本支持通过环境变量控制启动内容：
- `START_BACKEND=1`（默认）启动 API
- `START_WORKER=1`（默认）启动 Worker
- `START_FRONTEND=1`（默认）启动前端
- `START_RABBITMQ=1`（默认）通过 docker compose 拉起 RabbitMQ
- `START_REDIS=1`（默认）尝试启动本地 redis-server
- `CONFIG_PATH` 指定后端配置，默认对接 compose-local 配置

示例仅启动后端 + Worker：
```bash
START_FRONTEND=0 ./start.sh
```

### 前端生产构建

```bash
cd frontend
npm run build
```
产物输出到 `frontend/dist`，由 Nginx 提供服务。

---

## 配置说明

后端配置为 YAML 文件，通过环境变量 `CONFIG_PATH` 指定路径。共有三份预设配置：

| 配置文件 | 适用场景 | 关键差异 |
|---------|---------|---------|
| `configs/config.yaml` | 本机原生 MySQL | MySQL host `localhost:3306`，pprof 启用 |
| `configs/config.compose-local.yaml` | 本地开发对接 compose 依赖 | MySQL host `localhost:3307`（compose 映射），pprof 启用 |
| `configs/config.docker.yaml` | 容器内运行 | MySQL host `mysql`，Redis host `redis`，RabbitMQ host `rabbitmq`，pprof 禁用 |

`config/loadconfig.go` 中的 `LoadLocalDev` 在配置文件不存在时会自动回退到硬编码的默认本地配置。

---

## 代码组织与模块划分

后端采用**按领域分层**的组织方式，每个业务模块一个目录，内部通常包含：
- `entity.go` — 数据模型（GORM struct + request/response struct）
- `repo.go` — 数据访问层（直接操作 `*gorm.DB`）
- `service.go` — 业务逻辑层（调用 Repo，处理缓存/MQ/降级）
- `handler.go` — HTTP Handler 层（Gin 路由处理函数）

**现有模块：**
- `account` — 注册、登录、改名、改密、登出、按 ID/用户名查询
- `video` — 视频发布、作者视频列表、视频详情（含本地缓存 + Redis + MySQL 三级缓存）
- `video/like_*` — 点赞/取消点赞、判断是否点赞、我点赞的视频列表
- `video/comment_*` — 评论发布/删除、评论列表
- `social` — 关注/取关、粉丝列表、关注列表
- `feed` — 最新视频流、点赞数排序流、热度榜、关注流

**公共基础设施：**
- `internal/http/router.go` 是**唯一的依赖注入中心**，负责初始化所有 Repository / Service / Handler / MQ / Middleware，并注册路由。
- `internal/middleware/jwt/jwt.go` 提供两种鉴权中间件：
  - `JWTAuth` — 强制要求合法 token，否则 401
  - `SoftJWTAuth` — 允许不带 token；带了 token 则必须合法，否则 401（用于 Feed 匿名浏览）
- `internal/middleware/ratelimit/` 基于 Redis 的滑动窗口限流，支持按 IP 或按账号限流。

---

## 测试策略

当前项目**测试覆盖较薄**，仅有 2 个测试文件：
- `backend/internal/middleware/redis/redis_test.go` — 使用 `miniredis` 测试限流计数器的 TTL 行为
- `backend/internal/observability/pprof_test.go` — pprof 服务器的基本可用性测试

**测试规范：**
- 后端测试文件命名遵循 Go 惯例 `*_test.go`，与被测代码同包。
- 需要 Redis 的单元测试优先使用 `github.com/alicebob/miniredis/v2` 内存版，避免依赖外部 Redis。
- 测试用 `t.Parallel()` 标注无状态测试以提高速度。

**接口测试：**
- `test/postman.json` 提供完整的 Postman Collection，包含变量预置与部分测试脚本。推荐在功能变更后导入 Postman 跑通全链路。

**建议补充方向：**
- Service 层的核心业务逻辑单元测试（使用 `sqlmock` 或 SQLite in-memory 隔离 DB）
- Handler 层的 HTTP 集成测试（使用 `httptest` + Gin 的 `ServeHTTP`）
- Worker 的消息消费逻辑测试

---

## 开发规范

### 语言与注释
- 代码注释、文档、Commit Message 默认使用**中文**。
- 保持与现有代码一致的注释风格。

### API 风格
- **所有接口统一使用 POST 方法**（包括查询类接口），请求体为 JSON。
- 统一返回格式：`{ "message": "..." }` 或 `{ "error": "..." }`，HTTP Status 200 表示业务成功，非 200 表示失败。
- 上传文件接口使用 `multipart/form-data`（`uploadVideo`、`uploadCover`）。
- 静态文件（视频/封面）通过后端 `/static` 路由暴露，文件存储在 `backend/.run/uploads/`。

### 鉴权与 Token
- 登录成功后后端将 JWT 写入 `account.token` 字段并同步缓存到 Redis（`account:<id>`，TTL 24h）。
- 改名/改密/登出会清空 `account.token` 并删除 Redis 缓存，使旧 JWT 立即失效。
- 鉴权中间件优先查 Redis，Redis 不可用则回退 MySQL 校验，校验通过后回填 Redis（**自愈机制**）。

### 缓存与降级
- Redis 是**可选依赖**。连接失败时自动降级走 MySQL，不会阻塞核心业务。
- 热点缓存（Feed 匿名流、视频详情）使用 Redis `SETNX` 分布式锁防止缓存击穿。
- 视频详情采用 **L1（本地缓存 3s）-> L2（Redis 5m）-> L3（MySQL）** 三级缓存架构。
- 热度榜使用分钟级 ZSET 滑动窗口 + 快照合并（`ZUNIONSTORE`），保证分页稳定性。
- 数据变更时（视频删除、点赞、评论等）主动 `DEL` 相关缓存，保证一致性。

### MQ 与异步
- RabbitMQ 也是**可选依赖**。发布失败时降级为**直写** MySQL 或 Redis，确保数据不丢失。
- 事件类型与 Exchange：
  - `like.events` / `like.like` | `like.unlike` → `LikeWorker`
  - `comment.events` / `comment.publish` | `comment.delete` → `CommentWorker`
  - `social.events` / `social.follow` | `social.unfollow` → `SocialWorker`
  - `video.popularity.events` / `video.popularity.update` → `PopularityWorker`
- Worker 进程在 `cmd/worker/main.go` 中启动，独立消费上述队列。

### 分页设计
- `/feed/listLatest`：时间戳游标 `latest_time`（Unix 秒）
- `/feed/listLikesCount`：**复合游标** `likes_count_before + id_before`，解决点赞数相同导致的排序不稳定
- `/feed/listByPopularity`：**快照分页** `as_of`（分钟级快照版本）+ `offset`，避免实时热度变化导致跳页/重复

---

## 安全注意事项

1. **密码存储**：使用 `bcrypt` 哈希，禁止明文存储或传输。
2. **JWT Secret**：`internal/auth/jwt.go` 中的签名密钥为硬编码示例（`my-secret-key`）。生产环境必须通过环境变量或安全秘钥管理服务注入。
3. **Token 撤销**：改名/改密/登出会立即使旧 token 失效，但 Redis 缓存缺失时依赖 MySQL 回退，需保证 DB 可用性。
4. **文件上传**：`uploadVideo` / `uploadCover` 保存到本地磁盘，生产环境应替换为对象存储（OSS/S3/MinIO）并做文件类型/大小校验。
5. **CORS / 代理**：本地开发通过 Vite 代理解决跨域；生产由 Nginx 统一反向代理，避免直接暴露后端端口。
6. **Rate Limit**：当前限流依赖 Redis，Redis 不可用时限流中间件会自动跳过（不阻断请求），生产需评估是否接受此降级行为。

---

## 常用命令速查

| 命令 | 说明 |
|------|------|
| `docker compose up -d --build` | 全量容器化启动 |
| `cd backend && go run ./cmd` | 启动 API 服务 |
| `cd backend && go run ./cmd/worker` | 启动 Worker |
| `cd frontend && npm install && npm run dev` | 启动前端开发服务器 |
| `cd backend && go test ./...` | 运行所有 Go 测试 |
| `./start.sh` | 本地一键启动（backend + worker + frontend + 依赖） |
| `START_FRONTEND=0 ./start.sh` | 仅启动后端与 Worker |
