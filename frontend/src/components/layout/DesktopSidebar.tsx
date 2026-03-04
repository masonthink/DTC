"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Home, Lightbulb, Users, Bot, PenSquare } from "lucide-react";
import { cn } from "@/lib/utils";

const navItems = [
  { href: "/dashboard", icon: Home, label: "首页" },
  { href: "/topics", icon: Lightbulb, label: "想法" },
  { href: "/connections", icon: Users, label: "搭子" },
  { href: "/agents", icon: Bot, label: "分身" },
];

export function DesktopSidebar() {
  const pathname = usePathname();

  return (
    <aside className="hidden md:flex flex-col w-[68px] flex-shrink-0 border-r border-border/60 bg-card/80 h-[calc(100vh-56px)] sticky top-14">
      {/* Navigation items */}
      <nav className="flex-1 flex flex-col items-center gap-1 px-2 pt-4">
        {navItems.map((item) => {
          const isActive =
            pathname === item.href ||
            (item.href !== "/dashboard" && pathname.startsWith(item.href + "/"));

          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex flex-col items-center justify-center w-full py-2 rounded-xl transition-all duration-150 group",
                isActive
                  ? "bg-primary/10 text-primary"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground"
              )}
              aria-current={isActive ? "page" : undefined}
            >
              <item.icon
                className="w-5 h-5 mb-0.5"
                strokeWidth={isActive ? 2.25 : 1.75}
              />
              <span
                className={cn(
                  "text-[10px] leading-none",
                  isActive ? "font-semibold" : "font-medium"
                )}
              >
                {item.label}
              </span>
            </Link>
          );
        })}
      </nav>

      {/* CTA: Submit topic */}
      <div className="px-2 pb-4">
        <Link
          href="/topics/submit"
          className={cn(
            "flex flex-col items-center justify-center w-full py-2.5 rounded-xl transition-all duration-200",
            "bg-primary text-primary-foreground shadow-primary-sm",
            "hover:bg-primary/90 hover:shadow-primary-md",
            "active:scale-95"
          )}
        >
          <PenSquare className="w-5 h-5 mb-0.5" strokeWidth={2} />
          <span className="text-[10px] leading-none font-semibold">
            发布想法
          </span>
        </Link>
      </div>
    </aside>
  );
}
