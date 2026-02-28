"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";
import { tokenStorage } from "@/lib/api";

interface User {
  id: string;
  phone?: string;
  email?: string;
  display_name?: string;
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  setUser: (user: User) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      setUser: (user) => set({ user, isAuthenticated: true }),
      logout: () => {
        tokenStorage.clear();
        set({ user: null, isAuthenticated: false });
      },
    }),
    { name: "auth-store" }
  )
);
