# Product Innovation Strategist - Agent Memory

## Project: Digital Twin Community (DTC / Concors)

### Product Overview
- AI-driven social discussion platform with "digital twin" (AI Agent) concept
- Users create AI agents representing their professional background and thinking style
- Users submit topics; system matches relevant agents for 4-round structured AI discussions
- Generates analysis reports with consensus, divergence, key questions, and action items
- Users can request real-person connections based on discussion results

### Tech Stack
- Frontend: Next.js 14 + TypeScript + shadcn/ui (deployed on Vercel, domain: concors.cc)
- Backend: Go + Echo + Asynq
- DB: PostgreSQL + Redis + Qdrant (vector DB)
- LLM: Kimi (Moonshot) primary + DeepSeek backup
- Embedding: Jina embeddings-v3

### Key Product Docs
- `产品愿景.md` - Vision doc, north star
- `产品问题与优化方案.md` - 8 identified problems with solutions
- `产品命名与域名调研.md` - Naming research, chose "Concors" (Latin: like-minded)
- `Phase3-功能优先级RICE评分.md` / `Phase4-指标体系.md` / `Phase5-产品路线图.md`

### Domain & Naming
- Product name: Concors (from Latin "concors" = con + cordis = same heart)
- Domain: concors.cc (API: api.concors.cc)
- .ai domain researched but concors.ai status unclear
- Slogan candidates: "Find your concors" / "让思想连接有价值的人"

### Frontend Architecture
- Mobile-first PWA design (max-w-screen-sm)
- Bottom navigation: 首页/话题/发布/连接/分身
- All copy in Chinese
- Key user flows: register -> create agent -> submit topic -> view discussion -> view report -> request connection

### Analysis Done (2026-03-03)
- Completed full product assessment and copy audit
- See detailed analysis in conversation output
