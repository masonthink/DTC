"use client";

import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuthStore } from "@/store/auth";
import { tokenStorage } from "@/lib/api";
import {
  LogOut,
  User,
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
        { icon: Bot, label: "我的分身", href: "/agents", sub: "管理你的数字分身" },
        { icon: BookOpen, label: "话题记录", href: "/topics", sub: "查看所有讨论话题" },
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
    <div className="min-h-screen bg-slate-950">
      {/* Header */}
      <div className="bg-gradient-to-b from-indigo-950/40 to-slate-950 px-4 pt-12 pb-6">
        <div className="flex items-center gap-4">
          {/* Avatar */}
          <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-indigo-600 to-purple-600 flex items-center justify-center text-white font-bold text-xl shadow-lg shadow-indigo-600/20 flex-shrink-0">
            {initials}
          </div>

          <div className="flex-1 min-w-0">
            {user?.display_name && (
              <h2 className="text-white font-bold text-xl">{user.display_name}</h2>
            )}
            <p className="text-slate-400 text-sm mt-0.5 truncate">
              {user?.phone ? `📱 ${user.phone}` : user?.email ? `✉️ ${user.email}` : "未设置联系方式"}
            </p>
          </div>
        </div>
      </div>

      <div className="px-4 space-y-4">
        {menuSections.map((section) => (
          <div key={section.title}>
            <p className="text-xs font-medium text-slate-500 uppercase tracking-wider mb-2 px-1">
              {section.title}
            </p>
            <div className="bg-slate-900 border border-slate-700/50 rounded-2xl overflow-hidden">
              {section.items.map((item, idx) => (
                <div key={item.label}>
                  {item.href ? (
                    <Link
                      href={item.href}
                      className="flex items-center gap-3 px-4 py-4 hover:bg-slate-800/50 active:bg-slate-800 transition-colors"
                    >
                      <div className="w-9 h-9 rounded-xl bg-slate-800 flex items-center justify-center flex-shrink-0">
                        <item.icon className="w-4.5 h-4.5 text-slate-400" size={18} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-white text-sm font-medium">{item.label}</p>
                        {item.sub && (
                          <p className="text-slate-500 text-xs mt-0.5">{item.sub}</p>
                        )}
                      </div>
                      <ChevronRight className="w-4 h-4 text-slate-600" />
                    </Link>
                  ) : (
                    <div className="flex items-center gap-3 px-4 py-4 opacity-50">
                      <div className="w-9 h-9 rounded-xl bg-slate-800 flex items-center justify-center flex-shrink-0">
                        <item.icon className="w-4.5 h-4.5 text-slate-400" size={18} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-white text-sm font-medium">{item.label}</p>
                        {item.sub && (
                          <p className="text-slate-500 text-xs mt-0.5">{item.sub}</p>
                        )}
                      </div>
                      <span className="text-xs text-slate-600 bg-slate-800 px-2 py-0.5 rounded-full">
                        即将上线
                      </span>
                    </div>
                  )}
                  {idx < section.items.length - 1 && (
                    <div className="h-px bg-slate-800 mx-4" />
                  )}
                </div>
              ))}
            </div>
          </div>
        ))}

        {/* About */}
        <div className="bg-slate-900 border border-slate-700/50 rounded-2xl px-4 py-4">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-indigo-600/20 border border-indigo-500/20 flex items-center justify-center text-lg flex-shrink-0">
              🤖
            </div>
            <div className="flex-1">
              <p className="text-white text-sm font-semibold">Concors</p>
              <p className="text-slate-500 text-xs">让思想连接有价值的人</p>
            </div>
            <span className="text-xs text-slate-600">v0.1.0</span>
          </div>
        </div>

        {/* Logout */}
        <button
          onClick={handleLogout}
          className="w-full flex items-center justify-center gap-2 py-4 rounded-2xl border border-red-500/20 bg-red-500/5 text-red-400 hover:bg-red-500/10 transition-colors active:scale-[0.98] text-sm font-medium"
        >
          <LogOut className="w-4 h-4" />
          退出登录
        </button>

        <div className="pb-4" />
      </div>
    </div>
  );
}
