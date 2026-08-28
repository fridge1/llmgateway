use super::{ToolStatus, CurrentConfig};
use super::common::{find_command, run_version};

fn read_hermes_config() -> Option<(String, bool, Option<String>)> {
    let home = dirs::home_dir()?;

    let config_path = home.join(".hermes").join("config.yaml");
    let config_content = std::fs::read_to_string(config_path).ok()?;
    let base_url = config_content.lines()
        .find(|line| line.trim().starts_with("base_url:"))
        .and_then(|line| line.splitn(2, ':').nth(1))
        .map(|s| s.trim().to_string())?;

    let current_model = config_content.lines()
        .skip_while(|line| !line.trim().starts_with("model:"))
        .skip(1)
        .take_while(|line| line.starts_with(' ') || line.starts_with('\t'))
        .find(|line| line.trim().starts_with("name:"))
        .and_then(|line| line.splitn(2, ':').nth(1))
        .map(|s| s.trim().to_string());

    let env_path = home.join(".hermes").join(".env");
    let has_key = std::fs::read_to_string(env_path).ok()
        .map(|content| content.lines().any(|line| {
            let trimmed = line.trim();
            trimmed.starts_with("OPENAI_API_KEY=") && trimmed.len() > "OPENAI_API_KEY=".len()
        }))
        .unwrap_or(false);

    Some((base_url, has_key, current_model))
}

pub fn detect(gateway_url: &str) -> Result<Option<ToolStatus>, String> {
    let path = match find_command("hermes") {
        Some(p) => p,
        None => return Ok(None),
    };

    let version_output = run_version(&path)?;

    let expected_url = format!("{}/v1", gateway_url.trim_end_matches('/'));
    let (configured, current_config) = match read_hermes_config() {
        Some((base_url, has_key, current_model)) => {
            let is_configured = base_url == expected_url && has_key;
            (is_configured, Some(CurrentConfig { base_url, has_key, current_model }))
        }
        None => (false, None),
    };

    Ok(Some(ToolStatus {
        tool: "hermes_agent".to_string(),
        path: path.to_string_lossy().to_string(),
        version: version_output,
        configured,
        current_config,
    }))
}
