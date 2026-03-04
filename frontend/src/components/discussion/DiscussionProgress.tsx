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
                className={`w-2.5 h-2.5 rounded-full border-2 transition-all duration-300 ${
                  m.state === "completed"
                    ? "bg-primary border-primary"
                    : m.state === "active"
                    ? "bg-primary border-primary/40 animate-pulse-glow"
                    : "bg-muted border-border"
                }`}
              />
            </div>
            {/* Connector */}
            {i < milestones.length - 1 && (
              <div
                className={`flex-1 h-0.5 transition-all duration-300 ${
                  m.state === "completed" ? "bg-primary" : "bg-border"
                }`}
              />
            )}
          </div>
        ))}
      </div>

      {/* Labels */}
      <div className="flex justify-between mt-2">
        {milestones.map((m) => (
          <span
            key={m.label}
            className={`text-[10px] font-medium leading-none ${
              m.state === "active"
                ? "text-primary"
                : m.state === "completed"
                ? "text-muted-foreground"
                : "text-muted-foreground/50"
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
    { label: "想法提交", shortLabel: "提交" },
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
    pending_matching: "正在匹配四个讨论分身...",
    matching: "正在匹配四个讨论分身...",
    matched: "分身匹配就绪，讨论即将开始",
    discussion_active: "四个分身正在深度讨论中...",
    report_generating: "正在生成你的专属讨论报告...",
  };

  const msg = messages[status];
  if (!msg) return null;

  return (
    <p className="text-[11px] text-muted-foreground mt-2.5 flex items-center gap-1.5">
      <span className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse flex-shrink-0" />
      {msg}
    </p>
  );
}
