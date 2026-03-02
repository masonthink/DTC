"use client";

import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuthStore } from "@/store/auth";
import {
  LogOut,
  ChevronRight,
  Bot,
  BookOpen,
  Shield,
  Bell,
} from "lucide-react";

export default function ProfilePage() {
  const router = useRouter();
  const { user, logout } = useAuthStore();

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  const initials = user?.display_name
    ? user.display_name.slice(0, 2)
    : user?.phone?.slice(-4) ?? "??";

  const menuSections = [
    {
      title: "账号",
      items: [
        { icon: Bot, label: "我的背景", href: "/agents", sub: "管理你的专业背景档案" },
        { icon: BookOpen, label: "问题记录", href: "/topics", sub: "查看所有提交的问题" },
      ],
    },
    {
      title: "设置",
      items: [
        { icon: Bell, label: "通知设置", href: null, sub: "即将上线" },
        { icon: Shield, label: "隐私说明", href: null, sub: "了解数据如何被保护" },
      ],
    },
  ];

  return (
    <div className="min-h-screen bg-background">
      {/* Profile header — uses a deep primary gradient for visual impact */}
      <div className="relative bg-gradient-to-b from-primary/90 to-primary/70 px-4 pt-12 pb-8 overflow-hidden">
        {/* Decorative background elements */}
        <div className="absolute inset-0 pointer-events-none overflow-hidden">
          <div className="absolute -top-8 -right-8 w-48 h-48 rounded-full bg-white/5" />
          <div className="absolute bottom-0 left-0 w-32 h-32 rounded-full bg-black/10" />
        </div>

        <div className="relative flex items-center gap-4">
          {/* Avatar */}
          <div className="w-16 h-16 rounded-2xl bg-white/20 backdrop-blur-sm border border-white/25 flex items-center justify-center text-white font-bold text-[20px] flex-shrink-0 shadow-lg">
            {initials}
          </div>

          <div className="flex-1 min-w-0">
            {user?.display_name && (
              <h2 className="text-white font-bold text-[20px] mb-0.5">{user.display_name}</h2>
            )}
            <p className="text-white/70 text-[13px] truncate">
              {user?.phone
                ? user.phone
                : user?.email
                ? user.email
                : "未设置联系方式"}
            </p>
          </div>
        </div>
      </div>

      <div className="px-4 pt-5 space-y-4">
        {menuSections.map((section) => (
          <div key={section.title}>
            <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2 px-1">
              {section.title}
            </p>
            <div className="bg-card border border-border rounded-2xl overflow-hidden shadow-xs">
              {section.items.map((item, idx) => (
                <div key={item.label}>
                  {item.href ? (
                    <Link
                      href={item.href}
                      className="flex items-center gap-3 px-4 py-4 hover:bg-muted/50 active:bg-muted transition-colors"
                    >
                      <div className="w-9 h-9 rounded-xl bg-muted border border-border flex items-center justify-center flex-shrink-0">
                        <item.icon className="text-muted-foreground" size={16} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-foreground text-[13px] font-medium">{item.label}</p>
                        {item.sub && (
                          <p className="text-muted-foreground text-[11px] mt-0.5">{item.sub}</p>
                        )}
                      </div>
                      <ChevronRight className="w-4 h-4 text-muted-foreground/50" />
                    </Link>
                  ) : (
                    <div className="flex items-center gap-3 px-4 py-4 opacity-50">
                      <div className="w-9 h-9 rounded-xl bg-muted border border-border flex items-center justify-center flex-shrink-0">
                        <item.icon className="text-muted-foreground" size={16} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-foreground text-[13px] font-medium">{item.label}</p>
                        {item.sub && (
                          <p className="text-muted-foreground text-[11px] mt-0.5">{item.sub}</p>
                        )}
                      </div>
                      <span className="text-[10px] font-medium text-muted-foreground bg-muted border border-border px-2 py-0.5 rounded-full">
                        即将上线
                      </span>
                    </div>
                  )}
                  {idx < section.items.length - 1 && (
                    <div className="h-px bg-border/60 mx-4" />
                  )}
                </div>
              ))}
            </div>
          </div>
        ))}

        {/* About */}
        <div className="bg-card border border-border rounded-2xl px-4 py-4 shadow-xs">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-primary/10 border border-primary/15 flex items-center justify-center text-base flex-shrink-0">
              🤖
            </div>
            <div className="flex-1">
              <p className="text-foreground text-[13px] font-semibold">Concors</p>
              <p className="text-muted-foreground text-[11px] mt-0.5">AI 多角度分析，帮你做更好的决策</p>
            </div>
            <span className="text-[11px] text-muted-foreground font-mono">v0.1.0</span>
          </div>
        </div>

        {/* Logout */}
        <button
          onClick={handleLogout}
          className="w-full flex items-center justify-center gap-2 py-4 rounded-2xl border border-red-200 bg-red-50 text-red-600 hover:bg-red-100 transition-colors duration-150 active:scale-[0.98] text-[13px] font-medium"
        >
          <LogOut className="w-4 h-4" />
          退出登录
        </button>

        <div className="pb-4" />
      </div>
    </div>
  );
}
