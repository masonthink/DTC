import type { TopicStatus } from "@/lib/api";

interface Props {
  status: TopicStatus;
}

const STATUS_CONFIG: Record<
  TopicStatus,
  { label: string; classes: string; pulse?: boolean }
> = {
  pending_matching: {
    label: "匹配中",
    classes: "bg-amber-50 text-amber-600 border-amber-200/80",
    pulse: true,
  },
  matching: {
    label: "匹配中",
    classes: "bg-amber-50 text-amber-600 border-amber-200/80",
    pulse: true,
  },
  matched: {
    label: "已匹配",
    classes: "bg-blue-50 text-blue-600 border-blue-200/80",
  },
  discussion_active: {
    label: "讨论中",
    classes: "bg-primary/8 text-primary border-primary/20",
    pulse: true,
  },
  report_generating: {
    label: "生成报告",
    classes: "bg-violet-50 text-violet-600 border-violet-200/80",
    pulse: true,
  },
  completed: {
    label: "已完成",
    classes: "bg-emerald-50 text-emerald-600 border-emerald-200/80",
  },
  failed: {
    label: "失败",
    classes: "bg-red-50 text-red-500 border-red-200/80",
  },
  cancelled: {
    label: "已取消",
    classes: "bg-muted text-muted-foreground border-border",
  },
};

export function TopicStatusBadge({ status }: Props) {
  const config = STATUS_CONFIG[status] ?? STATUS_CONFIG.cancelled;
  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium border flex-shrink-0 ${config.classes}`}
    >
      {config.pulse && (
        <span className="w-1.5 h-1.5 rounded-full bg-current animate-pulse" />
      )}
      {config.label}
    </span>
  );
}
