# Digital Twin Community · Project Master Plan

> 本文档为项目唯一主文档，包含团队架构与海外增长计划。
> Target: 1,000 paid users, break-even within 12 months
> Market: English-first, indie dev / PM / founder niche
> Created: 2026-02-26

---

## Table of Contents

### Part A — Agent Team Architecture
- [A1. Architecture Overview](#a1-architecture-overview)
- [A2. Agent Roster & Skills](#a2-agent-roster--skills)
- [A3. Collaboration Workflow](#a3-collaboration-workflow)

### Part B — Overseas Growth Plan
- [B1. Assumptions & Constraints](#b1-assumptions--constraints)
- [B2. Financial Model (12-Month P&L)](#b2-financial-model)
- [B3. Phase 1: Seed (Week 1–12)](#b3-phase-1-seed-week-112--0--500-users)
- [B4. Phase 2: Growth (Month 4–6)](#b4-phase-2-growth-month-46--500--5000-users)
- [B5. Phase 3: Monetize (Month 7–9)](#b5-phase-3-monetize-month-79--5000--15000-users)
- [B6. Phase 4: Scale to 1K Paid (Month 10–12)](#b6-phase-4-scale-month-1012--15000--25000-users)
- [B7. Sensitivity Analysis & Pivot Playbook](#b7-sensitivity-analysis--pivot-playbook)
- [B8. Operational Toolchain & Cadence](#b8-operational-toolchain--cadence)

---
---

# Part A — Agent Team Architecture

> 定义项目的 AI Agent 协作团队：10 个专职 Agent + 1 个 Orchestrator。

## A1. Architecture Overview

```
┌──────────────────────────────────────────────────────────────┐
│                    Orchestrator Agent                         │
│          任务拆解 / 优先级调度 / 跨 Agent 协调               │
└───┬──────────┬──────────┬──────────┬──────────┬─────────────┘
    │          │          │          │          │
┌───▼──┐  ┌───▼──┐  ┌───▼──┐  ┌───▼──┐  ┌───▼──────┐
│Product│  │ UX   │  │ Dev  │  │ Ops  │  │  QA /    │
│Agent  │  │Agent │  │Team  │  │Team  │  │ Security │
└──────┘  └──────┘  └──────┘  └──────┘  └──────────┘
    │          │          │          │          │
    └──────────┴──────────┴──────────┴──────────┘
                    EventBus / 共享任务看板
```

### Agent 职责总表

| Agent | 本项目职责 |
|-------|-----------|
| **Orchestrator Agent** | 接收需求、任务拆解、分配给子 Agent、汇总结果、处理冲突 |
| **Product Agent** | 维护 PRD、用户故事、功能优先级（RICE）、路线图 |
| **UX Agent** | 核心流程设计、低保真线框图、可用性评估、文案审查 |
| **Backend Agent** | Go 模块开发、API 实现、数据库 Schema、LLM 集成 |
| **Architect Agent** | 技术架构决策、模块边界、性能瓶颈识别、ADR 生成 |
| **CI/CD Agent** | Cloud Build 流水线、Cloud Deploy 配置、Canary 发布 |
| **Infrastructure Agent** | GKE 配置、Cloud SQL/Memorystore、Qdrant 部署、IaC |
| **Observability Agent** | Cloud Monitoring 大盘、告警规则、Sentry 接入 |
| **QA Agent** | 测试用例生成、E2E 脚本（Playwright）、LLM 质量测试 |
| **Security Agent** | 隐私架构审查、匿名 ID 安全测试、依赖漏洞扫描 |
| **Operations Agent** | 种子用户运营、冷启动执行、增长机制设计、内容分发、留存优化、指标追踪 |

---

## A2. Agent Roster & Skills

### Orchestrator Agent

**职责：** 接收需求、任务拆解、分配给子 Agent、汇总结果、处理冲突

| Skill | 描述 |
|-------|------|
| `task-decompose` | 将产品需求拆解为可执行的子任务 DAG |
| `priority-schedule` | 基于业务优先级和依赖关系排队调度 |
| `conflict-resolve` | 处理多 Agent 并发修改同一资源的冲突 |
| `progress-report` | 生成研发进度报告，同步给 PM |
| `context-compress` | 压缩历史上下文，维护长期记忆 |

### Product Agent

**职责：** 维护 PRD、用户故事、功能优先级（RICE）、路线图

| Skill | 描述 |
|-------|------|
| `prd-maintain` | 维护产品需求文档，确保需求描述完整、可执行 |
| `user-story-write` | 编写用户故事及验收标准（Given-When-Then） |
| `feature-priority` | 使用 RICE 模型评估功能优先级，输出排序清单 |
| `roadmap-update` | 维护产品路线图，跟踪里程碑进度和 Go/No-Go 决策 |
| `stakeholder-sync` | 同步各方需求变更，管理需求变更日志 |

### UX Agent

**职责：** 核心流程设计、低保真线框图、可用性评估、文案审查

| Skill | 描述 |
|-------|------|
| `user-flow-design` | 设计核心用户流程（创建分身、提交 Topic、读报告、发起连接） |
| `wireframe-spec` | 生成低保真交互说明文档，对接开发实现 |
| `usability-eval` | 基于 Nielsen 十大可用性原则对界面方案进行评分 |
| `ux-audit` | 对已实现界面进行可用性审查，输出问题清单和优先级 |
| `a11y-check` | 无障碍合规检查（WCAG 2.1 AA 级） |
| `copy-review` | 界面文案审查（清晰度、一致性、引导性） |
| `interaction-pattern` | 推荐符合平台特性的交互模式（异步等待感、隐私渐进解锁） |

### Backend Agent

**职责：** Go 模块开发、API 实现、数据库 Schema、LLM 集成

| Skill | 描述 |
|-------|------|
| `api-scaffold` | 根据 OpenAPI Spec 生成 CRUD 接口骨架 |
| `db-schema-design` | 数据库建模（Cloud SQL），自动生成 Migration |
| `cache-strategy` | Memorystore 缓存设计（热点数据、缓存穿透防护） |
| `llm-integrate` | LLM 服务集成（Prompt 管理、流式响应、成本控制） |
| `code-review` | PR 自动审查（规范、逻辑、性能） |

### Architect Agent

**职责：** 技术架构决策、模块边界、性能瓶颈识别、ADR 生成

| Skill | 描述 |
|-------|------|
| `arch-review` | 评审设计方案，识别 SPOF（单点故障） |
| `tech-debt-scan` | 扫描代码库，量化技术债，生成偿还计划 |
| `capacity-model` | 基于用户增长预测容量需求 |
| `adr-generate` | 自动生成架构决策记录（ADR） |

### CI/CD Agent

**职责：** Cloud Build 流水线、Cloud Deploy 配置、Canary 发布

| Skill | 描述 |
|-------|------|
| `pipeline-build` | 生成/优化 Cloud Build 配置 |
| `build-cache` | 构建缓存策略，加速 CI 构建时间 |
| `artifact-publish` | 镜像构建、推送 Artifact Registry、版本打标 |
| `rollback-trigger` | 检测到发布失败自动回滚到上一个稳定版本 |
| `env-promote` | 控制 dev → staging → prod 的晋级流程 |

### Infrastructure Agent

**职责：** GKE 配置、Cloud SQL/Memorystore、Qdrant 部署、IaC

| Skill | 描述 |
|-------|------|
| `iac-generate` | 生成 Terraform 基础设施代码（GCP 资源） |
| `auto-scale` | 配置 GKE HPA/VPA，设置弹性伸缩策略 |
| `cost-optimize` | 分析 GCP 账单，推荐 Committed Use / Spot VM 方案 |
| `network-config` | VPC、子网、安全组、Cloud Load Balancing 配置 |
| `disaster-recovery` | 多可用区容灾配置，RTO/RPO 验证 |

### Observability Agent

**职责：** Cloud Monitoring 大盘、告警规则、Sentry 接入

| Skill | 描述 |
|-------|------|
| `metric-dashboard` | 自动生成 Cloud Monitoring 大盘（QPS/延迟/错误率/LLM 成本） |
| `alert-rule` | 基于 SLO 设置告警规则，减少告警噪音 |
| `log-anomaly` | Cloud Logging 异常检测，关联上下文定位根因 |
| `trace-analyze` | Cloud Trace 分布式链路分析，识别性能瓶颈 |
| `incident-response` | 告警触发后自动执行 Runbook，生成事故报告 |

### QA Agent

**职责：** 测试用例生成、E2E 脚本（Playwright）、LLM 质量测试

| Skill | 描述 |
|-------|------|
| `unit-test-gen` | 根据函数签名和逻辑自动生成单元测试 |
| `e2e-test-gen` | 生成 Playwright E2E 测试脚本 |
| `llm-quality-test` | LLM 输出质量评估（相关性、一致性、安全性） |
| `perf-test` | 生成 k6 压测脚本，分析压测报告 |
| `coverage-enforce` | 强制覆盖率门禁，阻断低质量 PR 合并 |

### Security Agent

**职责：** 隐私架构审查、匿名 ID 安全测试、依赖漏洞扫描

| Skill | 描述 |
|-------|------|
| `privacy-review` | 隐私架构审查，确保分身数据匿名化合规 |
| `anon-id-test` | 匿名 ID 体系安全测试，防止身份反推 |
| `dependency-audit` | 第三方依赖 CVE 扫描，自动升级补丁版本 |
| `sast-scan` | 静态代码安全扫描（OWASP Top 10） |
| `compliance-check` | 数据合规检查（GDPR / 隐私政策要求） |

### Operations Agent

**职责：** 种子用户运营、冷启动执行、增长机制设计、内容分发、留存优化、指标追踪

| Skill | 描述 |
|-------|------|
| `seed-user-recruit` | 种子用户招募与培育：筛选垂直圈子目标用户（独立开发者/PM），手动辅助创建高质量 Agent，确保 30 个种子 Agent 背景组合的行业代表性 |
| `content-distribute` | 外部内容分发：将讨论精华摘要脱敏后发布到社区（Indie Hackers / Hacker News / Twitter），追踪各渠道用户 Agent 创建完成率 vs 平均值差异 |
| `referral-program` | 邀请增长机制：设计和运营用户邀请体系（邀请→双方获额外 Topic 配额），追踪病毒系数 K 值（目标 Q2 ≥ 0.3 → Q4 ≥ 0.6），优化邀请转化漏斗 |
| `retention-lifecycle` | 用户留存生命周期管理：配置 T+0/1h/12h/48h 四阶段触达策略，运营周报摘要推送（Agent 动态汇总），识别流失节点并设计召回干预 |
| `growth-dashboard` | 增长指标看板运营：维护 AARRR 五层漏斗（获取→激活→留存→收入→推荐），每日监控 DAU/Topic 提交/有效连接，每月输出增长追踪报告（MAU 趋势、K 值、渠道 ROI、NPS） |
| `cold-start-playbook` | 冷启动 Playbook 执行：按三阶段策略推进（Phase 1 邀请制垂直起步 → Phase 2 社区联动引流 → Phase 3 推荐自增长），管理每阶段 Go/No-Go 决策点 |
| `community-ops` | 社区与 KOL 运营：建立外部社区合作关系，Q4 阶段执行 KOL/创业者圈层定向运营，运营"话题广场"公开内容，驱动内容传播飞轮 |

**Operations Agent Skills 设计依据：**

| Skill | 对应产品文档来源 |
|-------|-----------------|
| `seed-user-recruit` | 产品问题与优化方案 · 问题三（冷启动 Phase 1）|
| `content-distribute` | Phase5 路线图 · Q2 月 5（外部内容分发）+ Phase4 · 假设验证 1 |
| `referral-program` | Phase5 路线图 · Q2 月 5（邀请机制）+ Phase4 · 推荐层指标 |
| `retention-lifecycle` | 产品问题与优化方案 · 问题四（价值感知延迟）+ Phase5 · Q2 月 6（留存优化）|
| `growth-dashboard` | Phase4 指标体系 · 五大看板 + AARRR + OKR |
| `cold-start-playbook` | 产品问题与优化方案 · 问题三（三阶段破局）+ Phase5 · Q1 Go/No-Go |
| `community-ops` | Phase5 路线图 · Q4 月 10-11（KOL 运营 + 话题广场）|

---

## A3. Collaboration Workflow

```
Product Agent
  └─ feature-priority → 确认功能 P0/P1/P2

Orchestrator
  └─ task-decompose → 拆分为：API + 前端 + DB + 测试 + 基础设施

并行执行：
  UX Agent (wireframe-spec)     → 交互说明文档
  Backend Agent (api-scaffold)  → API 骨架 + DB Migration
  Architect Agent (arch-review) → 方案评审，识别 SPOF

Backend Agent 开发完成 → PR 提交

并行门禁：
  QA Agent (unit-test-gen)      → 自动生成单元测试
  Security Agent (sast-scan)    → 静态安全扫描
  UX Agent (ux-audit)           → 界面可用性检查

通过门禁 → CI/CD Agent
  └─ pipeline-build → Cloud Build 构建
  └─ env-promote → Staging 部署

QA Agent (e2e-test-gen + smoke-test) → E2E 验证通过

CI/CD Agent
  └─ canary-deploy → 10% 流量 → Observability Agent 监控
  └─ 15min 无异常 → 全量放量

Observability Agent
  └─ metric-dashboard → 实时监控 QPS/错误率/LLM 成本
  └─ alert-rule → 异常自动触发 Runbook
```

---
---

# Part B — Overseas Growth Plan

> Status: Just launched, zero users
> Target: 1,000 paid users → break-even

## B1. Assumptions & Constraints

### Hard Constraints

| Constraint | Value | Implication |
|-----------|-------|-------------|
| Team size | 2 backend + 1 frontend + 1 product/growth (founder) | Founder does growth full-time from Day 1 |
| Funding runway | ~$50K pre-revenue budget | Must reach break-even before Month 12 or raise |
| LLM model | Claude Haiku for routine, Sonnet for reports | Cost control is survival constraint |
| Infrastructure | GCP, start minimal → scale with users | No over-provisioning |

### Key Assumptions (to be validated)

| # | Assumption | Validation Method | Deadline |
|---|-----------|-------------------|----------|
| A1 | Indie devs will create AI Agents representing themselves | Seed user completion rate | Week 6 |
| A2 | AI-to-AI discussion produces insights users value | Report satisfaction score | Week 8 |
| A3 | Users will connect with strangers based on Agent discussions | First effective connection | Week 10 |
| A4 | Users will pay $12/mo for unlimited Topics + priority matching | Founding member pre-sales | Month 5 |
| A5 | Content from discussions can drive organic traffic | First blog post conversion rate | Month 4 |

**If A1-A3 fail by Week 12 → pause growth, pivot product mechanics.**

---

## B2. Financial Model

### 12-Month P&L Projection (Monthly Granularity)

```
              M1     M2     M3     M4     M5     M6     M7     M8     M9     M10    M11    M12
─────────────────────────────────────────────────────────────────────────────────────────────────
USERS
 New signups   15    85    200    400    600   1200   1800   2000   2500   2800   3000   3400
 Cumulative    15   100    300    700   1300   2500   4300   6300   8800  11600  14600  18000
 MAU           15    80    200    450    900   1800   3000   4200   5500   7000   8500  10000
 Paid users     0     0      0      0    30*    80*    200    350    500    650    830   1000
                                        (* = founding member pre-sales)

REVENUE ($)
 MRR            0     0      0      0    270    720   2400   4200   6000   7800   9960  12000
 Cumulative     0     0      0      0    270    990   3390   7590  13590  21390  31350  43350

COSTS ($)
 LLM API       50   200    500    900   1500   2500   3500   4500   5500   6000   6500   7000
 Infra        200   200    300    400    500    700    900   1100   1300   1500   1700   2000
 SaaS tools   100   100    100    150    150    200    250    250    300    300    300    300
 Paid acq.      0     0      0      0      0      0      0    500   1000   1500   2000   2000
 Total        350   500    900   1450   2150   3400   4650   6350   8100   9300  10500  11300

NET ($)      -350  -500   -900  -1450  -1880  -2680  -2250  -2150  -2100  -1500   -540   +700
Cum. burn    -350  -850  -1750  -3200  -5080  -7760 -10010 -12160 -14260 -15760 -16300 -15600
─────────────────────────────────────────────────────────────────────────────────────────────────
Total cash needed before break-even: ~$16,300
Break-even month: Month 12 (MRR $12K > costs $11.3K)
```

### Pricing Tiers

```
Free tier:
  - 1 Agent
  - 1 Topic/month
  - 48h report delivery
  - 2 connections/month

Pro tier ($12/mo or $108/yr):
  - Unlimited Topics
  - Priority matching (Pro Agents matched first)
  - 12h fast reports
  - Unlimited connections
  - Weekly Agent activity digest
  - "Pro" badge on Agent profile

Team tier (future, $29/mo per seat):
  - Shared Agent library
  - Team topic boards
  - Collaboration features
```

### LLM Cost Control (Survival-Critical)

| Tactic | Savings | When |
|--------|---------|------|
| **Haiku for Agent chat**, Sonnet only for final reports | -40% vs all-Sonnet | Day 1 |
| **Cache Agent personality prompts** — same user = reuse system prompt across Topics | -15% | Day 1 |
| **Cap discussion to 4 rounds** for free tier, 6 for Pro | -20% free tier cost | Month 7 |
| **Batch discussions** — run all Agent turns in parallel, not sequential | Faster + cheaper (fewer round trips) | Day 1 |
| **Tier-based model routing**: routine role-play → Haiku; final synthesis → Sonnet | -25% overall | Month 3 |
| **Per-user monthly cost cap**: if a free user exceeds $0.50 LLM cost, queue their Topics | Prevents whale cost | Month 4 |

**Unit economics:**
```
Free user:   $0.30/mo LLM cost (1 Topic × 5 agents × 4 rounds × Haiku)
Pro user:    $2.00/mo LLM cost (avg 3 Topics × 5 agents × 6 rounds × mixed)
Blended at 1K paid + 9K free active: ($0.30 × 9000 + $2.00 × 1000) / 10000 = $0.47/user
Total LLM at 10K MAU: $4,700/mo → model shows $7K (includes buffer for spikes)
```

---

## B3. Phase 1: Seed (Week 1–12) → 0 → 500 Users

### Phase Goals

| Goal | Metric | Target |
|------|--------|--------|
| Validate core value loop | First effective connection | By Week 10 |
| Prove Agent creation works | Completion rate | ≥ 70% |
| Prove discussion quality | Report satisfaction | ≥ 3.5/5 |
| Build seed community | Active users | 300 MAU by Week 12 |

### Week 1–2: Pre-Launch Prep

**Goal:** Prepare all growth assets before inviting anyone

| Day | Task | Owner | Output |
|-----|------|-------|--------|
| D1 | Set up landing page (Framer/Next.js) with waitlist | Frontend | Live URL + Tally waitlist form |
| D1 | Create founder Twitter/X account, write pinned thread explaining the product | Founder | 1 thread, 8+ tweets |
| D2 | Write "Why I'm building this" post | Founder | Indie Hackers draft |
| D3 | Identify 100 target seed users: search Twitter for #buildinpublic, Indie Hackers profiles, HN Show HN posters who match criteria | Founder | Spreadsheet: name, handle, background, relevance score (1-5) |
| D4-5 | Craft personalized DM templates (3 variants by segment: dev, PM, founder) | Founder | 3 DM templates |
| D6-7 | Send first 30 DMs to top-scored targets | Founder | 30 DMs sent |
| D8-10 | Follow up non-responders, engage with their content first (like, reply) before re-DMing | Founder | Build rapport |
| D11-14 | Product Hunt "Coming Soon" page live | Founder | PH page URL |

**DM Template (Indie Developer variant):**
```
Hey [name] — I've been following your work on [project].

I'm building something I think you'd find interesting: an AI "digital twin"
that represents your background/expertise and joins async discussions with
other people's twins. The idea is your twin networks for you — finds
relevant people, discusses ideas, and only surfaces connections worth your time.

We're inviting 30 builders for early access. Would you be interested?
I'd personally help set up your twin and would love your feedback.
```

**Expected:** 30 DMs → 15 replies → 10 confirmed seed users

**If <5 confirm:** Value prop messaging is wrong. Rewrite DMs, try different segments.

---

### Week 3–4: First 30 Seeds Onboarding

**Goal:** 30 active Agents, first discussion cycles running

| Task | Detail | Success Metric |
|------|--------|---------------|
| **1:1 onboarding calls** | 15-min video call with each seed user. Walk through Agent creation, submit their first Topic together. Record common friction points. | 30 Agents created |
| **Curate Topic seeding** | If users don't submit Topics, you submit 5 interesting ones (sourced from HN/Twitter trends). Assign them to the pool. | ≥ 10 active Topics |
| **Monitor first discussions** | Read every single AI discussion output. Flag quality issues immediately for backend team. | Quality log spreadsheet |
| **Daily Discord check-in** | Create private Discord for seeds. Post daily: "Here's what happened in discussions today" | Discord active |
| **Collect feedback** | After first report delivered, send 3-question survey: (1) Was this useful? (2) Was your Agent accurate? (3) Would you connect with anyone? | ≥ 20 responses |

**Daily founder routine (Week 3-4):**
```
09:00  Check overnight discussion outputs, flag quality issues
09:30  Reply to all seed user messages in Discord
10:00  Onboarding call #1
10:30  Onboarding call #2
11:00  Write today's "community digest" for Discord
14:00  Review + share any interesting discussion highlights on Twitter
15:00  Send 10 more DMs to waitlist / new targets
16:00  Sync with backend on quality issues
```

**Critical quality gate (end of Week 4):**
```
□ 30 Agents created                           → if <20, onboarding flow is broken
□ ≥ 10 Topics with completed discussions       → if <5, matching/discussion engine broken
□ ≥ 15 reports delivered                       → if <10, report pipeline broken
□ Report satisfaction ≥ 3.0/5                  → if <3.0, LLM quality needs urgent work
□ ≥ 3 users say "I'd want to connect"         → if 0, core value hypothesis failing
```

**If failing:** Do NOT proceed to Week 5. Spend Week 5-6 fixing the broken step. Growth on broken product = wasted effort.

---

### Week 5–6: Expand to 100 Users

**Goal:** Open invite-only access beyond hand-held onboarding

| Task | Detail |
|------|--------|
| **Seed users invite 3 friends each** | Ask each active seed: "Know 2-3 people who'd find this valuable?" Provide personal invite links |
| **Publish "Building in Public" Post #1** | Indie Hackers: "I got 30 people to create AI twins of themselves — here's what happened" with real (anonymized) discussion excerpts |
| **Twitter thread #1** | "I built a platform where your AI twin discusses ideas with other people's AI twins. Here's the wildest discussion so far:" (screenshot + link) |
| **Open waitlist drip** | Start letting waitlist signups in, 10/day, monitor activation rate |
| **Self-serve onboarding** | Remove 1:1 call requirement. Add in-product tooltips + 2-minute video guide |

**Expected:** 30 seeds × 2 invites = 60 + 20 from IH post + 20 from waitlist = 100 users

**Daily tracking:**
```
New signups today:        ___
Agent creation started:   ___
Agent creation completed: ___ (completion rate: ___%)
Topics submitted today:   ___
Reports delivered today:  ___
Connections requested:    ___
```

---

### Week 7–8: Product Hunt Launch

**Goal:** Big spike to 300+ users

**Preparation checklist (Week 6, in parallel):**

| Asset | Spec | Status |
|-------|------|--------|
| Demo video | 45-sec, show: create Agent → submit Topic → get report → see connection recommendation | [ ] |
| Screenshots | 5 screens: Agent creation, discussion in progress, report, connection flow, Agent profile | [ ] |
| Tagline | "Your AI twin networks while you sleep" | [ ] |
| Description | 3 paragraphs: problem, solution, how it works | [ ] |
| First comment | Founder story: why built this, what surprised us, what's next | [ ] |
| Hunter | Recruit a known PH hunter (reach out Week 5) or self-hunt | [ ] |
| Upvote base | Ask 100 existing users + friends to upvote at 12:01 AM PST | [ ] |

**Launch day protocol:**
```
00:01 PST  Launch goes live (Tuesday or Wednesday — best days)
00:05      Post first comment (founder story)
00:30      Tweet announcement, tag relevant people
01:00      Post to Discord: "We're live on PH, would appreciate your support"
06:00      Reply to every single PH comment within 1 hour
12:00      Mid-day Twitter update: "We're #X on Product Hunt!"
18:00      Share any interesting user stories from the day
24:00      Publish results thread on Twitter regardless of outcome
```

**Realistic PH outcomes:**
```
Scenario A (Top 5):   400-800 signups, 200+ upvotes
Scenario B (Top 10):  150-400 signups, 100+ upvotes
Scenario C (Top 20):  50-150 signups
Scenario D (Flop):    <50 signups → still valuable for the backlink + credibility
```

**Post-PH activation sprint (Week 8):**
- All PH signups get email sequence: Day 0 welcome → Day 1 "create your Agent" → Day 3 "submit your first idea" → Day 7 "here's what your Agent has been up to"
- Monitor PH cohort activation separately: what % create Agent? What % submit Topic?
- If PH activation < 50%, the onboarding flow needs work (too much friction for cold traffic)

---

### Week 9–10: First Connection Milestone

**Goal:** 10+ effective connections — the moment the product is "real"

| Action | Purpose |
|--------|---------|
| Manually review all discussion reports, identify highest-potential connections | Ensure best matches aren't missed |
| Reach out to users who received good reports: "Did you see the connection recommendation?" | Push through the funnel |
| If connections are happening → screenshot (with permission), share on Twitter | Social proof |
| If connections NOT happening → user interviews: what's blocking? | Diagnose and fix |

**Connection failure diagnosis tree:**
```
Users not reading reports?
  → Push notification / email not working → fix delivery
  → Reports not interesting → LLM quality issue → fix prompts

Users reading but not requesting connection?
  → Don't see value in connecting → report doesn't convey why
  → Scared to connect with stranger → trust/safety issue
  → Connection flow too clunky → simplify UI

Users requesting but other side not accepting?
  → Notification not reaching recipient → fix delivery
  → Request doesn't explain context well → improve request template
  → Recipients not engaged → re-engagement campaign
```

---

### Week 11–12: Consolidate & Go/No-Go

**Goal:** Reach 300+ MAU, complete Go/No-Go assessment

**Actions:**
- Run 20 user interviews (mix of active, churned, never-activated)
- Calculate all metrics for Go/No-Go
- Write internal "Phase 1 retrospective" document
- Identify top 3 product issues to fix before Phase 2

**Go/No-Go Decision (Week 12) — HARD GATE:**

| Criterion | Target | Actual | Pass? |
|-----------|--------|--------|-------|
| Registered users | ≥ 300 | ___ | |
| Agent creation completion rate | ≥ 60% | ___ | |
| Topics with completed discussions | ≥ 100 | ___ | |
| Report satisfaction score | ≥ 3.5/5 | ___ | |
| Effective connections (total) | ≥ 10 | ___ | |
| Users who'd recommend (from interviews) | ≥ 50% | ___ | |

**Decision rules:**
```
6/6 pass  → Full speed Phase 2
4-5 pass  → Phase 2 with reduced scope, fix failing metrics first
2-3 pass  → Extend Phase 1 by 4 weeks, deep-fix issues
0-1 pass  → Serious pivot discussion. Product-market fit not found.
```

---

## B4. Phase 2: Growth (Month 4–6) → 500 → 5,000 Users

### Month 4: Fix & Polish (No Growth Push Yet)

**Rule: Do not scale what isn't working.**

| Week | Focus | Actions |
|------|-------|---------|
| W13 | Fix top 3 issues from Phase 1 retro | Backend/frontend sprint |
| W14 | Onboarding optimization | Reduce Agent creation to <3 min. A/B test question flow. Target: completion rate ≥ 75% |
| W15 | Report quality overhaul | Rewrite LLM prompts based on user feedback. Implement satisfaction rating in-product. Target: ≥ 4.0/5 |
| W16 | Email/notification tuning | Implement T+0/1h/12h/48h notification sequence. Measure open rates. Target: report open rate ≥ 60% |

**Metric checkpoints (end of Month 4):**
```
□ Agent creation completion: ≥ 75% (was ≥ 60% in Phase 1)
□ Report satisfaction: ≥ 4.0/5
□ Report open rate: ≥ 60%
□ D7 retention: ≥ 35%
□ Monthly connections per 100 MAU: ≥ 5
```

### Month 5: Turn on Growth Channels (One at a Time)

**Important: Test channels sequentially, not simultaneously.** You need to measure each channel's contribution cleanly.

**Week 17-18: Content marketing test**

| Content Piece | Channel | Effort | Expected Signups |
|--------------|---------|--------|-----------------|
| "What 5 AI agents said about [trending topic]" thread | Twitter/X | 2h | 30-50 |
| "We built AI digital twins for 300 people — here's what surprised us" | Indie Hackers | 4h | 80-150 |
| Same content adapted | Hacker News | 1h | 50-200 (high variance) |

**Per-channel full-funnel tracking:**
```
Impressions → Clicks → Signups → Agent Created → Topic Submitted → Report Read

If signups are high but activation is low → that channel brings low-intent users → deprioritize.
```

**Week 19-20: Referral program launch**
```
Referral v1 (simple):
  User A shares personal invite link
  User B signs up + creates Agent
  → User A gets +1 bonus Topic this month
  → User B gets +1 bonus Topic this month

Measure:
  - % of users who share their link (target: ≥ 15%)
  - Invite → signup conversion (target: ≥ 25%)
  - K-value = share_rate × invites_per_sharer × conversion = 0.15 × 3 × 0.25 = 0.11
```

**Week 21: Founding Member pre-sale**
```
Email to all active users:

  Subject: "You're invited: Founding Member pricing — locked forever"

  Body:
  - We're launching Pro in 2 months
  - As an early user, you can lock in $89/year ($7.42/mo) forever
  - Pro will be $144/yr ($12/mo) at launch
  - Founding members also get: [badge], [priority matching], [early features]
  - Limited to first 100 members

Target: 30-50 pre-paid users (validates willingness to pay)
If <10 buy: pricing or value prop is wrong. Interview non-buyers. Adjust before Month 7.
```

### Month 6: Double Down on Winners

**By now you know which channels work.** Example scenario:
```
Channel performance:
  Twitter/X threads:    150 signups/post, 45% activation → WINNER
  Indie Hackers posts:  100 signups/post, 55% activation → WINNER
  Hacker News:          200 signups once, 30% activation → SPIKE but unreliable
  Reddit:               40 signups/post, 25% activation → NOT WORTH IT
  Referral:             K=0.15 → NEEDS OPTIMIZATION
```

**Actions:**
- 2× frequency on winning channels
- Kill underperforming channels
- Referral optimization: test different incentives, simplify sharing UX
- Start building SEO content library (10 blog posts, targeting long-tail keywords)

**Month 6 targets:**
```
□ 2,500 cumulative users
□ 1,800 MAU
□ 50-80 founding members pre-paid
□ K-value ≥ 0.2
□ At least 1 channel producing ≥ 200 signups/month reliably
□ D30 retention ≥ 25%
```

---

## B5. Phase 3: Monetize (Month 7–9) → 5,000 → 15,000 Users

### Month 7: Pro Launch

**Paywall UX — when users hit the wall:**
```
Page: "Submit a Topic"
  ↓
User already submitted 1 Topic this month
  ↓
Modal:
  ┌────────────────────────────────────────────┐
  │  You've used your free Topic this month    │
  │                                            │
  │  ● Upgrade to Pro — $12/mo                │
  │    Unlimited Topics, priority matching,    │
  │    12h fast reports                        │
  │                                            │
  │    [Upgrade to Pro]  [Maybe later]         │
  │                                            │
  │  Your next free Topic refreshes in 23 days │
  └────────────────────────────────────────────┘
```

**"Maybe later" is important.** Don't trap users. They'll convert when ready.

**Conversion monitoring (daily):**
```
Free users who hit paywall today:           ___
  → Converted immediately:                  ___ (instant conversion rate: ___%)
  → Clicked "Maybe later":                  ___
  → Converted within 7 days:                ___ (delayed conversion rate: ___%)
  → Never converted:                        ___

Target: 8-12% of users who hit paywall convert within 30 days.
```

**If <5%:** perceived value gap between free and Pro isn't big enough. Options:
- Add more Pro-exclusive features
- Show a "preview" of what Pro matching looks like (FOMO)
- Lower price to $9/mo for first 3 months
- Test $8/mo permanently

**Pro user cohort analysis (Week 3-4):**
```
Paid → still active Day 7:    ___% (target: ≥ 90%)
Paid → still active Day 30:   ___% (target: ≥ 85%)
Paid → submitted 2+ Topics:   ___% (target: ≥ 70%)
Paid → made a connection:     ___% (target: ≥ 40%)

Monthly churn target: ≤ 8% (= ≥ 92% monthly retention)
```

### Month 8: Conversion Optimization Sprint

| Experiment | Hypothesis | Metric | Duration |
|-----------|-----------|--------|----------|
| A/B: Paywall at Topic #2 vs at report blur | Blurring reports creates more FOMO | Conversion rate | 2 weeks |
| A/B: $12/mo vs $10/mo vs $15/mo | Find optimal price point | Revenue per 1000 users | 2 weeks |
| Pro badge visibility | Pro badge on Agents creates social proof | Upgrade rate among users who see Pro Agents | 2 weeks |
| Upgrade nudge email | Email when free user's Agent is matched but can't join | Email → upgrade conversion | 2 weeks |

### Month 9: Retention & Churn Prevention

```
Churn risk signals (automated detection):

  Level 1 (Yellow): No login in 7 days
    → Email "Your Agent joined 2 new discussions this week — see what happened"

  Level 2 (Orange): No login in 14 days + no Topic submitted
    → Email "We miss you" + offer 1 bonus free Topic

  Level 3 (Red): Pro user, no activity in 21 days
    → Personal email from founder. "Is everything okay? Would love your feedback."

  Level 4 (Critical): Cancellation initiated
    → Exit survey (mandatory, 3 questions) + offer 50% off next month
```

**Month 9 financial checkpoint:**
```
Pro users:         500
MRR:               $6,000
Monthly costs:     $8,100
Monthly net:       -$2,100

Critical question: Can we reach 1K paid in 3 months?
  Current growth rate: ~100 new paid/month
  Need: ~170 new paid/month in M10-12
  Gap:  70/month → need to increase paid acquisition or conversion rate
```

---

## B6. Phase 4: Scale (Month 10–12) → 15,000 → 25,000 Users

### Growth Levers to Close the Gap

| Lever | Current | Target | How |
|-------|---------|--------|-----|
| Organic signups/month | 1,500 | 2,500 | SEO content engine + 2nd PH launch |
| Referral signups/month | 300 | 800 | Referral v2: Pro users invite friends to 2-week free Pro trial |
| Paid acquisition/month | 0 | 500 | Twitter/X ads, $2K/mo budget, target CAC ≤ $4 |
| Free → Pro conversion | 8% | 12% | Paywall optimization from M8 experiments |
| Monthly churn | 7% | 5% | Churn prevention system from M9 |

### Month 10: SEO + Paid Acquisition

**SEO content engine:**
```
Week 1: Publish 5 blog posts from top anonymized discussions
  Target keywords:
  - "find a technical co-founder"
  - "async networking for developers"
  - "AI-powered professional networking"
  - "indie developer networking"
  - "startup idea feedback tool"

Week 2-4: Publish 3 posts/week (total 15 posts this month)
  Content source: Every good discussion report → anonymize → blog post
  Post template:
    Title: "What 5 AI experts think about [topic]"
    Body: Discussion highlights with different perspectives
    CTA: "Want your AI twin to join discussions like this? Create yours free →"
```

**Paid acquisition (start small):**
```
Twitter/X ads:
  Budget: $500/week = $2,000/month
  Targeting: followers of @IndieHackers, @levelsio, @marckohlbrugge, #buildinpublic
  Creative: 30-sec video demo
  Expected: 500 signups at $4 CAC

  Rule: If CAC > $6 after 2 weeks → pause, optimize creative
  Rule: If CAC < $3 → double budget
```

### Month 11: KOL + Partnerships

**Micro-influencer program:**
```
Target: 10 creators with 5K-50K followers in indie dev / startup space
Offer: Free Pro for life + $100 per video/thread that drives ≥ 50 signups
Budget: $1,000-2,000 total (performance-based)
Expected: 5 accept → 3 publish → 1,000 signups → 80 paid
```

**Newsletter partnerships:**
```
Target newsletters:
  - Indie Hackers newsletter (sponsor slot ~$500)
  - TLDR newsletter (sponsor slot ~$1,000)
  - Founder-focused Substacks (free exchange: they try product, write about it)
Expected: 1,000 signups from newsletter placements
```

### Month 12: Final Push to 1K Paid

```
Week 1: "Year-end special" — $99/year (17% off annual) for 2 weeks only
Week 2: Email all free users who hit paywall 2+ times: personal pitch
Week 3: Referral contest — top 3 referrers get lifetime Pro
Week 4: Audit and optimize every step of the paid conversion funnel
```

**Final scorecard:**
```
Total registered:    18,000 (±3,000)
MAU:                 10,000 (±2,000)
Paid users:          1,000 (target)
MRR:                 $12,000
Monthly costs:       $11,300
Net:                 +$700/mo → break-even ✅

If ahead of target:  Raise prices to $15/mo for new users (grandfather existing)
If behind (800 paid): Extend timeline by 2 months, cut paid acquisition costs
If far behind (500):  Reassess pricing, consider $8/mo to increase volume
```

---

## B7. Sensitivity Analysis & Pivot Playbook

### Scenario: Low Activation (Agent creation < 50%)

```
Diagnosis:
  - Questionnaire too long → test 3-question version
  - Users don't understand what an "Agent" is → rewrite onboarding copy
  - Users feel creepy about AI representing them → add more control/preview

Pivot actions:
  Week 1: Run 10 user interviews specifically about the creation flow
  Week 2: A/B test simplified flow (3 questions vs current)
  Week 3: Implement winner
  Week 4: Re-measure. If still < 50%, consider "auto-create Agent from LinkedIn" flow.
```

### Scenario: Low Discussion Quality (satisfaction < 3.0/5)

```
Diagnosis:
  - Agents sound too generic → need more personality injection
  - Discussions too agreeable → strengthen "challenger" role prompt
  - Reports too long / boring → redesign report format

Pivot actions:
  - Immediately: Founder reads every report, scores them, identifies patterns
  - Week 1: Rewrite all role prompts (challenger, supporter, questioner)
  - Week 2: Test "human-in-the-loop" — founder edits 50 Agent responses to train
  - Week 3: Deploy improved prompts, A/B test vs old
  - Target: 4.0/5 within 3 weeks
```

### Scenario: Nobody Wants to Pay (founding members < 10)

```
Diagnosis:
  - Users love free but don't value enough to pay → feature gap
  - Price too high → test $8/mo, $5/mo
  - Timing wrong → users haven't experienced enough value yet

Pivot actions:
  - Survey: "What would make this worth $12/mo to you?"
  - Test pricing: offer $5/mo to 100 users, measure take rate
  - Add value: priority matching is killer feature — make it dramatically better for Pro
  - Consider usage-based: $3/Topic instead of subscription
  - Nuclear option: pivot to B2B (team/company plans) if consumer won't pay
```

### Scenario: Growth Stalls at 2K Users

```
Diagnosis:
  - Organic channels exhausted
  - Referral not working (K < 0.1)
  - No product-market fit for growth

Pivot actions:
  - Segment analysis: which user type is most active? Double down on that niche.
  - Geography: try Japan, Korea indie dev scene
  - Product pivot: add "public Agent profiles" as standalone value
  - Distribution pivot: integrate with existing communities (Slack/Discord bot)
```

### Decision Timeline

```
Week 6:   Can users create Agents?                     YES → continue  NO → fix onboarding
Week 8:   Are discussions valuable?                     YES → continue  NO → fix LLM quality
Week 10:  Are connections happening?                    YES → continue  NO → fix connection UX
Week 12:  Go/No-Go gate                                PASS → Phase 2  FAIL → extend/pivot
Month 5:  Will users pay? (founding members)            YES → continue  NO → reprice/repackage
Month 7:  Pro launch conversion ≥ 5%?                   YES → continue  NO → optimize or lower price
Month 9:  On track for 1K paid? (≥ 500 paid)           YES → continue  NO → reassess timeline
Month 12: 1K paid achieved?                             YES → done      NO → extend 2 months
```

---

## B8. Operational Toolchain & Cadence

### Tools Budget (Month 1 → Month 7)

| Tool | Purpose | M1 Cost | M7 Cost |
|------|---------|---------|---------|
| Vercel | Frontend hosting | $0 | $20 |
| GCP | Backend + DB | $200 | $900 |
| Anthropic API | LLM | $50 | $3,500 |
| Resend | Transactional email | $0 | $20 |
| PostHog | Analytics + A/B testing | $0 | $0-150 |
| Sentry | Error tracking | $0 | $0 |
| Stripe | Payments | — | 2.9%+30¢ |
| Customer.io | Lifecycle email | — | $100 |
| Discord | Community | $0 | $0 |
| Typefully | Social scheduling | $0 | $10 |
| **Total** | | **~$250** | **~$4,700** |

### Founder Weekly Cadence

**Every Monday (1 hour):**
```
□ Review last week's metrics (signups, activation, retention, MRR)
□ Update tracking spreadsheet
□ Identify biggest drop-off in funnel → that's this week's priority
□ Plan content for the week (2 Twitter threads + 1 long-form post)
```

**Every Friday (30 min):**
```
□ Read all user feedback received this week
□ Tag feedback: [onboarding] [quality] [pricing] [feature-request] [bug]
□ Pick top 1 feedback item → create ticket for next week
□ Write weekend Twitter thread summarizing the week (building in public)
```

**Monthly (2 hours):**
```
□ Full metrics review against this plan
□ Update financial model with actuals
□ Recalculate runway
□ Adjust next month's targets if needed
□ Write monthly retrospective (internal)
```

---

## 12-Month Summary

```
Month 1-3   Seed & Validate     500 users      $0 MRR        -$1.5K/mo
Month 4-6   Growth Engine       5K users       $1.2K MRR     -$2.3K/mo
Month 7-9   Monetize            15K users      $6K MRR       approaching break-even
Month 10-12 Scale               18K+ users     $12K MRR      +$700/mo ✅

→ 1,000 paid users
→ Break-even
→ Total pre-revenue investment: ~$16,300
→ Ready for sustainable growth or Series A
```
