use super::ConfigResult;
#[cfg(not(windows))]
use super::shell_profile::{get_shell_profile_path, backup_file, write_env_block};
use std::fs;

pub fn configure(gateway_url: &str, api_key: &str, model: Option<&str>) -> ConfigResult {
    let base_url = format!("{}/v1", gateway_url.trim_end_matches('/'));
    let result = (|| -> Result<(), String> {
        let codex_dir = dirs::home_dir().ok_or("无法获取 HOME 目录")?.join(".codex");
        if !codex_dir.exists() {
            fs::create_dir_all(&codex_dir).map_err(|e| format!("创建 .codex 目录失败: {}", e))?;
        }

        // 1. Set OPENAI_API_KEY env var
        #[cfg(not(windows))]
        {
            let profile = get_shell_profile_path()?;
            backup_file(&profile)?;
            write_env_block(&profile, "codex_cli", &[
                ("OPENAI_API_KEY", api_key),
            ])?;
        }
        #[cfg(windows)]
        {
            super::windows_env::write_env_vars("codex_cli", &[
                ("OPENAI_API_KEY", api_key),
            ])?;
        }

        // 2. Write gateway provider to config.toml with experimental_bearer_token
        let config_path = codex_dir.join("config.toml");
        let content = fs::read_to_string(&config_path).unwrap_or_default();

        if content.contains("[model_providers.gateway]") {
            let new_content = rewrite_gateway_section(&content, &base_url, api_key);
            fs::write(&config_path, new_content).map_err(|e| format!("写入 config.toml 失败: {}", e))?;
        } else {
            let section = format!(
                "\nmodel_provider = \"gateway\"\n\n\
                 [model_providers.gateway]\n\
                 name = \"LLM Gateway\"\n\
                 base_url = \"{}\"\n\
                 experimental_bearer_token = \"{}\"\n\
                 wire_api = \"responses\"\n",
                base_url, api_key
            );
            let new_content = if content.is_empty() {
                section
            } else {
                format!("{}\n{}", content.trim_end(), section)
            };
            fs::write(&config_path, new_content).map_err(|e| format!("写入 config.toml 失败: {}", e))?;
        }

        // 3. Delete auth.json — Codex will use experimental_bearer_token directly
        let auth_path = codex_dir.join("auth.json");
        if auth_path.exists() {
            fs::remove_file(&auth_path).map_err(|e| format!("删除 auth.json 失败: {}", e))?;
        }

        // 4. Set root-level model in config.toml if specified
        if let Some(m) = model {
            let content = fs::read_to_string(&config_path).unwrap_or_default();
            let new_content = set_root_model(&content, m);
            fs::write(&config_path, new_content).map_err(|e| format!("写入 config.toml 失败: {}", e))?;
        }

        Ok(())
    })();

    ConfigResult {
        tool: "codex_cli".to_string(),
        success: result.is_ok(),
        message: result.err().unwrap_or_else(|| "配置成功".to_string()),
    }
}

fn rewrite_gateway_section(content: &str, base_url: &str, api_key: &str) -> String {
    let mut new_content = String::new();
    let mut in_gateway = false;
    let mut wrote_base_url = false;
    let mut wrote_token = false;

    for line in content.lines() {
        if line.trim() == "[model_providers.gateway]" {
            in_gateway = true;
            wrote_base_url = false;
            wrote_token = false;
            new_content.push_str(line);
            new_content.push('\n');
        } else if in_gateway {
            let trimmed = line.trim();
            if trimmed.starts_with("base_url") {
                new_content.push_str(&format!("base_url = \"{}\"\n", base_url));
                wrote_base_url = true;
            } else if trimmed.starts_with("experimental_bearer_token") {
                new_content.push_str(&format!("experimental_bearer_token = \"{}\"\n", api_key));
                wrote_token = true;
            } else if trimmed.starts_with("env_key") {
                // Remove env_key, replaced by experimental_bearer_token
            } else if trimmed.starts_with("profile") {
                // Remove stale profile field from gateway section
            } else if trimmed.starts_with('[') {
                if !wrote_base_url {
                    new_content.push_str(&format!("base_url = \"{}\"\n", base_url));
                }
                if !wrote_token {
                    new_content.push_str(&format!("experimental_bearer_token = \"{}\"\n", api_key));
                }
                in_gateway = false;
                new_content.push_str(line);
                new_content.push('\n');
            } else if trimmed.is_empty() {
                // Skip blank lines inside gateway section to keep it clean
            } else {
                new_content.push_str(line);
                new_content.push('\n');
            }
        } else {
            new_content.push_str(line);
            new_content.push('\n');
        }
    }

    if in_gateway {
        if !wrote_base_url {
            new_content.push_str(&format!("base_url = \"{}\"\n", base_url));
        }
        if !wrote_token {
            new_content.push_str(&format!("experimental_bearer_token = \"{}\"\n", api_key));
        }
    }

    new_content
}

fn set_root_model(content: &str, model: &str) -> String {
    let model_line = format!("model = \"{}\"", model);
    let mut result = String::new();
    let mut replaced = false;
    let mut in_section = false;

    for line in content.lines() {
        let trimmed = line.trim();
        if trimmed.starts_with('[') {
            in_section = true;
        }
        if !in_section && trimmed.starts_with("model") && !trimmed.starts_with("model_provider") {
            if !replaced {
                result.push_str(&model_line);
                result.push('\n');
                replaced = true;
            }
        } else {
            result.push_str(line);
            result.push('\n');
        }
    }

    if !replaced {
        let provider_line = "model_provider = ";
        if let Some(pos) = result.find(provider_line) {
            let end = result[pos..].find('\n').map(|i| pos + i + 1).unwrap_or(result.len());
            result.insert_str(end, &format!("{}\n", model_line));
        } else {
            result.insert_str(0, &format!("{}\n", model_line));
        }
    }

    result
}
