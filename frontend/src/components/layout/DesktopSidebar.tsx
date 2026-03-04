"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Home, BookOpen, PlusCircle, Users, User } from "lucide-react";
import { cn } from "@/lib/utils";

const tabs = [
  { href: "/dashboard", icon: Home, label: "首页" },
  { href: "/topics", icon: BookOpen, label: "想法" },
  { href: "/topics/submit", icon: PlusCircle, label: "发布", primary: true },
  { href: "/connections", icon: Users, label: "搭子" },
  { href: "/profile", icon: User, label: "我的" },
];

export function DesktopSidebar() {
  const pathname = usePathname();

  return (
    <aside className="hidden md:flex flex-col w-60 flex-shrink-0 border-r border-border/60 bg-card/50 h-screen sticky top-0">
      {/* Logo */}
      <div className="px-5 h-14 flex items-center gap-2 border-b border-border/40">
        <div className="w-7 h-7 rounded-lg bg-primary-gradient flex items-center justify-center flex-shrink-0">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
            <circle cx="4" cy="4" r="2.5" fill="white" fillOpacity="0.9" />
            <circle cx="10" cy="4" r="2.5" fill="white" fillOpacity="0.6" />
            <circle cx="7" cy="10" r="2.5" fill="white" fillOpacity="0.75" />
          </svg>
        </div>
        <span className="text-[15px] font-semibold tracking-tight text-foreground">
          Concors
        </span>
        <span className="text-[9px] font-bold text-primary bg-primary/10 border border-primary/20 px-1.5 py-0.5 rounded-full uppercase tracking-wider leading-none">
          Beta
        </span>
      </div>

      {/* Nav items */}
      <nav className="flex-1 px-3 py-4 space-y-1">
        {tabs.map((tab) => {
          const isActive =
            pathname === tab.href ||
            (tab.href !== "/dashboard" && pathname.startsWith(tab.href + "/"));

          if (tab.primary) {
            return (
              <Link
                key={tab.href}
                href={tab.href}
                className={cn(
                  "flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-150",
                  "bg-primary text-primary-foreground shadow-primary-sm hover:bg-primary/90",
                  isActive && "shadow-primary-md"
                )}
              >
                <tab.icon className="w-[18px] h-[18px]" strokeWidth={2.5} />
                <span className="text-[13px] font-semibold">{tab.label}</span>
              </Link>
            );
          }

          return (
            <Link
              key={tab.href}
              href={tab.href}
              className={cn(
                "flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-150",
                isActive
                  ? "bg-primary/8 text-primary font-medium"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground"
              )}
            >
              <tab.icon
                className="w-[18px] h-[18px]"
                strokeWidth={isActive ? 2.25 : 1.75}
              />
              <span className="text-[13px]">{tab.label}</span>
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
