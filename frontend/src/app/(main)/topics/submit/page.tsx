"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { agentApi, topicApi, extractApiError } from "@/lib/api";
import useSWR from "swr";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { cn } from "@/lib/utils";

const TOPIC_TYPES = [
  { value: "business_idea", label: "商业想法", emoji: "💡", desc: "创业方向、产品概念验证" },
  { value: "career_decision", label: "职业决策", emoji: "💼", desc: "跳槽、转行、晋升" },
  { value: "tech_choice", label: "技术选型", emoji: "⚙️", desc: "架构、工具、技术栈" },
  { value: "product_design", label: "产品设计", emoji: "🎨", desc: "功能、交互、用户体验" },
  { value: "investment", label: "投资判断", emoji: "📈", desc: "项目评估、市场分析" },
  { value: "other", label: "其他", emoji: "💬", desc: "其他深度想法" },
];

const schema = z.object({
  agent_id: z.string().min(1, "请选择一个分身"),
  topic_type: z.string().min(1, "请选择想法类型"),
  title: z.string().min(5, "想法标题至少5个字").max(200),
  description: z.string().min(20, "想法描述至少20个字").max(2000),
  background: z.string().max(1000).optional(),
});

type FormData = z.infer<typeof schema>;

export default function SubmitTopicPage() {
  const router = useRouter();
  const { data: agents } = useSWR("agents", agentApi.list);
  const [step, setStep] = useState<1 | 2>(1);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<FormData>({ resolver: zodResolver(schema) });

  const selectedType = watch("topic_type");
  const title = watch("title");
  const description = watch("description");
  const canProceed =
    watch("agent_id") && selectedType && title?.length >= 5 && description?.length >= 20;

  const onSubmit = async (data: FormData) => {
    setSubmitting(true);
    setError(null);
    try {
      const topic = await topicApi.submit(data);
      router.push(`/topics/${topic.id}`);
    } catch (err) {
      setError(extractApiError(err));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-card/92 backdrop-blur-2xl border-b border-border/60">
        <div className="px-4 py-4 flex items-center gap-3">
          {step === 2 ? (
            <button
              onClick={() => setStep(1)}
              className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-muted text-muted-foreground hover:text-foreground transition-all duration-150"
            >
              <ArrowLeft className="w-5 h-5" />
            </button>
          ) : (
            <Link
              href="/dashboard"
              className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-muted text-muted-foreground hover:text-foreground transition-all duration-150"
            >
              <ArrowLeft className="w-5 h-5" />
            </Link>
          )}
          <div className="flex-1">
            <h1 className="text-[15px] font-bold text-foreground tracking-tight">
              {step === 1 ? "提交想法" : "确认提交"}
            </h1>
          </div>
          {/* Step indicator */}
          <div className="flex gap-1.5 items-center">
            {[1, 2].map((s) => (
              <div
                key={s}
                className={cn(
                  "h-1.5 rounded-full transition-all duration-300",
                  s === step ? "w-5 bg-primary" : s < step ? "w-1.5 bg-primary/60" : "w-1.5 bg-border"
                )}
              />
            ))}
          </div>
        </div>
      </div>

      <form onSubmit={handleSubmit(onSubmit)}>
        {/* Step 1 */}
        {step === 1 && (
          <div className="px-4 pt-5 space-y-5">
            {/* Select agent */}
            <div>
              <label className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                选择分身
              </label>
              {agents && agents.length > 0 ? (
                <div className="space-y-2">
                  {agents.map((agent) => (
                    <label
                      key={agent.id}
                      className={cn(
                        "flex items-center gap-3 p-3.5 rounded-2xl border cursor-pointer transition-all duration-150 active:scale-[0.99]",
                        watch("agent_id") === agent.id
                          ? "border-primary bg-primary/6 shadow-xs"
                          : "border-border bg-card hover:border-primary/30 hover:bg-muted/40"
                      )}
                    >
                      <input
                        type="radio"
                        value={agent.id}
                        {...register("agent_id")}
                        className="sr-only"
                      />
                      <div className="w-9 h-9 rounded-xl bg-primary/10 border border-primary/15 flex items-center justify-center text-lg flex-shrink-0">
                        🤖
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-foreground text-[13px] font-semibold">{agent.display_name}</p>
                        <p className="text-muted-foreground text-[11px] mt-0.5 truncate">
                          {agent.industries.slice(0, 2).join(" · ")}
                        </p>
                      </div>
                      <div
                        className={cn(
                          "w-5 h-5 rounded-full border-2 flex-shrink-0 transition-all duration-150",
                          watch("agent_id") === agent.id
                            ? "border-primary bg-primary"
                            : "border-border"
                        )}
                      >
                        {watch("agent_id") === agent.id && (
                          <svg viewBox="0 0 20 20" fill="currentColor" className="text-primary-foreground w-full h-full p-0.5">
                            <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                          </svg>
                        )}
                      </div>
                    </label>
                  ))}
                </div>
              ) : (
                <div className="bg-card border border-border rounded-2xl p-5 text-center">
                  <p className="text-muted-foreground text-[13px] mb-3">还没有分身，先创建一个</p>
                  <Link
                    href="/agents/create"
                    className="text-primary text-[13px] font-medium hover:text-primary/80 transition-colors"
                  >
                    创建分身 →
                  </Link>
                </div>
              )}
              {errors.agent_id && (
                <p className="text-red-500 text-[11px] mt-1.5">{errors.agent_id.message}</p>
              )}
            </div>

            {/* Topic type */}
            <div>
              <label className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                想法类型
              </label>
              <div className="grid grid-cols-2 gap-2">
                {TOPIC_TYPES.map((t) => (
                  <button
                    key={t.value}
                    type="button"
                    onClick={() => setValue("topic_type", t.value)}
                    className={cn(
                      "text-left p-3.5 rounded-2xl border text-[12px] transition-all duration-150 active:scale-95",
                      selectedType === t.value
                        ? "border-primary bg-primary/8 shadow-xs"
                        : "border-border bg-card hover:border-primary/30 hover:bg-muted/40"
                    )}
                  >
                    <div className="text-lg mb-1">{t.emoji}</div>
                    <div className={cn("font-semibold mb-0.5 transition-colors", selectedType === t.value ? "text-primary" : "text-foreground")}>{t.label}</div>
                    <div className="text-muted-foreground leading-tight text-[11px]">{t.desc}</div>
                  </button>
                ))}
              </div>
              {errors.topic_type && (
                <p className="text-red-500 text-[11px] mt-1.5">{errors.topic_type.message}</p>
              )}
            </div>

            {/* Title */}
            <div>
              <label htmlFor="topic-title" className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                想法标题{" "}
                {(title?.length ?? 0) < 5 ? (
                  <span className="text-red-500 normal-case font-normal">
                    还差 {5 - (title?.length ?? 0)} 个字
                  </span>
                ) : (
                  <span className="text-muted-foreground/60 normal-case font-normal">
                    {title?.length}/200
                  </span>
                )}
              </label>
              <input
                id="topic-title"
                {...register("title")}
                placeholder="简洁地描述你的想法"
                className="w-full bg-background border border-border focus:border-primary rounded-xl px-4 py-3 text-foreground placeholder-muted-foreground/60 text-[13px] focus:outline-none focus:ring-2 focus:ring-primary/15 transition-all duration-150"
              />
              {errors.title && (
                <p className="text-red-500 text-[11px] mt-1.5">{errors.title.message}</p>
              )}
            </div>

            {/* Description */}
            <div>
              <label htmlFor="topic-description" className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                详细描述{" "}
                {(description?.length ?? 0) < 20 ? (
                  <span className="text-red-500 normal-case font-normal">
                    还差 {20 - (description?.length ?? 0)} 个字
                  </span>
                ) : (
                  <span className="text-muted-foreground/60 normal-case font-normal">
                    {description?.length}/2000
                  </span>
                )}
              </label>
              <textarea
                id="topic-description"
                {...register("description")}
                placeholder="描述你的具体想法、当前思考、以及你最希望讨论的方向..."
                rows={5}
                className="w-full bg-background border border-border focus:border-primary rounded-xl px-4 py-3 text-foreground placeholder-muted-foreground/60 text-[13px] focus:outline-none focus:ring-2 focus:ring-primary/15 transition-all duration-150 resize-none"
              />
              {errors.description && (
                <p className="text-red-500 text-[11px] mt-1.5">{errors.description.message}</p>
              )}
            </div>

            {/* Background */}
            <div>
              <label htmlFor="topic-background" className="block text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                补充背景 <span className="text-muted-foreground/60 normal-case font-normal">（选填）</span>
              </label>
              <textarea
                id="topic-background"
                {...register("background")}
                placeholder="你的行业背景、已有资源、限制条件..."
                rows={3}
                className="w-full bg-background border border-border focus:border-primary rounded-xl px-4 py-3 text-foreground placeholder-muted-foreground/60 text-[13px] focus:outline-none focus:ring-2 focus:ring-primary/15 transition-all duration-150 resize-none"
              />
            </div>

            <div className="pb-4" />
          </div>
        )}

        {/* Step 2 — confirmation */}
        {step === 2 && (
          <div className="px-4 pt-5 space-y-4">
            {/* Preview card */}
            <div className="bg-card border border-border rounded-2xl p-5 shadow-xs">
              <p className="text-[11px] font-semibold text-primary uppercase tracking-wider mb-3">想法预览</p>
              <p className="text-foreground font-semibold text-[14px] mb-2 leading-snug">{watch("title")}</p>
              <p className="text-muted-foreground text-[13px] line-clamp-5 leading-relaxed">
                {watch("description")}
              </p>
            </div>

            {/* Timeline preview */}
            <div className="bg-card border border-border rounded-2xl p-5 shadow-xs">
              <p className="text-[13px] font-semibold text-foreground mb-4">接下来会发生什么</p>
              <div className="space-y-3">
                {[
                  { time: "T+0", label: "系统为你匹配背景最相关的分身" },
                  { time: "T+1min", label: "多个分身基于各自背景结构化讨论" },
                  { time: "T+5min", label: "生成报告，推荐最值得认识的真人搭子" },
                ].map((item, i) => (
                  <div key={item.time} className="flex items-center gap-3">
                    <span className="text-primary font-mono text-[11px] w-14 flex-shrink-0 font-medium">
                      {item.time}
                    </span>
                    <div className="w-px h-4 bg-border flex-shrink-0" />
                    <span className="text-muted-foreground text-[13px]">{item.label}</span>
                  </div>
                ))}
              </div>
            </div>

            {error && (
              <div className="bg-red-500/8 border border-red-500/20 rounded-xl px-4 py-3 text-red-600 text-[13px]">
                {error}
              </div>
            )}

            <div className="pb-4" />
          </div>
        )}

        {/* Sticky bottom action */}
        <div className="sticky bottom-0 px-4 pb-8 pt-4 bg-gradient-to-t from-background via-background/95 to-transparent">
          {step === 1 ? (
            <button
              type="button"
              onClick={() => setStep(2)}
              disabled={!canProceed}
              className="w-full bg-primary hover:bg-primary/90 disabled:opacity-40 disabled:cursor-not-allowed text-primary-foreground font-semibold py-4 rounded-2xl transition-all duration-150 active:scale-[0.98] shadow-primary-md"
            >
              下一步：确认提交
            </button>
          ) : (
            <button
              type="submit"
              disabled={submitting}
              className="w-full bg-primary hover:bg-primary/90 disabled:opacity-50 text-primary-foreground font-semibold py-4 rounded-2xl transition-all duration-150 active:scale-[0.98] shadow-primary-md"
            >
              {submitting ? "提交中..." : "确认提交想法"}
            </button>
          )}
        </div>
      </form>
    </div>
  );
}
