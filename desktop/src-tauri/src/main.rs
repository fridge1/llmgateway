// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use desktop_lib::state::AppState;
use desktop_lib::commands;
use desktop_lib::tray;

fn main() {
    let gateway_url = std::env::var("LLM_GATEWAY_URL")
        .unwrap_or_else(|_| "https://your-domain.com".to_string());

    tauri::Builder::default()
        .manage(AppState::new(gateway_url))
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_opener::init())
        .setup(|app| {
            tray::setup_tray(app.handle())?;
            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                api.prevent_close();
                let _ = window.hide();
            }
        })
        .invoke_handler(tauri::generate_handler![
            commands::auth::login,
            commands::auth::register,
            commands::auth::check_token,
            commands::auth::logout,
            commands::auth::get_me,
            commands::scanner::scan_tools,
            commands::configurator::configure_tools,
            commands::configurator::configure_tool,
            commands::configurator::clear_tool_config,
            commands::gateway::get_balance,
            commands::gateway::get_stats,
            commands::gateway::list_keys,
            commands::gateway::create_key,
            commands::gateway::list_models,
            commands::gateway::api_request,
            commands::gateway::api_request_unauth,
            commands::gateway::api_request_multipart,
            commands::gateway::playground_stream,
            commands::gateway::playground_stream_abort,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
