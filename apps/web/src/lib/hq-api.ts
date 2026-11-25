import { apiFetch, ApiError } from "@/lib/api";

export { ApiError };

export async function hqFetch<T>(path: string, opts: { method?: string; body?: unknown } = {}): Promise<T> {
  return apiFetch<T>(path.startsWith("/api/") ? path : `/api/v1/hq${path}`, opts);
}

export type HQOverview = {
  range: { from: string; to: string; key: string };
  generated_at: string;
  deploy_version: string | null;
  service_health: ServiceRow[];
  unresolved_issues: number | null;
  critical_issues: number | null;
  affected_stores: number | null;
  api_success_rate: number | null;
  api_request_count: number | null;
  p95_latency_ms: number | null;
  failed_jobs: number | null;
  stuck_jobs: number | null;
  charts: {
    error_occurrences: { t: string; count: number }[];
    api_error_rate: { t: string; rate: number | null; errors: number; requests: number }[];
    api_latency: { t: string; avg_ms: number | null; max_ms: number | null; p95_ms: number | null }[];
    errors_by_module: { name: string; count: number }[];
    severity: { name: string; count: number }[];
  };
  recent_critical_issues: Issue[];
  recent_failed_jobs: JobRow[];
  most_affected_stores: { store_id: string; store_name?: string; store_slug?: string; count: number }[];
  unavailable: string[];
};

export type ServiceRow = {
  key: string;
  name: string;
  status: string;
  detail?: string | null;
  checked_at?: string | null;
  latency_ms?: number | null;
};

export type Issue = {
  id: string;
  fingerprint: string;
  title: string;
  severity: string;
  source: string;
  module: string;
  error_code: string;
  status: string;
  first_seen_at: string;
  last_seen_at: string;
  occurrence_count: number;
  affected_store_count: number;
  deploy_version?: string | null;
  admin_notes?: string | null;
};

export type JobRow = {
  id: string;
  queue: string;
  job_type: string;
  status: string;
  store_id?: string | null;
  attempts: number;
  max_attempts?: number | null;
  last_error?: string | null;
  created_at: string;
  updated_at: string;
  stuck: boolean;
};

export function fmtNum(n: number | null | undefined): string {
  if (n === null || n === undefined) return "—";
  return new Intl.NumberFormat("en-US").format(n);
}

export function fmtPct(n: number | null | undefined): string {
  if (n === null || n === undefined) return "—";
  return `${n.toFixed(2)}%`;
}

export function fmtMs(n: number | null | undefined): string {
  if (n === null || n === undefined) return "—";
  return `${Math.round(n)} ms`;
}

export function statusTone(status: string): string {
  switch (status) {
    case "operational":
    case "resolved":
      return "text-success";
    case "degraded":
    case "investigating":
    case "warning":
      return "text-warning";
    case "unavailable":
    case "critical":
    case "error":
    case "failed":
    case "dead":
      return "text-error";
    default:
      return "text-foreground-muted";
  }
}
