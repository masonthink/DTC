"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import useSWR, { mutate } from "swr";
import { agentApi, type Agent } from "@/lib/api";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowLeft, Edit2, Check, X, Star, MessageSquare, Zap } from "lucide-react";
import Link from "next/link";
import { toast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";

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
  questioner: "text-red-600 bg-red-50 border-red-100",
  supporter: "text-emerald-600 bg-emerald-50 border-emerald-100",
  supplementer: "text-blue-600 bg-blue-50 border-blue-100",
  inquirer: "text-violet-600 bg-violet-50 border-violet-100",
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
      <div className="min-h-screen bg-background flex flex-col items-center justify-center px-4">
        <div className="w-16 h-16 rounded-2xl bg-muted border border-border flex items-center justify-center mb-4 text-2xl">
          🤖
        </div>
        <p className="text-foreground font-semibold text-[15px] mb-2">分身不存在</p>
        <Link href="/agents" className="text-primary text-[13px] hover:text-primary/80 transition-colors">返回分身列表</Link>
      </div>
    );
  }

  const typeConfig = AGENT_TYPE_OPTIONS.find((o) => o.value === agent.agent_type);

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
          <p className="flex-1 text-[13px] text-muted-foreground">分身详情</p>
          {!editing ? (
            <button
              onClick={startEdit}
              className="flex items-center gap-1.5 text-primary text-[13px] font-medium hover:text-primary/80 transition-colors px-3 py-1.5 rounded-lg hover:bg-primary/8"
            >
              <Edit2 className="w-3.5 h-3.5" />
              编辑
            </button>
          ) : (
            <div className="flex items-center gap-2">
              <button
                onClick={cancelEdit}
                className="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-muted text-muted-foreground transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
              <button
                onClick={saveEdit}
                disabled={saving}
                className="flex items-center gap-1.5 bg-primary hover:bg-primary/90 disabled:opacity-50 text-primary-foreground text-[12px] font-medium px-3 py-1.5 rounded-lg transition-all duration-150 shadow-primary-sm"
              >
                <Check className="w-3.5 h-3.5" />
                {saving ? "保存中" : "保存"}
              </button>
            </div>
          )}
        </div>
      </div>

      <div className="px-4 pt-5 space-y-4 pb-6">
        {/* Identity card */}
        <div className="bg-card border border-border rounded-2xl p-5 shadow-xs">
          <div className="flex items-start gap-4">
            <div className="w-16 h-16 rounded-2xl bg-primary/10 border border-primary/15 flex items-center justify-center text-3xl flex-shrink-0">
              {typeConfig?.emoji ?? "🤖"}
            </div>
            <div className="flex-1 min-w-0">
              {editing ? (
                <input
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                  className="w-full bg-background border border-border focus:border-primary rounded-xl px-3 py-2 text-foreground text-[16px] font-semibold outline-none focus:ring-2 focus:ring-primary/15 transition-all duration-150 mb-2"
                  placeholder="分身名称"
                  maxLength={30}
                />
              ) : (
                <h1 className="text-[18px] font-bold text-foreground mb-1.5">{agent.display_name}</h1>
              )}

              {editing ? (
                <div className="flex flex-wrap gap-2 mt-1">
                  {AGENT_TYPE_OPTIONS.map((opt) => (
                    <button
                      key={opt.value}
                      onClick={() => setAgentType(opt.value)}
                      className={cn(
                        "flex items-center gap-1 text-[11px] px-3 py-1.5 rounded-full border transition-all duration-150",
                        agentType === opt.value
                          ? "bg-primary border-primary text-primary-foreground"
                          : "bg-muted border-border text-muted-foreground hover:border-primary/30"
                      )}
                    >
                      {opt.emoji} {opt.label}
                    </button>
                  ))}
                </div>
              ) : (
                <span className="inline-flex items-center gap-1 text-[11px] bg-primary/8 text-primary border border-primary/15 px-2.5 py-1 rounded-full font-medium">
                  {typeConfig?.emoji} {typeConfig?.label}
                </span>
              )}
            </div>
          </div>

          {/* Anon ID */}
          <div className="mt-4 pt-4 border-t border-border/60 flex items-center justify-between">
            <span className="text-[11px] text-muted-foreground">匿名 ID</span>
            <span className="font-mono text-[12px] text-foreground/70">{agent.anon_id}</span>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 gap-3">
          <StatCard
            icon={<MessageSquare className="w-4.5 h-4.5 text-primary" />}
            value={String(agent.discussion_count)}
            label="次讨论"
          />
          <StatCard
            icon={<Star className="w-4.5 h-4.5 text-amber-500" />}
            value={agent.quality_score > 0 ? agent.quality_score.toFixed(1) : "—"}
            label="质量分"
          />
          <StatCard
            icon={<Zap className="w-4.5 h-4.5 text-emerald-500" />}
            value={`${agent.experience_years}年`}
            label="经验"
          />
        </div>

        {/* Social footprint */}
        {agent.discussion_count > 0 && (
          <div className="bg-gradient-to-r from-primary/6 to-violet-50/60 border border-primary/12 rounded-2xl px-5 py-4">
            <p className="text-[12px] text-foreground/80 leading-relaxed">
              <span className="font-semibold text-primary">{agent.display_name}</span> 已代表你参与了{" "}
              <span className="font-semibold">{agent.discussion_count}</span> 次深度讨论
              {agent.quality_score > 0 && (
                <>，质量评分 <span className="font-semibold text-amber-600">{agent.quality_score.toFixed(1)}</span></>
              )}
              。讨论越多，越容易找到合适的搭子。
            </p>
          </div>
        )}

        {/* Thinking style */}
        <div className="bg-card border border-border rounded-2xl p-5 shadow-xs">
          <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-4">
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
          <div className="bg-card border border-border rounded-2xl p-5 shadow-xs space-y-4">
            {agent.industries.length > 0 && (
              <div>
                <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  行业领域
                </p>
                <div className="flex flex-wrap gap-2">
                  {agent.industries.map((ind) => (
                    <span key={ind} className="text-[11px] bg-muted text-muted-foreground border border-border px-2.5 py-1 rounded-full">
                      {ind}
                    </span>
                  ))}
                </div>
              </div>
            )}
            {agent.skills.length > 0 && (
              <div>
                <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  专业技能
                </p>
                <div className="flex flex-wrap gap-2">
                  {agent.skills.map((skill) => (
                    <span key={skill} className="text-[11px] bg-primary/8 text-primary border border-primary/15 px-2.5 py-1 rounded-full font-medium">
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
          <div className="bg-card border border-border rounded-2xl p-5 shadow-xs">
            <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-4">
              问卷信息
            </p>
            <div className="space-y-3">
              {agent.questionnaire.bio && (
                <div>
                  <p className="text-[11px] text-muted-foreground mb-1.5">个人简介</p>
                  <p className="text-[13px] text-foreground/80 leading-relaxed">{agent.questionnaire.bio}</p>
                </div>
              )}
              {agent.questionnaire.current_role && (
                <div className="flex justify-between items-center pt-2 border-t border-border/60">
                  <span className="text-[11px] text-muted-foreground">当前职位</span>
                  <span className="text-[12px] text-foreground font-medium">{agent.questionnaire.current_role}</span>
                </div>
              )}
              {agent.questionnaire.preferred_role && (
                <div className="flex justify-between items-center pt-2 border-t border-border/60">
                  <span className="text-[11px] text-muted-foreground">偏好角色</span>
                  <span className={cn("text-[11px] px-2 py-0.5 rounded-full border font-medium", ROLE_COLORS[agent.questionnaire.preferred_role] ?? "text-muted-foreground")}>
                    {PREFERRED_ROLE_LABELS[agent.questionnaire.preferred_role] ?? agent.questionnaire.preferred_role}
                  </span>
                </div>
              )}
              {agent.questionnaire.discussion_strength && (
                <div className="pt-2 border-t border-border/60">
                  <p className="text-[11px] text-muted-foreground mb-1.5">讨论优势</p>
                  <p className="text-[13px] text-foreground/80 leading-relaxed">{agent.questionnaire.discussion_strength}</p>
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
  questioner: "质疑者",
};

function ThinkingBar({ label, value }: { label: string; value: number }) {
  const pct = Math.round(value * 100);
  return (
    <div>
      <div className="flex justify-between items-center mb-1.5">
        <span className="text-[12px] text-muted-foreground">{label}</span>
        <span className="text-[11px] text-muted-foreground tabular-nums">{pct}%</span>
      </div>
      <div className="h-1.5 bg-muted rounded-full overflow-hidden">
        <div
          className="h-full bg-gradient-to-r from-primary to-violet-500 rounded-full transition-all duration-500"
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

function StatCard({ icon, value, label }: { icon: React.ReactNode; value: string; label: string }) {
  return (
    <div className="bg-card border border-border rounded-xl p-4 flex flex-col items-center gap-1.5 shadow-xs">
      {icon}
      <span className="text-[16px] font-bold text-foreground tabular-nums">{value}</span>
      <span className="text-[11px] text-muted-foreground">{label}</span>
    </div>
  );
}

function AgentSkeleton() {
  return (
    <div className="min-h-screen bg-background px-4 pt-4 space-y-4">
      <Skeleton className="h-10 w-full rounded-xl" />
      <Skeleton className="h-36 w-full rounded-2xl" />
      <div className="grid grid-cols-3 gap-3">
        <Skeleton className="h-24 rounded-xl" />
        <Skeleton className="h-24 rounded-xl" />
        <Skeleton className="h-24 rounded-xl" />
      </div>
      <Skeleton className="h-40 w-full rounded-2xl" />
    </div>
  );
}
