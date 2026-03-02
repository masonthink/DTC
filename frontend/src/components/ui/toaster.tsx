"use client";

import { useToast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";

export function Toaster() {
  const { toasts } = useToast();

  if (!toasts.length) return null;

  return (
    <div className="fixed top-4 left-0 right-0 z-[100] flex flex-col items-center gap-2 px-4 pointer-events-none">
      {toasts.map((t) => (
        <div
          key={t.id}
          className={cn(
            "w-full max-w-sm rounded-2xl px-4 py-3.5 shadow-lg pointer-events-auto border animate-unlock",
            t.variant === "destructive"
              ? "bg-red-600 border-red-500/50 text-white"
              : "bg-foreground border-foreground/10 text-background"
          )}
        >
          {t.title && (
            <p className="font-semibold text-[13px] leading-snug">{t.title}</p>
          )}
          {t.description && (
            <p className={cn("text-[12px] mt-0.5 leading-relaxed", t.variant === "destructive" ? "text-white/80" : "text-background/70")}>
              {t.description}
            </p>
          )}
        </div>
      ))}
    </div>
  );
}
