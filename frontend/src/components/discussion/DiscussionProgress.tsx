"use client";

import { useMemo } from "react";
import type { TopicStatus } from "@/lib/api";

interface Props {
  status: TopicStatus;
  submittedAt: string;
}

/**
 * DiscussionProgress shows the 48-hour timeline of a discussion
 * with visual indicators for each milestone.
 *
 * UX principle: 异步等待的仪式感 — users should feel something is happening.
 */
export function DiscussionProgress({ status, submittedAt }: Props) {
  const milestones = useMemo(
    () => buildMilestones(status, new Date(submittedAt)),
    [status, submittedAt]
  );

  return (
    <div className="mt-3">
      <div className="flex items-center gap-0">
        {milestones.map((m, i) => (
          <div key={m.label} className="flex items-center flex-1 last:flex-none">
            {/* Node */}
            <div className="relative flex-shrink-0">
              <div
                className={`w-3 h-3 rounded-full border-2 transition-all ${
                  m.state === "completed"
                    ? "bg-indigo-500 border-indigo-400"
                    : m.state === "active"
                    ? "bg-indigo-500 border-indigo-300 animate-pulse-glow"
                    : "bg-slate-200 border-slate-300"
                }`}
              />
              {/* Tooltip */}
              <div className="absolute bottom-4 left-1/2 -translate-x-1/2 hidden group-hover:block bg-slate-900 text-xs text-slate-300 px-2 py-1 rounded whitespace-nowrap z-10 shadow-lg">
                {m.label}
              </div>
            </div>
            {/* Connector */}
            {i < milestones.length - 1 && (
              <div
                className={`flex-1 h-0.5 ${
                  m.state === "completed" ? "bg-indigo-500" : "bg-slate-200"
                }`}
              />
            )}
          </div>
        ))}
      </div>
      {/* Labels */}
      <div className="flex justify-between mt-1.5">
        {milestones.map((m) => (
          <span
            key={m.label}
            className={`text-[10px] ${
              m.state === "active"
                ? "text-indigo-400 font-medium"
                : m.state === "completed"
                ? "text-slate-500"
                : "text-slate-600"
            }`}
          >
            {m.shortLabel}
          </span>
        ))}
      </div>
      {/* Time estimate */}
      <TimeEstimate status={status} submittedAt={new Date(submittedAt)} />
    </div>
  );
}

type MilestoneState = "completed" | "active" | "upcoming";

interface Milestone {
  label: string;
  shortLabel: string;
  state: MilestoneState;
}

function buildMilestones(status: TopicStatus, submittedAt: Date): Milestone[] {
  const order: TopicStatus[] = [
    "pending_matching",
    "matched",
    "discussion_active",
    "report_generating",
    "completed",
  ];
  const currentIdx = order.indexOf(status);

  return [
    { label: "话题提交", shortLabel: "提交" },
    { label: "匹配分身", shortLabel: "匹配" },
    { label: "讨论进行", shortLabel: "讨论" },
    { label: "生成报告", shortLabel: "报告" },
    { label: "完成", shortLabel: "完成" },
  ].map((m, i) => ({
    ...m,
    state: (i < currentIdx
      ? "completed"
      : i === currentIdx
      ? "active"
      : "upcoming") as MilestoneState,
  }));
}

function TimeEstimate({ status, submittedAt }: { status: TopicStatus; submittedAt: Date }) {
  const now = new Date();
  const hoursElapsed = (now.getTime() - submittedAt.getTime()) / 3_600_000;

  const messages: Partial<Record<TopicStatus, string>> = {
    pending_matching: "正在为你寻找最合适的分身...",
    matching: "正在为你寻找最合适的分身...",
    matched: `匹配完成，讨论将在 ${Math.max(0, Math.round(1.5 - hoursElapsed))} 小时后开始`,
    discussion_active: `讨论进行中，报告将在约 ${Math.max(0, Math.round(48 - hoursElapsed))} 小时后就绪`,
    report_generating: "正在生成你的专属分析报告...",
  };

  const msg = messages[status];
  if (!msg) return null;

  return (
    <p className="text-xs text-slate-500 mt-2 flex items-center gap-1">
      <span className="w-1.5 h-1.5 rounded-full bg-indigo-500 animate-pulse flex-shrink-0" />
      {msg}
    </p>
  );
}
