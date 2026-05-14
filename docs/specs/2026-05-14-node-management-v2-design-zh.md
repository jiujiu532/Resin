# 节点管理 V2 - 设计规格书

## 概述

将当前的平台级黑名单进化为完整的节点生命周期管理系统：
- 总节点池：禁用 / 启用 / 删除，操作联动所有平台
- 平台节点池：禁用 / 启用（仅影响本平台）+ 删除（全局联动）
- 平台订阅来源选择器（药丸式多选）
- 批量操作（复选框 + 全选）

---

## 1. 概念与层级关系

```
总节点池（所有订阅导入的全部节点）
  |
  +-- 平台 A（筛选后的子集：订阅来源 + 正则 + 地区 + 黑名单）
  |     |-- 启用的节点（参与路由）
  |     +-- 禁用的节点（存在但不参与路由）
  |
  +-- 平台 B（另一个筛选子集）
        |-- 启用的节点
        +-- 禁用的节点
```

**核心规则：**
- 总节点池 > 平台节点池（平台是总池的子集）
- 在总节点池中禁用/启用/删除 → 联动影响所有包含该节点的平台
- 在平台中禁用/启用 → 仅影响当前平台，不影响其他平台和总池
- 删除操作无论在哪里触发，都是全局彻底删除

---

## 2. 节点状态

每个节点有两个独立的状态维度：

| 维度 | 作用域 | 取值 |
|------|--------|------|
| 全局启用状态 | 总节点池 | 启用 / 禁用 |
| 平台启用状态 | 每个平台独立 | 启用 / 禁用（通过 blockedNodes 实现） |

一个节点在某平台中可路由的条件：`全局启用 AND 平台启用 AND 通过过滤规则 AND 健康`

---

## 3. 功能拆解

### 3.1 总节点池页面（原"节点池"改名）

**UI 变更：**
- 页面标题：改为"总节点池"
- 顶部工具栏：新增 3 个操作按钮 [禁用] [启用] [删除]
- 每行左侧：新增复选框
- 表头左侧：新增"全选"复选框
- 未选中任何节点时，按钮置灰不可点击

**操作说明：**
- **禁用（批量）：** 将选中的节点标记为全局禁用。所有包含这些节点的平台中，这些节点也会变为不可路由。持久化到数据库 `disabled_nodes` 表。
- **启用（批量）：** 将选中的节点从全局禁用列表中移除。注意：如果某平台单独禁用了该节点（平台级黑名单），平台级禁用不受影响。
- **删除（批量）：** 从总节点池中彻底删除节点。弹出危险确认对话框，带"我已了解风险，不再提醒"勾选框。删除后从所有订阅的 managed nodes、所有平台的黑名单、以及全局池中移除。

**后端 API：**
- `POST /api/v1/nodes/actions/disable` 请求体：`{"node_hashes": [...]}`
- `POST /api/v1/nodes/actions/enable` 请求体：`{"node_hashes": [...]}`
- `POST /api/v1/nodes/actions/delete` 请求体：`{"node_hashes": [...]}`

### 3.2 平台订阅来源选择器

**当前行为：** 平台通过正则表达式 + 地区过滤从整个总节点池中自动筛选节点。

**新行为（叠加）：** 平台新增 `subscription_sources` 字段（订阅 ID 列表）。当该字段非空时，只有来自选定订阅的节点才有资格进入该平台的可路由视图。正则和地区过滤仍然作为二次筛选条件。

**UI 变更（平台详情 > 配置 tab）：**
- 在正则过滤规则上方新增"订阅来源"区域
- 一个下拉框，列出所有已导入的订阅
- 用户选择订阅后，以药丸/标签形式展示在上方
- 每个药丸有 X 按钮可移除
- 不选择任何订阅 = 使用全部订阅（向后兼容）
- 正则过滤和地区过滤保留在下方，作为二次筛选

**数据模型：**
- `model.Platform` 新增 `SubscriptionSources []string`（订阅 ID 列表）
- 数据库迁移：`ALTER TABLE platforms ADD COLUMN subscription_sources_json TEXT NOT NULL DEFAULT '[]'`
- `evaluateNode` 新增检查：如果 SubscriptionSources 非空，节点必须属于其中至少一个订阅

### 3.3 平台节点池

**UI 变更（平台详情页）：**
- "可路由节点"链接改名为"{平台名}节点池"
- 显示：该平台下总节点数，下方标注启用个数和禁用个数
- 节点列表同样带复选框 + 批量操作
- 顶部工具栏：[禁用] [启用] [删除]

**操作说明：**
- **禁用（平台级）：** 将节点加入该平台的 `blocked_node_hashes`。仅影响当前平台，不影响其他平台。
- **启用（平台级）：** 将节点从该平台的 `blocked_node_hashes` 中移除。仅影响当前平台。
- **删除（全局联动）：** 与总节点池的删除行为一致。弹出危险确认对话框。从所有平台和总节点池中彻底删除。

**后端 API（复用已有 + 新增批量）：**
- 平台禁用 = 已有的 `POST /platforms/{id}/blocked-nodes`（拉黑）
- 平台启用 = 已有的 `DELETE /platforms/{id}/blocked-nodes/{hash}`（解除拉黑）
- 批量平台禁用 = 新增 `POST /platforms/{id}/blocked-nodes/batch`
- 批量平台启用 = 新增 `DELETE /platforms/{id}/blocked-nodes/batch`
- 删除 = 复用全局删除 `POST /api/v1/nodes/actions/delete`

### 3.4 删除确认对话框

**行为：**
- 模态对话框，危险风格（红色）
- 文案："删除操作不可恢复。选中的节点将从所有平台和总节点池中永久移除。"
- 显示即将删除的节点数量
- 右下角：勾选框"我已了解风险，不再提醒"
- 按钮：[取消] [确认删除]
- "不再提醒"偏好存储在浏览器 localStorage 中

---

## 4. 数据模型变更

### 4.1 新增：全局禁用节点表

在 state.db 中新建表存储全局禁用的节点 hash。

**数据库迁移 000007：**
```sql
CREATE TABLE IF NOT EXISTS disabled_nodes (
    node_hash TEXT PRIMARY KEY
);
```

### 4.2 平台模型更新

`model.Platform` 新增 `SubscriptionSources []string`。

**数据库迁移 000008：**
```sql
ALTER TABLE platforms ADD COLUMN subscription_sources_json TEXT NOT NULL DEFAULT '[]';
```

### 4.3 evaluateNode 更新

```
func evaluateNode(entry, subLookup, geoLookup) bool:
    0. 全局禁用 -> false
    1. 订阅被禁用 -> false
    2. 不健康 -> false
    3. subscription_sources 非空 且 节点不属于任何选定订阅 -> false
    4. 正则过滤不通过 -> false
    5. 出口 IP 未知 -> false
    6. 地区过滤不通过 -> false
    7. 无延迟数据 -> false
    8. 平台级黑名单 -> false
    return true
```

---

## 5. API 汇总

| 方法 | 路径 | 作用域 | 说明 |
|------|------|--------|------|
| POST | /api/v1/nodes/actions/disable | 全局 | 批量全局禁用节点 |
| POST | /api/v1/nodes/actions/enable | 全局 | 批量全局启用节点 |
| POST | /api/v1/nodes/actions/delete | 全局 | 批量永久删除节点 |
| GET | /api/v1/nodes | 全局 | 列出节点（新增 `globally_disabled` 过滤参数） |
| POST | /platforms/{id}/blocked-nodes | 平台 | 在平台中禁用节点（已有） |
| DELETE | /platforms/{id}/blocked-nodes/{hash} | 平台 | 在平台中启用节点（已有） |
| POST | /platforms/{id}/blocked-nodes/batch | 平台 | 批量在平台中禁用 |
| DELETE | /platforms/{id}/blocked-nodes/batch | 平台 | 批量在平台中启用 |

---

## 6. 前端变更汇总

| 页面 | 变更内容 |
|------|----------|
| 节点池页面 | 改名"总节点池"，新增复选框 + 全选 + 批量操作工具栏 |
| 平台详情 > 配置 | 新增订阅来源药丸选择器，放在正则/地区过滤上方 |
| 平台详情 > 监控 | "可路由节点"改为显示总数/启用数/禁用数 |
| 平台节点池视图 | 新增复选框 + 批量操作工具栏（禁用/启用/删除） |
| 删除确认弹窗 | 新建可复用的模态组件，带"不再提醒"localStorage |

---

## 7. 实施顺序（子项目）

1. **子项目 1：** 全局禁用/启用 + 删除 API + 数据库迁移 + evaluateNode 更新
2. **子项目 2：** 前端批量操作（复选框、工具栏、删除确认弹窗）用于总节点池
3. **子项目 3：** 平台订阅来源选择器（后端 + 前端）
4. **子项目 4：** 平台节点池视图 + 平台级禁用/启用 + 批量操作

每个子项目独立可部署、可测试。

---

## 8. 向后兼容性

- `subscription_sources` 为空 = 使用全部订阅（保持当前行为）
- `disabled_nodes` 表为空 = 所有节点启用（保持当前行为）
- 已有的 `blocked_node_hashes` 继续作为平台级禁用使用
- 无破坏性 API 变更，所有新端点都是新增的
