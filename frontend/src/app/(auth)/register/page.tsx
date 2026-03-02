"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { authApi, tokenStorage } from "@/lib/api";
import Link from "next/link";

const registerSchema = z
  .object({
    display_name: z.string().min(2, "昵称至少2个字"),
    phone: z.string().optional(),
    email: z.string().email("邮箱格式不正确").optional(),
    password: z.string().min(8, "密码至少8位"),
    confirm: z.string().min(8, "请确认密码"),
  })
  .refine((d) => d.phone || d.email, { message: "请填写手机号或邮箱" })
  .refine((d) => d.password === d.confirm, {
    message: "两次密码不一致",
    path: ["confirm"],
  });

type RegisterForm = z.infer<typeof registerSchema>;

export default function RegisterPage() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [mode, setMode] = useState<"phone" | "email">("phone");

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterForm>({ resolver: zodResolver(registerSchema) });

  const onSubmit = async (data: RegisterForm) => {
    setLoading(true);
    setError(null);
    try {
      const result = await authApi.register({
        display_name: data.display_name,
        phone: mode === "phone" ? data.phone : undefined,
        email: mode === "email" ? data.email : undefined,
        password: data.password,
      });
      tokenStorage.set(
        result.access_token,
        result.refresh_token,
        new Date(result.expires_at)
      );
      router.push("/agents/create");
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : "注册失败，请稍后重试";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-50 px-4 py-8">
      <div className="w-full max-w-sm">
        {/* Logo */}
        <div className="text-center mb-8">
          <Link href="/" className="inline-flex items-center gap-2 mb-6">
            <span className="text-2xl font-bold text-slate-900">
              <span className="text-indigo-600">C</span>oncors
            </span>
          </Link>
          <h1 className="text-xl font-bold text-slate-900">创建账号</h1>
          <p className="text-slate-500 mt-1 text-sm">加入数字分身社区</p>
        </div>

        {/* Card */}
        <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-sm">
          {/* Mode toggle */}
          <div className="flex rounded-xl bg-slate-100 p-1 mb-5">
            {(["phone", "email"] as const).map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => setMode(m)}
                className={`flex-1 py-2 text-sm rounded-lg transition-all font-medium ${
                  mode === m
                    ? "bg-white text-slate-900 shadow-sm"
                    : "text-slate-500 hover:text-slate-700"
                }`}
              >
                {m === "phone" ? "手机号" : "邮箱"}
              </button>
            ))}
          </div>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div>
              <label className="block text-sm text-slate-700 font-medium mb-1.5">昵称</label>
              <input
                {...register("display_name")}
                placeholder="你的昵称"
                className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-2.5 text-slate-900 placeholder-slate-400 focus:outline-none focus:border-indigo-500 focus:bg-white transition-colors text-sm"
              />
              {errors.display_name && (
                <p className="text-red-500 text-xs mt-1">{errors.display_name.message}</p>
              )}
            </div>

            {mode === "phone" ? (
              <div>
                <label className="block text-sm text-slate-700 font-medium mb-1.5">手机号</label>
                <input
                  {...register("phone")}
                  type="tel"
                  placeholder="请输入手机号"
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-2.5 text-slate-900 placeholder-slate-400 focus:outline-none focus:border-indigo-500 focus:bg-white transition-colors text-sm"
                />
                {errors.phone && (
                  <p className="text-red-500 text-xs mt-1">{errors.phone.message}</p>
                )}
              </div>
            ) : (
              <div>
                <label className="block text-sm text-slate-700 font-medium mb-1.5">邮箱</label>
                <input
                  {...register("email")}
                  type="email"
                  placeholder="请输入邮箱"
                  className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-2.5 text-slate-900 placeholder-slate-400 focus:outline-none focus:border-indigo-500 focus:bg-white transition-colors text-sm"
                />
                {errors.email && (
                  <p className="text-red-500 text-xs mt-1">{errors.email.message}</p>
                )}
              </div>
            )}

            <div>
              <label className="block text-sm text-slate-700 font-medium mb-1.5">密码</label>
              <input
                {...register("password")}
                type="password"
                placeholder="至少8位"
                className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-2.5 text-slate-900 placeholder-slate-400 focus:outline-none focus:border-indigo-500 focus:bg-white transition-colors text-sm"
              />
              {errors.password && (
                <p className="text-red-500 text-xs mt-1">{errors.password.message}</p>
              )}
            </div>

            <div>
              <label className="block text-sm text-slate-700 font-medium mb-1.5">确认密码</label>
              <input
                {...register("confirm")}
                type="password"
                placeholder="再次输入密码"
                className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-2.5 text-slate-900 placeholder-slate-400 focus:outline-none focus:border-indigo-500 focus:bg-white transition-colors text-sm"
              />
              {errors.confirm && (
                <p className="text-red-500 text-xs mt-1">{errors.confirm.message}</p>
              )}
            </div>

            {error && (
              <div className="bg-red-50 border border-red-200 rounded-xl px-4 py-3 text-red-600 text-sm">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed text-white font-semibold py-3 rounded-xl transition-all active:scale-[0.98] shadow-sm shadow-indigo-600/20 mt-1"
            >
              {loading ? "注册中..." : "创建账号"}
            </button>
          </form>

          <p className="text-center text-slate-500 text-sm mt-5">
            已有账号？{" "}
            <Link
              href="/login"
              className="text-indigo-600 hover:text-indigo-500 font-medium transition-colors"
            >
              立即登录
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
