"use client";

import { use } from "react";
import useSWR from "swr";
import { useRouter } from "next/navigation";
import { discussionApi, type DiscussionMessage } from "@/lib/api";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowLeft, Brain } from "lucide-react";
import { cn } from "@/lib/utils";

interface Props {
  params: Promise<{ id: string }>;
}

const ROLE_CONFIG: Record<string, { label: string; color: string; bg: string; border: string; emoji: string }> = {
  questioner: {
    label: "质疑者",
    emoji: "🔍",
    color: "text-red-400",
    bg: "bg-red-400/10",
    border: "border-red-400/20",
  },
  supporter: {
    label: "支持者",
    emoji: "✅",
    color: "text-emerald-400",
    bg: "bg-emerald-400/10",
    border: "border-emerald-400/20",
  },
  supplementer: {
    label: "补充者",
    emoji: "💡",
    color: "text-blue-400",
    bg: "bg-blue-400/10",
    border: "border-blue-400/20",
  },
  inquirer: {
    label: "提问者",
    emoji: "❓",
    color: "text-purple-400",
    bg: "bg-purple-400/10",
    border: "border-purple-400/20",
  },
};

export default function DiscussionPage({ params }: Props) {
  const { id } = use(params);
  const router = useRouter();

  const { data: discussion, isLoading: discussionLoading } = useSWR(
    `discussion:${id}`,
    () => discussionApi.get(id)
  );
  const { data: messages, isLoading: messagesLoading } = useSWR(
    `discussion:${id}:messages`,
    () => discussionApi.getMessages(id)
  );

  const isLoading = discussionLoading || messagesLoading;

  // Group messages by round
  const rounds = groupByRound(messages ?? []);

  return (
    <div className="min-h-screen bg-slate-950">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-slate-950/95 backdrop-blur-xl border-b border-slate-800">
        <div className="px-4 py-4 flex items-center gap-3">
          <button
            onClick={() => router.back()}
            className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-slate-800 text-slate-400 hover:text-white transition-colors"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-semibold text-white">讨论记录</p>
            {discussion && (
              <p className="text-xs text-slate-500 mt-0.5">
                第 {discussion.current_round} 轮 · {STATUS_LABELS[discussion.status] ?? discussion.status}
              </p>
            )}
          </div>
          <div className="w-8 h-8 bg-indigo-600/20 border border-indigo-500/20 rounded-xl flex items-center justify-center">
            <Brain className="w-4 h-4 text-indigo-400" />
          </div>
        </div>
      </div>

      {isLoading && <DiscussionSkeleton />}

      {!isLoading && messages?.length === 0 && (
        <div className="flex flex-col items-center justify-center py-20 text-center px-4">
          <p className="text-4xl mb-4">⏳</p>
          <p className="text-white font-medium mb-2">讨论尚未开始</p>
          <p className="text-slate-400 text-sm">AI 分身们还没有发言，请稍后再查看</p>
        </div>
      )}

      {/* Participants */}
      {!isLoading && discussion?.participants && discussion.participants.length > 0 && (
        <div className="px-4 pt-4">
          <p className="text-xs font-medium text-slate-500 uppercase tracking-wider mb-3">参与分身</p>
          <div className="flex gap-2 flex-wrap mb-5">
            {discussion.participants.map((p) => {
              const cfg = ROLE_CONFIG[p.role];
              return (
                <div
                  key={p.agent_id}
                  className={cn("flex items-center gap-1.5 px-3 py-1.5 rounded-full border text-xs", cfg?.bg, cfg?.border)}
                >
                  <span>{cfg?.emoji}</span>
                  <span className={cfg?.color}>{cfg?.label}</span>
                  <span className="text-slate-500 font-mono">{p.anon_id}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Rounds */}
      <div className="px-4 space-y-6 pb-6">
        {rounds.map(({ roundNum, msgs }) => (
          <div key={roundNum}>
            <div className="flex items-center gap-3 mb-3">
              <div className="h-px flex-1 bg-slate-800" />
              <span className="text-xs font-medium text-slate-500 uppercase tracking-wider">
                第 {roundNum} 轮
              </span>
              <div className="h-px flex-1 bg-slate-800" />
            </div>
            <div className="space-y-3">
              {msgs.map((msg, i) => (
                <MessageCard key={i} msg={msg} />
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function MessageCard({ msg }: { msg: DiscussionMessage }) {
  const cfg = ROLE_CONFIG[msg.role] ?? {
    label: msg.role,
    emoji: "💬",
    color: "text-slate-400",
    bg: "bg-slate-800",
    border: "border-slate-700",
  };

  return (
    <div className={cn("rounded-2xl border p-4", cfg.bg, cfg.border)}>
      {/* Role badge + confidence */}
      <div className="flex items-center justify-between mb-3">
        <span className={cn("flex items-center gap-1.5 text-xs font-medium", cfg.color)}>
          <span>{cfg.emoji}</span>
          {cfg.label}
        </span>
        {msg.confidence > 0 && (
          <span className="text-xs text-slate-500">
            置信度 {Math.round(msg.confidence * 100)}%
          </span>
        )}
      </div>

      {/* Content */}
      <p className="text-sm text-slate-200 leading-relaxed whitespace-pre-wrap">{msg.content}</p>

      {/* Key point */}
      {msg.key_point && (
        <div className="mt-3 pt-3 border-t border-slate-700/50">
          <p className="text-xs text-slate-500 mb-1">核心论点</p>
          <p className={cn("text-xs font-medium leading-relaxed", cfg.color)}>{msg.key_point}</p>
        </div>
      )}
    </div>
  );
}

const STATUS_LABELS: Record<string, string> = {
  PENDING_MATCHING: "等待匹配",
  ROUND_1_QUEUED: "第1轮排队中",
  ROUND_1_RUNNING: "第1轮进行中",
  ROUND_1_COMPLETED: "第1轮完成",
  ROUND_2_QUEUED: "第2轮排队中",
  ROUND_2_RUNNING: "第2轮进行中",
  ROUND_2_COMPLETED: "第2轮完成",
  ROUND_3_QUEUED: "第3轮排队中",
  ROUND_3_RUNNING: "第3轮进行中",
  ROUND_3_COMPLETED: "第3轮完成",
  ROUND_4_QUEUED: "第4轮排队中",
  ROUND_4_RUNNING: "第4轮进行中",
  ROUND_4_COMPLETED: "第4轮完成",
  REPORT_GENERATING: "生成报告中",
  COMPLETED: "已完成",
  DEGRADED: "已降级",
};

function groupByRound(
  messages: DiscussionMessage[]
): { roundNum: number; msgs: DiscussionMessage[] }[] {
  const map = new Map<number, DiscussionMessage[]>();
  for (const msg of messages) {
    const r = msg.round_num;
    if (!map.has(r)) map.set(r, []);
    map.get(r)!.push(msg);
  }
  return Array.from(map.entries())
    .sort(([a], [b]) => a - b)
    .map(([roundNum, msgs]) => ({ roundNum, msgs }));
}

function DiscussionSkeleton() {
  return (
    <div className="px-4 pt-4 space-y-4 animate-pulse">
      <div className="flex gap-2">
        {[1, 2, 3, 4].map((i) => <Skeleton key={i} className="h-8 w-24 rounded-full" />)}
      </div>
      {[1, 2, 3].map((i) => (
        <div key={i}>
          <Skeleton className="h-4 w-20 mx-auto mb-3" />
          <div className="space-y-3">
            <Skeleton className="h-28 w-full rounded-2xl" />
            <Skeleton className="h-28 w-full rounded-2xl" />
          </div>
        </div>
      ))}
    </div>
  );
}
