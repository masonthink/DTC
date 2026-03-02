"use client";

import useSWR from "swr";
import Link from "next/link";
import { agentApi, type Agent } from "@/lib/api";
import { Skeleton } from "@/components/ui/skeleton";
import { ChevronRight, Plus, Star, MessageSquare } from "lucide-react";

export default function AgentsPage() {
  const { data: agents, isLoading } = useSWR("agents", agentApi.list);

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-white/95 backdrop-blur-xl border-b border-slate-200">
        <div className="px-4 py-4 flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold text-slate-900">我的分身</h1>
            <p className="text-xs text-slate-500 mt-0.5">代表你参与深度讨论</p>
          </div>
          <Link
            href="/agents/create"
            className="flex items-center gap-1.5 bg-indigo-600 hover:bg-indigo-500 active:scale-95 text-white text-sm font-medium px-4 py-2 rounded-xl transition-all"
          >
            <Plus className="w-4 h-4" />
            创建分身
          </Link>
        </div>
      </div>

      <div className="px-4 pt-4 space-y-3">
        {isLoading && (
          <>
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
          </>
        )}

        {agents?.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="w-20 h-20 rounded-3xl bg-indigo-600/10 border border-indigo-500/20 flex items-center justify-center text-4xl mb-6">
              🤖
            </div>
            <h3 className="text-slate-900 font-semibold text-lg mb-2">
              还没有数字分身
            </h3>
            <p className="text-slate-400 text-sm max-w-xs mb-8 leading-relaxed">
              创建你的数字分身，它将代表你的思维方式和专业背景参与讨论
            </p>
            <Link
              href="/agents/create"
              className="bg-indigo-600 hover:bg-indigo-500 text-white px-8 py-3 rounded-xl font-medium transition-colors active:scale-95"
            >
              创建我的第一个分身
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
  const agentTypeLabel: Record<Agent["agent_type"], string> = {
    professional: "职场精英",
    entrepreneur: "创业者",
    investor: "投资人",
    generalist: "多面手",
  };

  return (
    <Link href={`/agents/${agent.id}`} className="block bg-white border border-slate-200 rounded-2xl p-4 active:bg-slate-200 transition-colors">
      <div className="flex items-start gap-4">
        {/* Avatar */}
        <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-indigo-600/30 to-purple-600/20 border border-indigo-500/20 flex items-center justify-center text-2xl flex-shrink-0">
          🤖
        </div>

        {/* Info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <h3 className="text-slate-900 font-semibold text-base truncate">
              {agent.display_name}
            </h3>
            <span className="text-xs bg-indigo-600/20 text-indigo-400 border border-indigo-600/30 px-2 py-0.5 rounded-full flex-shrink-0">
              {agentTypeLabel[agent.agent_type]}
            </span>
          </div>

          <p className="text-slate-400 text-xs mb-3 line-clamp-1">
            {agent.industries.slice(0, 3).join(" · ")}
          </p>

          <div className="flex items-center gap-4 text-xs text-slate-500">
            <span className="flex items-center gap-1">
              <MessageSquare className="w-3.5 h-3.5" />
              {agent.discussion_count} 次讨论
            </span>
            {agent.quality_score > 0 && (
              <span className="flex items-center gap-1 text-amber-400">
                <Star className="w-3.5 h-3.5 fill-current" />
                {agent.quality_score.toFixed(1)}
              </span>
            )}
            <span className="text-xs text-slate-600">
              #{agent.anon_id.slice(0, 8)}
            </span>
          </div>
        </div>

        <ChevronRight className="w-4 h-4 text-slate-600 flex-shrink-0 mt-1" />
      </div>
    </Link>
  );
}
