"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { authApi, tokenStorage } from "@/lib/api";
import Link from "next/link";

const loginSchema = z.object({
  phone: z.string().optional(),
  email: z.string().email().optional(),
  password: z.string().min(8, "密码至少8位"),
}).refine((data) => data.phone || data.email, {
  message: "请填写手机号或邮箱",
});

type LoginForm = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [mode, setMode] = useState<"phone" | "email">("phone");

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginForm>({ resolver: zodResolver(loginSchema) });

  const onSubmit = async (data: LoginForm) => {
    setLoading(true);
    setError(null);
    try {
      const result = await authApi.login(data);
      tokenStorage.set(result.access_token, result.refresh_token, new Date(result.expires_at));
      router.push("/dashboard");
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "登录失败，请检查账号密码";
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-50 px-4">
      <div className="w-full max-w-sm">
        {/* Logo */}
        <div className="text-center mb-8">
          <Link href="/" className="inline-flex items-center gap-2 mb-6">
            <span className="text-2xl font-bold text-slate-900">
              <span className="text-indigo-600">C</span>oncors
            </span>
          </Link>
          <h1 className="text-xl font-bold text-slate-900">欢迎回来</h1>
          <p className="text-slate-500 mt-1 text-sm">登录你的数字分身账号</p>
        </div>

        {/* Card */}
        <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-sm">
          {/* Mode toggle */}
          <div className="flex rounded-xl bg-slate-100 p-1 mb-5">
            {(["phone", "email"] as const).map((m) => (
              <button
                key={m}
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
                placeholder="请输入密码"
                className="w-full bg-slate-50 border border-slate-200 rounded-xl px-4 py-2.5 text-slate-900 placeholder-slate-400 focus:outline-none focus:border-indigo-500 focus:bg-white transition-colors text-sm"
              />
              {errors.password && (
                <p className="text-red-500 text-xs mt-1">{errors.password.message}</p>
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
              {loading ? "登录中..." : "登录"}
            </button>
          </form>

          <p className="text-center text-slate-500 text-sm mt-5">
            还没有账号？{" "}
            <Link href="/register" className="text-indigo-600 hover:text-indigo-500 font-medium transition-colors">
              立即注册
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
