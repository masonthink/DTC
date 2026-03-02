"use client";

import { useState } from "react";
import useSWR, { mutate } from "swr";
import { connectionApi, topicApi, type Connection } from "@/lib/api";
import Link from "next/link";
import { Skeleton } from "@/components/ui/skeleton";
import { formatDistanceToNow } from "date-fns";
import { zhCN } from "date-fns/locale";
import { Users, CheckCircle, XCircle, Clock, ChevronDown, ChevronUp } from "lucide-react";
import { toast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";

const STATUS_CONFIG = {
  pending: {
    label: "待回应",
    icon: Clock,
    color: "text-amber-600",
    bg: "bg-amber-50",
    border: "border-amber-100",
  },
  accepted: {
    label: "已接受",
    icon: CheckCircle,
    color: "text-emerald-600",
    bg: "bg-emerald-50",
    border: "border-emerald-100",
  },
  rejected: {
    label: "已拒绝",
    icon: XCircle,
    color: "text-muted-foreground",
    bg: "bg-muted/60",
    border: "border-border",
  },
  cancelled: {
    label: "已取消",
    icon: XCircle,
    color: "text-muted-foreground",
    bg: "bg-muted/60",
    border: "border-border",
  },
  expired: {
    label: "已过期",
    icon: Clock,
    color: "text-muted-foreground",
    bg: "bg-muted/60",
    border: "border-border",
  },
};

export default function ConnectionsPage() {
  const { data: connections, isLoading } = useSWR("connections", connectionApi.list);

  const pending = connections?.filter((c) => c.status === "pending") ?? [];
  const others = connections?.filter((c) => c.status !== "pending") ?? [];

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-card/92 backdrop-blur-2xl border-b border-border/60">
        <div className="px-4 py-4 flex items-center justify-between">
          <div>
            <h1 className="text-[18px] font-bold text-foreground tracking-tight">我的搭子</h1>
            <p className="text-[11px] text-muted-foreground mt-0.5">通过讨论发现的志同道合的人</p>
          </div>
          {pending.length > 0 && (
            <span className="bg-amber-500 text-white text-[11px] font-bold w-6 h-6 rounded-full flex items-center justify-center tabular-nums">
              {pending.length}
            </span>
          )}
        </div>
      </div>

      <div className="px-4 pt-4 space-y-4">
        {isLoading && (
          <>
            <Skeleton className="h-28 w-full rounded-2xl" />
            <Skeleton className="h-28 w-full rounded-2xl" />
          </>
        )}

        {!isLoading && !connections?.length && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="w-20 h-20 rounded-3xl bg-muted border border-border flex items-center justify-center mb-6">
              <Users className="w-8 h-8 text-muted-foreground" />
            </div>
            <h3 className="text-foreground font-semibold text-[15px] mb-2">还没有找到搭子</h3>
            <p className="text-muted-foreground text-[13px] max-w-xs leading-relaxed">
              提交一个你正在思考的话题，分身会帮你筛选出值得认识的人
            </p>
          </div>
        )}

        {/* Pending section */}
        {pending.length > 0 && (
          <div>
            <p className="text-[11px] font-semibold text-amber-600 uppercase tracking-wider mb-3">
              待处理 ({pending.length})
            </p>
            <div className="space-y-3">
              {pending.map((c) => (
                <ConnectionCard key={c.id} connection={c} />
              ))}
            </div>
          </div>
        )}

        {/* Other connections */}
        {others.length > 0 && (
          <div>
            {pending.length > 0 && (
              <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-3 mt-5">
                历史记录
              </p>
            )}
            <div className="space-y-3">
              {others.map((c) => (
                <ConnectionCard key={c.id} connection={c} />
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function ConnectionCard({ connection }: { connection: Connection }) {
  const [expanded, setExpanded] = useState(false);
  const [respondLoading, setRespondLoading] = useState(false);
  const [contact, setContact] = useState("");
  const cfg = STATUS_CONFIG[connection.status];
  const StatusIcon = cfg.icon;

  const handleRespond = async (accept: boolean) => {
    if (accept && !contact.trim()) {
      toast({ title: "请填写联系方式", variant: "destructive" });
      return;
    }
    setRespondLoading(true);
    try {
      await connectionApi.respond(connection.id, {
        accept,
        target_contact: accept ? contact : undefined,
      });
      toast({
        title: accept ? "已接受连接请求" : "已拒绝连接请求",
        description: accept ? "对方将收到你的联系方式" : undefined,
      });
      mutate("connections");
    } catch {
      toast({ title: "操作失败，请稍后重试", variant: "destructive" });
    } finally {
      setRespondLoading(false);
    }
  };

  const isPending = connection.status === "pending";

  return (
    <div
      className={cn(
        "rounded-2xl border p-4 shadow-xs",
        cfg.bg,
        cfg.border
      )}
    >
      <div className="flex items-start gap-3">
        <div
          className={cn(
            "w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0 border",
            cfg.bg,
            cfg.border
          )}
        >
          <StatusIcon className={cn("w-4.5 h-4.5", cfg.color)} />
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className={cn("text-[11px] font-medium px-2 py-0.5 rounded-full border", cfg.bg, cfg.color, cfg.border)}>
              {cfg.label}
            </span>
            <span className="text-[11px] text-muted-foreground">
              {formatDistanceToNow(new Date(connection.requested_at), {
                addSuffix: true,
                locale: zhCN,
              })}
            </span>
          </div>

          <p className="text-foreground text-[13px] font-medium">
            来自匿名分身的连接请求
          </p>

          {connection.request_message && (
            <p className="text-muted-foreground text-[12px] mt-1 line-clamp-2 leading-relaxed">
              &ldquo;{connection.request_message}&rdquo;
            </p>
          )}

          <p className="text-muted-foreground/70 text-[11px] mt-1">
            过期时间：
            {formatDistanceToNow(new Date(connection.expires_at), {
              addSuffix: true,
              locale: zhCN,
            })}
          </p>
        </div>

        {isPending && (
          <button
            onClick={() => setExpanded((v) => !v)}
            className="w-8 h-8 flex items-center justify-center rounded-xl hover:bg-black/5 text-muted-foreground transition-colors flex-shrink-0"
          >
            {expanded ? (
              <ChevronUp className="w-4 h-4" />
            ) : (
              <ChevronDown className="w-4 h-4" />
            )}
          </button>
        )}
      </div>

      {/* Respond form */}
      {isPending && expanded && (
        <div className="mt-4 pt-4 border-t border-border/60 space-y-3 animate-unlock">
          <div>
            <label htmlFor={`contact-${connection.id}`} className="block text-[12px] text-muted-foreground mb-1.5">
              你的联系方式（接受后对方可见）
            </label>
            <input
              id={`contact-${connection.id}`}
              value={contact}
              onChange={(e) => setContact(e.target.value)}
              placeholder="微信 / 邮箱 / 其他"
              className="w-full bg-background border border-border rounded-xl px-3.5 py-2.5 text-foreground text-[13px] placeholder-muted-foreground/60 focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 transition-all duration-150"
            />
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => handleRespond(false)}
              disabled={respondLoading}
              className="flex-1 py-2.5 rounded-xl border border-border text-muted-foreground text-[13px] font-medium hover:bg-muted/60 transition-all duration-150 disabled:opacity-50 active:scale-[0.98]"
            >
              拒绝
            </button>
            <button
              onClick={() => handleRespond(true)}
              disabled={respondLoading || !contact.trim()}
              className="flex-1 py-2.5 rounded-xl bg-primary hover:bg-primary/90 text-primary-foreground text-[13px] font-semibold transition-all duration-150 disabled:opacity-50 active:scale-[0.98] shadow-primary-sm"
            >
              {respondLoading ? "处理中..." : "接受连接"}
            </button>
          </div>
        </div>
      )}

      {/* Show contacts if accepted */}
      {connection.status === "accepted" && (
        <div className="mt-3 pt-3 border-t border-emerald-200/60">
          <ContactsView connectionId={connection.id} topicId={connection.topic_id} />
        </div>
      )}
    </div>
  );
}

function ContactsView({ connectionId, topicId }: { connectionId: string; topicId?: string }) {
  const { data } = useSWR(
    `contacts-${connectionId}`,
    () => connectionApi.getContacts(connectionId),
    { shouldRetryOnError: false }
  );

  const { data: topic } = useSWR(
    topicId ? `topic-${topicId}` : null,
    () => topicApi.get(topicId!),
    { shouldRetryOnError: false }
  );

  if (!data) return null;

  return (
    <div className="space-y-2">
      {topic && (
        <div className="mb-2">
          <p className="text-[11px] text-muted-foreground mb-1">你们是如何相遇的</p>
          <Link
            href={`/topics/${topicId}`}
            className="text-[12px] text-primary font-medium hover:text-primary/80 transition-colors"
          >
            讨论话题：{topic.title} →
          </Link>
        </div>
      )}
      {data.requester_contact && (
        <div>
          <p className="text-[11px] text-muted-foreground mb-0.5">对方联系方式</p>
          <p className="text-[13px] text-emerald-600 font-medium">{data.requester_contact}</p>
        </div>
      )}
      {data.target_contact && (
        <div>
          <p className="text-[11px] text-muted-foreground mb-0.5">我的联系方式（已发送）</p>
          <p className="text-[13px] text-foreground font-medium">{data.target_contact}</p>
        </div>
      )}
    </div>
  );
}
