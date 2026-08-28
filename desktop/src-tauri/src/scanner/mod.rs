pub mod common;
pub mod claude_code;
pub mod codex_cli;
pub mod openclaw;
pub mod hermes;

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolScanResult {
    pub tools: Vec<ToolStatus>,
    pub scan_errors: Vec<ScanError>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolStatus {
    pub tool: String,
    pub path: String,
    pub version: String,
    pub configured: bool,
    pub current_config: Option<CurrentConfig>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CurrentConfig {
    pub base_url: String,
    pub has_key: bool,
    pub current_model: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScanError {
    pub tool: String,
    pub error: String,
}

pub fn scan_all(gateway_url: &str) -> ToolScanResult {
    let mut tools = Vec::new();
    let mut scan_errors = Vec::new();

    let scanners: &[(&str, &dyn Fn(&str) -> Result<Option<ToolStatus>, String>)] = &[
        ("claude_code", &claude_code::detect),
        ("codex_cli", &codex_cli::detect),
        ("openclaw", &openclaw::detect),
        ("hermes_agent", &hermes::detect),
    ];

    for (name, detect_fn) in scanners {
        match detect_fn(gateway_url) {
            Ok(Some(status)) => tools.push(status),
            Ok(None) => {}
            Err(e) => scan_errors.push(ScanError { tool: name.to_string(), error: e }),
        }
    }

    ToolScanResult { tools, scan_errors }
}
