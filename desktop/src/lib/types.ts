export interface UserInfo {
  user_id?: string;
  phone: string;
  role: string;
}

export interface ToolStatus {
  tool: "claude_code" | "codex_cli" | "openclaw" | "hermes_agent";
  path: string;
  version: string;
  configured: boolean;
  current_config?: {
    base_url: string;
    has_key: boolean;
    current_model?: string;
  };
}

export interface Model {
  id: string;
  object?: string;
  created?: number;
  owned_by?: string;
}

export interface ScanError {
  tool: string;
  error: string;
}

export interface ToolScanResult {
  tools: ToolStatus[];
  scan_errors: ScanError[];
}
