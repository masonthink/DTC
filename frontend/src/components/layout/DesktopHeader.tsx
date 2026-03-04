"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { Bell, LogOut, Settings, User } from "lucide-react";
import { useAuthStore } from "@/store/auth";
import { useState, useRef, useEffect } from "react";
import { cn } from "@/lib/utils";

export function DesktopHeader() {
  const { user, logout } = useAuthStore();
  const router = useRouter();
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setDropdownOpen(false);
      }
    }
    if (dropdownOpen) {
      document.addEventListener("mousedown", handleClickOutside);
      return () =>
        document.removeEventListener("mousedown", handleClickOutside);
    }
  }, [dropdownOpen]);

  // Close on Escape key
  useEffect(() => {
    function handleEscape(event: KeyboardEvent) {
      if (event.key === "Escape") setDropdownOpen(false);
    }
    if (dropdownOpen) {
      document.addEventListener("keydown", handleEscape);
      return () => document.removeEventListener("keydown", handleEscape);
    }
  }, [dropdownOpen]);

  const initials = user?.display_name
    ? user.display_name.slice(0, 2)
    : user?.email
    ? user.email.slice(0, 2).toUpperCase()
    : "??";

  const handleLogout = () => {
    setDropdownOpen(false);
    logout();
    router.push("/login");
  };

  return (
    <header className="hidden md:flex items-center h-14 px-5 border-b border-border/60 bg-card/92 backdrop-blur-xl sticky top-0 z-30">
      {/* Left: Logo */}
      <Link href="/dashboard" className="flex items-center gap-2 mr-auto">
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
      </Link>

      {/* Right: Actions */}
      <div className="flex items-center gap-1">
        {/* Notifications */}
        <button
          aria-label="通知"
          className="w-9 h-9 flex items-center justify-center rounded-xl hover:bg-muted text-muted-foreground hover:text-foreground transition-all duration-150"
        >
          <Bell className="w-[18px] h-[18px]" />
        </button>

        {/* User avatar dropdown */}
        <div className="relative ml-1" ref={dropdownRef}>
          <button
            onClick={() => setDropdownOpen((prev) => !prev)}
            className={cn(
              "w-8 h-8 rounded-xl flex items-center justify-center text-[12px] font-bold transition-all duration-150",
              "bg-primary/10 text-primary border border-primary/20",
              "hover:bg-primary/15 hover:border-primary/30",
              dropdownOpen && "ring-2 ring-primary/30"
            )}
            aria-label="用户菜单"
            aria-expanded={dropdownOpen}
            aria-haspopup="true"
          >
            {initials}
          </button>

          {/* Dropdown menu */}
          {dropdownOpen && (
            <div className="absolute right-0 top-full mt-2 w-52 bg-card border border-border rounded-xl shadow-lg py-1 animate-scale-in z-50">
              {/* User info */}
              <div className="px-3 py-2.5 border-b border-border/60">
                {user?.display_name && (
                  <p className="text-[13px] font-semibold text-foreground truncate">
                    {user.display_name}
                  </p>
                )}
                <p className="text-[11px] text-muted-foreground truncate mt-0.5">
                  {user?.email || user?.phone || "未设置联系方式"}
                </p>
              </div>

              {/* Menu items */}
              <div className="py-1">
                <Link
                  href="/profile"
                  onClick={() => setDropdownOpen(false)}
                  className="flex items-center gap-2.5 px-3 py-2 text-[13px] text-foreground hover:bg-muted transition-colors"
                >
                  <User className="w-4 h-4 text-muted-foreground" />
                  个人中心
                </Link>
                <Link
                  href="/agents"
                  onClick={() => setDropdownOpen(false)}
                  className="flex items-center gap-2.5 px-3 py-2 text-[13px] text-foreground hover:bg-muted transition-colors"
                >
                  <Settings className="w-4 h-4 text-muted-foreground" />
                  账号设置
                </Link>
              </div>

              <div className="border-t border-border/60 pt-1">
                <button
                  onClick={handleLogout}
                  className="flex items-center gap-2.5 px-3 py-2 w-full text-[13px] text-red-500 hover:bg-red-50 transition-colors"
                >
                  <LogOut className="w-4 h-4" />
                  退出登录
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
