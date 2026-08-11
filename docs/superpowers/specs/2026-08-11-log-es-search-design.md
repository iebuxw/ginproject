# 操作日志 Elasticsearch 全文检索 设计文档

> 日期：2026-08-11　目的：学习 Elasticsearch（全文检索 + IK 中文分词）

## 背景与目标

本项目（Go Gin + Vue2 后端管理系统）已有操作日志（`OperationLog`）与登录日志模块，日志存 MySQL，查询仅支持 `module`/`method` 精确筛选。目标是以操作日志为载体引入 Elasticsearch 学习：

- 掌握 index/mapping、analyzer、IK 中文分词
- 掌握 `bool`/`multi_match`/`range`/`highlight` 查询 DSL
- 掌握 MySQL + ES 双写、查询走 ES 的实践
- 掌握 `bulk`、`from/size` 分页等概念（历史数据不导入，bulk 不在本次范围）

## 方案决策（已确认）

| 决策点 | 选择 | 说明 |
|---|---|---|
| 功能形态 | 方案 A：操作日志全文搜索 + 高亮 | 命中学习重点「全文检索 + 中文分词」 |
| ES 部署 | docker-compose 单节点 ES + Kibana | ES 7.17.15，无安全认证，便于学习 |
| 数据流向 | 双写 MySQL + ES，查询走 ES | MySQL 保留导出等旧逻辑 |
| 分词器 | 自定义镜像装 IK v7.17.15 | 与 ES 7.17.15 严格匹配 |
| 写入方式 | 同步写入 | 时序直观，先学通再谈异步 |
| 查询回退 | ES 失败回退 MySQL | 响应带 `data_source: "es"/"mysql"` 标记 |
| 历史数据 | 不导入 | 只同步新产生的日志 |

## 架构

```
middleware.OperationLogger ──> dao.LogDAO (MySQL)  双写
                        └──> es.LogRepo.Index (ES, Refresh=true)
GET /logs ──> LogController.List ──> ES 查询成功? ──> 返回 data_source=es
                                       └─失败→ LogService.FindPage (MySQL) → data_source=mysql
```

- 新增 `internal/es/`：client 封装、索引管理（对标 AutoMigrate）、LogRepo（写入/搜索）
- ES 不可用时不 fatal：写入跳过、查询回退，保证开发体验
- 文档 `_id` = MySQL 日志 ID，保证双写幂等

## 索引映射（operation_logs）

| 字段 | 类型 | 说明 |
|---|---|---|
| id / operator_id | long | |
| module / action / method / ip | keyword | 精确筛选 |
| path | text + fields.keyword | 可精确也可全文 |
| params / response | text | IK 分词，JSON 含中文演示佳 |
| duration | integer | |
| created_at | date, format `yyyy-MM-dd HH:mm:ss` | 与 `model.DateTime.MarshalJSON` 输出一致 |

- analyzer `ik_max_word`，search_analyzer `ik_smart`
- settings：`number_of_shards=1`、`number_of_replicas=0`
- 自定义 analyzer 注册 `ik_max_word`/`ik_smart`（对应 IK 内置类型）

## 查询 DSL

```
bool:
  must:   multi_match(keyword → path/params/response)，关键词空则省略
  filter: term(module)、term(method)、range(created_at gte/lte)
from/size 分页；sort created_at desc；highlight 对 path/params/response
```

返回 `{list, total, data_source}`；每条带 `highlight_path`/`highlight_params`/`highlight_response`。

## 安全

高亮片段经 `sanitizeHighlight` 处理：先占位 `<em>`，再 `html.EscapeString` 转义，最后还原 `<em>`，防止 `v-html` XSS。

## 错误处理

- ES 初始化失败：告警日志，不阻塞启动
- 写入失败：仅 `log.Printf` 告警，不阻断请求
- 查询失败：回退 MySQL 并返回 `data_source: mysql`

## 验证标准（ES 搜索真实生效）

1. 写入闭环：页面产生中文 POST 日志后，`curl` 查询 ES **立即命中**
2. `data_source: es`（证明走了 ES 而非回退）
3. 关键词搜索命中且高亮
4. method/时间范围筛选正确
5. `_analyze` 验证 IK 切词
6. 停 ES → 回退 mysql；起 ES → 恢复 es

## 学习点对照

- `multi_match`/`bool`/`range`/`highlight`、keyword vs text
- `ik_max_word` vs `ik_smart`、mapping/analyzer、refresh 语义
- 双写幂等（`_id`）、`from/size` 分页（`search_after` 为进阶备注）
