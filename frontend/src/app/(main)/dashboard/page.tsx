"use client";

import useSWR from "swr";
import { agentApi, topicApi, type Topic, type Agent } from "@/lib/api";
import Link from "next/link";
import { TopicStatusBadge } from "@/components/topic/TopicStatusBadge";
import { DiscussionProgress } from "@/components/discussion/DiscussionProgress";
import { Skeleton } from "@/components/ui/skeleton";
import { Bell, Sparkles, ChevronRight } from "lucide-react";

export default function DashboardPage() {
  const { data: agents, isLoading: agentsLoading } = useSWR("agents", agentApi.list);
  const { data: topicsData, isLoading: topicsLoading } = useSWR("topics", () =>
    topicApi.list({ limit: 10 })
  );

  const isLoading = agentsLoading || topicsLoading;
  const activeTopics = topicsData?.items?.filter(
    (t) => !["completed", "cancelled", "failed"].includes(t.status)
  );
  const completedTopics = topicsData?.items?.filter(
    (t) => t.status === "completed"
  );
  const hasAgent = agents && agents.length > 0;

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-card/92 backdrop-blur-2xl border-b border-border/60">
        <div className="px-4 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-7 h-7 rounded-lg bg-primary-gradient flex items-center justify-center flex-shrink-0">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <circle cx="4" cy="4" r="2.5" fill="white" fillOpacity="0.9" />
                <circle cx="10" cy="4" r="2.5" fill="white" fillOpacity="0.6" />
                <circle cx="7" cy="10" r="2.5" fill="white" fillOpacity="0.75" />
              </svg>
            </div>
            <div>
              <h1 className="text-[16px] font-semibold text-foreground tracking-tight leading-none">
                Concors
              </h1>
              <p className="text-[11px] text-muted-foreground mt-0.5">AI 多角度分析，帮你做更好的决策</p>
            </div>
          </div>
          <button aria-label="通知" className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-muted text-muted-foreground hover:text-foreground transition-all duration-150">
            <Bell className="w-[18px] h-[18px]" />
          </button>
        </div>
      </div>

      <div className="px-4 pt-5 space-y-5">
        {/* Onboarding banner */}
        {!isLoading && !hasAgent && (
          <div className="relative overflow-hidden bg-primary rounded-2xl p-5 shadow-primary-md">
            {/* Decorative circles */}
            <div className="absolute -top-8 -right-8 w-32 h-32 rounded-full bg-white/10" />
            <div className="absolute -bottom-6 -left-6 w-24 h-24 rounded-full bg-white/6" />

            <div className="relative">
              <div className="w-10 h-10 rounded-xl bg-white/15 flex items-center justify-center mb-3">
                <Sparkles className="w-5 h-5 text-white" />
              </div>
              <h2 className="text-white font-semibold text-[15px] mb-1.5 leading-snug">
                欢迎来到 Concors
              </h2>
              <p className="text-white/75 text-[13px] leading-relaxed mb-4">
                填写你的专业背景，AI 会基于你的视角，从四个不同角度分析你的问题。
              </p>
              <Link
                href="/agents/create"
                className="inline-flex items-center gap-1.5 bg-white hover:bg-white/95 text-primary px-4 py-2 rounded-xl text-[13px] font-semibold transition-all active:scale-95 shadow-sm"
              >
                开始设置
                <ChevronRight className="w-3.5 h-3.5" />
              </Link>
            </div>
          </div>
        )}

        {/* Loading skeleton */}
        {isLoading && (
          <div className="space-y-3">
            <Skeleton className="h-[88px] w-full rounded-2xl" />
            <Skeleton className="h-[96px] w-full rounded-2xl" />
            <Skeleton className="h-[96px] w-full rounded-2xl" />
          </div>
        )}

        {/* Agent quick view */}
        {!isLoading && hasAgent && (
          <section>
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-[13px] font-semibold text-foreground">我的背景档案</h2>
              <Link
                href="/agents"
                className="text-[12px] text-primary hover:text-primary/80 transition-colors flex items-center gap-0.5"
              >
                全部管理
                <ChevronRight className="w-3 h-3" />
              </Link>
            </div>
            <div className="space-y-2">
              {agents?.slice(0, 2).map((agent: Agent) => (
                <AgentCard key={agent.id} agent={agent} />
              ))}
            </div>
            <Link
              href="/topics/submit"
              className="flex items-center justify-center gap-2 border border-dashed border-border hover:border-primary/40 hover:bg-primary/4 rounded-2xl p-4 text-muted-foreground hover:text-primary transition-all duration-200 text-[13px] mt-2 active:scale-[0.99]"
            >
              <span className="text-base">+</span>
              提交新问题
            </Link>
          </section>
        )}

        {/* Active discussions */}
        {!isLoading && activeTopics && activeTopics.length > 0 && (
          <section>
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-[13px] font-semibold text-foreground flex items-center gap-2">
                进行中
                <span className="text-primary bg-primary/12 px-1.5 py-0.5 rounded-full text-[11px] font-semibold tabular-nums">
                  {activeTopics.length}
                </span>
              </h2>
              <Link
                href="/topics"
                className="text-[12px] text-primary hover:text-primary/80 transition-colors flex items-center gap-0.5"
              >
                查看全部
                <ChevronRight className="w-3 h-3" />
              </Link>
            </div>
            <div className="space-y-3">
              {activeTopics.map((topic: Topic) => (
                <ActiveTopicCard key={topic.id} topic={topic} />
              ))}
            </div>
          </section>
        )}

        {/* Completed discussions */}
        {!isLoading && completedTopics && completedTopics.length > 0 && (
          <section>
            <h2 className="text-[13px] font-semibold text-foreground mb-3">已完成报告</h2>
            <div className="space-y-2">
              {completedTopics.slice(0, 3).map((topic: Topic) => (
                <CompletedTopicCard key={topic.id} topic={topic} />
              ))}
            </div>
          </section>
        )}

        {/* Empty state */}
        {!isLoading && hasAgent && (!topicsData?.items || topicsData.items.length === 0) && (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <div className="w-16 h-16 rounded-2xl bg-primary/8 border border-primary/12 flex items-center justify-center mb-5">
              <svg width="28" height="28" viewBox="0 0 28 28" fill="none" className="text-primary">
                <path d="M4 22V8a2 2 0 012-2h16a2 2 0 012 2v14l-4-2-4 2-4-2-4 2z" stroke="currentColor" strokeWidth="1.75" strokeLinejoin="round"/>
                <path d="M9 12h10M9 16h6" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round"/>
              </svg>
            </div>
            <h3 className="text-foreground font-semibold text-[15px] mb-1.5">还没有提交过问题</h3>
            <p className="text-muted-foreground text-[13px] mb-6 max-w-[240px] leading-relaxed">
              提交一个问题，AI 会从四个不同角度帮你深度分析
            </p>
            <Link
              href="/topics/submit"
              className="bg-primary hover:bg-primary/90 text-primary-foreground px-7 py-2.5 rounded-xl text-[14px] font-semibold transition-all active:scale-95 shadow-primary-sm hover:shadow-primary-md"
            >
              提交第一个问题
            </Link>
          </div>
        )}

        <div className="pb-4" />
      </div>
    </div>
  );
}

function AgentCard({ agent }: { agent: Agent }) {
  const typeConfig: Record<Agent["agent_type"], { emoji: string; bg: string }> = {
    professional: { emoji: "💼", bg: "bg-blue-50 border-blue-100" },
    entrepreneur: { emoji: "🚀", bg: "bg-violet-50 border-violet-100" },
    investor: { emoji: "📈", bg: "bg-emerald-50 border-emerald-100" },
    generalist: { emoji: "🌐", bg: "bg-amber-50 border-amber-100" },
  };

  const cfg = typeConfig[agent.agent_type] ?? { emoji: "🤖", bg: "bg-muted border-border" };

  return (
    <div className="flex items-center gap-3 bg-card border border-border rounded-2xl p-3.5 shadow-xs">
      <div className={`w-11 h-11 rounded-xl border flex items-center justify-center text-xl flex-shrink-0 ${cfg.bg}`}>
        {cfg.emoji}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-foreground text-[14px] font-semibold truncate leading-none">{agent.display_name}</p>
        <p className="text-muted-foreground text-[12px] mt-1 truncate">
          {agent.industries.slice(0, 3).join(" · ")}
        </p>
      </div>
      <div className="text-right flex-shrink-0">
        <p className="text-primary text-[15px] font-bold leading-none">{agent.discussion_count}</p>
        <p className="text-muted-foreground text-[11px] mt-0.5">次讨论</p>
      </div>
    </div>
  );
}

function ActiveTopicCard({ topic }: { topic: Topic }) {
  return (
    <Link
      href={`/topics/${topic.id}`}
      className="block bg-card border border-border hover:border-primary/25 rounded-2xl p-4 transition-all duration-200 active:scale-[0.99] shadow-xs hover:shadow-sm"
    >
      <div className="flex items-start justify-between gap-3 mb-3">
        <h3 className="text-foreground text-[14px] font-medium line-clamp-2 flex-1 leading-snug">
          {topic.title}
        </h3>
        <TopicStatusBadge status={topic.status} />
      </div>
      <DiscussionProgress status={topic.status} submittedAt={topic.submitted_at} />
    </Link>
  );
}

function CompletedTopicCard({ topic }: { topic: Topic }) {
  return (
    <Link
      href={`/topics/${topic.id}`}
      className="flex items-center gap-3 bg-muted/60 border border-border/70 hover:border-emerald-400/30 hover:bg-emerald-50/50 rounded-2xl p-3.5 transition-all duration-200 active:scale-[0.99]"
    >
      <div className="w-9 h-9 rounded-xl bg-emerald-50 border border-emerald-100 flex items-center justify-center flex-shrink-0">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" className="text-emerald-500">
          <path d="M3 8l3 3 7-7" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round"/>
        </svg>
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-foreground/85 text-[13px] font-medium line-clamp-1">{topic.title}</p>
        {topic.report_ready_at && (
          <p className="text-muted-foreground text-[11px] mt-0.5">
            {new Date(topic.report_ready_at).toLocaleDateString("zh-CN")} 生成
          </p>
        )}
      </div>
      <span className="text-[12px] text-emerald-500 flex-shrink-0 font-medium">查看报告</span>
    </Link>
  );
}
