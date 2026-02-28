"use client";

import { useState } from "react";
import useSWR, { mutate } from "swr";
import { connectionApi, type Connection } from "@/lib/api";
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
    color: "text-amber-400",
    bg: "bg-amber-400/10",
    border: "border-amber-400/20",
  },
  accepted: {
    label: "已接受",
    icon: CheckCircle,
    color: "text-emerald-400",
    bg: "bg-emerald-400/10",
    border: "border-emerald-400/20",
  },
  rejected: {
    label: "已拒绝",
    icon: XCircle,
    color: "text-slate-500",
    bg: "bg-slate-800",
    border: "border-slate-700",
  },
  cancelled: {
    label: "已取消",
    icon: XCircle,
    color: "text-slate-500",
    bg: "bg-slate-800",
    border: "border-slate-700",
  },
  expired: {
    label: "已过期",
    icon: Clock,
    color: "text-slate-500",
    bg: "bg-slate-800",
    border: "border-slate-700",
  },
};

export default function ConnectionsPage() {
  const { data: connections, isLoading } = useSWR("connections", connectionApi.list);

  const pending = connections?.filter((c) => c.status === "pending") ?? [];
  const others = connections?.filter((c) => c.status !== "pending") ?? [];

  return (
    <div className="min-h-screen bg-slate-950">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-slate-950/95 backdrop-blur-xl border-b border-slate-800">
        <div className="px-4 py-4 flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold text-white">连接请求</h1>
            <p className="text-xs text-slate-500 mt-0.5">与志同道合的人建立联系</p>
          </div>
          {pending.length > 0 && (
            <span className="bg-amber-500 text-white text-xs font-bold w-6 h-6 rounded-full flex items-center justify-center">
              {pending.length}
            </span>
          )}
        </div>
      </div>

      <div className="px-4 pt-4 space-y-4">
        {isLoading && (
          <>
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
          </>
        )}

        {!isLoading && !connections?.length && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="w-20 h-20 rounded-3xl bg-slate-900 border border-slate-700/50 flex items-center justify-center mb-6">
              <Users className="w-8 h-8 text-slate-600" />
            </div>
            <h3 className="text-white font-semibold mb-2">暂无连接请求</h3>
            <p className="text-slate-400 text-sm max-w-xs leading-relaxed">
              完成话题讨论并查看报告后，可以向推荐的匿名分身发起连接请求
            </p>
          </div>
        )}

        {/* Pending section */}
        {pending.length > 0 && (
          <div>
            <p className="text-xs font-medium text-amber-400/80 uppercase tracking-wider mb-3">
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
              <p className="text-xs font-medium text-slate-500 uppercase tracking-wider mb-3 mt-5">
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
        "rounded-2xl border p-4",
        cfg.bg,
        cfg.border
      )}
    >
      <div className="flex items-start gap-3">
        <div
          className={cn(
            "w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0",
            cfg.bg,
            "border",
            cfg.border
          )}
        >
          <StatusIcon className={cn("w-5 h-5", cfg.color)} />
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className={cn("text-xs font-medium px-2 py-0.5 rounded-full", cfg.bg, cfg.color, "border", cfg.border)}>
              {cfg.label}
            </span>
            <span className="text-xs text-slate-600">
              {formatDistanceToNow(new Date(connection.requested_at), {
                addSuffix: true,
                locale: zhCN,
              })}
            </span>
          </div>

          <p className="text-white text-sm font-medium">
            来自匿名分身的连接请求
          </p>

          {connection.request_message && (
            <p className="text-slate-400 text-xs mt-1 line-clamp-2">
              &ldquo;{connection.request_message}&rdquo;
            </p>
          )}

          <p className="text-slate-600 text-xs mt-1">
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
            className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-slate-700/50 text-slate-400 transition-colors flex-shrink-0"
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
        <div className="mt-4 pt-4 border-t border-slate-700/50 space-y-3">
          <div>
            <label className="block text-xs text-slate-400 mb-1.5">
              你的联系方式（接受后对方可见）
            </label>
            <input
              value={contact}
              onChange={(e) => setContact(e.target.value)}
              placeholder="微信 / 邮箱 / 其他"
              className="w-full bg-slate-800 border border-slate-600 rounded-xl px-3 py-2.5 text-white text-sm placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition-colors"
            />
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => handleRespond(false)}
              disabled={respondLoading}
              className="flex-1 py-2.5 rounded-xl border border-slate-600 text-slate-400 text-sm font-medium hover:bg-slate-700/50 transition-colors disabled:opacity-50 active:scale-[0.98]"
            >
              拒绝
            </button>
            <button
              onClick={() => handleRespond(true)}
              disabled={respondLoading || !contact.trim()}
              className="flex-1 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-semibold transition-all disabled:opacity-50 active:scale-[0.98]"
            >
              {respondLoading ? "处理中..." : "接受连接"}
            </button>
          </div>
        </div>
      )}

      {/* Show contacts if accepted */}
      {connection.status === "accepted" && (
        <div className="mt-3 pt-3 border-t border-emerald-500/20">
          <ContactsView connectionId={connection.id} />
        </div>
      )}
    </div>
  );
}

function ContactsView({ connectionId }: { connectionId: string }) {
  const { data } = useSWR(
    `contacts-${connectionId}`,
    () => connectionApi.getContacts(connectionId),
    { shouldRetryOnError: false }
  );

  if (!data) return null;

  return (
    <div className="space-y-2">
      {data.requester_contact && (
        <div>
          <p className="text-xs text-slate-500">对方联系方式</p>
          <p className="text-sm text-emerald-400 font-medium">{data.requester_contact}</p>
        </div>
      )}
      {data.target_contact && (
        <div>
          <p className="text-xs text-slate-500">我的联系方式（已发送）</p>
          <p className="text-sm text-slate-300">{data.target_contact}</p>
        </div>
      )}
    </div>
  );
}
