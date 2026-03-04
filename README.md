# QuantForge

基于 **gin-vue-admin（Go + Vue）** 构建的企业级量化交易系统，支持策略在线开发、多资产回测、策略沙箱执行、模拟盘运行、统一风控与多租户权限隔离。

## 1. 项目目标

QuantForge 旨在构建一个可扩展、可审计、可管控的量化交易平台，核心能力包括：

- 策略在线开发与版本管理
- 多资产高性能回测
- 容器化策略沙箱执行
- 模拟盘交易与订单生命周期管理
- 策略级 / 账户级 / 租户级统一风控
- 企业级多租户 + RBAC 权限体系

## 2. 总体技术架构

```text
[ Web Frontend (Vue + Ant Design Vue) ]
                   ↓
[ API Gateway / gin-vue-admin Backend ]
                   ↓
-------------------------------------------------------
| User Service      | Strategy Service | Backtest Engine |
| Sim Engine        | Risk Engine      | Data Service    |
| Scheduler         | Audit Service    | Notification    |
-------------------------------------------------------
                   ↓
[ Redis | PostgreSQL | ClickHouse | Object Storage ]
```

### 2.1 技术选型

- **后端**：Golang（gin-vue-admin 主框架，REST API）
- **策略执行引擎**：Python（策略 SDK + 执行 Runtime）
- **服务通信**：gRPC（高吞吐内部服务通信）
- **缓存与队列**：Redis（缓存、任务队列、限流）
- **业务数据库**：PostgreSQL（租户、用户、订单、审计）
- **行情数据库**：ClickHouse（高频 K 线 / Tick）
- **沙箱**：Docker（资源限制 + 网络隔离）
- **前端**：Vue（gin-vue-admin）+ Ant Design Vue + WebSocket

## 3. 与 gin-vue-admin 的集成方案

### 3.1 模块划分建议

在 gin-vue-admin 后端采用模块化目录：

- `server/api/v1/quantforge/`：HTTP API
- `server/service/quantforge/`：业务服务层
- `server/model/quantforge/`：领域模型与 DTO
- `server/router/quantforge/`：路由注册
- `server/middleware/`：租户隔离、鉴权、审计中间件

前端采用：

- `web/src/view/quantforge/`：页面（策略、回测、模拟盘、风控）
- `web/src/api/quantforge/`：API 调用封装
- `web/src/pinia/modules/quantforge.ts`：状态管理

### 3.2 多租户隔离

- 在 JWT 中写入 `tenant_id`、`role`
- 后端统一中间件自动注入 `tenant_id` 过滤条件
- 所有业务表必须包含 `tenant_id`
- 严禁跨租户查询与写入

### 3.3 RBAC 角色

- `Admin`
- `Quant Developer`
- `Risk Manager`
- `Viewer`

权限维度：

- 策略创建
- 策略运行
- 回测查看
- 模拟盘下单
- 风控策略修改

## 4. 核心领域模块设计

### 4.1 用户与权限系统

#### 表结构（示例）

```sql
CREATE TABLE tenant (
    id UUID PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE "user" (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenant(id),
    username VARCHAR(64) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### 4.2 策略管理模块

功能：

- 创建策略
- 上传策略文件
- 版本管理
- 参数管理
- 一键触发回测

策略 Python 接口规范：

```python
class Strategy:
    def initialize(self, context):
        pass

    def handle_bar(self, context, bar):
        pass

    def on_order(self, context, order):
        pass
```

### 4.3 策略沙箱执行

每个策略运行在独立容器，需满足：

- CPU / 内存限制
- 禁止外网访问（`--network=none`）
- 限制文件系统访问（只读挂载 + 临时目录）
- 执行超时自动 kill

示例：

```bash
docker run \
  --memory=512m \
  --cpus=1 \
  --network=none \
  --read-only \
  strategy_runner
```

### 4.4 回测引擎

流程：

1. 加载历史数据
2. 按时间推进
3. 调用策略逻辑
4. 生成订单
5. 撮合成交
6. 更新账户权益
7. 记录回测结果

必须支持：

- 市价单、限价单
- 滑点模型（固定、百分比、成交量冲击）
- 手续费模型

滑点公式：

```text
成交价 = 市价 * (1 ± 滑点比例)
```

性能目标：

- 单任务百万级 K 线回测
- 并发 200+ 回测任务
- 多线程 / 分片读取数据

### 4.5 风控系统

风险维度：

- 最大回撤限制
- 单笔仓位比例
- 总杠杆限制
- 单日最大亏损
- 持仓集中度限制

执行示例：

```python
if current_drawdown > max_drawdown:
    force_close_all_positions()
    stop_strategy()
```

风控级别：

- 策略级
- 账户级
- 租户级

### 4.6 数据服务

- 日线：PostgreSQL
- 高频数据：ClickHouse
- 热点数据：Redis

标准接口：

- K 线查询
- 批量数据下载
- 因子计算接口
- 数据完整性校验

### 4.7 模拟盘系统

模拟账户表示例：

```sql
CREATE TABLE account (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    balance DECIMAL(20, 8) NOT NULL,
    equity DECIMAL(20, 8) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

订单状态机：

```text
CREATED -> SUBMITTED -> FILLED / CANCELLED
```

### 4.8 任务调度系统

基于 Redis Queue：

- 异步回测任务
- 自动重试
- 优先级队列

### 4.9 日志与审计

必须记录：

- 策略代码版本
- 回测参数
- 风控触发记录
- 每笔成交记录
- 用户操作日志

要求：审计日志不可删除（append-only + 冷备份）。

## 5. 安全设计

- JWT 认证
- 强制 HTTPS
- 策略沙箱隔离
- API 限流
- SQL 注入防护
- XSS 防护

## 6. 性能指标

| 指标 | 要求 |
|---|---|
| API 延迟 | < 50ms |
| 回测百万 K 线 | < 5s |
| 并发用户 | 1000 |
| 数据可用性 | 99.9% |

## 7. 部署建议

- Kubernetes 部署
- 策略容器自动扩缩容（HPA）
- 计算节点与数据节点分离
- 回测任务与实时模拟盘任务资源隔离

## 8. 里程碑规划（建议）

- **M1（基础能力）**：租户 + RBAC + 策略 CRUD + 回测最小闭环
- **M2（性能增强）**：并发回测、ClickHouse 优化、任务调度完善
- **M3（生产就绪）**：统一风控、审计闭环、K8s 高可用部署

---

> 当前仓库作为 QuantForge 的架构与需求基线文档，后续可按上述模块逐步落地到 gin-vue-admin 前后端代码结构。

## 9. 开发启动（第一阶段落地）

当前仓库已开始提供一个可运行的后端雏形（采用与 gin-vue-admin 一致的分层思路）：

- `server/cmd/main.go`：应用入口
- `server/internal/router`：路由装配
- `server/internal/api`：HTTP Handler
- `server/internal/service`：业务逻辑
- `server/internal/model`：领域模型
- `server/internal/middleware`：租户与角色中间件

### 9.1 本地运行

```bash
cd server
go run ./cmd
```

默认监听 `:8080`。

### 9.2 示例接口

1. 健康检查

```bash
curl http://localhost:8080/health
```

2. 创建策略（需要 Admin / Quant Developer）

```bash
curl -X POST http://localhost:8080/api/v1/strategies \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-a" \
  -H "X-Role: Quant Developer" \
  -d '{"name":"demo-strategy","version":"v1","parameters":{"symbol":"BTCUSDT"}}'
```

3. 查询策略列表

```bash
curl http://localhost:8080/api/v1/strategies \
  -H "X-Tenant-ID: tenant-a"
```

4. 更新策略（需要 Admin / Quant Developer）

```bash
curl -X PUT http://localhost:8080/api/v1/strategies/stg_1 \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-a" \
  -H "X-Role: Quant Developer" \
  -d '{"version":"v2","status":"ACTIVE"}'
```

5. 删除策略（仅 Admin）

```bash
curl -X DELETE http://localhost:8080/api/v1/strategies/stg_1 \
  -H "X-Tenant-ID: tenant-a" \
  -H "X-Role: Admin"
```


6. 触发回测（需要 Admin / Quant Developer）

```bash
curl -X POST http://localhost:8080/api/v1/backtests \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-a" \
  -H "X-Role: Quant Developer" \
  -d '{"strategyId":"stg_1","priority":8,"maxRetries":2,"parameters":{"start":"2024-01-01","end":"2024-12-31"}}'
```

7. 查询回测任务列表

```bash
curl http://localhost:8080/api/v1/backtests \
  -H "X-Tenant-ID: tenant-a"
```


8. 创建模拟账户（需要 Admin / Quant Developer）

```bash
curl -X POST http://localhost:8080/api/v1/sim/accounts \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-a" \
  -H "X-Role: Quant Developer" \
  -d '{"balance":100000}'
```

9. 模拟下单（状态机会自动推进到 FILLED）

```bash
curl -X POST http://localhost:8080/api/v1/sim/orders \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-a" \
  -H "X-Role: Quant Developer" \
  -d '{"accountId":"acc_1","symbol":"BTCUSDT","side":"BUY","qty":0.1,"price":50000}'
```

10. 查询模拟订单

```bash
curl http://localhost:8080/api/v1/sim/orders \
  -H "X-Tenant-ID: tenant-a"
```


11. 配置租户风控规则（需要 Admin / Risk Manager）

```bash
curl -X PUT http://localhost:8080/api/v1/risk/rules \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-a" \
  -H "X-Role: Risk Manager" \
  -d '{"maxOrderNotional":100000,"maxDailyLoss":200000,"maxPositionPercent":0.3}'
```

12. 查询租户风控规则

```bash
curl http://localhost:8080/api/v1/risk/rules \
  -H "X-Tenant-ID: tenant-a"
```

> 说明：当前实现为启动阶段原型（内存存储），后续将迁移到 PostgreSQL / Redis / ClickHouse 并逐步接入 gin-vue-admin 现有权限与代码生成体系。


## 10. 当前完成度（M1 对照）

### 已完成（原型）

- 多租户上下文与 RBAC 鉴权中间件（基于 `X-Tenant-ID` / `X-Role`）
- 策略管理：创建、查询、更新、删除
- 回测任务：触发、查询、优先级字段、失败重试（内存队列 + worker）
- 模拟盘：账户创建、下单、订单查询，订单状态自动流转
- 风控：租户级规则配置 + 下单前风控校验

### 未完成（下一步重点）

- 持久化存储（PostgreSQL / Redis / ClickHouse）替换内存实现
- JWT / HTTPS / 限流 / 审计不可篡改存储
- 策略沙箱容器执行（Docker 资源限制 + 网络/文件隔离）
- 真正撮合与行情驱动（滑点、手续费、持仓与权益实时计算）
- 任务调度高可用（重试策略、优先级队列持久化、失败告警）
- 前端（gin-vue-admin web）页面与接口联调

