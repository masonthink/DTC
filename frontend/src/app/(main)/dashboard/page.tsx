"use client";

import useSWR from "swr";
import { agentApi, topicApi, type Topic, type Agent } from "@/lib/api";
import Link from "next/link";
import { TopicStatusBadge } from "@/components/topic/TopicStatusBadge";
import { DiscussionProgress } from "@/components/discussion/DiscussionProgress";
import { Skeleton } from "@/components/ui/skeleton";
import { Bell } from "lucide-react";

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
    <div className="min-h-screen bg-slate-50">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-white/95 backdrop-blur-xl border-b border-slate-200">
        <div className="px-4 py-4 flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold text-slate-900">
              <span className="text-indigo-400">C</span>oncors
            </h1>
            <p className="text-xs text-slate-500 mt-0.5">让思想连接有价值的人</p>
          </div>
          <button className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-slate-100 text-slate-400 hover:text-slate-900 transition-colors">
            <Bell className="w-5 h-5" />
          </button>
        </div>
      </div>

      <div className="px-4 pt-4 space-y-5">
        {/* Onboarding banner */}
        {!isLoading && !hasAgent && (
          <div className="bg-gradient-to-br from-indigo-600/25 via-indigo-600/10 to-purple-600/10 border border-indigo-500/30 rounded-2xl p-5">
            <div className="text-3xl mb-3">👋</div>
            <h2 className="text-slate-900 font-semibold text-base mb-1.5">
              欢迎来到 Concors
            </h2>
            <p className="text-slate-700 text-sm leading-relaxed mb-4">
              创建你的数字分身，让它代表你的专业背景和思维方式，与其他分身展开深度讨论。
            </p>
            <Link
              href="/agents/create"
              className="inline-flex items-center gap-2 bg-indigo-600 hover:bg-indigo-500 text-white px-5 py-2.5 rounded-xl text-sm font-semibold transition-all active:scale-95 shadow-lg shadow-indigo-600/25"
            >
              创建我的分身 →
            </Link>
          </div>
        )}

        {/* Loading skeleton */}
        {isLoading && (
          <div className="space-y-3">
            <Skeleton className="h-20 w-full" />
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-24 w-full" />
          </div>
        )}

        {/* Agent quick view */}
        {!isLoading && hasAgent && (
          <section>
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-sm font-semibold text-slate-900">我的分身</h2>
              <Link
                href="/agents"
                className="text-indigo-400 text-xs hover:text-indigo-700 transition-colors"
              >
                全部管理 →
              </Link>
            </div>
            <div className="space-y-2">
              {agents?.slice(0, 2).map((agent: Agent) => (
                <AgentCard key={agent.id} agent={agent} />
              ))}
            </div>
            <Link
              href="/topics/submit"
              className="flex items-center justify-center gap-2 border border-dashed border-slate-200 hover:border-indigo-500/60 hover:bg-indigo-600/5 rounded-2xl p-4 text-slate-500 hover:text-indigo-400 transition-all text-sm mt-2 active:scale-[0.99]"
            >
              + 提交新话题
            </Link>
          </section>
        )}

        {/* Active discussions */}
        {!isLoading && activeTopics && activeTopics.length > 0 && (
          <section>
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-sm font-semibold text-slate-900">
                进行中{" "}
                <span className="text-indigo-400 bg-indigo-600/15 px-2 py-0.5 rounded-full text-xs ml-1">
                  {activeTopics.length}
                </span>
              </h2>
              <Link
                href="/topics"
                className="text-indigo-400 text-xs hover:text-indigo-700 transition-colors"
              >
                查看全部 →
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
            <h2 className="text-sm font-semibold text-slate-900 mb-3">已完成报告</h2>
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
            <div className="text-5xl mb-4">💬</div>
            <h3 className="text-slate-900 font-semibold mb-2">还没有任何讨论</h3>
            <p className="text-slate-400 text-sm mb-6 max-w-xs leading-relaxed">
              提交一个话题，让你的分身和其他分身展开深度的多角度讨论
            </p>
            <Link
              href="/topics/submit"
              className="bg-indigo-600 hover:bg-indigo-500 text-white px-8 py-3 rounded-xl text-sm font-semibold transition-all active:scale-95 shadow-lg shadow-indigo-600/20"
            >
              提交第一个话题
            </Link>
          </div>
        )}

        <div className="pb-4" />
      </div>
    </div>
  );
}

function AgentCard({ agent }: { agent: Agent }) {
  const typeEmojis: Record<Agent["agent_type"], string> = {
    professional: "💼",
    entrepreneur: "🚀",
    investor: "📈",
    generalist: "🌐",
  };

  return (
    <div className="flex items-center gap-3 bg-white border border-slate-200 rounded-2xl p-3.5">
      <div className="w-11 h-11 rounded-xl bg-indigo-600/15 border border-indigo-500/20 flex items-center justify-center text-xl flex-shrink-0">
        {typeEmojis[agent.agent_type] ?? "🤖"}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-slate-900 text-sm font-semibold truncate">{agent.display_name}</p>
        <p className="text-slate-500 text-xs mt-0.5 truncate">
          {agent.industries.slice(0, 3).join(" · ")}
        </p>
      </div>
      <div className="text-right flex-shrink-0">
        <p className="text-indigo-400 text-sm font-semibold">{agent.discussion_count}</p>
        <p className="text-slate-600 text-xs">次讨论</p>
      </div>
    </div>
  );
}

function ActiveTopicCard({ topic }: { topic: Topic }) {
  return (
    <Link
      href={`/topics/${topic.id}`}
      className="block bg-white border border-slate-200 hover:border-indigo-500/30 rounded-2xl p-4 transition-all active:scale-[0.99]"
    >
      <div className="flex items-start justify-between gap-3 mb-3">
        <h3 className="text-slate-900 text-sm font-medium line-clamp-2 flex-1">
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
      className="flex items-center gap-3 bg-slate-50 border border-slate-200/60 hover:border-emerald-500/20 rounded-2xl p-3.5 transition-all active:scale-[0.99]"
    >
      <div className="w-8 h-8 rounded-lg bg-emerald-500/10 flex items-center justify-center flex-shrink-0">
        <span className="text-sm">✅</span>
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-slate-700 text-sm font-medium line-clamp-1">{topic.title}</p>
        {topic.report_ready_at && (
          <p className="text-slate-600 text-xs mt-0.5">
            {new Date(topic.report_ready_at).toLocaleDateString("zh-CN")} 生成
          </p>
        )}
      </div>
      <span className="text-xs text-emerald-400 flex-shrink-0">查看报告 →</span>
    </Link>
  );
}
