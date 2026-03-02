"use client";

import { use } from "react";
import useSWR from "swr";
import { reportApi, connectionApi, type Report, type RecommendedAgent } from "@/lib/api";
import { useState } from "react";

interface Props {
  params: Promise<{ id: string }>;
}

export default function ReportPage({ params }: Props) {
  const { id } = use(params);
  const { data: report, isLoading } = useSWR(`report:${id}`, () => reportApi.get(id));

  if (isLoading) return <ReportSkeleton />;
  if (!report) return <div className="text-center text-slate-400 py-16">报告不存在</div>;

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <span className="text-xs text-indigo-400 uppercase tracking-wider font-medium">
          讨论分析报告
        </span>
        <div className="mt-2 flex items-center gap-3">
          <div className="flex-1">
            <div className="flex items-center gap-2 text-sm text-slate-400">
              <span>质量评分</span>
              <QualityScore score={report.quality_score} />
            </div>
            <p className="text-xs text-slate-500 mt-0.5">
              {new Date(report.generated_at).toLocaleDateString("zh-CN", {
                year: "numeric",
                month: "long",
                day: "numeric",
              })} 生成
            </p>
          </div>
        </div>
      </div>

      {/* Summary */}
      <section className="bg-slate-100/80 border border-slate-200 rounded-2xl p-6 mb-6">
        <h2 className="text-slate-900 font-semibold mb-4">📋 讨论摘要</h2>
        <div className="text-slate-700 text-sm leading-relaxed whitespace-pre-line">
          {report.summary}
        </div>
      </section>

      {/* Opinion matrix */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
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
        <section className="bg-slate-100/60 border border-slate-200 rounded-2xl p-5 mb-6">
          <h3 className="text-slate-700 font-medium text-sm mb-3 flex items-center gap-2">
            <span>🔍</span> 值得关注的盲点
          </h3>
          <ul className="space-y-2">
            {report.blind_spots.map((item, i) => (
              <li key={i} className="text-slate-400 text-sm flex items-start gap-2">
                <span className="text-slate-600 mt-0.5 flex-shrink-0">·</span>
                {item}
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Recommended connections */}
      {report.recommended_agents.length > 0 && (
        <section className="mb-6">
          <h2 className="text-slate-900 font-semibold mb-4">🤝 推荐连接</h2>
          <div className="space-y-3">
            {report.recommended_agents.map((agent) => (
              <RecommendedAgentCard key={agent.agent_id} agent={agent} />
            ))}
          </div>
        </section>
      )}

      {/* User rating */}
      <UserRating reportId={id} currentRating={report.user_rating} />
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
    emerald: "border-emerald-500/20 bg-emerald-500/5",
    amber: "border-amber-500/20 bg-amber-500/5",
    blue: "border-blue-500/20 bg-blue-500/5",
    indigo: "border-indigo-500/20 bg-indigo-500/5",
  };

  return (
    <div className={`border rounded-xl p-4 ${colorMap[color]}`}>
      <h3 className="text-slate-700 font-medium text-sm mb-3">
        {icon} {title}
      </h3>
      <ul className="space-y-2">
        {items.map((item, i) => (
          <li key={i} className="text-slate-400 text-xs leading-relaxed flex items-start gap-1.5">
            <span className="text-slate-600 mt-0.5 flex-shrink-0">·</span>
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
      ? "text-emerald-400"
      : score >= 6
      ? "text-amber-400"
      : "text-red-400";
  return (
    <span className={`font-bold ${color}`}>
      {score.toFixed(1)}<span className="text-slate-600 font-normal">/10</span>
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
    } catch {
      // handle error
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-slate-100/80 border border-slate-200 rounded-xl p-4">
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <span className="text-slate-900 text-sm font-mono font-medium">{agent.anon_id}</span>
            <span className="text-xs text-slate-500">
              综合得分 {(agent.final_score * 100).toFixed(0)}%
            </span>
          </div>
          <div className="flex flex-wrap gap-1">
            {agent.reasons.map((r, i) => (
              <span
                key={i}
                className="text-xs bg-indigo-500/10 text-indigo-400 px-2 py-0.5 rounded-full"
              >
                {r}
              </span>
            ))}
          </div>
        </div>

        {!sent ? (
          <button
            onClick={() => setShowConnect(!showConnect)}
            className="flex-shrink-0 text-xs bg-indigo-600 hover:bg-indigo-500 text-white px-3 py-1.5 rounded-lg transition-colors"
          >
            申请连接
          </button>
        ) : (
          <span className="flex-shrink-0 text-xs text-emerald-400">已发送</span>
        )}
      </div>

      {/* Connection request form */}
      {showConnect && !sent && (
        <div className="mt-4 pt-4 border-t border-slate-200 space-y-3 animate-unlock">
          <p className="text-xs text-slate-400">
            申请连接后，对方会收到通知。双方确认后才会交换联系方式。
          </p>
          <input
            value={contact}
            onChange={(e) => setContact(e.target.value)}
            placeholder="你的联系方式（微信/邮箱/手机）"
            className="w-full bg-slate-200/60 border border-slate-300 rounded-lg px-3 py-2 text-sm text-slate-900 placeholder-slate-400 focus:outline-none focus:border-indigo-500"
          />
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            placeholder="申请理由（选填）"
            rows={2}
            className="w-full bg-slate-200/60 border border-slate-300 rounded-lg px-3 py-2 text-sm text-slate-900 placeholder-slate-400 focus:outline-none focus:border-indigo-500 resize-none"
          />
          <button
            onClick={handleConnect}
            disabled={loading || !contact.trim()}
            className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-sm py-2 rounded-lg transition-colors"
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
    } catch {
      // ignore
    }
  };

  return (
    <div className="text-center py-6 border-t border-slate-200">
      <p className="text-slate-400 text-sm mb-3">这份报告对你有帮助吗？</p>
      <div className="flex justify-center gap-2">
        {[1, 2, 3, 4, 5].map((n) => (
          <button
            key={n}
            onClick={() => handleRate(n)}
            className={`w-8 h-8 rounded-lg text-sm transition-all ${
              rating >= n
                ? "bg-amber-500 text-white"
                : "bg-slate-100 text-slate-500 hover:bg-slate-200"
            }`}
          >
            ★
          </button>
        ))}
      </div>
      {saved && (
        <p className="text-xs text-slate-500 mt-2">感谢反馈！</p>
      )}
    </div>
  );
}

function ReportSkeleton() {
  return (
    <div className="max-w-3xl mx-auto px-4 py-8 animate-pulse">
      <div className="h-4 bg-slate-100 rounded w-1/4 mb-4" />
      <div className="h-48 bg-slate-100 rounded-2xl mb-6" />
      <div className="grid grid-cols-2 gap-4 mb-6">
        {[...Array(4)].map((_, i) => (
          <div key={i} className="h-32 bg-slate-100 rounded-xl" />
        ))}
      </div>
    </div>
  );
}
