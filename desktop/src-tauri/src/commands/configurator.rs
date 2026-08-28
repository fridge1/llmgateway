use tauri::State;
use crate::state::AppState;
use crate::config_writer::{self, ConfigResult};
use crate::gateway_client::GatewayClient;

#[tauri::command]
pub async fn configure_tools(
    state: State<'_, AppState>,
    tools: Vec<String>,
) -> Result<Vec<ConfigResult>, String> {
    let token = state.token.lock().unwrap().clone()
        .ok_or("未登录")?;
    let url = state.gateway_url.lock().unwrap().clone();
    let client = GatewayClient::new(&url);

    let api_key = client.get_or_create_key(&token).await
        .map_err(|e| e.to_string())?;

    Ok(config_writer::configure_all(&url, &api_key, &tools))
}

#[tauri::command]
pub async fn configure_tool(
    state: State<'_, AppState>,
    tool: String,
    model: Option<String>,
) -> Result<ConfigResult, String> {
    let token = state.token.lock().unwrap().clone()
        .ok_or("未登录")?;
    let url = state.gateway_url.lock().unwrap().clone();
    let client = GatewayClient::new(&url);

    let api_key = client.get_or_create_key_for_tool(&token, &tool).await
        .map_err(|e| e.to_string())?;

    Ok(config_writer::configure_one(&url, &api_key, &tool, model.as_deref()))
}

#[tauri::command]
pub async fn clear_tool_config(tool: String) -> Result<(), String> {
    match tool.as_str() {
        "claude_code" => {
            #[cfg(not(windows))]
            {
                let profile = crate::config_writer::shell_profile::get_shell_profile_path()?;
                crate::config_writer::shell_profile::remove_env_block(&profile, &tool)
            }
            #[cfg(windows)]
            {
                crate::config_writer::windows_env::remove_env_vars(
                    "claude_code",
                    &["ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_MODEL"],
                )
            }
        }
        "codex_cli" => {
            #[cfg(not(windows))]
            {
                let profile = crate::config_writer::shell_profile::get_shell_profile_path()?;
                crate::config_writer::shell_profile::remove_env_block(&profile, &tool)?;
            }
            #[cfg(windows)]
            {
                crate::config_writer::windows_env::remove_env_vars(
                    "codex_cli",
                    &["OPENAI_API_KEY"],
                )?;
            }
            let home = dirs::home_dir().ok_or("无法获取 HOME 目录")?;
            let config_path = home.join(".codex").join("config.toml");
            if config_path.exists() {
                let content = std::fs::read_to_string(&config_path).map_err(|e| e.to_string())?;
                let new_content = remove_codex_gateway_config(&content);
                std::fs::write(&config_path, new_content).map_err(|e| e.to_string())?;
            }
            Ok(())
        }
        "openclaw" => {
            let home = dirs::home_dir().ok_or("无法获取 HOME 目录")?;
            let config_path = home.join(".openclaw").join("openclaw.json");
            if !config_path.exists() { return Ok(()); }
            let content = std::fs::read_to_string(&config_path).map_err(|e| e.to_string())?;
            let mut config: serde_json::Value = serde_json::from_str(&content).map_err(|e| e.to_string())?;
            if let Some(providers) = config.pointer_mut("/models/providers/anthropic") {
                if let Some(obj) = providers.as_object_mut() {
                    obj.remove("baseUrl");
                    obj.remove("apiKey");
                }
            }
            if let Some(models) = config.get_mut("models") {
                if let Some(obj) = models.as_object_mut() {
                    obj.remove("default");
                }
            }
            let formatted = serde_json::to_string_pretty(&config).map_err(|e| e.to_string())?;
            std::fs::write(&config_path, formatted).map_err(|e| e.to_string())
        }
        "hermes_agent" => {
            let home = dirs::home_dir().ok_or("无法获取 HOME 目录")?;
            let config_path = home.join(".hermes").join("config.yaml");
            if config_path.exists() {
                let content = std::fs::read_to_string(&config_path).map_err(|e| e.to_string())?;
                if let Some(start) = content.find("model:") {
                    let rest = &content[start..];
                    let end = rest.lines().skip(1)
                        .position(|line| !line.is_empty() && !line.starts_with(' ') && !line.starts_with('\t'))
                        .map(|pos| {
                            rest.lines().skip(1).take(pos).map(|l| l.len() + 1).sum::<usize>() + rest.lines().next().unwrap().len() + 1
                        })
                        .unwrap_or(rest.len());
                    let new_content = format!("{}{}", &content[..start], &content[start + end..]);
                    std::fs::write(&config_path, new_content.trim().to_owned() + "\n").map_err(|e| e.to_string())?;
                }
            }
            let env_path = home.join(".hermes").join(".env");
            if env_path.exists() {
                let content = std::fs::read_to_string(&env_path).map_err(|e| e.to_string())?;
                let new_content: String = content.lines()
                    .filter(|l| !l.trim().starts_with("OPENAI_API_KEY="))
                    .collect::<Vec<_>>()
                    .join("\n");
                std::fs::write(&env_path, new_content.trim().to_owned() + "\n").map_err(|e| e.to_string())?;
            }
            Ok(())
        }
        _ => Err(format!("未知工具: {}", tool)),
    }
}

fn remove_codex_gateway_config(content: &str) -> String {
    let mut result = String::new();
    let mut skip_gateway = false;
    let mut removed_model = false;
    let mut removed_provider = false;

    for line in content.lines() {
        let trimmed = line.trim();
        if trimmed == "[model_providers.gateway]" {
            skip_gateway = true;
            continue;
        }
        if skip_gateway {
            if trimmed.starts_with('[') {
                skip_gateway = false;
                result.push_str(line);
                result.push('\n');
            }
            continue;
        }
        if !removed_model && !trimmed.starts_with('[') && trimmed.starts_with("model") && !trimmed.starts_with("model_provider") {
            removed_model = true;
            continue;
        }
        if !removed_provider && trimmed.starts_with("model_provider") {
            removed_provider = true;
            continue;
        }
        result.push_str(line);
        result.push('\n');
    }

    result
}
