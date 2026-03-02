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
    <div className="min-h-screen bg-slate-50">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-white/95 backdrop-blur-xl border-b border-slate-200">
        <div className="px-4 pt-4 pb-0 flex items-center justify-between">
          <h1 className="text-xl font-bold text-slate-900">我的话题</h1>
          <Link
            href="/topics/submit"
            className="flex items-center gap-1.5 text-indigo-400 text-sm font-medium active:opacity-70"
          >
            <Plus className="w-4 h-4" />
            提交
          </Link>
        </div>

        {/* Tabs */}
        <div className="flex px-4 pt-3">
          {tabs.map((t) => (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`flex items-center gap-1.5 pb-3 px-1 mr-6 text-sm font-medium border-b-2 transition-colors ${
                tab === t.key
                  ? "text-slate-900 border-indigo-500"
                  : "text-slate-500 border-transparent hover:text-slate-700"
              }`}
            >
              {t.label}
              {t.count !== undefined && t.count > 0 && (
                <span
                  className={`text-xs px-1.5 py-0.5 rounded-full ${
                    tab === t.key
                      ? "bg-indigo-600/30 text-indigo-700"
                      : "bg-slate-300 text-slate-400"
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
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-24 w-full" />
          </>
        )}

        {!isLoading && displayed.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="text-4xl mb-4">
              {tab === "active" ? "⏳" : tab === "completed" ? "✅" : "💬"}
            </div>
            <p className="text-slate-400 text-sm">
              {tab === "active" ? "暂无进行中的话题" : tab === "completed" ? "暂无已完成的话题" : "还没有话题"}
            </p>
            {tab !== "completed" && (
              <Link
                href="/topics/submit"
                className="mt-6 bg-indigo-600 hover:bg-indigo-500 text-white px-6 py-2.5 rounded-xl text-sm font-medium transition-colors active:scale-95"
              >
                提交第一个话题
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
      className={`block rounded-2xl border p-4 transition-all active:scale-[0.99] ${
        isCompleted
          ? "bg-slate-50 border-slate-200/60 hover:border-slate-300/60"
          : "bg-white border-slate-200 hover:border-slate-300"
      }`}
    >
      <div className="flex items-start justify-between gap-3 mb-2.5">
        <h3 className="text-slate-900 text-sm font-medium line-clamp-2 flex-1">
          {topic.title}
        </h3>
        <TopicStatusBadge status={topic.status} />
      </div>

      {topic.description && (
        <p className="text-slate-500 text-xs line-clamp-2 mb-3">
          {topic.description}
        </p>
      )}

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs text-slate-600 bg-slate-100 px-2 py-0.5 rounded-full">
            {TOPIC_TYPE_LABELS[topic.topic_type] ?? topic.topic_type}
          </span>
          {topic.tags?.slice(0, 2).map((tag) => (
            <span key={tag} className="text-xs text-slate-500">
              #{tag}
            </span>
          ))}
        </div>
        <span className="text-xs text-slate-600 flex-shrink-0">
          {formatDistanceToNow(new Date(topic.submitted_at), {
            addSuffix: true,
            locale: zhCN,
          })}
        </span>
      </div>

      {isCompleted && topic.report_ready_at && (
        <div className="mt-3 pt-3 border-t border-slate-200 flex items-center gap-2">
          <span className="text-xs text-emerald-400">✓ 报告已生成</span>
          <span className="text-xs text-slate-600">→ 查看报告</span>
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
