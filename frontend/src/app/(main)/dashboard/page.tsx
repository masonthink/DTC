"use client";

import useSWR from "swr";
import { agentApi, topicApi, type Topic, type Agent } from "@/lib/api";
import Link from "next/link";
import { TopicStatusBadge } from "@/components/topic/TopicStatusBadge";
import { DiscussionProgress } from "@/components/discussion/DiscussionProgress";

export default function DashboardPage() {
  const { data: agents } = useSWR("agents", agentApi.list);
  const { data: topicsData } = useSWR("topics", () =>
    topicApi.list({ limit: 10 })
  );

  const activeTopics = topicsData?.items?.filter(
    (t) => !["completed", "cancelled", "failed"].includes(t.status)
  );
  const completedTopics = topicsData?.items?.filter(
    (t) => t.status === "completed"
  );

  const hasAgent = agents && agents.length > 0;

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      {/* Welcome banner */}
      {!hasAgent && (
        <div className="mb-8 bg-gradient-to-r from-indigo-600/20 to-purple-600/20 border border-indigo-500/30 rounded-2xl p-6">
          <h2 className="text-lg font-semibold text-white mb-2">
            👋 欢迎来到数字分身社区
          </h2>
          <p className="text-slate-300 text-sm mb-4">
            第一步：创建你的数字分身，让它代表你参与深度讨论
          </p>
          <Link
            href="/agents/create"
            className="inline-flex items-center gap-2 bg-indigo-600 hover:bg-indigo-500 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors"
          >
            创建我的分身 →
          </Link>
        </div>
      )}

      {/* Agent quick view */}
      {hasAgent && (
        <section className="mb-8">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-base font-semibold text-white">我的分身</h2>
            <Link href="/agents" className="text-indigo-400 text-sm hover:text-indigo-300">
              管理 →
            </Link>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {agents?.slice(0, 2).map((agent: Agent) => (
              <AgentCard key={agent.id} agent={agent} />
            ))}
            <Link
              href="/topics/submit"
              className="flex items-center justify-center gap-2 border border-dashed border-slate-600 hover:border-indigo-500 rounded-xl p-4 text-slate-400 hover:text-indigo-400 transition-colors text-sm"
            >
              + 提交新话题
            </Link>
          </div>
        </section>
      )}

      {/* Active discussions */}
      {activeTopics && activeTopics.length > 0 && (
        <section className="mb-8">
          <h2 className="text-base font-semibold text-white mb-4">
            进行中的讨论 ({activeTopics.length})
          </h2>
          <div className="space-y-3">
            {activeTopics.map((topic: Topic) => (
              <ActiveTopicCard key={topic.id} topic={topic} />
            ))}
          </div>
        </section>
      )}

      {/* Completed discussions */}
      {completedTopics && completedTopics.length > 0 && (
        <section>
          <h2 className="text-base font-semibold text-white mb-4">
            已完成的报告
          </h2>
          <div className="space-y-3">
            {completedTopics.map((topic: Topic) => (
              <CompletedTopicCard key={topic.id} topic={topic} />
            ))}
          </div>
        </section>
      )}

      {/* Empty state */}
      {hasAgent && (!topicsData?.items || topicsData.items.length === 0) && (
        <div className="text-center py-16">
          <div className="text-4xl mb-4">💬</div>
          <h3 className="text-white font-medium mb-2">还没有讨论</h3>
          <p className="text-slate-400 text-sm mb-6">
            提交一个话题，让你的分身和其他分身展开深度讨论
          </p>
          <Link
            href="/topics/submit"
            className="inline-flex items-center gap-2 bg-indigo-600 hover:bg-indigo-500 text-white px-6 py-2.5 rounded-lg text-sm font-medium transition-colors"
          >
            提交我的第一个话题
          </Link>
        </div>
      )}
    </div>
  );
}

function AgentCard({ agent }: { agent: Agent }) {
  return (
    <div className="bg-slate-800/60 border border-slate-700 rounded-xl p-4">
      <div className="flex items-start gap-3">
        <div className="w-10 h-10 rounded-lg bg-indigo-600/20 border border-indigo-500/30 flex items-center justify-center text-lg flex-shrink-0">
          🤖
        </div>
        <div className="min-w-0">
          <p className="text-white text-sm font-medium truncate">{agent.display_name}</p>
          <p className="text-slate-400 text-xs mt-0.5">
            {agent.industries.slice(0, 2).join(" · ")}
          </p>
          <p className="text-slate-500 text-xs mt-1">
            参与了 {agent.discussion_count} 次讨论
          </p>
        </div>
      </div>
    </div>
  );
}

function ActiveTopicCard({ topic }: { topic: Topic }) {
  return (
    <Link
      href={`/topics/${topic.id}`}
      className="block bg-slate-800/60 border border-slate-700 hover:border-slate-600 rounded-xl p-4 transition-colors"
    >
      <div className="flex items-start justify-between gap-3 mb-3">
        <h3 className="text-white text-sm font-medium line-clamp-2">{topic.title}</h3>
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
      className="block bg-slate-800/40 border border-slate-700/50 hover:border-slate-600 rounded-xl p-4 transition-colors"
    >
      <div className="flex items-start justify-between gap-3">
        <h3 className="text-slate-300 text-sm font-medium line-clamp-1">{topic.title}</h3>
        <span className="text-xs text-emerald-400 bg-emerald-400/10 px-2 py-0.5 rounded-full flex-shrink-0">
          报告已就绪
        </span>
      </div>
      {topic.report_ready_at && (
        <p className="text-slate-500 text-xs mt-2">
          {new Date(topic.report_ready_at).toLocaleDateString("zh-CN")} 生成
        </p>
      )}
    </Link>
  );
}
