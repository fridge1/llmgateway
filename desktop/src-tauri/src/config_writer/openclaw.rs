use super::ConfigResult;

pub fn configure(gateway_url: &str, api_key: &str, model: Option<&str>) -> ConfigResult {
    let result = (|| -> Result<(), String> {
        let home = dirs::home_dir().ok_or("无法获取 HOME 目录")?;
        let config_path = home.join(".openclaw").join("openclaw.json");

        let mut config: serde_json::Value = if config_path.exists() {
            let content = std::fs::read_to_string(&config_path).map_err(|e| e.to_string())?;
            serde_json::from_str(&content).unwrap_or(serde_json::json!({}))
        } else {
            if let Some(parent) = config_path.parent() {
                std::fs::create_dir_all(parent).map_err(|e| e.to_string())?;
            }
            serde_json::json!({})
        };

        config["models"]["providers"]["anthropic"]["baseUrl"] = serde_json::json!(gateway_url);
        config["models"]["providers"]["anthropic"]["apiKey"] = serde_json::json!(api_key);
        config["models"]["providers"]["anthropic"]["api"] = serde_json::json!("anthropic-messages");
        if let Some(m) = model {
            config["models"]["default"] = serde_json::json!(m);
        }

        let formatted = serde_json::to_string_pretty(&config).map_err(|e| e.to_string())?;
        std::fs::write(&config_path, formatted).map_err(|e| e.to_string())
    })();

    ConfigResult {
        tool: "openclaw".to_string(),
        success: result.is_ok(),
        message: result.err().unwrap_or_else(|| "配置成功".to_string()),
    }
}
