use super::{ToolStatus, CurrentConfig};
use super::common::{find_command, run_version};

pub fn detect(gateway_url: &str) -> Result<Option<ToolStatus>, String> {
    let path = match find_command("codex") {
        Some(p) => p,
        None => return Ok(None),
    };

    let version_output = run_version(&path)?;

    let expected_url = format!("{}/v1", gateway_url.trim_end_matches('/'));

    let (config_base_url, has_token, current_model) = read_config_toml_gateway();
    let configured = config_base_url.as_deref() == Some(expected_url.as_str()) && has_token;

    Ok(Some(ToolStatus {
        tool: "codex_cli".to_string(),
        path: path.to_string_lossy().to_string(),
        version: version_output,
        configured,
        current_config: config_base_url.map(|url| CurrentConfig { base_url: url, has_key: has_token, current_model }),
    }))
}

fn read_config_toml_gateway() -> (Option<String>, bool, Option<String>) {
    let home = match dirs::home_dir() {
        Some(h) => h,
        None => return (None, false, None),
    };
    let content = match std::fs::read_to_string(home.join(".codex").join("config.toml")) {
        Ok(c) => c,
        Err(_) => return (None, false, None),
    };

    let mut in_gateway = false;
    let mut base_url = None;
    let mut has_token = false;
    let mut current_model = None;

    for line in content.lines() {
        let trimmed = line.trim();
        if trimmed == "[model_providers.gateway]" {
            in_gateway = true;
        } else if in_gateway {
            if trimmed.starts_with('[') {
                break;
            }
            if trimmed.starts_with("base_url") {
                if let Some(val) = trimmed.split('=').nth(1) {
                    base_url = Some(val.trim().trim_matches('"').to_string());
                }
            } else if trimmed.starts_with("experimental_bearer_token") {
                if let Some(val) = trimmed.split('=').nth(1) {
                    has_token = !val.trim().trim_matches('"').is_empty();
                }
            }
        } else if !trimmed.starts_with('[') && trimmed.starts_with("model") && !trimmed.starts_with("model_provider") {
            if let Some(val) = trimmed.split('=').nth(1) {
                let v = val.trim().trim_matches('"');
                if !v.is_empty() {
                    current_model = Some(v.to_string());
                }
            }
        }
    }
    (base_url, has_token, current_model)
}
