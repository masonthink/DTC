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
            "w-full max-w-sm rounded-2xl p-4 shadow-2xl pointer-events-auto border animate-unlock",
            t.variant === "destructive"
              ? "bg-red-950 border-red-700/50 text-white"
              : "bg-slate-800 border-slate-600/50 text-white"
          )}
        >
          {t.title && (
            <p className="font-semibold text-sm">{t.title}</p>
          )}
          {t.description && (
            <p className="text-sm text-slate-400 mt-0.5">{t.description}</p>
          )}
        </div>
      ))}
    </div>
  );
}
