"use client";

import { useState } from "react";
import useSWR from "swr";
import Link from "next/link";
import { topicApi, type Topic } from "@/lib/api";
import { TopicStatusBadge } from "@/components/topic/TopicStatusBadge";
import { Skeleton } from "@/components/ui/skeleton";
import { formatDistanceToNow } from "date-fns";
import { zhCN } from "date-fns/locale";
import { Plus } from "lucide-react";

type Tab = "active" | "completed" | "all";

export default function TopicsPage() {
  const [tab, setTab] = useState<Tab>("active");
  const { data, isLoading } = useSWR("topics-all", () =>
    topicApi.list({ limit: 50 })
  );

  const items = data?.items ?? [];
  const activeItems = items.filter(
    (t) => !["completed", "cancelled", "failed"].includes(t.status)
  );
  const completedItems = items.filter((t) => t.status === "completed");
  const displayed =
    tab === "active" ? activeItems : tab === "completed" ? completedItems : items;

  const tabs: { key: Tab; label: string; count?: number }[] = [
    { key: "active", label: "进行中", count: activeItems.length },
    { key: "completed", label: "已完成", count: completedItems.length },
    { key: "all", label: "全部", count: items.length },
  ];

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="sticky top-0 md:top-14 z-10 bg-card/92 backdrop-blur-2xl border-b border-border/60">
        <div className="px-4 pt-4 pb-0 flex items-center justify-between">
          <h1 className="text-[18px] font-bold text-foreground tracking-tight">我的想法</h1>
          <Link
            href="/topics/submit"
            className="flex items-center gap-1.5 text-primary text-[13px] font-medium active:opacity-70 hover:text-primary/80 transition-colors"
          >
            <Plus className="w-4 h-4" />
            提交想法
          </Link>
        </div>

        {/* Tabs */}
        <div className="flex px-4 pt-3">
          {tabs.map((t) => (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`flex items-center gap-1.5 pb-3 px-1 mr-6 text-[13px] font-medium border-b-2 transition-all duration-150 ${
                tab === t.key
                  ? "text-foreground border-primary"
                  : "text-muted-foreground border-transparent hover:text-foreground/70"
              }`}
            >
              {t.label}
              {t.count !== undefined && t.count > 0 && (
                <span
                  className={`text-[11px] px-1.5 py-0.5 rounded-full font-semibold tabular-nums ${
                    tab === t.key
                      ? "bg-primary/15 text-primary"
                      : "bg-muted text-muted-foreground"
                  }`}
                >
                  {t.count}
                </span>
              )}
            </button>
          ))}
        </div>
      </div>

      <div className="px-4 pt-4 space-y-3">
        {isLoading && (
          <>
            <Skeleton className="h-24 w-full rounded-2xl" />
            <Skeleton className="h-24 w-full rounded-2xl" />
            <Skeleton className="h-24 w-full rounded-2xl" />
          </>
        )}

        {!isLoading && displayed.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="w-14 h-14 rounded-2xl bg-muted flex items-center justify-center mb-4 text-2xl">
              {tab === "active" ? "⏳" : tab === "completed" ? "✅" : "💬"}
            </div>
            <p className="text-muted-foreground text-[13px]">
              {tab === "active" ? "暂无进行中的想法" : tab === "completed" ? "暂无已完成的想法" : "提交一个想法，让分身帮你找搭子"}
            </p>
            {tab !== "completed" && (
              <Link
                href="/topics/submit"
                className="mt-6 bg-primary hover:bg-primary/90 text-primary-foreground px-6 py-2.5 rounded-xl text-[13px] font-medium transition-all active:scale-95 shadow-primary-sm"
              >
                提交第一个想法
              </Link>
            )}
          </div>
        )}

        {displayed.map((topic) => (
          <TopicCard key={topic.id} topic={topic} />
        ))}
      </div>
    </div>
  );
}

function TopicCard({ topic }: { topic: Topic }) {
  const isCompleted = topic.status === "completed";

  return (
    <Link
      href={`/topics/${topic.id}`}
      className={`block rounded-2xl border p-4 transition-all duration-200 active:scale-[0.99] shadow-xs ${
        isCompleted
          ? "bg-muted/50 border-border/60 hover:border-border"
          : "bg-card border-border hover:border-primary/25 hover:shadow-sm"
      }`}
    >
      <div className="flex items-start justify-between gap-3 mb-2.5">
        <h3 className="text-foreground text-[14px] font-medium line-clamp-2 flex-1 leading-snug">
          {topic.title}
        </h3>
        <TopicStatusBadge status={topic.status} />
      </div>

      {topic.description && (
        <p className="text-muted-foreground text-[12px] line-clamp-2 mb-3 leading-relaxed">
          {topic.description}
        </p>
      )}

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-[11px] text-foreground/60 bg-muted px-2 py-0.5 rounded-full">
            {TOPIC_TYPE_LABELS[topic.topic_type] ?? topic.topic_type}
          </span>
          {topic.tags?.slice(0, 2).map((tag) => (
            <span key={tag} className="text-[11px] text-muted-foreground">
              #{tag}
            </span>
          ))}
        </div>
        <span className="text-[11px] text-muted-foreground flex-shrink-0">
          {formatDistanceToNow(new Date(topic.submitted_at), {
            addSuffix: true,
            locale: zhCN,
          })}
        </span>
      </div>

      {isCompleted && topic.report_ready_at && (
        <div className="mt-3 pt-3 border-t border-border/60 flex items-center gap-2">
          <svg width="13" height="13" viewBox="0 0 13 13" fill="none" className="text-emerald-500 flex-shrink-0">
            <path d="M2 6.5l3 3 6-6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
          <span className="text-[12px] text-emerald-500 font-medium">报告已生成</span>
          <span className="text-[12px] text-muted-foreground ml-auto">查看报告 →</span>
        </div>
      )}
    </Link>
  );
}

const TOPIC_TYPE_LABELS: Record<string, string> = {
  business_idea: "商业想法",
  career_decision: "职业决策",
  tech_choice: "技术选型",
  product_design: "产品设计",
  investment: "投资判断",
  other: "其他",
};
