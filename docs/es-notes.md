# ES 增删改查速查笔记

本项目 ES 只用做**操作日志索引**（`operation_logs`），代码全在 `internal/es/` 三文件：
`client.go`(连接) / `index.go`(建索引) / `log_repo.go`(增查)。

MySQL 是**唯一数据源**，ES 是**读优化副本**，失败降级不阻断。

## 0. 建索引（对标"建表"）

- `es.EnsureIndex`（`internal/es/index.go:47`），启动时 `main.go:52` 调用，幂等（已存在则跳过）
- mapping 定义在 `index.go:18`，**上线前定死，建后难改**
- 索引名 `LogIndex = "operation_logs"`（`index.go:13`）

**记住**：对标 GORM AutoMigrate，自动建，但改 mapping = 重建索引。

## 1. 增（写一条日志）→ `LogRepo.Index`

`internal/es/log_repo.go:27`

| 要点 | 说明 |
|---|---|
| `DocumentID` | = **MySQL 主键**，同 id 覆盖→**双写幂等**（对标 `ON DUPLICATE KEY UPDATE`） |
| `Refresh:"true"` | 写入立即可见；默认要等 1s（近实时 NRT）。日志量小才敢开 |
| 触发点 | `middleware/logger.go:50`，每次**非 GET** 请求：先写 MySQL 再同步写 ES；ES 失败只 `log.Printf` 告警，不阻断请求 |

**记住**：id 复用主键防重复，ES 挂了不影响业务，日志重新写会自然补回。

## 2. 查（全文检索）→ `LogRepo.Search`

`internal/es/log_repo.go:71`，DSL 由 `buildBoolQuery`（`log_repo.go:139`）组装：

```
bool
├─ must   多字段关键词全文检索（IK 分词）→ multi_match on path/params/response
└─ filter 精确筛选（不打分，可缓存）：
          ├─ term   module / method    （这俩是 keyword，整体匹配）
          └─ range  created_at  gte/lte
附带：highlight 高亮(<em>)、sort created_at desc、from/size 分页
```

入口 `GET /logs`（`controller/log_controller.go:42`）：**优 ES，失败/未启用回退 MySQL**，响应带 `data_source`(`es`/`mysql`) 由前端展示。

**记住**：must=模糊搜，filter=精确筛；搜词走 text 分词，筛选用 keyword。

## 3. 删 / 改（本项目未实现）

日志 append-only，没做删改。若要做，底层 API：
- 删单条：`esapi.DeleteRequest{Index, DocumentID}`
- 改单条：`esapi.UpdateRequest{Index, DocumentID, Body:{doc:{...}}}`
- 按条件删：`esapi.DeleteByQueryRequest{Index, Body}`

难点：要与 MySQL 双写一致，改哪边、失败怎么补偿，新手别碰。

**记住**：项目里"删改"不存在，ES 只有"写入覆盖 + 查询"。

## 4. 概念速查

- **text**：分词后存，模糊搜（`LIKE %词%`）｜**keyword**：整块存，精确匹配（`=`）按此选型
- **IK 分词**：`analyzer=ik_max_word`（写入，切细→召回全）、`search_analyzer=ik_smart`（搜索，切准→误伤少）；两者分开配
- **`path.keyword` 子字段**：mapping 有（`index.go:36`）、**代码从未查询**，纯教学对比；项目里真实在用的 keyword 是 `module`/`method`（`log_repo.go:151,154` 的 term）

## 5. 调用链

```
启动  main.go → es.NewClient → EnsureIndex（建索引）
增    请求 → middleware.OperationLogger → dao.Create(MySQL) + es.Index(ES)
查    GET /logs → controller.List → service.SearchFromES → es.Search
                                        └─ 失败回退 MySQL FindPage
```

## 6. 易忘点

- IK 插件必须与 ES **严格同版本**（当前均 7.17.15）
- ES 不可用：查询自动回退 MySQL；写入仅告警
- `es.NewLogRepo(cli)` 可传 nil，内部 `Enabled()` 判断整体降级
- 高亮片段经 `sanitizeHighlight` 转义防 XSS，前端 `v-html` 只放行 `<em>`