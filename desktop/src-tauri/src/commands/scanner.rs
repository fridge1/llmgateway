use tauri::State;
use crate::state::AppState;
use crate::scanner::{self, ToolScanResult};

#[tauri::command]
pub async fn scan_tools(state: State<'_, AppState>) -> Result<ToolScanResult, String> {
    let url = state.gateway_url.lock().unwrap().clone();
    Ok(scanner::scan_all(&url))
}
