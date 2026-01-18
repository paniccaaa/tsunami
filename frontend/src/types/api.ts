export interface AttackConfig {
  url: string;
  method: string;
  body?: string;
  headers?: string[];
  rate: string;
  duration: string;
  timeout: string;
  workers: number;
  connections: number;
}

export interface StartResponse {
  id: string;
  status: string;
  started_at: string;
}

export interface StopResponse {
  id: string;
  status: string;
  stopped_at: string;
}

export interface LatencyPercentiles {
  p50: string;
  p90: string;
  p95: string;
  p99: string;
}

export interface MetricsPayload {
  total_requests: number;
  successes: number;
  failures: number;
  current_rps: number;
  average_latency: string;
  latency_percentiles?: LatencyPercentiles;
  status_codes: Record<number, number>;
}

export interface StatusResponse {
  id?: string;
  status: string;
  started_at?: string;
  elapsed_time?: string;
  metrics?: MetricsPayload;
}

export interface ConfigPayload {
  url: string;
  method: string;
  body?: string;
  headers?: string[];
  rate: string;
  duration: string;
  workers: number;
  connections: number;
}

export interface SummaryPayload {
  total_requests: number;
  successful_requests: number;
  failed_requests: number;
  total_elapsed_time: string;
  average_latency: string;
  throughput_rps: number;
}

export interface ResultsResponse {
  id: string;
  status: string;
  config: ConfigPayload;
  summary: SummaryPayload;
  latency_percentiles?: LatencyPercentiles;
  status_codes: Record<number, number>;
  timestamp: string;
}

export interface WSMessage {
  type: 'metrics' | 'started' | 'completed' | 'stopped' | 'error';
  timestamp: string;
  data?: MetricsPayload | { error: string };
}

export type SessionStatus = 'idle' | 'running' | 'completed' | 'stopped' | 'error';
