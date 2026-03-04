"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { agentApi, extractApiError, type CreateAgentRequest } from "@/lib/api";
import { ArrowLeft, ChevronRight } from "lucide-react";
import Link from "next/link";
import { toast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";

// ─── Schema ───────────────────────────────────────────────────────────────────

const schema = z.object({
  agent_type: z.enum(["professional", "entrepreneur", "investor", "generalist"]),
  display_name: z.string().min(2, "至少2个字").max(20, "最多20个字"),
  questionnaire: z.object({
    primary_industry: z.string().min(1, "请选择行业"),
    years_experience: z.number().min(0).max(40),
    current_role: z.string().min(2, "请填写职位"),
    expertise: z.array(z.string()).min(1, "至少选一项"),
    problem_approach: z.string().min(1, "请选择"),
    decision_style: z.string().min(1, "请选择"),
    risk_tolerance: z.number().min(1).max(10),
    innovation_focus: z.number().min(1).max(10),
    preferred_role: z.string().min(1, "请选择"),
    discussion_strength: z.string().min(5, "至少5个字"),
    bio: z.string().min(20, "至少20个字").max(500, "最多500字"),
  }),
});

type FormValues = z.infer<typeof schema>;

// ─── Config ───────────────────────────────────────────────────────────────────

const AGENT_TYPES = [
  { value: "professional", label: "职场精英", emoji: "💼", desc: "专注于职场发展与专业成长" },
  { value: "entrepreneur", label: "创业者", emoji: "🚀", desc: "拥抱不确定性，善于创新突破" },
  { value: "investor", label: "投资人", emoji: "📈", desc: "系统思考，关注长期价值" },
  { value: "generalist", label: "多面手", emoji: "🌐", desc: "跨界思维，连接不同领域" },
];

const INDUSTRIES = [
  "科技/互联网", "金融/投资", "医疗/健康", "教育",
  "创业/创投", "设计/创意", "法律/咨询", "制造/工程",
  "媒体/内容", "零售/消费", "房地产", "其他",
];

const EXPERTISE_OPTIONS = [
  "产品设计", "技术开发", "市场营销", "商业战略",
  "数据分析", "融资投资", "团队管理", "用户研究",
  "内容创作", "销售运营", "财务规划", "法律合规",
];

const PROBLEM_APPROACHES = [
  { value: "analytical", label: "系统分析，逻辑推导" },
  { value: "intuitive", label: "直觉驱动，快速决策" },
  { value: "collaborative", label: "广泛讨论，集体智慧" },
  { value: "experimental", label: "快速试验，数据验证" },
];

const DECISION_STYLES = [
  { value: "data_driven", label: "数据驱动" },
  { value: "principle_based", label: "原则导向" },
  { value: "experience_based", label: "经验优先" },
  { value: "consensus_based", label: "共识优先" },
];

const DISCUSSION_ROLES = [
  { value: "questioner", label: "追问型", emoji: "❓", desc: "喜欢追问到底，挖掘本质" },
  { value: "supporter", label: "建设型", emoji: "🔧", desc: "擅长提出具体可行的方案" },
  { value: "supplementer", label: "发散型", emoji: "💡", desc: "善于引入新视角和跨界思考" },
  { value: "inquirer", label: "分析型", emoji: "📊", desc: "偏好数据和逻辑推理" },
];

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function AgentCreatePage() {
  const router = useRouter();
  const [step, setStep] = useState(0);
  const [loading, setLoading] = useState(false);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    control,
    formState: { errors },
    trigger,
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      agent_type: "professional",
      questionnaire: {
        years_experience: 3,
        risk_tolerance: 5,
        innovation_focus: 5,
        expertise: [],
      },
    },
  });

  const agentType = watch("agent_type");
  const expertise = watch("questionnaire.expertise") ?? [];
  const riskTolerance = watch("questionnaire.risk_tolerance") ?? 5;
  const innovationFocus = watch("questionnaire.innovation_focus") ?? 5;

  const steps = [
    { title: "类型", fields: ["agent_type", "display_name"] },
    { title: "背景", fields: ["questionnaire.primary_industry", "questionnaire.years_experience", "questionnaire.current_role", "questionnaire.expertise"] },
    { title: "风格", fields: ["questionnaire.problem_approach", "questionnaire.decision_style", "questionnaire.risk_tolerance", "questionnaire.innovation_focus", "questionnaire.preferred_role", "questionnaire.discussion_strength", "questionnaire.bio"] },
  ];

  const goNext = async () => {
    const fields = steps[step].fields as Parameters<typeof trigger>[0];
    const ok = await trigger(fields);
    if (ok) setStep((s) => s + 1);
  };

  const onSubmit = async (data: FormValues) => {
    setLoading(true);
    try {
      await agentApi.create(data as CreateAgentRequest);
      toast({ title: "分身创建成功！", description: "现在可以提交想法了" });
      router.push("/dashboard");
    } catch (err) {
      toast({ title: "创建失败", description: extractApiError(err), variant: "destructive" });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-card/92 backdrop-blur-2xl border-b border-border/60">
        <div className="px-4 py-4 flex items-center gap-3">
          {step > 0 ? (
            <button
              onClick={() => setStep((s) => s - 1)}
              className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-muted text-muted-foreground hover:text-foreground transition-all duration-150"
            >
              <ArrowLeft className="w-5 h-5" />
            </button>
          ) : (
            <Link
              href="/agents"
              className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-muted text-muted-foreground hover:text-foreground transition-all duration-150"
            >
              <ArrowLeft className="w-5 h-5" />
            </Link>
          )}
          <div className="flex-1">
            <h1 className="text-[15px] font-bold text-foreground tracking-tight">创建数字分身</h1>
          </div>
          {/* Step dots */}
          <div className="flex gap-1.5 items-center">
            {steps.map((_, i) => (
              <div
                key={i}
                className={cn(
                  "h-1.5 rounded-full transition-all duration-300",
                  i === step
                    ? "w-5 bg-primary"
                    : i < step
                    ? "w-1.5 bg-primary/60"
                    : "w-1.5 bg-border"
                )}
              />
            ))}
          </div>
        </div>
      </div>

      <form onSubmit={handleSubmit(onSubmit)}>
        <div className="px-4 pt-6 pb-4">
          {/* ── Step 0: Type & Name ── */}
          {step === 0 && (
            <div className="space-y-6">
              <div>
                <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-3">
                  分身类型
                </p>
                <div className="grid grid-cols-2 gap-3">
                  {AGENT_TYPES.map((t) => (
                    <button
                      key={t.value}
                      type="button"
                      onClick={() => setValue("agent_type", t.value as FormValues["agent_type"])}
                      className={cn(
                        "p-4 rounded-2xl border text-left transition-all duration-150 active:scale-95",
                        agentType === t.value
                          ? "border-primary bg-primary/8 shadow-xs"
                          : "border-border bg-card hover:border-primary/30 hover:bg-muted/40"
                      )}
                    >
                      <span className="text-2xl block mb-2">{t.emoji}</span>
                      <p className={cn("text-[13px] font-semibold transition-colors", agentType === t.value ? "text-primary" : "text-foreground")}>{t.label}</p>
                      <p className="text-muted-foreground text-[11px] mt-0.5 leading-tight">{t.desc}</p>
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label htmlFor="agent-display-name" className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  分身名称
                </label>
                <input
                  id="agent-display-name"
                  {...register("display_name")}
                  placeholder="给你的分身起个名字"
                  className="w-full bg-background border border-border focus:border-primary rounded-xl px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-primary/15 transition-all duration-150 text-[13px]"
                />
                {errors.display_name && (
                  <p className="text-red-500 text-[11px] mt-1.5">{errors.display_name.message}</p>
                )}
              </div>
            </div>
          )}

          {/* ── Step 1: Professional background ── */}
          {step === 1 && (
            <div className="space-y-5">
              <div>
                <label className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  主要行业
                </label>
                <Controller
                  name="questionnaire.primary_industry"
                  control={control}
                  render={({ field }) => (
                    <div className="grid grid-cols-3 gap-2">
                      {INDUSTRIES.map((ind) => (
                        <button
                          key={ind}
                          type="button"
                          onClick={() => field.onChange(ind)}
                          className={cn(
                            "py-2.5 px-3 rounded-xl text-[11px] font-medium border transition-all duration-150 active:scale-95",
                            field.value === ind
                              ? "border-primary bg-primary/8 text-primary"
                              : "border-border bg-card text-muted-foreground hover:border-primary/30 hover:bg-muted/40"
                          )}
                        >
                          {ind}
                        </button>
                      ))}
                    </div>
                  )}
                />
                {errors.questionnaire?.primary_industry && (
                  <p className="text-red-500 text-[11px] mt-1.5">{errors.questionnaire.primary_industry.message}</p>
                )}
              </div>

              <div>
                <label className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  工作年限：<span className="text-foreground normal-case font-medium">{watch("questionnaire.years_experience")} 年</span>
                </label>
                <input
                  type="range"
                  min="0"
                  max="25"
                  step="1"
                  {...register("questionnaire.years_experience", { valueAsNumber: true })}
                  className="w-full"
                />
                <div className="flex justify-between text-[11px] text-muted-foreground mt-1">
                  <span>应届</span>
                  <span>25年+</span>
                </div>
              </div>

              <div>
                <label htmlFor="current-role" className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  当前职位
                </label>
                <input
                  id="current-role"
                  {...register("questionnaire.current_role")}
                  placeholder="如：产品经理、CTO、创始人"
                  className="w-full bg-background border border-border focus:border-primary rounded-xl px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-primary/15 transition-all duration-150 text-[13px]"
                />
                {errors.questionnaire?.current_role && (
                  <p className="text-red-500 text-[11px] mt-1.5">{errors.questionnaire.current_role.message}</p>
                )}
              </div>

              <div>
                <label className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  专业能力 <span className="text-muted-foreground/60 normal-case font-normal">（多选）</span>
                </label>
                <div className="flex flex-wrap gap-2">
                  {EXPERTISE_OPTIONS.map((opt) => (
                    <button
                      key={opt}
                      type="button"
                      onClick={() => {
                        const next = expertise.includes(opt)
                          ? expertise.filter((e) => e !== opt)
                          : [...expertise, opt];
                        setValue("questionnaire.expertise", next);
                      }}
                      className={cn(
                        "px-3 py-1.5 rounded-full text-[11px] font-medium border transition-all duration-150 active:scale-95",
                        expertise.includes(opt)
                          ? "border-primary bg-primary/8 text-primary"
                          : "border-border bg-card text-muted-foreground hover:border-primary/30"
                      )}
                    >
                      {opt}
                    </button>
                  ))}
                </div>
                {errors.questionnaire?.expertise && (
                  <p className="text-red-500 text-[11px] mt-1.5">{errors.questionnaire.expertise.message}</p>
                )}
              </div>
            </div>
          )}

          {/* ── Step 2: Thinking style & bio ── */}
          {step === 2 && (
            <div className="space-y-5">
              <div>
                <label className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  解决问题的方式
                </label>
                <Controller
                  name="questionnaire.problem_approach"
                  control={control}
                  render={({ field }) => (
                    <div className="space-y-2">
                      {PROBLEM_APPROACHES.map((opt) => (
                        <button
                          key={opt.value}
                          type="button"
                          onClick={() => field.onChange(opt.value)}
                          className={cn(
                            "w-full flex items-center gap-3 p-3.5 rounded-xl border text-left transition-all duration-150 active:scale-[0.99]",
                            field.value === opt.value
                              ? "border-primary bg-primary/8"
                              : "border-border bg-card hover:border-primary/30 hover:bg-muted/40"
                          )}
                        >
                          <div
                            className={cn(
                              "w-4 h-4 rounded-full border-2 flex-shrink-0 transition-all duration-150",
                              field.value === opt.value
                                ? "border-primary bg-primary"
                                : "border-border"
                            )}
                          />
                          <span className={cn("text-[13px] transition-colors", field.value === opt.value ? "text-foreground font-medium" : "text-muted-foreground")}>
                            {opt.label}
                          </span>
                        </button>
                      ))}
                    </div>
                  )}
                />
              </div>

              <div>
                <label className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  决策风格
                </label>
                <Controller
                  name="questionnaire.decision_style"
                  control={control}
                  render={({ field }) => (
                    <div className="grid grid-cols-2 gap-2">
                      {DECISION_STYLES.map((opt) => (
                        <button
                          key={opt.value}
                          type="button"
                          onClick={() => field.onChange(opt.value)}
                          className={cn(
                            "py-3 px-4 rounded-xl border text-[13px] font-medium transition-all duration-150 active:scale-95",
                            field.value === opt.value
                              ? "border-primary bg-primary/8 text-primary"
                              : "border-border bg-card text-muted-foreground hover:border-primary/30"
                          )}
                        >
                          {opt.label}
                        </button>
                      ))}
                    </div>
                  )}
                />
              </div>

              <div>
                <label className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  风险偏好：<span className="text-foreground normal-case font-medium">{riskTolerance}/10</span>
                </label>
                <input
                  type="range"
                  min="1"
                  max="10"
                  step="1"
                  {...register("questionnaire.risk_tolerance", { valueAsNumber: true })}
                  className="w-full"
                />
                <div className="flex justify-between text-[11px] text-muted-foreground mt-1">
                  <span>保守稳健</span>
                  <span>激进冒险</span>
                </div>
              </div>

              <div>
                <label className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  创新倾向：<span className="text-foreground normal-case font-medium">{innovationFocus}/10</span>
                </label>
                <input
                  type="range"
                  min="1"
                  max="10"
                  step="1"
                  {...register("questionnaire.innovation_focus", { valueAsNumber: true })}
                  className="w-full"
                />
                <div className="flex justify-between text-[11px] text-muted-foreground mt-1">
                  <span>守成稳定</span>
                  <span>颠覆创新</span>
                </div>
              </div>

              <div>
                <label className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  讨论风格偏好
                </label>
                <Controller
                  name="questionnaire.preferred_role"
                  control={control}
                  render={({ field }) => (
                    <div className="grid grid-cols-2 gap-2">
                      {DISCUSSION_ROLES.map((r) => (
                        <button
                          key={r.value}
                          type="button"
                          onClick={() => field.onChange(r.value)}
                          className={cn(
                            "p-3 rounded-xl border text-left transition-all duration-150 active:scale-95",
                            field.value === r.value
                              ? "border-primary bg-primary/8"
                              : "border-border bg-card hover:border-primary/30 hover:bg-muted/40"
                          )}
                        >
                          <span className="text-lg">{r.emoji}</span>
                          <p className={cn("text-[12px] font-semibold mt-1 transition-colors", field.value === r.value ? "text-primary" : "text-foreground")}>{r.label}</p>
                          <p className="text-muted-foreground text-[11px] leading-tight">{r.desc}</p>
                        </button>
                      ))}
                    </div>
                  )}
                />
              </div>

              <div>
                <label htmlFor="discussion-strength" className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  我的讨论优势
                </label>
                <input
                  id="discussion-strength"
                  {...register("questionnaire.discussion_strength")}
                  placeholder="如：能快速抓住问题本质，善于提出反例"
                  className="w-full bg-background border border-border focus:border-primary rounded-xl px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-primary/15 transition-all duration-150 text-[13px]"
                />
                {errors.questionnaire?.discussion_strength && (
                  <p className="text-red-500 text-[11px] mt-1.5">{errors.questionnaire.discussion_strength.message}</p>
                )}
              </div>

              <div>
                <label htmlFor="agent-bio" className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  分身简介 <span className="text-muted-foreground/60 normal-case font-normal">（20–500字）</span>
                </label>
                <textarea
                  id="agent-bio"
                  {...register("questionnaire.bio")}
                  rows={4}
                  placeholder="描述你的背景、思维方式和对问题的独特视角..."
                  className="w-full bg-background border border-border focus:border-primary rounded-xl px-4 py-3 text-foreground placeholder-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-primary/15 transition-all duration-150 text-[13px] resize-none"
                />
                {errors.questionnaire?.bio && (
                  <p className="text-red-500 text-[11px] mt-1.5">{errors.questionnaire.bio.message}</p>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Bottom action */}
        <div className="sticky bottom-0 px-4 pb-8 pt-4 bg-gradient-to-t from-background via-background/95 to-transparent">
          {step < steps.length - 1 ? (
            <button
              type="button"
              onClick={goNext}
              className="w-full flex items-center justify-center gap-2 bg-primary hover:bg-primary/90 text-primary-foreground font-semibold py-4 rounded-2xl transition-all duration-150 active:scale-[0.98] shadow-primary-md"
            >
              下一步
              <ChevronRight className="w-5 h-5" />
            </button>
          ) : (
            <button
              type="submit"
              disabled={loading}
              className="w-full bg-primary hover:bg-primary/90 disabled:opacity-50 text-primary-foreground font-semibold py-4 rounded-2xl transition-all duration-150 active:scale-[0.98] shadow-primary-md"
            >
              {loading ? "创建中..." : "完成，创建分身"}
            </button>
          )}
        </div>
      </form>
    </div>
  );
}
