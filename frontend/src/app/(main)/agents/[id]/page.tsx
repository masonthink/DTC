"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import useSWR, { mutate } from "swr";
import { agentApi, type Agent } from "@/lib/api";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowLeft, Edit2, Check, X, Star, MessageSquare, Zap } from "lucide-react";
import Link from "next/link";
import { toast } from "@/hooks/use-toast";

interface Props {
  params: Promise<{ id: string }>;
}

const AGENT_TYPE_OPTIONS: { value: Agent["agent_type"]; label: string; emoji: string }[] = [
  { value: "professional", label: "职场精英", emoji: "💼" },
  { value: "entrepreneur", label: "创业者", emoji: "🚀" },
  { value: "investor", label: "投资人", emoji: "📈" },
  { value: "generalist", label: "多面手", emoji: "🌐" },
];

const ROLE_COLORS: Record<string, string> = {
  questioner: "text-red-400 bg-red-400/10 border-red-400/20",
  supporter: "text-emerald-400 bg-emerald-400/10 border-emerald-400/20",
  supplementer: "text-blue-400 bg-blue-400/10 border-blue-400/20",
  inquirer: "text-purple-400 bg-purple-400/10 border-purple-400/20",
};

export default function AgentDetailPage({ params }: Props) {
  const { id } = use(params);
  const router = useRouter();
  const { data: agent, isLoading } = useSWR(`agents/${id}`, () => agentApi.get(id));
  const [editing, setEditing] = useState(false);
  const [displayName, setDisplayName] = useState("");
  const [agentType, setAgentType] = useState<Agent["agent_type"]>("professional");
  const [saving, setSaving] = useState(false);

  const startEdit = () => {
    if (!agent) return;
    setDisplayName(agent.display_name);
    setAgentType(agent.agent_type);
    setEditing(true);
  };

  const cancelEdit = () => setEditing(false);

  const saveEdit = async () => {
    if (!displayName.trim()) {
      toast({ title: "分身名称不能为空", variant: "destructive" });
      return;
    }
    setSaving(true);
    try {
      await agentApi.update(id, {
        agent_type: agentType,
        display_name: displayName.trim(),
      } as Parameters<typeof agentApi.update>[1]);
      await mutate(`agents/${id}`);
      await mutate("agents");
      setEditing(false);
      toast({ title: "保存成功" });
    } catch {
      toast({ title: "保存失败，请稍后重试", variant: "destructive" });
    } finally {
      setSaving(false);
    }
  };

  if (isLoading) return <AgentSkeleton />;
  if (!agent) {
    return (
      <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center px-4">
        <p className="text-4xl mb-4">🤖</p>
        <p className="text-white font-medium mb-2">分身不存在</p>
        <Link href="/agents" className="text-indigo-400 text-sm">返回分身列表</Link>
      </div>
    );
  }

  const typeConfig = AGENT_TYPE_OPTIONS.find((o) => o.value === agent.agent_type);

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
          <p className="flex-1 text-sm text-slate-400">分身详情</p>
          {!editing ? (
            <button
              onClick={startEdit}
              className="flex items-center gap-1.5 text-indigo-400 text-sm hover:text-indigo-300 transition-colors px-3 py-1.5 rounded-lg hover:bg-indigo-500/10"
            >
              <Edit2 className="w-4 h-4" />
              编辑
            </button>
          ) : (
            <div className="flex items-center gap-2">
              <button
                onClick={cancelEdit}
                className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-slate-800 text-slate-400 transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
              <button
                onClick={saveEdit}
                disabled={saving}
                className="flex items-center gap-1.5 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white text-sm font-medium px-3 py-1.5 rounded-lg transition-colors"
              >
                <Check className="w-4 h-4" />
                {saving ? "保存中" : "保存"}
              </button>
            </div>
          )}
        </div>
      </div>

      <div className="px-4 pt-5 space-y-4 pb-6">
        {/* Identity card */}
        <div className="bg-slate-900 border border-slate-700/50 rounded-2xl p-5">
          <div className="flex items-start gap-4">
            <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-indigo-600/30 to-purple-600/20 border border-indigo-500/20 flex items-center justify-center text-3xl flex-shrink-0">
              {typeConfig?.emoji ?? "🤖"}
            </div>
            <div className="flex-1 min-w-0">
              {editing ? (
                <input
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                  className="w-full bg-slate-800 border border-slate-600 focus:border-indigo-500 rounded-xl px-3 py-2 text-white text-lg font-semibold outline-none transition-colors mb-2"
                  placeholder="分身名称"
                  maxLength={30}
                />
              ) : (
                <h1 className="text-xl font-bold text-white mb-1">{agent.display_name}</h1>
              )}

              {editing ? (
                <div className="flex flex-wrap gap-2 mt-1">
                  {AGENT_TYPE_OPTIONS.map((opt) => (
                    <button
                      key={opt.value}
                      onClick={() => setAgentType(opt.value)}
                      className={`flex items-center gap-1 text-xs px-3 py-1.5 rounded-full border transition-all ${
                        agentType === opt.value
                          ? "bg-indigo-600 border-indigo-500 text-white"
                          : "bg-slate-800 border-slate-700 text-slate-400 hover:border-slate-600"
                      }`}
                    >
                      {opt.emoji} {opt.label}
                    </button>
                  ))}
                </div>
              ) : (
                <span className="text-xs bg-indigo-600/20 text-indigo-400 border border-indigo-600/30 px-2.5 py-1 rounded-full">
                  {typeConfig?.emoji} {typeConfig?.label}
                </span>
              )}
            </div>
          </div>

          {/* Anon ID */}
          <div className="mt-4 pt-4 border-t border-slate-700/50 flex items-center justify-between">
            <span className="text-xs text-slate-500">匿名 ID</span>
            <span className="font-mono text-sm text-slate-300">{agent.anon_id}</span>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 gap-3">
          <StatCard
            icon={<MessageSquare className="w-5 h-5 text-indigo-400" />}
            value={String(agent.discussion_count)}
            label="次讨论"
          />
          <StatCard
            icon={<Star className="w-5 h-5 text-amber-400" />}
            value={agent.quality_score > 0 ? agent.quality_score.toFixed(1) : "—"}
            label="质量分"
          />
          <StatCard
            icon={<Zap className="w-5 h-5 text-emerald-400" />}
            value={`${agent.experience_years}年`}
            label="经验"
          />
        </div>

        {/* Thinking style */}
        <div className="bg-slate-900 border border-slate-700/50 rounded-2xl p-5">
          <p className="text-xs font-medium text-slate-400 uppercase tracking-wider mb-4">
            思维风格
          </p>
          <div className="space-y-3">
            {Object.entries(agent.thinking_style).map(([key, value]) => (
              <ThinkingBar key={key} label={STYLE_LABELS[key] ?? key} value={value as number} />
            ))}
          </div>
        </div>

        {/* Industries & Skills */}
        {(agent.industries.length > 0 || agent.skills.length > 0) && (
          <div className="bg-slate-900 border border-slate-700/50 rounded-2xl p-5 space-y-4">
            {agent.industries.length > 0 && (
              <div>
                <p className="text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                  行业领域
                </p>
                <div className="flex flex-wrap gap-2">
                  {agent.industries.map((ind) => (
                    <span key={ind} className="text-xs bg-slate-800 text-slate-300 border border-slate-700 px-2.5 py-1 rounded-full">
                      {ind}
                    </span>
                  ))}
                </div>
              </div>
            )}
            {agent.skills.length > 0 && (
              <div>
                <p className="text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                  专业技能
                </p>
                <div className="flex flex-wrap gap-2">
                  {agent.skills.map((skill) => (
                    <span key={skill} className="text-xs bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 px-2.5 py-1 rounded-full">
                      {skill}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Questionnaire summary */}
        {agent.questionnaire && (
          <div className="bg-slate-900 border border-slate-700/50 rounded-2xl p-5">
            <p className="text-xs font-medium text-slate-400 uppercase tracking-wider mb-4">
              问卷信息
            </p>
            <div className="space-y-3">
              {agent.questionnaire.bio && (
                <div>
                  <p className="text-xs text-slate-500 mb-1">个人简介</p>
                  <p className="text-sm text-slate-300 leading-relaxed">{agent.questionnaire.bio}</p>
                </div>
              )}
              {agent.questionnaire.current_role && (
                <div className="flex justify-between items-center">
                  <span className="text-xs text-slate-500">当前职位</span>
                  <span className="text-xs text-slate-300">{agent.questionnaire.current_role}</span>
                </div>
              )}
              {agent.questionnaire.preferred_role && (
                <div className="flex justify-between items-center">
                  <span className="text-xs text-slate-500">偏好角色</span>
                  <span className={`text-xs px-2 py-0.5 rounded-full border ${ROLE_COLORS[agent.questionnaire.preferred_role] ?? "text-slate-400"}`}>
                    {PREFERRED_ROLE_LABELS[agent.questionnaire.preferred_role] ?? agent.questionnaire.preferred_role}
                  </span>
                </div>
              )}
              {agent.questionnaire.discussion_strength && (
                <div>
                  <p className="text-xs text-slate-500 mb-1">讨论优势</p>
                  <p className="text-sm text-slate-300 leading-relaxed">{agent.questionnaire.discussion_strength}</p>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

const STYLE_LABELS: Record<string, string> = {
  analytical: "分析力",
  creative: "创造力",
  critical: "批判力",
  collaborative: "协作力",
  questioning: "质疑力",
};

const PREFERRED_ROLE_LABELS: Record<string, string> = {
  critic: "质疑者",
  advocate: "支持者",
  explorer: "探索者",
  questioner: "提问者",
};

function ThinkingBar({ label, value }: { label: string; value: number }) {
  const pct = Math.round(value * 100);
  return (
    <div>
      <div className="flex justify-between items-center mb-1">
        <span className="text-xs text-slate-400">{label}</span>
        <span className="text-xs text-slate-500">{pct}%</span>
      </div>
      <div className="h-1.5 bg-slate-800 rounded-full overflow-hidden">
        <div
          className="h-full bg-gradient-to-r from-indigo-500 to-purple-500 rounded-full transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

function StatCard({ icon, value, label }: { icon: React.ReactNode; value: string; label: string }) {
  return (
    <div className="bg-slate-900 border border-slate-700/50 rounded-xl p-4 flex flex-col items-center gap-2">
      {icon}
      <span className="text-lg font-bold text-white">{value}</span>
      <span className="text-xs text-slate-500">{label}</span>
    </div>
  );
}

function AgentSkeleton() {
  return (
    <div className="min-h-screen bg-slate-950 px-4 pt-4 space-y-4 animate-pulse">
      <Skeleton className="h-10 w-full" />
      <Skeleton className="h-32 w-full" />
      <div className="grid grid-cols-3 gap-3">
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
      </div>
      <Skeleton className="h-40 w-full" />
    </div>
  );
}
