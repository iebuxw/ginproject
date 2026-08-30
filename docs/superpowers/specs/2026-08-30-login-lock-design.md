# 登录失败锁定 设计文档

日期：2026-08-30
状态：待实现

## 目标

同一用户名连续登录失败达到阈值后临时锁定该账号，锁定时长到期后自动解锁，无需管理员介入。防止对固定账号的暴力猜密码。

## 需求决策（已与需求方确认）

- **锁定维度**：按用户名（含不存在的用户名，防枚举差异）。不做 IP 维度。
- **处理方式**：自动解锁，TTL 到期即恢复，无需解锁接口与管理员操作。
- **参数配置**：`.env` 可配，`LOGIN_LOCK_MAX_ATTEMPTS`（默认 5）、`LOGIN_LOCK_DURATION`（分钟，默认 15）。
- **失败累计窗口 = 锁定时长**：15 分钟内累计 5 次失败即触发锁定（最简单语义）。

## 方案选择

选定方案 A（Redis 单 key），备选方案及取舍：

| 方案 | 说明 | 取舍 |
| --- | --- | --- |
| **A. Redis 单 key（选定）** | `login_fail:<username>` 存连续失败计数，TTL 兼任累计窗口与锁定时长 | 逻辑最简、TTL 到期自动解锁零额外代码；与项目登录态数据放 Redis 的现有模式（token 黑名单、告警限频）一致；AuthService 已持有 rdb，无需新依赖注入。缺点：Redis 重启丢计数（学习项目可接受） |
| B. Redis 双 key | 计数窗口与锁定时长分离 | 语义更精细但多一次往返、TTL 协调逻辑更绕，对本需求是过度设计 |
| C. DB 字段 | users 表加 failed_count/locked_until，需新增迁移 | 持久化但每次失败写 MySQL，与项目现有模式相悖 |

## 架构与数据流

```
登录请求 -> AuthController.Login -> AuthService.Login
  1. 开头检查 Redis GET login_fail:<username>
     - 计数 >= 阈值 -> 返回 ErrAccountLocked（fmt.Errorf %w 包装剩余分钟数）
     - 锁定期间即使密码正确也拒绝（检查先于密码校验）
  2. 密码错误 / 用户不存在 -> INCR login_fail:<username>
     - 首次失败（INCR 后 =1）：EXPIRE 设累计窗口（15 分钟）
     - 达阈值（INCR 后 = 阈值）：EXPIRE 刷新为完整锁定时长，此刻起锁定计时
  3. 登录成功 -> DEL login_fail:<username> 清零
  4. TTL 到期 key 自动消失 -> 解锁，计数归零
```

Controller 侧：`ErrAccountLocked` 照常记登录日志（status=0，message 含剩余分钟数）、返回 401 + 提示文案；**不触发告警邮件**（现有 `errors.Is(err, ErrInvalidCredentials)` 条件不变，锁定错误天然不满足）。

## 改动清单

| 文件 | 改动 |
| --- | --- |
| `internal/config/config.go` | `Config` 新增 `LoginLock LoginLockConfig`（MaxAttempts、DurationMinutes），读取两个新 env，默认 5/15 |
| `.env.example` | 补 `LOGIN_LOCK_MAX_ATTEMPTS`、`LOGIN_LOCK_DURATION` 及注释 |
| `internal/service/auth_service.go` | 新增哨兵错误 `ErrAccountLocked`；私有 Helper：`loginFailKey`、`isLocked`（返回锁定状态+剩余分钟）、`recordFailure`（INCR+EXPIRE）、`clearFailures`（DEL）；`Login` 接入上述逻辑 |
| `internal/controller/auth_controller.go` | Login 失败分支补注释（锁定不发邮件）；Swagger Description 补锁定行为说明 |
| `docs/`（swag 生成） | `swag init` 重新生成 |

**不动**：路由、请求/响应 JSON 结构、权限点、前端、迁移文件、登录日志表结构、告警邮件链路。

## 边界与约束

- Redis 仅用 GET/INCR/EXPIRE/TTL/DEL，兼容 redis:3.2-alpine（无 HSET 多字段）。
- **降级策略**：Redis 异常时 `isLocked`/`recordFailure`/`clearFailures` 静默失败，登录流程照常（锁定是加固，不是依赖）。
- 账号被禁用（`user.Status != 1`）既不计失败也不清零，维持现状。
- 剩余分钟数向上取整，避免提示"0 分钟后"。
- 锁定计数不存在的用户名同样生效（防枚举：对攻击者反馈与真实用户一致）。
- 不新增表、不新增迁移、不新增第三方依赖。
- 对外契约不变：`{code, message, data}` 响应结构不变，仅 message 文案新增。

## 错误信息

- 凭据错误（累计中）：`用户名或密码错误`（不变）
- 锁定中：`账号已锁定，请 N 分钟后再试`

## 验证

- `go build ./...` 通过。
- Docker 重建 go-app 后 curl 验证：连续 6 次错误密码，第 6 次起返回锁定提示；锁定期间正确密码被拒；`redis-cli GET/TTL login_fail:admin` 确认计数与剩余时间；DEL 后正确密码恢复登录；登录日志 message 记录对应文案。
- 前端登录页无需改动，message 直接渲染；手工回归项见实现计划末尾清单。
