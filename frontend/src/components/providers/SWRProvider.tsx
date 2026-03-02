"use client";

import { SWRConfig } from "swr";
import { toast } from "@/hooks/use-toast";
import { extractApiError } from "@/lib/api";

export function SWRProvider({ children }: { children: React.ReactNode }) {
  return (
    <SWRConfig
      value={{
        onError: (err) => {
          const message = extractApiError(err);
          toast({ title: "数据加载失败", description: message, variant: "destructive" });
        },
        shouldRetryOnError: false,
      }}
    >
      {children}
    </SWRConfig>
  );
}
