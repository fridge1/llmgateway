import { invoke } from "@tauri-apps/api/core";
import type { UserInfo, ToolScanResult } from "./types";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type serde_json_Value = any;

export interface ConfigResult {
  tool: string;
  success: boolean;
  message: string;
}

export const api = {
  login: (phone: string, password: string, remember: boolean) =>
    invoke<UserInfo>("login", { phone, password, remember }),

  register: (phone: string, code: string, password: string, adminToken?: string) =>
    invoke<UserInfo>("register", { phone, code, password, adminToken: adminToken ?? null }),

  sendSmsCode: (phone: string, purpose: string) =>
    invoke<serde_json_Value>("api_request_unauth", {
      method: "POST", path: "/api/sms/send", body: { phone, purpose },
    }),

  resetPassword: (phone: string, code: string, newPassword: string) =>
    invoke<serde_json_Value>("api_request_unauth", {
      method: "POST", path: "/api/reset-password", body: { phone, code, new_password: newPassword },
    }),

  checkToken: () => invoke<UserInfo | null>("check_token"),

  logout: () => invoke<void>("logout"),

  getMe: () => invoke<UserInfo>("get_me"),

  scanTools: () => invoke<ToolScanResult>("scan_tools"),

  configureTools: (tools: string[]) =>
    invoke<ConfigResult[]>("configure_tools", { tools }),

  configureTool: (tool: string, model?: string) =>
    invoke<ConfigResult>("configure_tool", { tool, model: model ?? null }),

  clearToolConfig: (tool: string) =>
    invoke<void>("clear_tool_config", { tool }),

  getBalance: () =>
    invoke<{ balance: number; gift_balance: number }>("get_balance"),

  getStats: () => invoke<any>("get_stats"),

  listKeys: () => invoke<{ keys: any[] }>("list_keys"),

  createKey: () => invoke<any>("create_key"),

  listModels: () => invoke<{ data: any[] }>("list_models"),
};
