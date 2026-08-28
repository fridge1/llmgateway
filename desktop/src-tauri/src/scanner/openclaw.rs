use super::{ToolStatus, CurrentConfig};
use super::common::{find_command, run_version};

fn read_openclaw_config() -> Option<(String, bool, Option<String>)> {
    let home = dirs::home_dir()?;
    let config_path = home.join(".openclaw").join("openclaw.json");
    let content = std::fs::read_to_string(config_path).ok()?;
    let json: serde_json::Value = serde_json::from_str(&content).ok()?;

    let base_url = json.pointer("/models/providers/anthropic/baseUrl")
        .and_then(|v| v.as_str())
        .map(|s| s.to_string())?;
    let has_key = json.pointer("/models/providers/anthropic/apiKey")
        .and_then(|v| v.as_str())
        .map(|s| !s.is_empty())
        .unwrap_or(false);
    let current_model = json.pointer("/models/default")
        .and_then(|v| v.as_str())
        .map(|s| s.to_string());

    Some((base_url, has_key, current_model))
}

pub fn detect(gateway_url: &str) -> Result<Option<ToolStatus>, String> {
    let path = match find_command("openclaw") {
        Some(p) => p,
        None => return Ok(None),
    };

    let version_output = run_version(&path)?;

    let (configured, current_config) = match read_openclaw_config() {
        Some((base_url, has_key, current_model)) => {
            let is_configured = base_url == gateway_url && has_key;
            (is_configured, Some(CurrentConfig { base_url, has_key, current_model }))
        }
        None => (false, None),
    };

    Ok(Some(ToolStatus {
        tool: "openclaw".to_string(),
        path: path.to_string_lossy().to_string(),
        version: version_output,
        configured,
        current_config,
    }))
}
