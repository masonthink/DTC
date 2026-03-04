"use client";

import useSWR from "swr";
import { useParams } from "next/navigation";
import Link from "next/link";
import { topicApi, type Topic } from "@/lib/api";
import { TopicStatusBadge } from "@/components/topic/TopicStatusBadge";
import { DiscussionProgress } from "@/components/discussion/DiscussionProgress";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowLeft, FileText, MessageSquare } from "lucide-react";
import { formatDistanceToNow, format } from "date-fns";
import { zhCN } from "date-fns/locale";

const TOPIC_TYPE_CONFIG: Record<string, { label: string; emoji: string }> = {
  business_idea: { label: "商业想法", emoji: "💡" },
  career_decision: { label: "职业决策", emoji: "💼" },
  tech_choice: { label: "技术选型", emoji: "⚙️" },
  product_design: { label: "产品设计", emoji: "🎨" },
  investment: { label: "投资判断", emoji: "📈" },
  other: { label: "其他", emoji: "💬" },
};

export default function TopicDetailPage() {
  const { id } = useParams<{ id: string }>();

  const { data: topic, isLoading } = useSWR(
    id ? `topics/${id}` : null,
    () => topicApi.get(id)
  );

  if (isLoading) return <TopicSkeleton />;
  if (!topic) return <NotFound />;

  const typeConfig = TOPIC_TYPE_CONFIG[topic.topic_type] ?? { label: topic.topic_type, emoji: "💬" };

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-card/92 backdrop-blur-2xl border-b border-border/60">
        <div className="px-4 py-4 flex items-center gap-3">
          <Link
            href="/topics"
            className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-muted text-muted-foreground hover:text-foreground transition-all duration-150"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <div className="flex-1 min-w-0">
            <p className="text-[11px] text-muted-foreground">想法详情</p>
          </div>
          <TopicStatusBadge status={topic.status} />
        </div>
      </div>

      <div className="px-4 pt-5 space-y-4 pb-6">
        {/* Title card */}
        <div className="bg-card border border-border rounded-2xl p-5 shadow-xs">
          <div className="flex items-start gap-3 mb-3">
            <div className="w-10 h-10 rounded-xl bg-muted flex items-center justify-center text-xl flex-shrink-0">
              {typeConfig.emoji}
            </div>
            <div className="flex-1">
              <p className="text-[11px] text-muted-foreground mb-1 font-medium">{typeConfig.label}</p>
              <h2 className="text-foreground font-semibold text-[15px] leading-snug">
                {topic.title}
              </h2>
            </div>
          </div>

          {topic.description && (
            <p className="text-muted-foreground text-[13px] leading-relaxed">
              {topic.description}
            </p>
          )}

          {topic.background && (
            <div className="mt-3 pt-3 border-t border-border/60">
              <p className="text-[11px] font-medium text-muted-foreground mb-1.5">背景信息</p>
              <p className="text-muted-foreground text-[13px] leading-relaxed">{topic.background}</p>
            </div>
          )}

          {topic.tags && topic.tags.length > 0 && (
            <div className="flex flex-wrap gap-2 mt-3 pt-3 border-t border-border/60">
              {topic.tags.map((tag) => (
                <span
                  key={tag}
                  className="text-[11px] text-muted-foreground bg-muted px-2.5 py-1 rounded-full border border-border"
                >
                  #{tag}
                </span>
              ))}
            </div>
          )}
        </div>

        {/* Progress */}
        <div className="bg-card border border-border rounded-2xl p-5 shadow-xs">
          <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-4">
            进度追踪
          </p>
          <DiscussionProgress
            status={topic.status}
            submittedAt={topic.submitted_at}
          />
        </div>

        {/* Timeline */}
        <div className="bg-card border border-border rounded-2xl p-5 shadow-xs">
          <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-4">
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

        {/* Report CTA */}
        {topic.status === "completed" && topic.report_ready_at && (
          <Link
            href={`/reports/${id}`}
            className="flex items-center justify-between bg-emerald-50 border border-emerald-100 rounded-2xl p-4 active:scale-[0.99] transition-all duration-150 shadow-xs group"
          >
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-emerald-100 flex items-center justify-center">
                <FileText className="w-5 h-5 text-emerald-600" />
              </div>
              <div>
                <p className="text-foreground font-semibold text-[13px]">查看讨论报告</p>
                <p className="text-emerald-600/70 text-[11px] mt-0.5">
                  已生成 · 点击查看完整内容
                </p>
              </div>
            </div>
            <ArrowLeft className="w-4 h-4 text-emerald-500 rotate-180 group-hover:translate-x-0.5 transition-transform duration-150" />
          </Link>
        )}

        {/* Discussion CTA */}
        {(topic.status === "discussion_active" || topic.status === "report_generating" || topic.status === "completed") && topic.discussion_id && (
          <Link
            href={`/discussions/${topic.discussion_id}`}
            className="flex items-center gap-3 bg-primary/5 border border-primary/15 rounded-2xl p-4 active:scale-[0.99] transition-all duration-150 shadow-xs group"
          >
            <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
              <MessageSquare className="w-5 h-5 text-primary" />
            </div>
            <div className="flex-1">
              <p className="text-foreground font-semibold text-[13px]">
                {topic.status === "discussion_active" ? "讨论进行中" : "查看讨论记录"}
              </p>
              <p className="text-primary/60 text-[11px] mt-0.5">
                {topic.status === "discussion_active"
                  ? "AI 分身们正在深度讨论你的想法"
                  : "查看 AI 分身们的完整讨论过程"}
              </p>
            </div>
            {topic.status === "discussion_active" ? (
              <div className="flex gap-1 items-center">
                {[0, 1, 2].map((i) => (
                  <div
                    key={i}
                    className="w-1.5 h-1.5 rounded-full bg-primary animate-bounce"
                    style={{ animationDelay: `${i * 0.15}s` }}
                  />
                ))}
              </div>
            ) : (
              <ArrowLeft className="w-4 h-4 text-primary rotate-180 group-hover:translate-x-0.5 transition-transform duration-150" />
            )}
          </Link>
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
      <span className="text-muted-foreground text-[12px]">{label}</span>
      <div className="text-right">
        <span className={`text-[12px] font-medium ${highlight ? "text-primary" : "text-foreground"}`}>
          {value}
        </span>
        {sub && <p className="text-[11px] text-muted-foreground mt-0.5">{sub}</p>}
      </div>
    </div>
  );
}

function TopicSkeleton() {
  return (
    <div className="min-h-screen bg-background px-4 pt-4 space-y-4">
      <Skeleton className="h-10 w-full rounded-xl" />
      <Skeleton className="h-48 w-full rounded-2xl" />
      <Skeleton className="h-32 w-full rounded-2xl" />
    </div>
  );
}

function NotFound() {
  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center px-4">
      <div className="w-16 h-16 rounded-2xl bg-muted border border-border flex items-center justify-center mb-4 text-2xl">
        🔍
      </div>
      <p className="text-foreground font-semibold text-[15px] mb-2">想法不存在</p>
      <Link href="/topics" className="text-primary text-[13px] hover:text-primary/80 transition-colors">
        返回想法列表
      </Link>
    </div>
  );
}
