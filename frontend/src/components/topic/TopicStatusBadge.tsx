import type { TopicStatus } from "@/lib/api";

interface Props {
  status: TopicStatus;
}

const STATUS_CONFIG: Record<
  TopicStatus,
  { label: string; classes: string }
> = {
  pending_matching: {
    label: "匹配中",
    classes: "bg-amber-400/10 text-amber-400 border-amber-400/20",
  },
  matching: {
    label: "匹配中",
    classes: "bg-amber-400/10 text-amber-400 border-amber-400/20",
  },
  matched: {
    label: "已匹配",
    classes: "bg-blue-400/10 text-blue-400 border-blue-400/20",
  },
  discussion_active: {
    label: "讨论中",
    classes: "bg-indigo-400/10 text-indigo-400 border-indigo-400/20",
  },
  report_generating: {
    label: "生成报告",
    classes: "bg-purple-400/10 text-purple-400 border-purple-400/20",
  },
  completed: {
    label: "已完成",
    classes: "bg-emerald-400/10 text-emerald-400 border-emerald-400/20",
  },
  failed: {
    label: "失败",
    classes: "bg-red-400/10 text-red-400 border-red-400/20",
  },
  cancelled: {
    label: "已取消",
    classes: "bg-slate-400/10 text-slate-400 border-slate-400/20",
  },
};

export function TopicStatusBadge({ status }: Props) {
  const config = STATUS_CONFIG[status] ?? STATUS_CONFIG.cancelled;
  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs border flex-shrink-0 ${config.classes}`}
    >
      {["matching", "discussion_active", "report_generating"].includes(status) && (
        <span className="w-1.5 h-1.5 rounded-full bg-current animate-pulse" />
      )}
      {config.label}
    </span>
  );
}
