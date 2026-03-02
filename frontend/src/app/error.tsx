"use client";

import { useEffect } from "react";
import { AlertTriangle } from "lucide-react";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center px-4 text-center">
      <div className="w-16 h-16 rounded-2xl bg-amber-50 border border-amber-100 flex items-center justify-center mb-5">
        <AlertTriangle className="w-7 h-7 text-amber-500" />
      </div>
      <h1 className="text-[18px] font-bold text-foreground mb-2">出现了一些问题</h1>
      <p className="text-muted-foreground text-[13px] mb-8 max-w-xs leading-relaxed">
        页面加载失败，请尝试刷新
      </p>
      <button
        onClick={reset}
        className="bg-primary hover:bg-primary/90 active:scale-95 text-primary-foreground px-6 py-3 rounded-xl font-semibold text-[13px] transition-all duration-150 shadow-primary-sm"
      >
        重新加载
      </button>
    </div>
  );
}
