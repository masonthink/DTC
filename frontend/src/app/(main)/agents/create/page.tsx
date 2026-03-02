"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { agentApi, type CreateAgentRequest } from "@/lib/api";
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
  { value: "questioner", label: "提问者", emoji: "❓", desc: "善于发现关键问题" },
  { value: "supporter", label: "支持者", emoji: "✅", desc: "补充论据，深化论点" },
  { value: "supplementer", label: "补充者", emoji: "💡", desc: "提供新视角和信息" },
  { value: "inquirer", label: "探究者", emoji: "🔍", desc: "深入追问，挖掘本质" },
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
      toast({ title: "分身创建成功！", description: "现在可以提交话题了" });
      router.push("/dashboard");
    } catch {
      toast({ title: "创建失败", description: "请稍后重试", variant: "destructive" });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-white/95 backdrop-blur-xl border-b border-slate-200">
        <div className="px-4 py-4 flex items-center gap-3">
          {step > 0 ? (
            <button
              onClick={() => setStep((s) => s - 1)}
              className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-slate-100 text-slate-400 hover:text-slate-900 transition-colors"
            >
              <ArrowLeft className="w-5 h-5" />
            </button>
          ) : (
            <Link
              href="/agents"
              className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-slate-100 text-slate-400 hover:text-slate-900 transition-colors"
            >
              <ArrowLeft className="w-5 h-5" />
            </Link>
          )}
          <div className="flex-1">
            <h1 className="text-lg font-bold text-slate-900">创建数字分身</h1>
          </div>
          {/* Step dots */}
          <div className="flex gap-1.5">
            {steps.map((_, i) => (
              <div
                key={i}
                className={cn(
                  "h-1.5 rounded-full transition-all",
                  i === step
                    ? "w-5 bg-indigo-500"
                    : i < step
                    ? "w-1.5 bg-indigo-700"
                    : "w-1.5 bg-slate-300"
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
                <p className="text-xs font-medium text-slate-400 uppercase tracking-wider mb-3">
                  分身类型
                </p>
                <div className="grid grid-cols-2 gap-3">
                  {AGENT_TYPES.map((t) => (
                    <button
                      key={t.value}
                      type="button"
                      onClick={() => setValue("agent_type", t.value as FormValues["agent_type"])}
                      className={cn(
                        "p-4 rounded-2xl border text-left transition-all active:scale-95",
                        agentType === t.value
                          ? "border-indigo-500 bg-indigo-600/10"
                          : "border-slate-200 bg-white hover:border-slate-300"
                      )}
                    >
                      <span className="text-2xl block mb-2">{t.emoji}</span>
                      <p className="text-slate-900 text-sm font-semibold">{t.label}</p>
                      <p className="text-slate-500 text-xs mt-0.5 leading-tight">{t.desc}</p>
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                  分身名称
                </label>
                <input
                  {...register("display_name")}
                  placeholder="给你的分身起个名字"
                  className="w-full bg-white border border-slate-200 focus:border-indigo-500 rounded-xl px-4 py-3 text-slate-900 placeholder-slate-400 focus:outline-none transition-colors text-sm"
                />
                {errors.display_name && (
                  <p className="text-red-400 text-xs mt-1.5">{errors.display_name.message}</p>
                )}
              </div>
            </div>
          )}

          {/* ── Step 1: Professional background ── */}
          {step === 1 && (
            <div className="space-y-5">
              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
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
                            "py-2.5 px-3 rounded-xl text-xs font-medium border transition-all active:scale-95",
                            field.value === ind
                              ? "border-indigo-500 bg-indigo-600/15 text-indigo-700"
                              : "border-slate-200 bg-white text-slate-400 hover:border-slate-300"
                          )}
                        >
                          {ind}
                        </button>
                      ))}
                    </div>
                  )}
                />
                {errors.questionnaire?.primary_industry && (
                  <p className="text-red-400 text-xs mt-1.5">{errors.questionnaire.primary_industry.message}</p>
                )}
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                  工作年限：{watch("questionnaire.years_experience")} 年
                </label>
                <input
                  type="range"
                  min="0"
                  max="25"
                  step="1"
                  {...register("questionnaire.years_experience", { valueAsNumber: true })}
                  className="w-full accent-indigo-500"
                />
                <div className="flex justify-between text-xs text-slate-500 mt-1">
                  <span>应届</span>
                  <span>25年+</span>
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                  当前职位
                </label>
                <input
                  {...register("questionnaire.current_role")}
                  placeholder="如：产品经理、CTO、创始人"
                  className="w-full bg-white border border-slate-200 focus:border-indigo-500 rounded-xl px-4 py-3 text-slate-900 placeholder-slate-400 focus:outline-none transition-colors text-sm"
                />
                {errors.questionnaire?.current_role && (
                  <p className="text-red-400 text-xs mt-1.5">{errors.questionnaire.current_role.message}</p>
                )}
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                  专业能力（多选）
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
                        "px-3 py-1.5 rounded-full text-xs font-medium border transition-all active:scale-95",
                        expertise.includes(opt)
                          ? "border-indigo-500 bg-indigo-600/15 text-indigo-700"
                          : "border-slate-200 bg-white text-slate-400 hover:border-slate-300"
                      )}
                    >
                      {opt}
                    </button>
                  ))}
                </div>
                {errors.questionnaire?.expertise && (
                  <p className="text-red-400 text-xs mt-1.5">{errors.questionnaire.expertise.message}</p>
                )}
              </div>
            </div>
          )}

          {/* ── Step 2: Thinking style & bio ── */}
          {step === 2 && (
            <div className="space-y-5">
              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
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
                            "w-full flex items-center gap-3 p-3.5 rounded-xl border text-left transition-all active:scale-[0.99]",
                            field.value === opt.value
                              ? "border-indigo-500 bg-indigo-600/10 text-slate-900"
                              : "border-slate-200 bg-white text-slate-400 hover:border-slate-300"
                          )}
                        >
                          <div
                            className={cn(
                              "w-4 h-4 rounded-full border-2 flex-shrink-0 transition-colors",
                              field.value === opt.value
                                ? "border-indigo-500 bg-indigo-500"
                                : "border-slate-300"
                            )}
                          />
                          <span className="text-sm">{opt.label}</span>
                        </button>
                      ))}
                    </div>
                  )}
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
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
                            "py-3 px-4 rounded-xl border text-sm font-medium transition-all active:scale-95",
                            field.value === opt.value
                              ? "border-indigo-500 bg-indigo-600/10 text-indigo-700"
                              : "border-slate-200 bg-white text-slate-400 hover:border-slate-300"
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
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                  风险偏好：{riskTolerance}/10
                </label>
                <input
                  type="range"
                  min="1"
                  max="10"
                  step="1"
                  {...register("questionnaire.risk_tolerance", { valueAsNumber: true })}
                  className="w-full accent-indigo-500"
                />
                <div className="flex justify-between text-xs text-slate-500 mt-1">
                  <span>保守稳健</span>
                  <span>激进冒险</span>
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                  创新倾向：{innovationFocus}/10
                </label>
                <input
                  type="range"
                  min="1"
                  max="10"
                  step="1"
                  {...register("questionnaire.innovation_focus", { valueAsNumber: true })}
                  className="w-full accent-indigo-500"
                />
                <div className="flex justify-between text-xs text-slate-500 mt-1">
                  <span>守成稳定</span>
                  <span>颠覆创新</span>
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                  讨论中的角色
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
                            "p-3 rounded-xl border text-left transition-all active:scale-95",
                            field.value === r.value
                              ? "border-indigo-500 bg-indigo-600/10"
                              : "border-slate-200 bg-white hover:border-slate-300"
                          )}
                        >
                          <span className="text-lg">{r.emoji}</span>
                          <p className="text-slate-900 text-xs font-semibold mt-1">{r.label}</p>
                          <p className="text-slate-500 text-xs leading-tight">{r.desc}</p>
                        </button>
                      ))}
                    </div>
                  )}
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                  我的讨论优势
                </label>
                <input
                  {...register("questionnaire.discussion_strength")}
                  placeholder="如：能快速抓住问题本质，善于提出反例"
                  className="w-full bg-white border border-slate-200 focus:border-indigo-500 rounded-xl px-4 py-3 text-slate-900 placeholder-slate-400 focus:outline-none transition-colors text-sm"
                />
                {errors.questionnaire?.discussion_strength && (
                  <p className="text-red-400 text-xs mt-1.5">{errors.questionnaire.discussion_strength.message}</p>
                )}
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                  分身简介 <span className="text-slate-600 normal-case">（20-500字）</span>
                </label>
                <textarea
                  {...register("questionnaire.bio")}
                  rows={4}
                  placeholder="描述你的背景、思维方式和对问题的独特视角..."
                  className="w-full bg-white border border-slate-200 focus:border-indigo-500 rounded-xl px-4 py-3 text-slate-900 placeholder-slate-400 focus:outline-none transition-colors text-sm resize-none"
                />
                {errors.questionnaire?.bio && (
                  <p className="text-red-400 text-xs mt-1.5">{errors.questionnaire.bio.message}</p>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Bottom action */}
        <div className="sticky bottom-0 px-4 pb-8 pt-4 bg-gradient-to-t from-white via-white to-transparent">
          {step < steps.length - 1 ? (
            <button
              type="button"
              onClick={goNext}
              className="w-full flex items-center justify-center gap-2 bg-indigo-600 hover:bg-indigo-500 text-white font-semibold py-4 rounded-2xl transition-all active:scale-[0.98] shadow-lg shadow-indigo-600/25"
            >
              下一步
              <ChevronRight className="w-5 h-5" />
            </button>
          ) : (
            <button
              type="submit"
              disabled={loading}
              className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-semibold py-4 rounded-2xl transition-all active:scale-[0.98] shadow-lg shadow-indigo-600/25"
            >
              {loading ? "创建中..." : "完成，创建分身"}
            </button>
          )}
        </div>
      </form>
    </div>
  );
}
