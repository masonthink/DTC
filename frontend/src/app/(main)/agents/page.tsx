"use client";

import useSWR from "swr";
import Link from "next/link";
import { agentApi, type Agent } from "@/lib/api";
import { Skeleton } from "@/components/ui/skeleton";
import { ChevronRight, Plus, Star, MessageSquare } from "lucide-react";

const AGENT_TYPE_CONFIG: Record<Agent["agent_type"], { label: string; emoji: string; bg: string }> = {
  professional: { label: "职场精英", emoji: "💼", bg: "bg-blue-50 border-blue-100" },
  entrepreneur: { label: "创业者", emoji: "🚀", bg: "bg-violet-50 border-violet-100" },
  investor: { label: "投资人", emoji: "📈", bg: "bg-emerald-50 border-emerald-100" },
  generalist: { label: "多面手", emoji: "🌐", bg: "bg-amber-50 border-amber-100" },
};

export default function AgentsPage() {
  const { data: agents, isLoading } = useSWR("agents", agentApi.list);

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-card/92 backdrop-blur-2xl border-b border-border/60">
        <div className="px-4 py-4 flex items-center justify-between">
          <div>
            <h1 className="text-[18px] font-bold text-foreground tracking-tight">我的分身</h1>
            <p className="text-[11px] text-muted-foreground mt-0.5">代表你参与讨论，帮你发现值得认识的人</p>
          </div>
          <Link
            href="/agents/create"
            className="flex items-center gap-1.5 bg-primary hover:bg-primary/90 active:scale-95 text-primary-foreground text-[13px] font-medium px-3.5 py-2 rounded-xl transition-all duration-200 shadow-primary-sm"
          >
            <Plus className="w-3.5 h-3.5" />
            创建分身
          </Link>
        </div>
      </div>

      <div className="px-4 pt-4 space-y-3">
        {isLoading && (
          <>
            <Skeleton className="h-28 w-full rounded-2xl" />
            <Skeleton className="h-28 w-full rounded-2xl" />
          </>
        )}

        {!isLoading && agents?.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="w-20 h-20 rounded-3xl bg-primary/8 border border-primary/12 flex items-center justify-center text-4xl mb-6">
              🤖
            </div>
            <h3 className="text-foreground font-semibold text-[17px] mb-2">
              还没有创建分身
            </h3>
            <p className="text-muted-foreground text-[13px] max-w-xs mb-8 leading-relaxed">
              创建你的数字分身，它会代表你参与讨论、帮你找到搭子
            </p>
            <Link
              href="/agents/create"
              className="bg-primary hover:bg-primary/90 text-primary-foreground px-8 py-3 rounded-xl font-medium text-[14px] transition-all active:scale-95 shadow-primary-sm hover:shadow-primary-md"
            >
              创建我的分身
            </Link>
          </div>
        )}

        {agents?.map((agent: Agent) => (
          <AgentCard key={agent.id} agent={agent} />
        ))}
      </div>
    </div>
  );
}

function AgentCard({ agent }: { agent: Agent }) {
  const cfg = AGENT_TYPE_CONFIG[agent.agent_type] ?? { label: "分身", emoji: "🤖", bg: "bg-muted border-border" };

  return (
    <Link
      href={`/agents/${agent.id}`}
      className="block bg-card border border-border rounded-2xl p-4 hover:border-primary/20 hover:shadow-sm transition-all duration-200 active:scale-[0.99] shadow-xs"
    >
      <div className="flex items-start gap-4">
        {/* Avatar */}
        <div className={`w-14 h-14 rounded-2xl border flex items-center justify-center text-2xl flex-shrink-0 ${cfg.bg}`}>
          {cfg.emoji}
        </div>

        {/* Info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <h3 className="text-foreground font-semibold text-[15px] truncate">
              {agent.display_name}
            </h3>
            <span className="text-[11px] bg-primary/10 text-primary border border-primary/15 px-2 py-0.5 rounded-full flex-shrink-0 font-medium">
              {cfg.label}
            </span>
          </div>

          <p className="text-muted-foreground text-[12px] mb-3 line-clamp-1">
            {agent.industries.slice(0, 3).join(" · ")}
          </p>

          <div className="flex items-center gap-4 text-[12px] text-muted-foreground">
            <span className="flex items-center gap-1">
              <MessageSquare className="w-3.5 h-3.5" />
              {agent.discussion_count} 次讨论
            </span>
            {agent.quality_score > 0 && (
              <span className="flex items-center gap-1 text-amber-500">
                <Star className="w-3.5 h-3.5 fill-current" />
                {agent.quality_score.toFixed(1)}
              </span>
            )}
            <span className="text-[11px] text-muted-foreground/60 font-mono">
              #{agent.anon_id.slice(0, 8)}
            </span>
          </div>
        </div>

        <ChevronRight className="w-4 h-4 text-muted-foreground/50 flex-shrink-0 mt-1" />
      </div>
    </Link>
  );
}
