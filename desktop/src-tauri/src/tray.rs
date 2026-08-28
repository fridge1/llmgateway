use tauri::{
    AppHandle, Emitter, Manager,
    tray::{TrayIconBuilder, MouseButton, MouseButtonState, TrayIconEvent},
    menu::{MenuBuilder, MenuItemBuilder},
};

pub fn setup_tray(app: &AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    let open = MenuItemBuilder::with_id("open", "打开主面板").build(app)?;
    let rescan = MenuItemBuilder::with_id("rescan", "重新扫描工具").build(app)?;
    let logout = MenuItemBuilder::with_id("logout", "退出登录").build(app)?;
    let quit = MenuItemBuilder::with_id("quit", "退出应用").build(app)?;

    let menu = MenuBuilder::new(app)
        .item(&open)
        .separator()
        .item(&rescan)
        .separator()
        .item(&logout)
        .item(&quit)
        .build()?;

    TrayIconBuilder::new()
        .icon(app.default_window_icon().unwrap().clone())
        .menu(&menu)
        .on_menu_event(move |app, event| {
            match event.id().as_ref() {
                "open" => {
                    if let Some(w) = app.get_webview_window("main") {
                        let _ = w.show();
                        let _ = w.set_focus();
                    }
                }
                "quit" => app.exit(0),
                "logout" => {
                    if let Some(w) = app.get_webview_window("main") {
                        let _ = w.emit("tray-logout", ());
                    }
                }
                "rescan" => {
                    if let Some(w) = app.get_webview_window("main") {
                        let _ = w.emit("tray-rescan", ());
                    }
                }
                _ => {}
            }
        })
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click { button: MouseButton::Left, button_state: MouseButtonState::Up, .. } = event {
                let app = tray.app_handle();
                if let Some(w) = app.get_webview_window("main") {
                    let _ = w.show();
                    let _ = w.set_focus();
                }
            }
        })
        .build(app)?;

    Ok(())
}
