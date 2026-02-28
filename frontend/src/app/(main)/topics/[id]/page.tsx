"use client";

import useSWR from "swr";
import { useParams } from "next/navigation";
import Link from "next/link";
import { topicApi, discussionApi, type Topic } from "@/lib/api";
import { TopicStatusBadge } from "@/components/topic/TopicStatusBadge";
import { DiscussionProgress } from "@/components/discussion/DiscussionProgress";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowLeft, FileText, MessageSquare } from "lucide-react";
import { formatDistanceToNow, format } from "date-fns";
import { zhCN } from "date-fns/locale";

const TOPIC_TYPE_LABELS: Record<string, string> = {
  business_idea: "💡 商业想法",
  career_decision: "💼 职业决策",
  tech_choice: "⚙️ 技术选型",
  product_design: "🎨 产品设计",
  investment: "📈 投资判断",
  other: "💬 其他",
};

const ROLE_LABELS: Record<string, string> = {
  questioner: "提问者",
  supporter: "支持者",
  supplementer: "补充者",
  inquirer: "探究者",
};

export default function TopicDetailPage() {
  const { id } = useParams<{ id: string }>();

  const { data: topic, isLoading } = useSWR(
    id ? `topics/${id}` : null,
    () => topicApi.get(id)
  );

  const { data: discussion } = useSWR(
    topic?.status === "discussion_active" && id
      ? `discussions-by-topic-${id}`
      : null,
    async () => {
      // We need to find discussion by topic - for now try common IDs
      return null;
    }
  );

  if (isLoading) return <TopicSkeleton />;
  if (!topic) return <NotFound />;

  return (
    <div className="min-h-screen bg-slate-950">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-slate-950/95 backdrop-blur-xl border-b border-slate-800">
        <div className="px-4 py-4 flex items-center gap-3">
          <Link
            href="/topics"
            className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-slate-800 text-slate-400 hover:text-white transition-colors"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <div className="flex-1 min-w-0">
            <p className="text-xs text-slate-500">话题详情</p>
          </div>
          <TopicStatusBadge status={topic.status} />
        </div>
      </div>

      <div className="px-4 pt-5 space-y-5 pb-4">
        {/* Title card */}
        <div className="bg-slate-900 border border-slate-700/50 rounded-2xl p-5">
          <div className="flex items-start gap-3 mb-3">
            <span className="text-2xl">
              {TOPIC_TYPE_LABELS[topic.topic_type]?.split(" ")[0] ?? "💬"}
            </span>
            <div className="flex-1">
              <p className="text-xs text-slate-500 mb-1">
                {TOPIC_TYPE_LABELS[topic.topic_type]?.slice(2) ?? topic.topic_type}
              </p>
              <h2 className="text-white font-semibold text-base leading-snug">
                {topic.title}
              </h2>
            </div>
          </div>

          {topic.description && (
            <p className="text-slate-400 text-sm leading-relaxed">
              {topic.description}
            </p>
          )}

          {topic.background && (
            <div className="mt-3 pt-3 border-t border-slate-700/50">
              <p className="text-xs text-slate-500 mb-1">背景信息</p>
              <p className="text-slate-400 text-sm leading-relaxed">{topic.background}</p>
            </div>
          )}

          {topic.tags && topic.tags.length > 0 && (
            <div className="flex flex-wrap gap-2 mt-3 pt-3 border-t border-slate-700/50">
              {topic.tags.map((tag) => (
                <span
                  key={tag}
                  className="text-xs text-slate-500 bg-slate-800 px-2.5 py-1 rounded-full"
                >
                  #{tag}
                </span>
              ))}
            </div>
          )}
        </div>

        {/* Progress */}
        <div className="bg-slate-900 border border-slate-700/50 rounded-2xl p-5">
          <p className="text-xs font-medium text-slate-400 uppercase tracking-wider mb-4">
            进度追踪
          </p>
          <DiscussionProgress
            status={topic.status}
            submittedAt={topic.submitted_at}
          />
        </div>

        {/* Timeline */}
        <div className="bg-slate-900 border border-slate-700/50 rounded-2xl p-5">
          <p className="text-xs font-medium text-slate-400 uppercase tracking-wider mb-4">
            时间线
          </p>
          <div className="space-y-3">
            <TimelineItem
              label="提交时间"
              value={format(new Date(topic.submitted_at), "MM月dd日 HH:mm", {
                locale: zhCN,
              })}
              sub={formatDistanceToNow(new Date(topic.submitted_at), {
                addSuffix: true,
                locale: zhCN,
              })}
            />
            {topic.matched_at && (
              <TimelineItem
                label="匹配成功"
                value={format(new Date(topic.matched_at), "MM月dd日 HH:mm", {
                  locale: zhCN,
                })}
                highlight
              />
            )}
            {topic.report_ready_at && (
              <TimelineItem
                label="报告生成"
                value={format(
                  new Date(topic.report_ready_at),
                  "MM月dd日 HH:mm",
                  { locale: zhCN }
                )}
                highlight
              />
            )}
          </div>
        </div>

        {/* Actions */}
        {topic.status === "completed" && topic.report_ready_at && (
          <Link
            href={`/reports/${id}`}
            className="flex items-center justify-between bg-emerald-600/10 border border-emerald-500/30 rounded-2xl p-5 active:scale-[0.99] transition-all"
          >
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-emerald-500/20 flex items-center justify-center">
                <FileText className="w-5 h-5 text-emerald-400" />
              </div>
              <div>
                <p className="text-white font-medium text-sm">查看讨论报告</p>
                <p className="text-emerald-400/70 text-xs mt-0.5">
                  已生成 · 点击查看完整内容
                </p>
              </div>
            </div>
            <ArrowLeft className="w-4 h-4 text-emerald-400 rotate-180" />
          </Link>
        )}

        {topic.status === "discussion_active" && (
          <div className="flex items-center gap-3 bg-indigo-600/10 border border-indigo-500/30 rounded-2xl p-5">
            <div className="w-10 h-10 rounded-xl bg-indigo-500/20 flex items-center justify-center">
              <MessageSquare className="w-5 h-5 text-indigo-400" />
            </div>
            <div>
              <p className="text-white font-medium text-sm">讨论进行中</p>
              <p className="text-indigo-400/70 text-xs mt-0.5">
                AI 分身们正在深度讨论你的话题
              </p>
            </div>
            <div className="ml-auto flex gap-1">
              {[0, 1, 2].map((i) => (
                <div
                  key={i}
                  className="w-1.5 h-1.5 rounded-full bg-indigo-400 animate-bounce"
                  style={{ animationDelay: `${i * 0.15}s` }}
                />
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function TimelineItem({
  label,
  value,
  sub,
  highlight,
}: {
  label: string;
  value: string;
  sub?: string;
  highlight?: boolean;
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-slate-500 text-xs">{label}</span>
      <div className="text-right">
        <span className={`text-xs font-medium ${highlight ? "text-indigo-400" : "text-slate-300"}`}>
          {value}
        </span>
        {sub && <p className="text-xs text-slate-600 mt-0.5">{sub}</p>}
      </div>
    </div>
  );
}

function TopicSkeleton() {
  return (
    <div className="min-h-screen bg-slate-950 px-4 pt-4 space-y-4">
      <Skeleton className="h-10 w-full" />
      <Skeleton className="h-48 w-full" />
      <Skeleton className="h-32 w-full" />
    </div>
  );
}

function NotFound() {
  return (
    <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center px-4">
      <p className="text-4xl mb-4">🔍</p>
      <p className="text-white font-medium mb-2">话题不存在</p>
      <Link href="/topics" className="text-indigo-400 text-sm">
        返回话题列表
      </Link>
    </div>
  );
}
