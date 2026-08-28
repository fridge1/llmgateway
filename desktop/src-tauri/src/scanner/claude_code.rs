use super::{ToolStatus, CurrentConfig};
use super::common::{find_command, run_version, read_env_var_from_shell_profile};

pub fn detect(gateway_url: &str) -> Result<Option<ToolStatus>, String> {
    let path = match find_command("claude") {
        Some(p) => p,
        None => return Ok(None),
    };

    let version_output = run_version(&path)?;
    if !version_output.to_lowercase().contains("claude") {
        return Ok(None);
    }

    let base_url = read_env_var_from_shell_profile("ANTHROPIC_BASE_URL");
    let has_key = read_env_var_from_shell_profile("ANTHROPIC_AUTH_TOKEN").is_some();
    let current_model = read_env_var_from_shell_profile("ANTHROPIC_MODEL");
    let configured = base_url.as_deref() == Some(gateway_url) && has_key;

    Ok(Some(ToolStatus {
        tool: "claude_code".to_string(),
        path: path.to_string_lossy().to_string(),
        version: version_output,
        configured,
        current_config: base_url.map(|url| CurrentConfig { base_url: url, has_key, current_model }),
    }))
}
