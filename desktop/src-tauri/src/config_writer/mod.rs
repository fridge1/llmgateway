pub mod shell_profile;
#[cfg(windows)]
pub mod windows_env;
pub mod claude_code;
pub mod codex_cli;
pub mod openclaw;
pub mod hermes;

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConfigResult {
    pub tool: String,
    pub success: bool,
    pub message: String,
}

pub fn configure_one(
    gateway_url: &str,
    api_key: &str,
    tool: &str,
    model: Option<&str>,
) -> ConfigResult {
    match tool {
        "claude_code" => claude_code::configure(gateway_url, api_key, model),
        "codex_cli" => codex_cli::configure(gateway_url, api_key, model),
        "openclaw" => openclaw::configure(gateway_url, api_key, model),
        "hermes_agent" => hermes::configure(gateway_url, api_key, model),
        _ => ConfigResult {
            tool: tool.to_string(),
            success: false,
            message: format!("未知工具: {}", tool),
        },
    }
}

pub fn configure_all(
    gateway_url: &str,
    api_key: &str,
    tools: &[String],
) -> Vec<ConfigResult> {
    tools.iter().map(|tool| configure_one(gateway_url, api_key, tool, None)).collect()
}
