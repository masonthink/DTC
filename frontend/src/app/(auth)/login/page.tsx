"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { authApi, tokenStorage, extractApiError } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
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
  const setUser = useAuthStore((s) => s.setUser);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [mode, setMode] = useState<"phone" | "email">("phone");

  const {
    register,
    handleSubmit,
    clearErrors,
    resetField,
    formState: { errors },
  } = useForm<LoginForm>({ resolver: zodResolver(loginSchema) });

  const onSubmit = async (data: LoginForm) => {
    setLoading(true);
    setError(null);
    try {
      const result = await authApi.login(data);
      tokenStorage.set(result.access_token, result.refresh_token, new Date(result.expires_at));
      if (result.user) {
        setUser(result.user);
      }
      // TODO: If backend doesn't return user in login response, add a GET /auth/me call here
      router.push("/dashboard");
    } catch (err: unknown) {
      setError(extractApiError(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-mesh flex items-center justify-center px-4 py-12">
      {/* Decorative blobs */}
      <div className="fixed top-0 left-0 right-0 bottom-0 pointer-events-none overflow-hidden">
        <div className="absolute -top-32 -right-32 w-96 h-96 rounded-full bg-primary/5 blur-3xl" />
        <div className="absolute -bottom-32 -left-32 w-96 h-96 rounded-full bg-violet-400/4 blur-3xl" />
      </div>

      <div className="relative w-full max-w-sm animate-reveal-up">
        {/* Logo */}
        <div className="text-center mb-8">
          <Link href="/" className="inline-flex items-center gap-2.5 mb-6 group">
            <div className="w-8 h-8 rounded-xl bg-primary-gradient flex items-center justify-center shadow-primary-sm group-hover:shadow-primary-md transition-shadow duration-200">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <circle cx="4.5" cy="4.5" r="2.8" fill="white" fillOpacity="0.9" />
                <circle cx="11.5" cy="4.5" r="2.8" fill="white" fillOpacity="0.6" />
                <circle cx="8" cy="11" r="2.8" fill="white" fillOpacity="0.75" />
              </svg>
            </div>
            <span className="text-lg font-semibold tracking-tight text-foreground">
              Concors
            </span>
          </Link>
          <h1 className="text-2xl font-bold text-foreground tracking-tight">欢迎回来</h1>
          <p className="text-muted-foreground mt-1.5 text-[14px]">登录你的 Concors 账号</p>
        </div>

        {/* Card */}
        <div className="bg-card border border-border rounded-2xl p-6 shadow-md">
          {/* Mode toggle */}
          <div className="flex rounded-xl bg-muted p-1 mb-5">
            {(["phone", "email"] as const).map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => {
                  setMode(m);
                  clearErrors();
                  resetField(m === "phone" ? "email" : "phone");
                }}
                className={`flex-1 py-2 text-[13px] rounded-lg transition-all duration-200 font-medium ${
                  mode === m
                    ? "bg-card text-foreground shadow-xs"
                    : "text-muted-foreground hover:text-foreground/80"
                }`}
              >
                {m === "phone" ? "手机号" : "邮箱"}
              </button>
            ))}
          </div>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {mode === "phone" ? (
              <div>
                <label htmlFor="phone" className="block text-[13px] font-medium text-foreground/80 mb-1.5">
                  手机号
                </label>
                <input
                  id="phone"
                  {...register("phone")}
                  type="tel"
                  placeholder="请输入手机号"
                  className="w-full bg-background border border-border rounded-xl px-4 py-2.5 text-foreground placeholder-muted-foreground/60 focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 transition-all duration-150 text-[14px]"
                />
                {errors.phone && (
                  <p className="text-destructive text-[12px] mt-1.5">{errors.phone.message}</p>
                )}
              </div>
            ) : (
              <div>
                <label htmlFor="email" className="block text-[13px] font-medium text-foreground/80 mb-1.5">
                  邮箱
                </label>
                <input
                  id="email"
                  {...register("email")}
                  type="email"
                  placeholder="请输入邮箱"
                  className="w-full bg-background border border-border rounded-xl px-4 py-2.5 text-foreground placeholder-muted-foreground/60 focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 transition-all duration-150 text-[14px]"
                />
                {errors.email && (
                  <p className="text-destructive text-[12px] mt-1.5">{errors.email.message}</p>
                )}
              </div>
            )}

            <div>
              <label htmlFor="password" className="block text-[13px] font-medium text-foreground/80 mb-1.5">
                密码
              </label>
              <input
                id="password"
                {...register("password")}
                type="password"
                placeholder="请输入密码"
                className="w-full bg-background border border-border rounded-xl px-4 py-2.5 text-foreground placeholder-muted-foreground/60 focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/15 transition-all duration-150 text-[14px]"
              />
              {errors.password && (
                <p className="text-destructive text-[12px] mt-1.5">{errors.password.message}</p>
              )}
            </div>

            {error && (
              <div className="bg-destructive/8 border border-destructive/20 rounded-xl px-4 py-3 text-destructive text-[13px] flex items-start gap-2.5">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" className="flex-shrink-0 mt-0.5">
                  <circle cx="8" cy="8" r="7" stroke="currentColor" strokeWidth="1.5"/>
                  <path d="M8 5v4M8 11v.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
                </svg>
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-primary hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed text-primary-foreground font-semibold py-3 rounded-xl transition-all duration-200 active:scale-[0.98] shadow-primary-sm hover:shadow-primary-md text-[15px] mt-1"
            >
              {loading ? (
                <span className="flex items-center justify-center gap-2">
                  <svg className="animate-spin w-4 h-4" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
                  </svg>
                  登录中
                </span>
              ) : "登录"}
            </button>
          </form>

          <div className="mt-5 pt-5 border-t border-border text-center">
            <p className="text-muted-foreground text-[13px]">
              还没有账号？{" "}
              <Link href="/register" className="text-primary hover:text-primary/80 font-medium transition-colors">
                立即注册
              </Link>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
