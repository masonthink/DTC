"use client";

import { useEffect } from "react";

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
    <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center px-4 text-center">
      <p className="text-5xl mb-6">⚠️</p>
      <h1 className="text-xl font-bold text-white mb-3">出现了一些问题</h1>
      <p className="text-slate-400 text-sm mb-8 max-w-xs">
        页面加载失败，请尝试刷新
      </p>
      <button
        onClick={reset}
        className="bg-indigo-600 hover:bg-indigo-500 active:scale-95 text-white px-6 py-3 rounded-xl font-medium transition-all"
      >
        重新加载
      </button>
    </div>
  );
}
