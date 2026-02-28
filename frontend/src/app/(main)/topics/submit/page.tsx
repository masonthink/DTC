"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { agentApi, topicApi } from "@/lib/api";
import useSWR from "swr";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";

const TOPIC_TYPES = [
  { value: "business_idea", label: "💡 商业想法", desc: "创业方向、产品概念验证" },
  { value: "career_decision", label: "💼 职业决策", desc: "跳槽、转行、晋升" },
  { value: "tech_choice", label: "⚙️ 技术选型", desc: "架构、工具、技术栈" },
  { value: "product_design", label: "🎨 产品设计", desc: "功能、交互、用户体验" },
  { value: "investment", label: "📈 投资判断", desc: "项目评估、市场分析" },
  { value: "other", label: "💬 其他", desc: "其他深度话题" },
];

const schema = z.object({
  agent_id: z.string().min(1, "请选择一个分身"),
  topic_type: z.string().min(1, "请选择话题类型"),
  title: z.string().min(5, "话题标题至少5个字").max(200),
  description: z.string().min(20, "话题描述至少20个字").max(2000),
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
    } catch {
      setError("提交失败，请稍后重试");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-slate-950/95 backdrop-blur-xl border-b border-slate-800">
        <div className="px-4 py-4 flex items-center gap-3">
          {step === 2 ? (
            <button
              onClick={() => setStep(1)}
              className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-slate-800 text-slate-400 hover:text-white transition-colors"
            >
              <ArrowLeft className="w-5 h-5" />
            </button>
          ) : (
            <Link
              href="/dashboard"
              className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-slate-800 text-slate-400 hover:text-white transition-colors"
            >
              <ArrowLeft className="w-5 h-5" />
            </Link>
          )}
          <div className="flex-1">
            <h1 className="text-lg font-bold text-white">
              {step === 1 ? "提交话题" : "确认提交"}
            </h1>
          </div>
          {/* Step indicator */}
          <div className="flex gap-1.5">
            {[1, 2].map((s) => (
              <div
                key={s}
                className={`h-1.5 rounded-full transition-all ${
                  s === step ? "w-5 bg-indigo-500" : s < step ? "w-1.5 bg-indigo-700" : "w-1.5 bg-slate-700"
                }`}
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
              <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                选择分身
              </label>
              {agents && agents.length > 0 ? (
                <div className="space-y-2">
                  {agents.map((agent) => (
                    <label
                      key={agent.id}
                      className={`flex items-center gap-3 p-3.5 rounded-2xl border cursor-pointer transition-all active:scale-[0.99] ${
                        watch("agent_id") === agent.id
                          ? "border-indigo-500 bg-indigo-500/10"
                          : "border-slate-700 bg-slate-900 hover:border-slate-600"
                      }`}
                    >
                      <input
                        type="radio"
                        value={agent.id}
                        {...register("agent_id")}
                        className="sr-only"
                      />
                      <div className="w-9 h-9 rounded-xl bg-indigo-600/20 flex items-center justify-center text-lg flex-shrink-0">
                        🤖
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-white text-sm font-semibold">{agent.display_name}</p>
                        <p className="text-slate-400 text-xs mt-0.5 truncate">
                          {agent.industries.slice(0, 2).join(" · ")}
                        </p>
                      </div>
                      <div
                        className={`w-5 h-5 rounded-full border-2 flex-shrink-0 transition-colors ${
                          watch("agent_id") === agent.id
                            ? "border-indigo-500 bg-indigo-500"
                            : "border-slate-600"
                        }`}
                      />
                    </label>
                  ))}
                </div>
              ) : (
                <div className="bg-slate-900 border border-slate-700 rounded-2xl p-5 text-center">
                  <p className="text-slate-400 text-sm mb-3">还没有分身，先创建一个</p>
                  <Link
                    href="/agents/create"
                    className="text-indigo-400 text-sm font-medium hover:text-indigo-300"
                  >
                    创建分身 →
                  </Link>
                </div>
              )}
              {errors.agent_id && (
                <p className="text-red-400 text-xs mt-1.5">{errors.agent_id.message}</p>
              )}
            </div>

            {/* Topic type */}
            <div>
              <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                话题类型
              </label>
              <div className="grid grid-cols-2 gap-2">
                {TOPIC_TYPES.map((t) => (
                  <button
                    key={t.value}
                    type="button"
                    onClick={() => setValue("topic_type", t.value)}
                    className={`text-left p-3.5 rounded-2xl border text-xs transition-all active:scale-95 ${
                      selectedType === t.value
                        ? "border-indigo-500 bg-indigo-500/10"
                        : "border-slate-700 bg-slate-900 hover:border-slate-600"
                    }`}
                  >
                    <div className="font-semibold text-white mb-0.5">{t.label}</div>
                    <div className="text-slate-500 leading-tight">{t.desc}</div>
                  </button>
                ))}
              </div>
              {errors.topic_type && (
                <p className="text-red-400 text-xs mt-1.5">{errors.topic_type.message}</p>
              )}
            </div>

            {/* Title */}
            <div>
              <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                话题标题{" "}
                <span className="text-slate-600 normal-case font-normal">
                  {title?.length ?? 0}/200
                </span>
              </label>
              <input
                {...register("title")}
                placeholder="简洁地描述核心问题"
                className="w-full bg-slate-900 border border-slate-700 focus:border-indigo-500 rounded-xl px-4 py-3 text-white placeholder-slate-500 text-sm focus:outline-none transition-colors"
              />
              {errors.title && (
                <p className="text-red-400 text-xs mt-1.5">{errors.title.message}</p>
              )}
            </div>

            {/* Description */}
            <div>
              <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                详细描述{" "}
                <span className="text-slate-600 normal-case font-normal">
                  {description?.length ?? 0}/2000
                </span>
              </label>
              <textarea
                {...register("description")}
                placeholder="描述具体问题、当前思考、以及你最希望获得什么角度的碰撞..."
                rows={5}
                className="w-full bg-slate-900 border border-slate-700 focus:border-indigo-500 rounded-xl px-4 py-3 text-white placeholder-slate-500 text-sm focus:outline-none transition-colors resize-none"
              />
              {errors.description && (
                <p className="text-red-400 text-xs mt-1.5">{errors.description.message}</p>
              )}
            </div>

            {/* Background */}
            <div>
              <label className="block text-xs font-medium text-slate-400 uppercase tracking-wider mb-2">
                补充背景 <span className="text-slate-600 normal-case font-normal">（选填）</span>
              </label>
              <textarea
                {...register("background")}
                placeholder="你的行业背景、已有资源、限制条件..."
                rows={3}
                className="w-full bg-slate-900 border border-slate-700 focus:border-indigo-500 rounded-xl px-4 py-3 text-white placeholder-slate-500 text-sm focus:outline-none transition-colors resize-none"
              />
            </div>

            <div className="pb-4" />
          </div>
        )}

        {/* Step 2 */}
        {step === 2 && (
          <div className="px-4 pt-5 space-y-5">
            <div className="bg-slate-900 border border-slate-700/50 rounded-2xl p-5">
              <h3 className="text-white font-semibold mb-3">话题预览</h3>
              <p className="text-white font-medium mb-2">{watch("title")}</p>
              <p className="text-slate-400 text-sm line-clamp-5 leading-relaxed">
                {watch("description")}
              </p>
            </div>

            <div className="bg-slate-900 border border-slate-700/50 rounded-2xl p-5 space-y-3">
              <h3 className="text-slate-300 font-medium text-sm">接下来会发生什么</h3>
              {[
                { time: "T+0.5h", label: "为你匹配最合适的4位数字分身" },
                { time: "T+1h", label: "推送「匹配预告」通知" },
                { time: "T+1.5h~12h", label: "分身们展开4轮深度讨论" },
                { time: "T+48h", label: "你的专属分析报告出炉" },
              ].map((item) => (
                <div key={item.time} className="flex items-center gap-3 text-sm">
                  <span className="text-indigo-400 font-mono text-xs w-16 flex-shrink-0">
                    {item.time}
                  </span>
                  <span className="text-slate-400">{item.label}</span>
                </div>
              ))}
            </div>

            {error && (
              <div className="bg-red-500/10 border border-red-500/30 rounded-xl px-4 py-3 text-red-400 text-sm">
                {error}
              </div>
            )}

            <div className="pb-4" />
          </div>
        )}

        {/* Sticky bottom action */}
        <div className="sticky bottom-0 px-4 pb-8 pt-4 bg-gradient-to-t from-slate-950 via-slate-950 to-transparent">
          {step === 1 ? (
            <button
              type="button"
              onClick={() => setStep(2)}
              disabled={!canProceed}
              className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-40 disabled:cursor-not-allowed text-white font-semibold py-4 rounded-2xl transition-all active:scale-[0.98] shadow-lg shadow-indigo-600/20"
            >
              下一步：确认提交
            </button>
          ) : (
            <button
              type="submit"
              disabled={submitting}
              className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-semibold py-4 rounded-2xl transition-all active:scale-[0.98] shadow-lg shadow-indigo-600/20"
            >
              {submitting ? "提交中..." : "确认提交话题"}
            </button>
          )}
        </div>
      </form>
    </div>
  );
}
