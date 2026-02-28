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
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-900 via-indigo-950 to-slate-900">
      <div className="w-full max-w-md px-6 py-8">
        {/* Logo */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-indigo-600 mb-4 animate-float">
            <span className="text-2xl">🤖</span>
          </div>
          <h1 className="text-2xl font-bold text-white">创建账号</h1>
          <p className="text-slate-400 mt-1 text-sm">加入数字分身社区</p>
        </div>

        {/* Card */}
        <div className="bg-slate-800/60 backdrop-blur border border-slate-700 rounded-2xl p-6">
          {/* Mode toggle */}
          <div className="flex rounded-lg bg-slate-700/50 p-1 mb-5">
            {(["phone", "email"] as const).map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => setMode(m)}
                className={`flex-1 py-2 text-sm rounded-md transition-all ${
                  mode === m
                    ? "bg-indigo-600 text-white shadow"
                    : "text-slate-400 hover:text-slate-200"
                }`}
              >
                {m === "phone" ? "手机号" : "邮箱"}
              </button>
            ))}
          </div>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {/* Display name */}
            <div>
              <label className="block text-sm text-slate-300 mb-1.5">昵称</label>
              <input
                {...register("display_name")}
                placeholder="你的昵称"
                className="w-full bg-slate-700/50 border border-slate-600 rounded-lg px-4 py-2.5 text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition-colors"
              />
              {errors.display_name && (
                <p className="text-red-400 text-xs mt-1">{errors.display_name.message}</p>
              )}
            </div>

            {/* Phone or email */}
            {mode === "phone" ? (
              <div>
                <label className="block text-sm text-slate-300 mb-1.5">手机号</label>
                <input
                  {...register("phone")}
                  type="tel"
                  placeholder="请输入手机号"
                  className="w-full bg-slate-700/50 border border-slate-600 rounded-lg px-4 py-2.5 text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition-colors"
                />
                {errors.phone && (
                  <p className="text-red-400 text-xs mt-1">{errors.phone.message}</p>
                )}
              </div>
            ) : (
              <div>
                <label className="block text-sm text-slate-300 mb-1.5">邮箱</label>
                <input
                  {...register("email")}
                  type="email"
                  placeholder="请输入邮箱"
                  className="w-full bg-slate-700/50 border border-slate-600 rounded-lg px-4 py-2.5 text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition-colors"
                />
                {errors.email && (
                  <p className="text-red-400 text-xs mt-1">{errors.email.message}</p>
                )}
              </div>
            )}

            {/* Password */}
            <div>
              <label className="block text-sm text-slate-300 mb-1.5">密码</label>
              <input
                {...register("password")}
                type="password"
                placeholder="至少8位"
                className="w-full bg-slate-700/50 border border-slate-600 rounded-lg px-4 py-2.5 text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition-colors"
              />
              {errors.password && (
                <p className="text-red-400 text-xs mt-1">{errors.password.message}</p>
              )}
            </div>

            {/* Confirm password */}
            <div>
              <label className="block text-sm text-slate-300 mb-1.5">确认密码</label>
              <input
                {...register("confirm")}
                type="password"
                placeholder="再次输入密码"
                className="w-full bg-slate-700/50 border border-slate-600 rounded-lg px-4 py-2.5 text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 transition-colors"
              />
              {errors.confirm && (
                <p className="text-red-400 text-xs mt-1">{errors.confirm.message}</p>
              )}
            </div>

            {error && (
              <div className="bg-red-500/10 border border-red-500/30 rounded-lg px-4 py-3 text-red-400 text-sm">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium py-3 rounded-xl transition-colors active:scale-[0.98] mt-2"
            >
              {loading ? "注册中..." : "创建账号"}
            </button>
          </form>

          <p className="text-center text-slate-400 text-sm mt-5">
            已有账号？{" "}
            <Link
              href="/login"
              className="text-indigo-400 hover:text-indigo-300 transition-colors"
            >
              立即登录
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
