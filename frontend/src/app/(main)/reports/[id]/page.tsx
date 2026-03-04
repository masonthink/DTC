"use client";

import useSWR from "swr";
import { reportApi, connectionApi, extractApiError, type Report, type RecommendedAgent } from "@/lib/api";
import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { toast } from "@/hooks/use-toast";

export default function ReportPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { data: report, isLoading } = useSWR(`report:${id}`, () => reportApi.get(id));

  if (isLoading) return <ReportSkeleton />;
  if (!report) return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <p className="text-muted-foreground text-[14px]">报告不存在</p>
    </div>
  );

  return (
    <div className="min-h-screen bg-background">
      {/* Sticky header */}
      <div className="sticky top-0 z-10 bg-card/92 backdrop-blur-2xl border-b border-border/60">
        <div className="max-w-3xl mx-auto px-4 py-4 flex items-center gap-3">
          <button
            onClick={() => router.back()}
            className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-muted text-muted-foreground hover:text-foreground transition-all duration-150"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <p className="flex-1 text-[14px] font-semibold text-foreground">讨论分析报告</p>
        </div>
      </div>

      <div className="max-w-3xl mx-auto px-4 py-6 space-y-5">
        {/* Report meta */}
        <div className="bg-card border border-border rounded-2xl p-5 shadow-xs">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-[11px] font-semibold text-primary uppercase tracking-wider mb-1">分析报告</p>
              <p className="text-[12px] text-muted-foreground">
                {new Date(report.generated_at).toLocaleDateString("zh-CN", {
                  year: "numeric",
                  month: "long",
                  day: "numeric",
                })} 生成
              </p>
            </div>
            <div className="text-right">
              <p className="text-[11px] text-muted-foreground mb-0.5">质量评分</p>
              <QualityScore score={report.quality_score} />
            </div>
          </div>
        </div>

        {/* Summary */}
        <section className="bg-card border border-border rounded-2xl p-5 shadow-xs">
          <h2 className="text-[14px] font-semibold text-foreground mb-3 flex items-center gap-2">
            <span className="w-5 h-5 rounded bg-primary/10 flex items-center justify-center text-xs">📋</span>
            讨论摘要
          </h2>
          <div className="text-[13px] text-foreground/80 leading-relaxed whitespace-pre-line">
            {report.summary}
          </div>
        </section>

        {/* Recommended connections — 前置，最高视觉权重 */}
        {report.recommended_agents.length > 0 && (
          <section className="bg-gradient-to-br from-primary/8 via-violet-50 to-primary/5 border border-primary/20 rounded-2xl p-5 shadow-sm">
            <h2 className="text-[15px] font-bold text-foreground mb-1 flex items-center gap-2">
              <span>🤝</span> 本次讨论最值得认识的搭子
            </h2>
            <p className="text-[12px] text-muted-foreground mb-4">讨论中发现了和你观点互补、背景契合的人</p>
            <div className="space-y-3">
              {report.recommended_agents.map((agent) => (
                <RecommendedAgentCard key={agent.agent_id} agent={agent} />
              ))}
            </div>
          </section>
        )}

        {/* Opinion matrix */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <OpinionSection
            icon="✅"
            title="核心共识"
            items={report.consensus_points}
            color="emerald"
          />
          <OpinionSection
            icon="⚡"
            title="主要分歧"
            items={report.divergence_points}
            color="amber"
          />
          <OpinionSection
            icon="❓"
            title="关键疑问"
            items={report.key_questions}
            color="blue"
          />
          <OpinionSection
            icon="🎯"
            title="行动建议"
            items={report.action_items}
            color="indigo"
          />
        </div>

        {/* Blind spots */}
        {report.blind_spots.length > 0 && (
          <section className="bg-muted/60 border border-border rounded-2xl p-5">
            <h3 className="text-[13px] font-semibold text-foreground mb-3 flex items-center gap-2">
              <span>🔍</span> 值得关注的盲点
            </h3>
            <ul className="space-y-2.5">
              {report.blind_spots.map((item, i) => (
                <li key={i} className="text-muted-foreground text-[13px] flex items-start gap-2.5 leading-relaxed">
                  <span className="w-1 h-1 rounded-full bg-muted-foreground/50 mt-2 flex-shrink-0" />
                  {item}
                </li>
              ))}
            </ul>
          </section>
        )}

        {/* User rating */}
        <UserRating reportId={id} currentRating={report.user_rating} />
      </div>
    </div>
  );
}

function OpinionSection({
  icon,
  title,
  items,
  color,
}: {
  icon: string;
  title: string;
  items: string[];
  color: "emerald" | "amber" | "blue" | "indigo";
}) {
  const colorMap = {
    emerald: "border-emerald-100 bg-emerald-50",
    amber: "border-amber-100 bg-amber-50",
    blue: "border-blue-100 bg-blue-50",
    indigo: "border-primary/15 bg-primary/5",
  };

  const textColorMap = {
    emerald: "text-emerald-600",
    amber: "text-amber-600",
    blue: "text-blue-600",
    indigo: "text-primary",
  };

  return (
    <div className={`border rounded-2xl p-4 shadow-xs ${colorMap[color]}`}>
      <h3 className={`${textColorMap[color]} font-semibold text-[13px] mb-3 flex items-center gap-1.5`}>
        <span>{icon}</span> {title}
      </h3>
      <ul className="space-y-2">
        {items.map((item, i) => (
          <li key={i} className="text-foreground/75 text-[12px] leading-relaxed flex items-start gap-2">
            <span className={`${textColorMap[color]} mt-1.5 flex-shrink-0`}>
              <span className="block w-1 h-1 rounded-full bg-current" />
            </span>
            {item}
          </li>
        ))}
      </ul>
    </div>
  );
}

function QualityScore({ score }: { score: number }) {
  const color =
    score >= 8
      ? "text-emerald-600"
      : score >= 6
      ? "text-amber-600"
      : "text-red-500";
  return (
    <span className={`text-[18px] font-bold ${color} leading-none`}>
      {score.toFixed(1)}
      <span className="text-muted-foreground font-normal text-[13px]">/10</span>
    </span>
  );
}

function RecommendedAgentCard({ agent }: { agent: RecommendedAgent }) {
  const [showConnect, setShowConnect] = useState(false);
  const [contact, setContact] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);

  const handleConnect = async () => {
    if (!contact.trim()) return;
    setLoading(true);
    try {
      await connectionApi.request({
        target_agent_id: agent.agent_id,
        request_message: message,
        requester_contact: contact,
      });
      setSent(true);
    } catch (err) {
      toast({ title: "连接请求失败", description: extractApiError(err), variant: "destructive" });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-card border border-border rounded-2xl p-4 shadow-xs">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-2">
            <span className="text-foreground text-[13px] font-mono font-semibold">{agent.anon_id}</span>
            <span className="text-[11px] text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
              契合度 {(agent.final_score * 100).toFixed(0)}%
            </span>
          </div>
          <div className="flex flex-wrap gap-1.5">
            {agent.reasons.map((r, i) => (
              <span
                key={i}
                className="text-[11px] bg-primary/8 text-primary px-2 py-0.5 rounded-full border border-primary/15 font-medium"
              >
                {r}
              </span>
            ))}
          </div>
        </div>

        {!sent ? (
          <button
            onClick={() => setShowConnect(!showConnect)}
            className="flex-shrink-0 text-[12px] bg-primary hover:bg-primary/90 text-primary-foreground px-3.5 py-1.5 rounded-xl transition-all duration-150 font-medium shadow-primary-sm"
          >
            认识 TA
          </button>
        ) : (
          <span className="flex-shrink-0 text-[12px] text-emerald-600 font-medium">✓ 已发送</span>
        )}
      </div>

      {/* Connection request form */}
      {showConnect && !sent && (
        <div className="mt-4 pt-4 border-t border-border space-y-3 animate-unlock">
          <p className="text-[12px] text-muted-foreground leading-relaxed">
            发送认识请求后，对方会收到通知。双方确认后才会交换联系方式。
          </p>
          <input
            value={contact}
            onChange={(e) => setContact(e.target.value)}
            placeholder="你的联系方式（微信/邮箱/手机）"
            className="w-full bg-background border border-border rounded-xl px-3.5 py-2.5 text-[13px] text-foreground placeholder-muted-foreground/60 focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 transition-all duration-150"
          />
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            placeholder="申请理由（选填）"
            rows={2}
            className="w-full bg-background border border-border rounded-xl px-3.5 py-2.5 text-[13px] text-foreground placeholder-muted-foreground/60 focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 transition-all duration-150 resize-none"
          />
          <button
            onClick={handleConnect}
            disabled={loading || !contact.trim()}
            className="w-full bg-primary hover:bg-primary/90 disabled:opacity-50 text-primary-foreground text-[13px] py-2.5 rounded-xl transition-all duration-150 font-medium shadow-primary-sm"
          >
            {loading ? "发送中..." : "确认发送申请"}
          </button>
        </div>
      )}
    </div>
  );
}

function UserRating({ reportId, currentRating }: { reportId: string; currentRating?: number }) {
  const [rating, setRating] = useState(currentRating ?? 0);
  const [saved, setSaved] = useState(!!currentRating);

  const handleRate = async (score: number) => {
    setRating(score);
    try {
      await reportApi.rate(reportId, { rating: score });
      setSaved(true);
    } catch (err) {
      toast({ title: "评分提交失败", description: extractApiError(err), variant: "destructive" });
    }
  };

  return (
    <div className="text-center py-6 border-t border-border">
      <p className="text-muted-foreground text-[13px] mb-4">这份报告对你有帮助吗？</p>
      <div className="flex justify-center gap-2">
        {[1, 2, 3, 4, 5].map((n) => (
          <button
            key={n}
            onClick={() => handleRate(n)}
            className={`w-9 h-9 rounded-xl text-base transition-all duration-150 active:scale-90 ${
              rating >= n
                ? "bg-amber-400 text-white shadow-sm"
                : "bg-muted text-muted-foreground hover:bg-amber-50 hover:text-amber-400 border border-border"
            }`}
          >
            ★
          </button>
        ))}
      </div>
      {saved && (
        <p className="text-[12px] text-muted-foreground mt-3">感谢你的反馈！</p>
      )}
    </div>
  );
}

function ReportSkeleton() {
  return (
    <div className="min-h-screen bg-background">
      <div className="h-14 bg-card border-b border-border" />
      <div className="max-w-3xl mx-auto px-4 py-6 space-y-4">
        <div className="h-20 bg-muted rounded-2xl animate-pulse" />
        <div className="h-40 bg-muted rounded-2xl animate-pulse" />
        <div className="grid grid-cols-2 gap-3">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-32 bg-muted rounded-2xl animate-pulse" />
          ))}
        </div>
      </div>
    </div>
  );
}
