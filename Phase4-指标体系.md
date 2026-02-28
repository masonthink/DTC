# 数字分身社区 · Phase 4 指标体系

> 最后更新：2026-02-25

---

## 北极星指标

**月有效连接数** — 同一月内，双方真人完成联系方式交换的次数

选择原因：
- 代表平台交付了真实价值（不是虚假活跃）
- 无法被刷数据（需双方主动确认）
- 是所有上游指标的最终验证

阶段目标：

| 阶段 | 用户规模 | 月有效连接目标 |
|------|---------|--------------|
| MVP 期 | 0 → 500 用户 | ≥ 50 |
| 成长期 | 500 → 5,000 用户 | ≥ 500 |
| 规模期 | 5,000 → 10 万用户 | ≥ 5,000 |

---

## 指标树（北极星拆解）

```
月有效连接数
│
├── 连接申请数 × 双向确认率
│         │
│         连接申请数 = 报告读完数 × 连接意向率
│                   │
│                   报告读完数 = 报告发送数 × 报告打开率
│                             │
│                             报告发送数 = Topic 提交数 × 匹配成功率
│                                       │
│                                       Topic 提交数 = MAU × 人均提交率
│                                                   │
│                                                   MAU = 注册数 × 激活率 × 留存率
│
└── 连接质量分（用户主观评价，防止无效连接污染北极星）
```

---

## AARRR 漏斗指标

### Acquisition 获客

| 指标 | 定义 | 目标值（MVP期） |
|------|------|---------------|
| 新注册用户数 | 每日/周/月注册量 | 月增 200+ |
| 渠道转化率 | 各渠道访问→注册 | > 15% |
| 邀请转化率 | 收到邀请→完成注册 | > 40% |
| CAC | 获取一个用户的平均成本 | < ¥30 |

### Activation 激活

激活漏斗目标：

| 步骤 | 目标转化率 |
|------|-----------|
| 注册 → 完成 Agent 创建问卷 | > 70% |
| 完成问卷 → 预览并确认 Agent | > 85% |
| 注册后 7 日内 → 提交第一个 Topic | > 50% |
| 收到报告后 24h 内 → 打开第一份完整报告 | > 65% |
| 看完首份报告 → 产生第一次连接意向 | > 25% |

### Retention 留存

| 指标 | 定义 | 目标值 |
|------|------|--------|
| D7 留存率 | 注册 7 天后仍活跃 | > 40% |
| D30 留存率 | 注册 30 天后仍活跃 | > 25% |
| 月活跃定义 | 当月打开过至少一份报告 | — |
| Topic 月均提交数 | 每个活跃用户每月提交量 | > 1.5 个 |
| 报告打开率 | 收到报告→打开 | > 60% |
| Agent 纠正率 | 活跃用户中主动纠正过 Agent 的比例 | > 30% |

### Revenue 变现（V2 阶段）

| 指标 | 定义 | 目标值（V2期） |
|------|------|--------------|
| Pro 转化率 | 活跃用户→付费订阅 | > 5% |
| ARPU | 月均每用户收入 | ¥15（含免费用户） |
| 付费用户续订率 | 月度续订 | > 80% |
| MRR 增长率 | 月经常性收入环比增长 | > 20% |

### Referral 传播

| 指标 | 定义 | 目标值 |
|------|------|--------|
| 邀请发送率 | 活跃用户中发出邀请的比例 | > 20% |
| 病毒系数 K | 每个用户平均带来的新用户数 | > 0.4 |
| NPS | 净推荐值 | > 40 |
| 有效连接后邀请率 | 连接成功后发邀请的比例 | > 35% |

---

## OKR 拆解（按季度）

### Q1：验证核心价值（MVP 上线，目标 500 用户）

**O1 证明「用想法找到对的人」这个价值是真实的**
- KR1 月有效连接数 ≥ 50
- KR2 Agent 创建完成率 ≥ 70%
- KR3 首份报告打开率 ≥ 60%
- KR4 D7 留存率 ≥ 40%

**O2 建立种子用户基础**
- KR1 招募并激活 30 个高质量种子 Agent
- KR2 收集 20 份用户深度访谈
- KR3 NPS ≥ 40（种子用户群）

### Q2：扩大规模（目标 5,000 用户）

**O1 实现可持续的用户增长**
- KR1 MAU 达到 2,000
- KR2 月有效连接数 ≥ 500
- KR3 病毒系数 K ≥ 0.3
- KR4 D30 留存率 ≥ 25%

**O2 提升讨论与匹配质量**
- KR1 用户对报告的满意度评分 ≥ 4.0/5
- KR2 匹配满意率（用户认为匹配相关）≥ 70%
- KR3 Agent 纠正率 ≥ 30%

### Q3：验证变现（目标 2 万用户）

**O1 上线 Pro 订阅，验证付费意愿**
- KR1 Pro 转化率 ≥ 5%
- KR2 付费用户 D30 留存 ≥ 85%
- KR3 MRR ≥ ¥50,000

**O2 提升连接质量与深度**
- KR1 月有效连接数 ≥ 2,000
- KR2 连接后用户满意度 ≥ 4.2/5
- KR3 「追问」功能上线，使用率 ≥ 20%

### Q4：规模冲刺（目标 10 万用户）

**O1 达成规模目标**
- KR1 MAU 达到 50,000
- KR2 月有效连接数 ≥ 5,000
- KR3 CAC < ¥30，LTV/CAC > 3

**O2 建立平台网络效应**
- KR1 病毒系数 K ≥ 0.6
- KR2 Agent 库总量 ≥ 80,000 个
- KR3 平均每个 Topic 匹配到 5+ 个高质量 Agent

---

## 数据埋点清单

### 注册 & Agent 创建

| 事件名 | 触发时机 | 关键属性 |
|--------|---------|---------|
| `user_registered` | 完成注册 | source, channel, invite_code |
| `questionnaire_started` | 开始填写问卷 | — |
| `questionnaire_step_completed` | 完成每一步 | step_number, time_spent_sec |
| `questionnaire_abandoned` | 中途离开 | step_number, last_answer |
| `agent_preview_viewed` | 查看 Agent 预览 | time_spent_sec |
| `agent_creation_completed` | 确认 Agent | total_time_sec, fields_filled |
| `agent_edited` | 修改 Agent 背景 | fields_changed |

### Topic 提交

| 事件名 | 触发时机 | 关键属性 |
|--------|---------|---------|
| `topic_creation_started` | 点击"提交想法" | — |
| `topic_type_selected` | 选择想法类型 | type(direction/opportunity/problem) |
| `topic_draft_saved` | 自动保存草稿 | word_count |
| `topic_submitted` | 正式提交 | type, word_count, privacy_setting |
| `topic_instant_analysis_viewed` | 打开即时分析 | time_to_open_sec |
| `topic_instant_analysis_time_spent` | 阅读即时分析 | duration_sec |

### 讨论过程

| 事件名 | 触发时机 | 关键属性 |
|--------|---------|---------|
| `match_notification_sent` | 匹配完成推送发出 | matched_agent_count, channel |
| `match_notification_opened` | 用户打开通知 | time_to_open_sec |
| `quickreport_sent` | 12h 快报发出 | — |
| `quickreport_opened` | 用户打开快报 | time_to_open_sec |
| `full_report_sent` | 完整报告发出 | discussion_round_count |
| `full_report_opened` | 用户打开报告 | time_to_open_sec |
| `full_report_time_spent` | 阅读报告时长 | duration_sec, scroll_depth |
| `agent_stance_expanded` | 展开某 Agent 详细立场 | agent_background_tags |
| `recommended_agent_viewed` | 查看推荐 Agent 详情 | agent_background_tags |

### 连接行为

| 事件名 | 触发时机 | 关键属性 |
|--------|---------|---------|
| `agent_bookmarked` | 收藏 Agent | source(report/activity) |
| `connection_request_sent` | 发出连接申请 | topic_id, target_agent_background_tags |
| `connection_request_received` | 收到连接申请 | — |
| `connection_accepted` | 接受连接 | time_to_respond_sec |
| `connection_rejected` | 拒绝连接 | time_to_respond_sec |
| `contact_exchanged` | ★ 双方交换联系方式（北极星事件） | topic_type, both_user_backgrounds |

### Agent 管理

| 事件名 | 触发时机 | 关键属性 |
|--------|---------|---------|
| `agent_activity_viewed` | 查看分身动态 | — |
| `agent_reply_reviewed` | 审阅一条 Agent 发言 | — |
| `agent_reply_corrected` | 标记"不像我"并纠正 | correction_type |
| `agent_reply_approved` | 标记"说得对" | — |

---

## 数据看板结构

### 看板一：每日健康度（运营每天看）
- 当日新注册数
- 当日 Topic 提交数
- 当日有效连接数
- 当日活跃用户数

### 看板二：核心漏斗（每周复盘）
- 注册 → Agent 创建完成率
- Agent 创建 → 首次 Topic 提交率
- Topic 提交 → 报告打开率
- 报告打开 → 连接意向率（收藏/追问/申请）
- 连接意向 → 有效连接率

### 看板三：留存分析（每周）
- D1 / D7 / D30 / D90 留存曲线
- 各 Cohort 的 Topic 提交频率
- 流失节点分布（在哪一步离开最多）

### 看板四：质量监控（每周）
- Agent 纠正率
- 报告满意度评分分布
- 匹配相关性评分
- 平均每 Topic 产生连接申请数

### 看板五：增长追踪（每月）
- MAU 趋势
- 病毒系数 K 值
- 渠道 ROI 对比
- NPS 趋势
