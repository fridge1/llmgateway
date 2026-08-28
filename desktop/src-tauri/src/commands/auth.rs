use tauri::State;
use keyring::Entry;
use crate::state::AppState;
use crate::gateway_client::{GatewayClient, UserInfo};

const KEYRING_SERVICE: &str = "llm-gateway-desktop";
const KEYRING_USER: &str = "jwt-token";

fn get_keyring() -> Result<Entry, String> {
    Entry::new(KEYRING_SERVICE, KEYRING_USER).map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn login(
    state: State<'_, AppState>,
    phone: String,
    password: String,
    remember: bool,
) -> Result<UserInfo, String> {
    let url = state.gateway_url.lock().unwrap().clone();
    let client = GatewayClient::new(&url);
    let resp = client.login(&phone, &password, remember).await.map_err(|e| e.to_string())?;

    get_keyring()?.set_password(&resp.token).map_err(|e| e.to_string())?;
    *state.token.lock().unwrap() = Some(resp.token.clone());

    let user = client.get_me(&resp.token).await.map_err(|e| e.to_string())?;
    Ok(user)
}

#[tauri::command]
pub async fn check_token(state: State<'_, AppState>) -> Result<Option<UserInfo>, String> {
    let token = match get_keyring()?.get_password() {
        Ok(t) => t,
        Err(_) => return Ok(None),
    };

    let url = state.gateway_url.lock().unwrap().clone();
    let client = GatewayClient::new(&url);

    match client.get_me(&token).await {
        Ok(user) => {
            *state.token.lock().unwrap() = Some(token);
            Ok(Some(user))
        }
        Err(_) => {
            let _ = get_keyring()?.delete_credential();
            Ok(None)
        }
    }
}

#[tauri::command]
pub async fn logout(state: State<'_, AppState>) -> Result<(), String> {
    *state.token.lock().unwrap() = None;
    let _ = get_keyring()?.delete_credential();
    Ok(())
}

#[tauri::command]
pub async fn register(
    state: State<'_, AppState>,
    phone: String,
    code: String,
    password: String,
    admin_token: Option<String>,
) -> Result<UserInfo, String> {
    let url = state.gateway_url.lock().unwrap().clone();
    let client = GatewayClient::new(&url);

    let mut body = serde_json::json!({
        "phone": phone,
        "code": code,
        "password": password,
    });
    if let Some(t) = admin_token {
        body["admin_token"] = serde_json::Value::String(t);
    }

    let resp = client.request("POST", "/api/register", Some(body), None)
        .await.map_err(|e| e.to_string())?;

    let token = resp.get("token").and_then(|t| t.as_str())
        .ok_or("注册响应中缺少 token")?;

    get_keyring()?.set_password(token).map_err(|e| e.to_string())?;
    *state.token.lock().unwrap() = Some(token.to_string());

    let user = client.get_me(token).await.map_err(|e| e.to_string())?;
    Ok(user)
}

#[tauri::command]
pub async fn get_me(state: State<'_, AppState>) -> Result<UserInfo, String> {
    let token = state.token.lock().unwrap().clone()
        .ok_or("未登录".to_string())?;
    let url = state.gateway_url.lock().unwrap().clone();
    let client = GatewayClient::new(&url);
    client.get_me(&token).await.map_err(|e| e.to_string())
}
