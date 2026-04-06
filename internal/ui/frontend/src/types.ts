export interface Session {
  session_id: string;
  started_at: string;
  ended_at: string;
  duration_ms: number;
  client_addr: string;
  target_host: string;
  target_port: string;
  dial_status: string;
  error: string;
  exchanges: InterceptedExchange[];
}

export interface InterceptedExchange {
  timestamp: string;
  method: string;
  url: string;
  request_headers: Record<string, string>;
  request_body: string;
  detected_files: FileRef[];
  blocked: boolean;
  block_reason: string;
  status_code: number;
  response_headers: Record<string, string>;
  response_body: string;
}

export interface FileRef {
  path: string;
  source: string;
}

export interface StoreStats {
  total_sessions: number;
  blocked_count: number;
  file_detections: number;
}

export interface ContentPrompt {
  id: string;
  session_id: string;
  url: string;
  matched_keyword: string;
  file_paths: string[];
}

export type PromptAction = 'allow_once' | 'allow_always' | 'block_once' | 'block_always';
