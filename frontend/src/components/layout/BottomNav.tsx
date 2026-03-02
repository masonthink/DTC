"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Home, BookOpen, PlusCircle, Users, User } from "lucide-react";
import { cn } from "@/lib/utils";

const tabs = [
  { href: "/dashboard", icon: Home, label: "首页" },
  { href: "/topics", icon: BookOpen, label: "话题" },
  { href: "/topics/submit", icon: PlusCircle, label: "发布", primary: true },
  { href: "/connections", icon: Users, label: "连接" },
  { href: "/agents", icon: User, label: "分身" },
];

export function BottomNav() {
  const pathname = usePathname();

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 bg-white/95 backdrop-blur-xl border-t border-slate-200/80">
      <div
        className="flex items-end justify-around max-w-screen-sm mx-auto"
        style={{ paddingBottom: "max(env(safe-area-inset-bottom), 8px)" }}
      >
        {tabs.map((tab) => {
          const isActive =
            pathname === tab.href ||
            (tab.href !== "/dashboard" && pathname.startsWith(tab.href + "/"));

          if (tab.primary) {
            return (
              <Link
                key={tab.href}
                href={tab.href}
                className="flex flex-col items-center pb-2 pt-1 px-3"
              >
                <div
                  className={cn(
                    "w-12 h-12 rounded-2xl flex items-center justify-center shadow-lg -mt-5 transition-all active:scale-95",
                    isActive
                      ? "bg-indigo-500 shadow-indigo-500/30"
                      : "bg-indigo-600 shadow-indigo-600/20"
                  )}
                >
                  <tab.icon className="w-6 h-6 text-white" strokeWidth={2} />
                </div>
              </Link>
            );
          }

          return (
            <Link
              key={tab.href}
              href={tab.href}
              className={cn(
                "flex flex-col items-center gap-1 pt-3 pb-1 px-4 min-w-[56px] transition-colors active:opacity-70",
                isActive ? "text-indigo-600" : "text-slate-400"
              )}
            >
              <tab.icon
                className={cn("w-5 h-5 transition-all", isActive && "scale-110")}
                strokeWidth={isActive ? 2.5 : 1.75}
              />
              <span className="text-[10px] font-medium leading-none">{tab.label}</span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
