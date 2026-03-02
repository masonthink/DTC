"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Home, BookOpen, PlusCircle, Users, User } from "lucide-react";
import { cn } from "@/lib/utils";

const tabs = [
  { href: "/dashboard", icon: Home, label: "首页" },
  { href: "/topics", icon: BookOpen, label: "问题" },
  { href: "/topics/submit", icon: PlusCircle, label: "提交", primary: true },
  { href: "/connections", icon: Users, label: "人脉" },
  { href: "/profile", icon: User, label: "我的" },
];

export function BottomNav() {
  const pathname = usePathname();

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50">
      {/* Frosted glass background with gradient mask at top */}
      <div className="absolute inset-0 bg-card/92 backdrop-blur-2xl border-t border-border/60" />

      <div
        className="relative flex items-end justify-around max-w-screen-sm mx-auto"
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
                aria-label="提交问题"
                className="flex flex-col items-center pb-2 pt-1 px-3"
              >
                <div
                  className={cn(
                    "w-12 h-12 rounded-2xl flex items-center justify-center -mt-5 transition-all duration-200",
                    "bg-primary shadow-primary-sm active:scale-90",
                    isActive && "shadow-primary-md"
                  )}
                >
                  <tab.icon className="w-5 h-5 text-primary-foreground" strokeWidth={2.5} />
                </div>
                <span className="text-[10px] leading-none font-medium text-muted-foreground mt-1">
                  {tab.label}
                </span>
              </Link>
            );
          }

          return (
            <Link
              key={tab.href}
              href={tab.href}
              className={cn(
                "flex flex-col items-center gap-1 pt-3 pb-1 px-4 min-w-[56px] transition-all duration-150 active:opacity-60",
                isActive ? "text-primary" : "text-muted-foreground"
              )}
            >
              <tab.icon
                className={cn(
                  "w-[22px] h-[22px] transition-all duration-150",
                  isActive && "scale-105"
                )}
                strokeWidth={isActive ? 2.5 : 1.75}
              />
              <span className={cn(
                "text-[10px] leading-none font-medium",
                isActive ? "opacity-100" : "opacity-70"
              )}>
                {tab.label}
              </span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
