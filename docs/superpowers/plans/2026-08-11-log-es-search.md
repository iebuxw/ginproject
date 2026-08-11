# 操作日志 ES 全文检索 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: 使用 superpowers:executing-plans 或 subagent-driven-development 按任务执行。步骤用 `- [ ]` 追踪。

**Goal:** 给操作日志加 Elasticsearch 全文搜索（IK 中文分词 + 高亮），双写 MySQL+ES，查询走 ES 失败回退 MySQL。
**Architecture:** 新增 `internal/es` 数据访问层（对标现有 dao）。`OperationLogger` 中间件同步双写；`GET /logs` 优先查 ES（`bool` + `multi_match` + `range` + `highlight`），ES 不可用/失败时回退 GORM MySQL 查询并返回 `data_source` 标记。docker-compose 加单节点 ES 7.17.15（自定义镜像装 IK v7.17.15）+ Kibana。
**Tech Stack:** `github.com/elastic/go-elasticsearch/v7`、elasticsearch 7.17.15、IK v7.17.15、kibana 7.17.15、Vue 2 + Element UI。

## 文件结构总览

| 文件 | 动作 | 职责 |
|---|---|---|
| `docker/elasticsearch.Dockerfile` | Create | 装 IK 的自定义 ES 镜像 |
| `docker-compose.yml` | Modify | 加 elasticsearch + kibana 服务 |
| `.env` / `.env.example` | Modify | 加 ES_HOST/PORT/USERNAME/PASSWORD |
| `internal/config/config.go` | Modify | 加 ElasticsearchConfig |
| `internal/es/client.go` | Create | ES client 封装 + Ping |
| `internal/es/index.go` | Create | 索引常量 + mapping + EnsureIndex |
| `internal/es/log_repo.go` | Create | Index 写入 / Search 查询 + 高亮 sanitize |
| `internal/middleware/logger.go` | Modify | 双写 ES |
| `internal/service/log_service.go` | Modify | 注入 es.LogRepo，加 SearchFromES/ESEnabled |
| `internal/controller/log_controller.go` | Modify | List 改走 ES + 回退 |
| `internal/router/router.go` | Modify | Setup 加 logRepo 参数，更新 OperationLogger 调用 |
| `cmd/server/main.go` | Modify | 初始化 ES、装配 logRepo |
| `web/src/views/log/index.vue` | Modify | 搜索框/时间范围/高亮渲染 |
| `AGENTS.md` | Modify | 端口 9200/5601、架构、注意事项 |

---

### Task 1: docker 基础设施（ES + Kibana + IK）

**Files:**
- Create: `docker/elasticsearch.Dockerfile`
- Modify: `docker-compose.yml`

- [ ] **Step 1: 创建自定义 ES 镜像（装 IK v7.17.15）**

`docker/elasticsearch.Dockerfile`：
```dockerfile
FROM elasticsearch:7.17.15

ARG IK_VERSION=7.17.15

USER root
RUN set -eux; \
    yum install -y curl unzip; \
    curl -L -o /tmp/ik.zip \
      https://github.com/medcl/elasticsearch-analysis-ik/releases/download/v${IK_VERSION}/elasticsearch-analysis-ik-${IK_VERSION}.zip; \
    /usr/share/elasticsearch/bin/elasticsearch-plugin install --batch file:///tmp/ik.zip; \
    rm -f /tmp/ik.zip; \
    yum remove -y curl unzip

USER elasticsearch
```

- [ ] **Step 2: docker-compose.yml 加两个服务**

`elasticsearch` + `kibana` 服务（配置见下），`volumes:` 段追加 `es-data:`。

- [ ] **Step 3: 启动并验证 IK 已装**

```bash
docker compose build elasticsearch
docker compose up -d elasticsearch kibana
curl "http://localhost:9200/_cat/plugins?v"
```
Expected：`analysis-ik` 出现在插件列表，集群 health `yellow`（单节点正常）。

- [ ] **Step 4: Commit**

---

### Task 2: 配置与依赖

**Files:**
- Modify: `go.mod`（`go get` 后自动更新）
- Modify: `internal/config/config.go`
- Modify: `.env`、`.env.example`

- [ ] **Step 1:** `go get github.com/elastic/go-elasticsearch/v7@v7.17.10`
- [ ] **Step 2:** config.go 加 `ElasticsearchConfig{Host,Port,Username,Password}` + `Addr()` 方法 + `Load()` 映射
- [ ] **Step 3:** `.env.example` 追加 ES 变量（`ES_HOST=elasticsearch`）；本地 `.env` 同段 `ES_HOST=127.0.0.1`
- [ ] **Step 4:** `go build ./...` 验证通过
- [ ] **Step 5: Commit**

---

### Task 3: `internal/es` 数据访问层

**Files:**
- Create: `internal/es/client.go`
- Create: `internal/es/index.go`
- Create: `internal/es/log_repo.go`

- [ ] **Step 1: client.go** — `NewClient(cfg) (*Client, error)`、`Ping()`、导出 `RawClient()`（Task 4 跨包使用）
- [ ] **Step 2: index.go** — 索引常量 `operation_logs`、`logMapping`（见 spec 映射表，`created_at` format `yyyy-MM-dd HH:mm:ss`）、`EnsureIndex(cli)` 幂等创建
- [ ] **Step 3: log_repo.go** — `LogRepo{cli}`；`Enabled()`；`Index(log)`（`_id`=MySQL ID，**Refresh: "true"** 保证改完立即可搜）；`Search(ctx, q)` 组装 bool DSL + 解析高亮；`SearchQuery`/`SearchHitDoc` 类型；`buildBoolQuery`/`join`/`sanitizeHighlight`
- [ ] **Step 4:** `go build ./...` 验证通过
- [ ] **Step 5: Commit**

---

### Task 4: 双写 + 查询改造 + 装配

**Files:**
- Modify: `internal/middleware/logger.go`
- Modify: `internal/service/log_service.go`
- Modify: `internal/controller/log_controller.go`
- Modify: `internal/router/router.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: middleware/logger.go** — `OperationLogger(logDAO, logRepo)` 同步双写，ES 失败仅告警
- [ ] **Step 2: service/log_service.go** — 注入 `logRepo`，加 `ESEnabled()`、`SearchFromES(q)`
- [ ] **Step 3: controller/log_controller.go** — `List` 先 ES（keyword/start_time/end_time）成功返回 `data_source: es`，失败回退 `FindPage` 返回 `data_source: mysql`
- [ ] **Step 4: router/router.go** — `Setup` 加 `logRepo` 参数，更新所有 `OperationLogger(logDAO, logRepo)`
- [ ] **Step 5: main.go** — 初始化 ES client（失败仅告警不 fatal）、`EnsureIndex`、装配 logRepo 进 service/router
- [ ] **Step 6:** `go build ./... && go vet ./...` 验证通过
- [ ] **Step 7: Commit**

---

### Task 5: 前端改造

**Files:**
- Modify: `web/src/views/log/index.vue`

- [ ] **Step 1:** 表单加关键词输入框、`el-date-picker` 时间范围（value-format `yyyy-MM-dd HH:mm:ss`）、数据源 tag
- [ ] **Step 2:** path/params 列改用 `v-html` 渲染 `highlight_path`/`highlight_params`（fallback 原值）；`fetchData` 传 keyword/start_time/end_time；记录 `dataSource`
- [ ] **Step 3:** `::v-deep em { color:#e6a23c; font-style:normal }` 高亮样式
- [ ] **Step 4:** `cd web && npm run build` 验证通过
- [ ] **Step 5: Commit**

---

### Task 6: 更新 AGENTS.md

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1:** 固定端口补 ES 9200 / Kibana 5601；架构补 `internal/es/`；新增操作日志双写说明 + ES/IK 版本严格匹配提醒
- [ ] **Step 2: Commit**

---

### Task 7: 端到端验证清单（不提交）

1. `docker compose up -d` 全部服务
2. `go run cmd/server/main.go` 本地启动（.env 需 `ES_HOST=127.0.0.1`）
3. 登录 `admin/admin`，做几次 POST/PUT/DELETE（含中文 params）
4. `curl` ES 立即命中新日志；日志页 `data_source: es`、关键词命中高亮
5. 停 ES → 回退 `mysql`；起 ES → 恢复 `es`
6. `_analyze` 验证 IK 切词
