use super::ConfigResult;

pub fn configure(gateway_url: &str, api_key: &str, model: Option<&str>) -> ConfigResult {
    let url = format!("{}/v1", gateway_url.trim_end_matches('/'));
    let result = (|| -> Result<(), String> {
        let home = dirs::home_dir().ok_or("无法获取 HOME 目录")?;
        let hermes_dir = home.join(".hermes");

        if !hermes_dir.exists() {
            std::fs::create_dir_all(&hermes_dir).map_err(|e| e.to_string())?;
        }

        // Write config.yaml (only modify model node)
        let config_path = hermes_dir.join("config.yaml");
        let mut content = std::fs::read_to_string(&config_path).unwrap_or_default();

        let model_block = if let Some(m) = model {
            format!(
                "model:\n  provider: custom\n  base_url: {}\n  name: {}\n",
                url, m
            )
        } else {
            format!(
                "model:\n  provider: custom\n  base_url: {}\n",
                url
            )
        };

        if content.contains("model:") {
            if let Some(start) = content.find("model:") {
                let rest = &content[start..];
                let end = rest.lines().skip(1)
                    .position(|line| !line.is_empty() && !line.starts_with(' ') && !line.starts_with('\t'))
                    .map(|pos| {
                        rest.lines().skip(1).take(pos).map(|l| l.len() + 1).sum::<usize>() + rest.lines().next().unwrap().len() + 1
                    })
                    .unwrap_or(rest.len());
                content = format!("{}{}{}", &content[..start], model_block, &content[start + end..]);
            }
        } else {
            content = format!("{}\n{}", content.trim_end(), model_block);
        }

        std::fs::write(&config_path, content.trim_end().to_owned() + "\n").map_err(|e| e.to_string())?;

        // Write .env (append or replace OPENAI_API_KEY)
        let env_path = hermes_dir.join(".env");
        let env_content = std::fs::read_to_string(&env_path).unwrap_or_default();
        let new_env: String = env_content.lines()
            .filter(|line| !line.trim().starts_with("OPENAI_API_KEY="))
            .collect::<Vec<_>>()
            .join("\n");
        let new_env = format!("{}\nOPENAI_API_KEY={}\n", new_env.trim(), api_key);
        std::fs::write(&env_path, new_env).map_err(|e| e.to_string())
    })();

    ConfigResult {
        tool: "hermes_agent".to_string(),
        success: result.is_ok(),
        message: result.err().unwrap_or_else(|| "配置成功".to_string()),
    }
}
