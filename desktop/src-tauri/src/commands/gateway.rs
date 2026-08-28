use tauri::{Emitter, State};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use crate::state::AppState;
use crate::gateway_client::{GatewayClient, SseEvent};

#[tauri::command]
pub async fn get_balance(state: State<'_, AppState>) -> Result<serde_json::Value, String> {
    let (url, token) = get_context(&state)?;
    let client = GatewayClient::new(&url);
    client.get_balance(&token).await.map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn get_stats(state: State<'_, AppState>) -> Result<serde_json::Value, String> {
    let (url, token) = get_context(&state)?;
    let client = GatewayClient::new(&url);
    client.get_stats(&token).await.map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn list_keys(state: State<'_, AppState>) -> Result<serde_json::Value, String> {
    let (url, token) = get_context(&state)?;
    let client = GatewayClient::new(&url);
    client.list_keys(&token).await.map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn create_key(state: State<'_, AppState>) -> Result<serde_json::Value, String> {
    let (url, token) = get_context(&state)?;
    let client = GatewayClient::new(&url);
    client.create_key(&token).await.map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn list_models(state: State<'_, AppState>) -> Result<serde_json::Value, String> {
    let (url, token) = get_context(&state)?;
    let client = GatewayClient::new(&url);
    client.list_models(&token).await.map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn api_request(
    method: String,
    path: String,
    body: Option<serde_json::Value>,
    state: State<'_, AppState>,
) -> Result<serde_json::Value, String> {
    let (url, token) = get_context(&state)?;
    let client = GatewayClient::new(&url);
    client.request(&method, &path, body, Some(&token)).await.map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn api_request_unauth(
    method: String,
    path: String,
    body: Option<serde_json::Value>,
    state: State<'_, AppState>,
) -> Result<serde_json::Value, String> {
    let url = state.gateway_url.lock().unwrap().clone();
    let client = GatewayClient::new(&url);
    client.request(&method, &path, body, None).await.map_err(|e| e.to_string())
}

/// One file part for a multipart upload. `data` is the raw file bytes.
#[derive(serde::Deserialize)]
pub struct MultipartFile {
    pub field: String,
    pub filename: String,
    pub mime: String,
    pub data: Vec<u8>,
}

/// Forwards a multipart/form-data POST (used by image edit, which uploads files).
#[tauri::command]
pub async fn api_request_multipart(
    path: String,
    text_fields: Vec<(String, String)>,
    files: Vec<MultipartFile>,
    state: State<'_, AppState>,
) -> Result<serde_json::Value, String> {
    let (url, token) = get_context(&state)?;
    let client = GatewayClient::new(&url);
    let file_fields = files
        .into_iter()
        .map(|f| (f.field, f.filename, f.mime, f.data))
        .collect();
    client
        .request_multipart(&path, text_fields, file_fields, &token)
        .await
        .map_err(|e| e.to_string())
}

fn get_context(state: &State<'_, AppState>) -> Result<(String, String), String> {
    let url = state.gateway_url.lock().unwrap().clone();
    let token = state.token.lock().unwrap().clone().ok_or("未登录")?;
    Ok((url, token))
}

/// Streams a chat completion, emitting per-`request_id` events to the frontend:
///   - `playground-sse-delta-{request_id}`    payload: { text }
///   - `playground-sse-complete-{request_id}` payload: { usage }
///   - `playground-sse-error-{request_id}`    payload: { message }
///
/// The JWT stays in Rust; only chunk text crosses the IPC boundary.
#[tauri::command]
pub async fn playground_stream(
    window: tauri::Window,
    state: State<'_, AppState>,
    request_id: String,
    body: serde_json::Value,
) -> Result<(), String> {
    let (url, token) = get_context(&state)?;

    let cancel = Arc::new(AtomicBool::new(false));
    state
        .stream_cancels
        .lock()
        .unwrap()
        .insert(request_id.clone(), cancel.clone());

    let client = GatewayClient::new(&url);
    let delta_event = format!("playground-sse-delta-{}", request_id);
    let complete_event = format!("playground-sse-complete-{}", request_id);
    let error_event = format!("playground-sse-error-{}", request_id);

    let mut usage: Option<serde_json::Value> = None;
    let result = client
        .stream_completions(body, &token, cancel, |ev| match ev {
            SseEvent::Delta(text) => {
                let _ = window.emit(&delta_event, serde_json::json!({ "text": text }));
            }
            SseEvent::Usage(u) => {
                usage = Some(u);
            }
            _ => {}
        })
        .await;

    state.stream_cancels.lock().unwrap().remove(&request_id);

    match result {
        Ok(()) => {
            let _ = window.emit(&complete_event, serde_json::json!({ "usage": usage }));
            Ok(())
        }
        Err(e) => {
            let _ = window.emit(&error_event, serde_json::json!({ "message": e.to_string() }));
            Err(e.to_string())
        }
    }
}

/// Signals an in-flight [`playground_stream`] to stop.
#[tauri::command]
pub fn playground_stream_abort(state: State<'_, AppState>, request_id: String) {
    if let Some(flag) = state.stream_cancels.lock().unwrap().get(&request_id) {
        flag.store(true, Ordering::Relaxed);
    }
}
