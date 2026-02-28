"use client";

import * as React from "react";

export interface Toast {
  id: string;
  title?: string;
  description?: string;
  variant?: "default" | "destructive";
  duration?: number;
}

type ToastInput = Omit<Toast, "id">;

type State = { toasts: Toast[] };

const listeners: Array<(state: State) => void> = [];
let memoryState: State = { toasts: [] };

function dispatch(action: { type: "add" | "remove"; toast?: Toast; id?: string }) {
  if (action.type === "add" && action.toast) {
    memoryState = { toasts: [action.toast, ...memoryState.toasts].slice(0, 3) };
  } else if (action.type === "remove") {
    memoryState = {
      toasts: memoryState.toasts.filter((t) => t.id !== action.id),
    };
  }
  listeners.forEach((l) => l(memoryState));
}

function toast(input: ToastInput) {
  const id = Math.random().toString(36).slice(2);
  dispatch({ type: "add", toast: { ...input, id } });
  setTimeout(() => {
    dispatch({ type: "remove", id });
  }, input.duration ?? 4000);
}

function useToast() {
  const [state, setState] = React.useState<State>(memoryState);

  React.useEffect(() => {
    listeners.push(setState);
    return () => {
      const idx = listeners.indexOf(setState);
      if (idx > -1) listeners.splice(idx, 1);
    };
  }, []);

  return { toasts: state.toasts, toast };
}

export { useToast, toast };
