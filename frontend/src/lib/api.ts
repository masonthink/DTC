/**
 * API client for digital-twin-community backend.
 * Uses axios with JWT auth token injection and automatic token refresh.
 */
import axios, { AxiosError, AxiosInstance, InternalAxiosRequestConfig } from "axios";
import Cookies from "js-cookie";
import { toast } from "@/hooks/use-toast";

const BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

// ─── Token management ──────────────────────────────────────────────────────

const TOKEN_KEY = "dtc_access_token";
const REFRESH_KEY = "dtc_refresh_token";

export const tokenStorage = {
  getAccess: () => Cookies.get(TOKEN_KEY) ?? null,
  getRefresh: () => Cookies.get(REFRESH_KEY) ?? null,
  set: (access: string, refresh: string, expiresAt: Date) => {
    Cookies.set(TOKEN_KEY, access, {
      expires: new Date(expiresAt),
      sameSite: "Strict",
      secure: process.env.NODE_ENV === "production",
    });
    Cookies.set(REFRESH_KEY, refresh, {
      expires: 30,
      sameSite: "Strict",
      secure: process.env.NODE_ENV === "production",
    });
  },
  clear: () => {
    Cookies.remove(TOKEN_KEY);
    Cookies.remove(REFRESH_KEY);
  },
};

// ─── Axios instance ────────────────────────────────────────────────────────

const api: AxiosInstance = axios.create({
  baseURL: `${BASE_URL}/api/v1`,
  timeout: 30_000,
  headers: { "Content-Type": "application/json" },
});

// Request interceptor: inject Bearer token
api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = tokenStorage.getAccess();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor: handle 401 → refresh
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value: string) => void;
  reject: (reason?: unknown) => void;
}> = [];

function processQueue(error: Error | null, token: string | null = null) {
  failedQueue.forEach(({ resolve, reject }) => {
    if (error) {
      reject(error);
    } else {
      resolve(token!);
    }
  });
  failedQueue = [];
}

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

    if (error.response?.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        return new Promise<string>((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        }).then((token) => {
          originalRequest.headers.Authorization = `Bearer ${token}`;
          return api(originalRequest);
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      const refreshToken = tokenStorage.getRefresh();
      if (!refreshToken) {
        tokenStorage.clear();
        toast({ title: "登录已过期", description: "请重新登录", variant: "destructive" });
        const returnUrl = encodeURIComponent(window.location.pathname + window.location.search);
        window.location.href = `/login?returnUrl=${returnUrl}`;
        return Promise.reject(error);
      }

      try {
        const { data } = await axios.post(`${BASE_URL}/api/v1/auth/refresh`, {
          refresh_token: refreshToken,
        });
        tokenStorage.set(data.access_token, data.refresh_token, new Date(data.expires_at));
        processQueue(null, data.access_token);
        originalRequest.headers.Authorization = `Bearer ${data.access_token}`;
        return api(originalRequest);
      } catch (refreshError) {
        processQueue(refreshError as Error);
        tokenStorage.clear();
        toast({ title: "登录已过期", description: "请重新登录", variant: "destructive" });
        const returnUrl = encodeURIComponent(window.location.pathname + window.location.search);
        window.location.href = `/login?returnUrl=${returnUrl}`;
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);

export default api;

// ─── Error extraction utility ─────────────────────────────────────────────

export function extractApiError(err: unknown): string {
  if (axios.isAxiosError(err)) {
    const data = err.response?.data;
    if (data?.message) return data.message;
    if (data?.error) return data.error;
    if (err.response?.status === 429) return "请求过于频繁，请稍后再试";
    if (err.response?.status === 500) return "服务器错误，请稍后重试";
  }
  if (err instanceof Error) return err.message;
  return "发生未知错误";
}

// ─── Auth ──────────────────────────────────────────────────────────────────

export const authApi = {
  register: (data: {
    phone?: string;
    email?: string;
    password: string;
    display_name: string;
  }) => api.post("/auth/register", data).then((r) => r.data),

  login: (data: { phone?: string; email?: string; password: string }) =>
    api.post("/auth/login", data).then((r) => r.data),

  refresh: (refreshToken: string) =>
    api.post("/auth/refresh", { refresh_token: refreshToken }).then((r) => r.data),
};

// ─── Agents ────────────────────────────────────────────────────────────────

export const agentApi = {
  create: (data: CreateAgentRequest) =>
    api.post("/agents", data).then((r) => r.data as Agent),

  list: () => api.get("/agents").then((r) => r.data as Agent[]),

  get: (id: string) => api.get(`/agents/${id}`).then((r) => r.data as Agent),

  update: (id: string, data: Partial<CreateAgentRequest>) =>
    api.put(`/agents/${id}`, data).then((r) => r.data as Agent),
};

// ─── Topics ────────────────────────────────────────────────────────────────

export const topicApi = {
  submit: (data: SubmitTopicRequest) =>
    api.post("/topics", data).then((r) => r.data as Topic),

  list: (params?: { limit?: number; offset?: number }) =>
    api.get("/topics", { params }).then((r) => r.data as TopicListResponse),

  get: (id: string) => api.get(`/topics/${id}`).then((r) => r.data as Topic),

  cancel: (id: string) => api.delete(`/topics/${id}`).then((r) => r.data),
};

// ─── Discussions ───────────────────────────────────────────────────────────

export const discussionApi = {
  get: (id: string) =>
    api.get(`/discussions/${id}`).then((r) => r.data as Discussion),

  getMessages: (id: string) =>
    api.get(`/discussions/${id}/messages`).then((r) => r.data as DiscussionMessage[]),
};

// ─── Reports ───────────────────────────────────────────────────────────────

export const reportApi = {
  get: (id: string) => api.get(`/reports/${id}`).then((r) => r.data as Report),

  rate: (id: string, data: { rating: number; feedback?: string }) =>
    api.post(`/reports/${id}/rating`, data).then((r) => r.data),
};

// ─── Connections ───────────────────────────────────────────────────────────

export const connectionApi = {
  request: (data: RequestConnectionRequest) =>
    api.post("/connections", data).then((r) => r.data as Connection),

  list: () => api.get("/connections").then((r) => r.data as Connection[]),

  respond: (id: string, data: { accept: boolean; target_contact?: string }) =>
    api.post(`/connections/${id}/respond`, data).then((r) => r.data as Connection),

  getContacts: (id: string) =>
    api.get(`/connections/${id}/contacts`).then((r) => r.data as ConnectionContacts),
};

// ─── Types ─────────────────────────────────────────────────────────────────

export interface Agent {
  id: string;
  user_id: string;
  agent_type: "professional" | "entrepreneur" | "investor" | "generalist";
  display_name: string;
  industries: string[];
  skills: string[];
  thinking_style: Record<string, number>;
  experience_years: number;
  anon_id: string;
  quality_score: number;
  discussion_count: number;
  created_at: string;
  questionnaire?: {
    primary_industry: string;
    years_experience: number;
    current_role: string;
    expertise: string[];
    problem_approach: string;
    decision_style: string;
    risk_tolerance: number;
    innovation_focus: number;
    preferred_role: string;
    discussion_strength: string;
    bio: string;
    additional_context?: string;
  };
}

export interface CreateAgentRequest {
  agent_type: Agent["agent_type"];
  display_name: string;
  questionnaire: {
    primary_industry: string;
    years_experience: number;
    current_role: string;
    expertise: string[];
    problem_approach: string;
    decision_style: string;
    risk_tolerance: number;
    innovation_focus: number;
    preferred_role: string;
    discussion_strength: string;
    bio: string;
  };
}

export interface Topic {
  id: string;
  submitter_user_id: string;
  submitter_agent_id: string;
  topic_type: string;
  title: string;
  description: string;
  background?: string;
  tags: string[];
  status: TopicStatus;
  submitted_at: string;
  matched_at?: string;
  report_ready_at?: string;
  discussion_id?: string;
  questionnaire?: Record<string, unknown>;
}

export type TopicStatus =
  | "pending_matching"
  | "matching"
  | "matched"
  | "discussion_active"
  | "report_generating"
  | "completed"
  | "failed"
  | "cancelled";

export interface TopicListResponse {
  items: Topic[];
  total: number;
}

export interface SubmitTopicRequest {
  agent_id: string;
  topic_type: string;
  title: string;
  description: string;
  background?: string;
  tags?: string[];
}

export interface Discussion {
  id: string;
  topic_id: string;
  status: string;
  current_round: number;
  participants: DiscussionParticipant[];
  is_degraded: boolean;
}

export interface DiscussionParticipant {
  agent_id: string;
  anon_id: string;
  role: "questioner" | "supporter" | "supplementer" | "inquirer";
}

export interface DiscussionMessage {
  round_num: number;
  agent_id: string;
  role: DiscussionParticipant["role"];
  content: string;
  key_point: string;
  addressed_to: DiscussionParticipant["role"];
  confidence: number;
  similarity_to_prev?: number;
  was_rewritten?: boolean;
  model_used?: string;
}

export interface Report {
  id: string;
  discussion_id: string;
  topic_id: string;
  summary: string;
  consensus_points: string[];
  divergence_points: string[];
  key_questions: string[];
  action_items: string[];
  blind_spots: string[];
  recommended_agents: RecommendedAgent[];
  quality_score: number;
  user_rating?: number;
  generated_at: string;
}

export interface RecommendedAgent {
  agent_id: string;
  anon_id: string;
  final_score: number;
  reasons: string[];
}

export interface Connection {
  id: string;
  requester_user_id: string;
  target_user_id: string;
  topic_id?: string;
  status: "pending" | "accepted" | "rejected" | "cancelled" | "expired";
  request_message?: string;
  requested_at: string;
  expires_at: string;
}

export interface ConnectionContacts {
  requester_contact: string;
  target_contact: string;
}

export interface RequestConnectionRequest {
  target_agent_id: string;
  topic_id?: string;
  request_message?: string;
  requester_contact: string;
}
