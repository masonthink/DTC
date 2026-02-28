-- =============================================================================
-- 数字分身社区 · 初始数据库 Schema
-- Migration: 001_initial_schema
-- =============================================================================

-- 启用扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- Schema 定义
-- =============================================================================
-- 主业务数据在 public schema
-- 匿名 ID 映射在独立 anon schema（最高安全）
CREATE SCHEMA IF NOT EXISTS anon;

-- =============================================================================
-- ENUM 类型
-- =============================================================================
CREATE TYPE user_status AS ENUM ('active', 'suspended', 'deleted');
CREATE TYPE agent_type AS ENUM ('professional', 'entrepreneur', 'investor', 'generalist');
CREATE TYPE topic_status AS ENUM (
    'pending_matching',
    'matching',
    'matched',
    'discussion_active',
    'report_generating',
    'completed',
    'failed',
    'cancelled'
);
CREATE TYPE discussion_status AS ENUM (
    'pending_matching',
    'round_1_queued', 'round_1_running', 'round_1_completed',
    'round_2_queued', 'round_2_running', 'round_2_completed',
    'round_3_queued', 'round_3_running', 'round_3_completed',
    'round_4_queued', 'round_4_running', 'round_4_completed',
    'report_generating',
    'completed',
    'degraded',
    'failed'
);
CREATE TYPE discussion_role AS ENUM ('questioner', 'supporter', 'supplementer', 'inquirer');
CREATE TYPE connection_status AS ENUM ('pending', 'accepted', 'rejected', 'cancelled', 'expired');
CREATE TYPE notification_type AS ENUM (
    'match_preview',      -- T+1h 匹配预告
    'discussion_update',  -- T+12h 讨论快报
    'report_ready',       -- T+48h 完整报告
    'connection_request', -- 连接申请
    'connection_accepted' -- 连接接受
);
CREATE TYPE notification_channel AS ENUM ('fcm', 'email', 'in_app');
CREATE TYPE notification_status AS ENUM ('pending', 'sent', 'failed', 'skipped');
CREATE TYPE llm_provider AS ENUM ('anthropic', 'openai', 'deepseek');

-- =============================================================================
-- users 表
-- =============================================================================
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone           VARCHAR(20) UNIQUE,
    email           VARCHAR(255) UNIQUE,
    display_name    VARCHAR(100) NOT NULL,
    avatar_url      TEXT,
    status          user_status NOT NULL DEFAULT 'active',
    fcm_token       TEXT,                    -- Firebase Cloud Messaging token
    timezone        VARCHAR(50) DEFAULT 'Asia/Shanghai',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at  TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT users_contact_check CHECK (phone IS NOT NULL OR email IS NOT NULL)
);

CREATE INDEX idx_users_phone ON users(phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_last_active ON users(last_active_at DESC);

-- =============================================================================
-- agents 表（数字分身）
-- =============================================================================
CREATE TABLE agents (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_type      agent_type NOT NULL DEFAULT 'professional',
    display_name    VARCHAR(100),            -- 分身显示名（可选，用于界面）

    -- 问卷数据 (JSONB 存储完整问卷答案)
    questionnaire   JSONB NOT NULL DEFAULT '{}',

    -- 标签化字段（冗余存储，用于过滤和匹配）
    industries      TEXT[] NOT NULL DEFAULT '{}',
    skills          TEXT[] NOT NULL DEFAULT '{}',
    thinking_style  JSONB NOT NULL DEFAULT '{}',  -- {analytical, creative, critical, collaborative}
    experience_years INT DEFAULT 0,

    -- 向量相关
    anon_id         VARCHAR(20) UNIQUE NOT NULL,   -- 匿名 ID: AGT-XXXXXXXX
    qdrant_point_id UUID,                          -- Qdrant 向量点 ID
    embedding_updated_at TIMESTAMPTZ,

    -- 质量评分
    quality_score   DECIMAL(3,2) DEFAULT 0.00,     -- 0-5.00
    discussion_count INT DEFAULT 0,
    connection_count INT DEFAULT 0,

    -- 活跃状态
    is_active       BOOLEAN NOT NULL DEFAULT true,
    last_active_at  TIMESTAMPTZ,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agents_user_id ON agents(user_id);
CREATE INDEX idx_agents_anon_id ON agents(anon_id);
CREATE INDEX idx_agents_industries ON agents USING GIN(industries);
CREATE INDEX idx_agents_skills ON agents USING GIN(skills);
CREATE INDEX idx_agents_active ON agents(is_active, last_active_at DESC);
CREATE INDEX idx_agents_quality ON agents(quality_score DESC) WHERE is_active = true;

-- =============================================================================
-- topics 表（讨论主题）
-- =============================================================================
CREATE TABLE topics (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submitter_user_id   UUID NOT NULL REFERENCES users(id),
    submitter_agent_id  UUID NOT NULL REFERENCES agents(id),

    topic_type          VARCHAR(50) NOT NULL,  -- business_idea, career_decision, tech_choice, etc.
    title               VARCHAR(500) NOT NULL,
    description         TEXT NOT NULL,
    background          TEXT,                  -- 提交者补充的背景
    tags                TEXT[] DEFAULT '{}',

    -- 向量
    topic_embedding     BYTEA,                 -- 序列化的 embedding 向量
    qdrant_point_id     UUID,

    status              topic_status NOT NULL DEFAULT 'pending_matching',

    -- 时间节点
    submitted_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    matched_at          TIMESTAMPTZ,
    discussion_started_at TIMESTAMPTZ,
    report_ready_at     TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,

    -- 通知状态
    notified_1h         BOOLEAN NOT NULL DEFAULT false,
    notified_12h        BOOLEAN NOT NULL DEFAULT false,
    notified_48h        BOOLEAN NOT NULL DEFAULT false,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_topics_submitter ON topics(submitter_user_id);
CREATE INDEX idx_topics_status ON topics(status);
CREATE INDEX idx_topics_submitted_at ON topics(submitted_at DESC);
CREATE INDEX idx_topics_pending_notification ON topics(status, notified_1h, notified_12h, notified_48h)
    WHERE status NOT IN ('cancelled', 'failed');

-- =============================================================================
-- discussions 表（讨论会话）
-- =============================================================================
CREATE TABLE discussions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    topic_id        UUID NOT NULL REFERENCES topics(id),

    status          discussion_status NOT NULL DEFAULT 'pending_matching',
    current_round   INT NOT NULL DEFAULT 0,  -- 0=未开始，1-4=当前轮次
    error_count     INT NOT NULL DEFAULT 0,

    -- 参与者 (JSONB 存储角色分配)
    -- [{agent_id, role, anon_id, assigned_at}]
    participants    JSONB NOT NULL DEFAULT '[]',

    -- 降级处理
    is_degraded     BOOLEAN NOT NULL DEFAULT false,
    degraded_reason TEXT,

    -- 统计
    total_tokens    INT DEFAULT 0,
    total_cost_usd  DECIMAL(10,6) DEFAULT 0,

    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_discussions_topic ON discussions(topic_id);
CREATE INDEX idx_discussions_status ON discussions(status);
CREATE INDEX idx_discussions_created ON discussions(created_at DESC);

-- =============================================================================
-- discussion_messages 表（讨论发言）
-- =============================================================================
CREATE TABLE discussion_messages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    discussion_id   UUID NOT NULL REFERENCES discussions(id),
    agent_id        UUID NOT NULL REFERENCES agents(id),
    round_number    INT NOT NULL CHECK (round_number BETWEEN 1 AND 4),
    role            discussion_role NOT NULL,

    -- 内容
    content         TEXT NOT NULL,
    key_point       TEXT NOT NULL,           -- 核心观点摘要（用于报告聚合）
    addressed_to    discussion_role,         -- 本轮回应对象
    confidence      DECIMAL(3,2),            -- 0-1 置信度

    -- 相似度检测
    similarity_to_prev DECIMAL(4,3),         -- 与前轮 cosine similarity
    was_rewritten   BOOLEAN DEFAULT false,   -- 是否触发了重写

    -- LLM 信息
    model_used      VARCHAR(100) NOT NULL,
    prompt_tokens   INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    cost_usd        DECIMAL(10,6) DEFAULT 0,

    -- 幂等键
    idempotency_key VARCHAR(255) UNIQUE NOT NULL, -- discussion_id+round+role

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_discussion ON discussion_messages(discussion_id);
CREATE INDEX idx_messages_discussion_round ON discussion_messages(discussion_id, round_number);
CREATE INDEX idx_messages_agent ON discussion_messages(agent_id);
CREATE INDEX idx_messages_idempotency ON discussion_messages(idempotency_key);

-- =============================================================================
-- reports 表（讨论报告）
-- =============================================================================
CREATE TABLE reports (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    discussion_id       UUID NOT NULL REFERENCES discussions(id),
    topic_id            UUID NOT NULL REFERENCES topics(id),

    -- 报告内容
    summary             TEXT,                -- 800-1200 字报告正文
    consensus_points    JSONB DEFAULT '[]',  -- 共识观点列表
    divergence_points   JSONB DEFAULT '[]',  -- 分歧观点列表
    key_questions       JSONB DEFAULT '[]',  -- 关键问题
    action_items        JSONB DEFAULT '[]',  -- 行动建议
    blind_spots         JSONB DEFAULT '[]',  -- 盲点/未考虑因素

    -- 连接推荐
    -- [{agent_id, anon_id, score, reasons: {insight, complementary, collaboration, activity}}]
    recommended_agents  JSONB DEFAULT '[]',

    -- 质量评估
    quality_score       DECIMAL(3,1),        -- LLM 自评 1-10
    user_rating         INT,                 -- 用户评分 1-5
    user_feedback       TEXT,

    -- 生成信息
    model_used          VARCHAR(100),
    total_tokens        INT DEFAULT 0,
    generation_attempts INT DEFAULT 1,

    generated_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_reports_discussion ON reports(discussion_id);
CREATE INDEX idx_reports_topic ON reports(topic_id);
CREATE INDEX idx_reports_quality ON reports(quality_score DESC);

-- =============================================================================
-- connections 表（连接申请）
-- =============================================================================
CREATE TABLE connections (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    requester_user_id       UUID NOT NULL REFERENCES users(id),
    target_user_id          UUID NOT NULL REFERENCES users(id),
    requester_agent_id      UUID NOT NULL REFERENCES agents(id),
    target_agent_id         UUID NOT NULL REFERENCES agents(id),
    topic_id                UUID REFERENCES topics(id),     -- 来源 Topic

    status                  connection_status NOT NULL DEFAULT 'pending',

    -- 申请信息
    request_message         TEXT,                           -- 申请理由（可选）

    -- 联系方式（AES-256-GCM 加密，Cloud KMS 管理密钥）
    requester_contact_enc   BYTEA,                         -- 加密后的联系方式
    target_contact_enc      BYTEA,                         -- 加密后的联系方式
    requester_contact_iv    BYTEA,                         -- 加密 IV
    target_contact_iv       BYTEA,                         -- 加密 IV

    -- 时间节点
    requested_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    responded_at            TIMESTAMPTZ,
    expires_at              TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '7 days'),

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT no_self_connection CHECK (requester_user_id != target_user_id)
);

CREATE INDEX idx_connections_requester ON connections(requester_user_id);
CREATE INDEX idx_connections_target ON connections(target_user_id);
CREATE INDEX idx_connections_status ON connections(status);
CREATE INDEX idx_connections_topic ON connections(topic_id);
-- 防止重复申请
CREATE UNIQUE INDEX idx_connections_unique ON connections(requester_user_id, target_user_id, topic_id)
    WHERE status NOT IN ('rejected', 'cancelled', 'expired');

-- =============================================================================
-- anon_id_mappings 表（独立 Schema，最高安全）
-- =============================================================================
CREATE TABLE anon.anon_id_mappings (
    anon_id         VARCHAR(20) NOT NULL,
    agent_id        UUID NOT NULL,
    discussion_id   UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (anon_id, discussion_id)
);

CREATE INDEX idx_anon_mappings_agent ON anon.anon_id_mappings(agent_id);
CREATE INDEX idx_anon_mappings_discussion ON anon.anon_id_mappings(discussion_id);

-- audit_log 表（所有对 anon_id_mappings 的访问都必须记录）
CREATE TABLE anon.access_audit_log (
    id              BIGSERIAL PRIMARY KEY,
    accessed_by     VARCHAR(100) NOT NULL,  -- 服务/用户标识
    action          VARCHAR(50) NOT NULL,   -- 'resolve_anon_id', 'decrypt_contact'
    anon_id         VARCHAR(20),
    discussion_id   UUID,
    connection_id   UUID,
    ip_address      INET,
    reason          TEXT,
    accessed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_accessed_at ON anon.access_audit_log(accessed_at DESC);
CREATE INDEX idx_audit_log_anon_id ON anon.access_audit_log(anon_id);

-- =============================================================================
-- notifications 表
-- =============================================================================
CREATE TABLE notifications (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID NOT NULL REFERENCES users(id),
    topic_id            UUID REFERENCES topics(id),

    notification_type   notification_type NOT NULL,
    channel             notification_channel NOT NULL,
    status              notification_status NOT NULL DEFAULT 'pending',

    title               VARCHAR(255),
    body                TEXT,
    data                JSONB DEFAULT '{}',  -- 额外数据（如 report_id）

    -- 发送结果
    external_id         VARCHAR(255),        -- FCM message_id / SendGrid message_id
    error_message       TEXT,
    retry_count         INT DEFAULT 0,

    scheduled_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at             TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_status ON notifications(status, scheduled_at)
    WHERE status = 'pending';
CREATE INDEX idx_notifications_topic ON notifications(topic_id);

-- =============================================================================
-- llm_cost_logs 表（LLM 成本追踪）
-- =============================================================================
CREATE TABLE llm_cost_logs (
    id              BIGSERIAL PRIMARY KEY,
    provider        llm_provider NOT NULL,
    model           VARCHAR(100) NOT NULL,
    module          VARCHAR(50) NOT NULL,   -- 调用来源模块
    reference_id    UUID,                   -- 关联的 discussion_id / report_id 等
    prompt_tokens   INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    total_tokens    INT NOT NULL DEFAULT 0,
    cost_usd        DECIMAL(10,6) NOT NULL DEFAULT 0,
    cached          BOOLEAN DEFAULT false,
    duration_ms     INT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cost_logs_created ON llm_cost_logs(created_at DESC);
CREATE INDEX idx_cost_logs_module ON llm_cost_logs(module, created_at DESC);
CREATE INDEX idx_cost_logs_reference ON llm_cost_logs(reference_id);

-- =============================================================================
-- 函数：自动更新 updated_at
-- =============================================================================
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为需要的表添加触发器
DO $$
DECLARE
    t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['users', 'agents', 'topics', 'discussions', 'reports', 'connections', 'notifications'] LOOP
        EXECUTE format('CREATE TRIGGER trg_%s_updated_at
            BEFORE UPDATE ON %s
            FOR EACH ROW EXECUTE FUNCTION update_updated_at()', t, t);
    END LOOP;
END;
$$;

-- =============================================================================
-- 初始种子数据：通才种子 Agent（冷启动用）
-- =============================================================================
INSERT INTO users (id, email, display_name, status)
VALUES ('00000000-0000-0000-0000-000000000001', 'seed@digital-twin.internal', 'System Seed', 'active');
