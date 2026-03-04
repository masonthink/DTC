"use client";

import useSWR from "swr";
import { useParams, useRouter } from "next/navigation";
import { discussionApi, type DiscussionMessage } from "@/lib/api";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowLeft, Brain } from "lucide-react";
import { cn } from "@/lib/utils";

// Color palette for distinguishing participants (by index, not role)
const PARTICIPANT_COLORS = [
  { color: "text-red-600", bg: "bg-red-50", border: "border-red-100", dot: "bg-red-400" },
  { color: "text-emerald-600", bg: "bg-emerald-50", border: "border-emerald-100", dot: "bg-emerald-400" },
  { color: "text-blue-600", bg: "bg-blue-50", border: "border-blue-100", dot: "bg-blue-400" },
  { color: "text-violet-600", bg: "bg-violet-50", border: "border-violet-100", dot: "bg-violet-400" },
];

export default function DiscussionPage() {
  const { id } = useParams<{ id: string }>();
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

  // Build agent_id → index mapping for consistent colors
  const agentIndexMap = new Map<string, number>();
  discussion?.participants?.forEach((p, i) => {
    agentIndexMap.set(p.agent_id, i);
  });

  // Build agent_id → anon_id mapping
  const agentAnonMap = new Map<string, string>();
  discussion?.participants?.forEach((p) => {
    agentAnonMap.set(p.agent_id, p.anon_id);
  });

  // Group messages by round
  const rounds = groupByRound(messages ?? []);

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-card/92 backdrop-blur-2xl border-b border-border/60">
        <div className="px-4 py-4 flex items-center gap-3">
          <button
            onClick={() => router.back()}
            className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-muted text-muted-foreground hover:text-foreground transition-all duration-150"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <div className="flex-1 min-w-0">
            <p className="text-[14px] font-semibold text-foreground">讨论记录</p>
            {discussion && (
              <p className="text-[11px] text-muted-foreground mt-0.5">
                第 {discussion.current_round} 轮 · {STATUS_LABELS[discussion.status] ?? discussion.status}
              </p>
            )}
          </div>
          <div className="w-8 h-8 bg-primary/10 border border-primary/20 rounded-xl flex items-center justify-center">
            <Brain className="w-4 h-4 text-primary" />
          </div>
        </div>
      </div>

      {isLoading && <DiscussionSkeleton />}

      {!isLoading && messages?.length === 0 && (
        <div className="flex flex-col items-center justify-center py-20 text-center px-4">
          <div className="w-14 h-14 rounded-2xl bg-muted flex items-center justify-center mb-4 text-2xl">
            ⏳
          </div>
          <p className="text-foreground font-medium text-[15px] mb-1.5">讨论尚未开始</p>
          <p className="text-muted-foreground text-[13px]">AI 分身们还没有发言，请稍后再查看</p>
        </div>
      )}

      {/* Participants */}
      {!isLoading && discussion?.participants && discussion.participants.length > 0 && (
        <div className="px-4 pt-5">
          <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-3">参与分身</p>
          <div className="flex gap-2 flex-wrap mb-5">
            {discussion.participants.map((p, i) => {
              const palette = PARTICIPANT_COLORS[i % PARTICIPANT_COLORS.length];
              return (
                <div
                  key={p.agent_id}
                  className={cn(
                    "flex items-center gap-1.5 px-2.5 py-1.5 rounded-full border text-[11px] font-medium",
                    palette.bg, palette.border, palette.color
                  )}
                >
                  <span className={cn("w-2 h-2 rounded-full", palette.dot)} />
                  <span className="font-mono">{p.anon_id}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Rounds */}
      <div className="px-4 space-y-8 pb-6">
        {rounds.map(({ roundNum, msgs }) => (
          <div key={roundNum}>
            {/* Round divider */}
            <div className="flex items-center gap-3 mb-4">
              <div className="h-px flex-1 bg-border" />
              <div className="flex items-center gap-2 px-3 py-1 bg-muted rounded-full">
                <div className="w-1.5 h-1.5 rounded-full bg-primary" />
                <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
                  第 {roundNum} 轮
                </span>
              </div>
              <div className="h-px flex-1 bg-border" />
            </div>

            <div className="space-y-3">
              {msgs.map((msg, i) => (
                <MessageCard
                  key={i}
                  msg={msg}
                  anonId={agentAnonMap.get(msg.agent_id) ?? msg.agent_id.slice(0, 6)}
                  colorIndex={agentIndexMap.get(msg.agent_id) ?? 0}
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function MessageCard({ msg, anonId, colorIndex }: { msg: DiscussionMessage; anonId: string; colorIndex: number }) {
  const palette = PARTICIPANT_COLORS[colorIndex % PARTICIPANT_COLORS.length];

  return (
    <div className={cn("rounded-2xl border p-4", palette.bg, palette.border)}>
      {/* Speaker badge + confidence */}
      <div className="flex items-center justify-between mb-3">
        <span className={cn("flex items-center gap-1.5 text-[12px] font-semibold", palette.color)}>
          <span className={cn("w-2 h-2 rounded-full", palette.dot)} />
          <span className="font-mono">{anonId}</span>
        </span>
        {msg.confidence > 0 && (
          <span className="text-[11px] text-muted-foreground bg-white/60 px-2 py-0.5 rounded-full border border-white/80">
            置信度 {Math.round(msg.confidence * 100)}%
          </span>
        )}
      </div>

      {/* Structured content */}
      <StructuredContent content={msg.content} accentColor={palette.color} />

      {/* Key point */}
      {msg.key_point && (
        <div className="mt-3 pt-3 border-t border-white/50">
          <p className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider mb-1">核心论点</p>
          <p className={cn("text-[12px] font-medium leading-relaxed", palette.color)}>{msg.key_point}</p>
        </div>
      )}
    </div>
  );
}

// Parse structured content sections like 【立场】, 【论据】, 【回应】, 【延伸问题】
function parseStructuredSections(content: string): { label: string; body: string }[] | null {
  const sectionPattern = /【([^】]+)】/g;
  const matches: RegExpExecArray[] = [];
  let m: RegExpExecArray | null;
  while ((m = sectionPattern.exec(content)) !== null) {
    matches.push(m);
  }
  if (matches.length === 0) return null; // fallback to plain text

  const sections: { label: string; body: string }[] = [];
  for (let i = 0; i < matches.length; i++) {
    const label = matches[i][1];
    const start = matches[i].index + matches[i][0].length;
    const end = i + 1 < matches.length ? matches[i + 1].index : content.length;
    sections.push({ label, body: content.slice(start, end).trim() });
  }
  return sections;
}

const SECTION_STYLES: Record<string, { icon: string; labelColor: string }> = {
  "立场": { icon: "🎯", labelColor: "text-primary" },
  "论据": { icon: "📊", labelColor: "text-blue-600" },
  "回应": { icon: "💬", labelColor: "text-amber-600" },
  "延伸问题": { icon: "❓", labelColor: "text-violet-600" },
};

function StructuredContent({ content, accentColor }: { content: string; accentColor: string }) {
  const sections = parseStructuredSections(content);

  // Fallback: render as plain text if not structured
  if (!sections) {
    return <p className="text-[13px] text-foreground/85 leading-relaxed whitespace-pre-wrap">{content}</p>;
  }

  return (
    <div className="space-y-2.5">
      {sections.map((sec, i) => {
        const style = SECTION_STYLES[sec.label] ?? { icon: "•", labelColor: accentColor };
        return (
          <div key={i}>
            <p className={cn("text-[11px] font-semibold mb-1 flex items-center gap-1", style.labelColor)}>
              <span>{style.icon}</span> {sec.label}
            </p>
            <div className="text-[13px] text-foreground/80 leading-relaxed whitespace-pre-wrap pl-4">
              {sec.body}
            </div>
          </div>
        );
      })}
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
    <div className="px-4 pt-5 space-y-4">
      <div className="flex gap-2">
        {[1, 2, 3, 4].map((i) => <Skeleton key={i} className="h-8 w-24 rounded-full" />)}
      </div>
      {[1, 2, 3].map((i) => (
        <div key={i}>
          <Skeleton className="h-4 w-20 mx-auto mb-4 rounded-full" />
          <div className="space-y-3">
            <Skeleton className="h-28 w-full rounded-2xl" />
            <Skeleton className="h-28 w-full rounded-2xl" />
          </div>
        </div>
      ))}
    </div>
  );
}
