use super::ConfigResult;
#[cfg(not(windows))]
use super::shell_profile::{get_shell_profile_path, backup_file, write_env_block};

pub fn configure(gateway_url: &str, api_key: &str, model: Option<&str>) -> ConfigResult {
    let result = (|| -> Result<(), String> {
        let mut env_vars: Vec<(&str, &str)> = vec![
            ("ANTHROPIC_BASE_URL", gateway_url),
            ("ANTHROPIC_AUTH_TOKEN", api_key),
        ];
        if let Some(m) = model {
            env_vars.push(("ANTHROPIC_MODEL", m));
        }

        #[cfg(not(windows))]
        {
            let profile = get_shell_profile_path()?;
            backup_file(&profile)?;
            write_env_block(&profile, "claude_code", &env_vars)
        }
        #[cfg(windows)]
        {
            super::windows_env::write_env_vars("claude_code", &env_vars)
        }
    })();

    ConfigResult {
        tool: "claude_code".to_string(),
        success: result.is_ok(),
        message: result.err().unwrap_or_else(|| "配置成功".to_string()),
    }
}
