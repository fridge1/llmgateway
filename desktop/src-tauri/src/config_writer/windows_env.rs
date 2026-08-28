use std::fs;
use std::path::PathBuf;
use std::process::Command;

use winreg::enums::*;
use winreg::RegKey;

fn env_registry_path() -> PathBuf {
    let home = dirs::home_dir().unwrap_or_default();
    home.join(".llm-gateway").join("env-registry.json")
}

fn load_registry() -> serde_json::Value {
    let path = env_registry_path();
    if path.exists() {
        fs::read_to_string(&path)
            .ok()
            .and_then(|s| serde_json::from_str(&s).ok())
            .unwrap_or(serde_json::json!({}))
    } else {
        serde_json::json!({})
    }
}

fn save_registry(data: &serde_json::Value) -> Result<(), String> {
    let path = env_registry_path();
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent).map_err(|e| format!("创建目录失败: {}", e))?;
    }
    let content = serde_json::to_string_pretty(data).map_err(|e| e.to_string())?;
    fs::write(&path, content).map_err(|e| format!("写入 env-registry.json 失败: {}", e))
}

fn try_write_system_registry(env_vars: &[(&str, &str)]) -> bool {
    let hklm = RegKey::predef(HKEY_LOCAL_MACHINE);
    let env_key = match hklm.open_subkey_with_flags(
        "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment",
        KEY_SET_VALUE | KEY_QUERY_VALUE,
    ) {
        Ok(k) => k,
        Err(_) => return false,
    };

    for (key, value) in env_vars {
        if env_key.set_value(*key, &value.to_string()).is_err() {
            return false;
        }
    }
    true
}

fn write_system_via_elevation(env_vars: &[(&str, &str)]) -> Result<(), String> {
    let tmp_dir = std::env::temp_dir();
    let script_path = tmp_dir.join("llm-gateway-setenv.ps1");

    let mut script = String::new();
    for (key, value) in env_vars {
        script.push_str(&format!(
            "[Environment]::SetEnvironmentVariable('{}', '{}', 'Machine')\n",
            key.replace('\'', "''"),
            value.replace('\'', "''"),
        ));
    }
    script.push_str("Remove-Item -Path $MyInvocation.MyCommand.Path -Force\n");

    fs::write(&script_path, &script)
        .map_err(|e| format!("写入临时脚本失败: {}", e))?;

    let output = Command::new("powershell.exe")
        .args([
            "-Command",
            &format!(
                "Start-Process powershell -Verb RunAs -Wait -ArgumentList '-ExecutionPolicy Bypass -File \"{}\"'",
                script_path.to_string_lossy()
            ),
        ])
        .output()
        .map_err(|e| format!("启动提权进程失败: {}", e))?;

    if output.status.success() {
        Ok(())
    } else {
        let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
        Err(format!("提权写入失败: {}", stderr))
    }
}

fn cleanup_user_level(var_names: &[&str]) {
    let hkcu = RegKey::predef(HKEY_CURRENT_USER);
    if let Ok(env_key) = hkcu.open_subkey_with_flags("Environment", KEY_SET_VALUE) {
        for name in var_names {
            let _ = env_key.delete_value(name);
        }
    }
}

pub fn write_env_vars(tool_name: &str, env_vars: &[(&str, &str)]) -> Result<(), String> {
    let var_names: Vec<String> = env_vars.iter().map(|(k, _)| k.to_string()).collect();
    let var_name_refs: Vec<&str> = var_names.iter().map(|s| s.as_str()).collect();

    // Collect previous values for backup
    let mut previous: serde_json::Map<String, serde_json::Value> = serde_json::Map::new();
    if let Ok(env_key) = RegKey::predef(HKEY_LOCAL_MACHINE).open_subkey(
        "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment",
    ) {
        for (key, _) in env_vars {
            if let Ok(old_val) = env_key.get_value::<String, _>(*key) {
                previous.insert(key.to_string(), serde_json::json!(old_val));
            }
        }
    }

    // Try direct system-level write first, elevate via UAC if needed
    if !try_write_system_registry(env_vars) {
        write_system_via_elevation(env_vars)?;
    }

    // Remove user-level duplicates to avoid conflicts
    cleanup_user_level(&var_name_refs);

    let mut registry = load_registry();
    registry[tool_name] = serde_json::json!({
        "vars": var_names,
        "previous": previous,
    });
    save_registry(&registry)?;

    broadcast_env_change();
    Ok(())
}

pub fn remove_env_vars(tool_name: &str, var_names: &[&str]) -> Result<(), String> {
    // Remove from system-level
    let hklm = RegKey::predef(HKEY_LOCAL_MACHINE);
    let direct_ok = if let Ok(env_key) = hklm.open_subkey_with_flags(
        "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment",
        KEY_SET_VALUE,
    ) {
        for name in var_names {
            let _ = env_key.delete_value(name);
        }
        true
    } else {
        false
    };

    if !direct_ok {
        let tmp_dir = std::env::temp_dir();
        let script_path = tmp_dir.join("llm-gateway-delenv.ps1");

        let mut script = String::new();
        for name in var_names {
            script.push_str(&format!(
                "[Environment]::SetEnvironmentVariable('{}', $null, 'Machine')\n",
                name.replace('\'', "''"),
            ));
        }
        script.push_str("Remove-Item -Path $MyInvocation.MyCommand.Path -Force\n");

        if let Ok(()) = fs::write(&script_path, &script) {
            let _ = Command::new("powershell.exe")
                .args([
                    "-Command",
                    &format!(
                        "Start-Process powershell -Verb RunAs -Wait -ArgumentList '-ExecutionPolicy Bypass -File \"{}\"'",
                        script_path.to_string_lossy()
                    ),
                ])
                .output();
        }
    }

    // Also clean user-level
    cleanup_user_level(var_names);

    let mut registry = load_registry();
    if let Some(obj) = registry.as_object_mut() {
        obj.remove(tool_name);
    }
    save_registry(&registry)?;

    broadcast_env_change();
    Ok(())
}

pub fn read_env_var(var_name: &str) -> Option<String> {
    // Check system-level first, then user-level
    let hklm = RegKey::predef(HKEY_LOCAL_MACHINE);
    if let Ok(env_key) = hklm.open_subkey(
        "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment",
    ) {
        if let Ok(val) = env_key.get_value::<String, _>(var_name) {
            return Some(val);
        }
    }
    let hkcu = RegKey::predef(HKEY_CURRENT_USER);
    let env_key = hkcu.open_subkey("Environment").ok()?;
    env_key.get_value::<String, _>(var_name).ok()
}

fn broadcast_env_change() {
    use std::ffi::OsStr;
    use std::os::windows::ffi::OsStrExt;

    #[link(name = "user32")]
    extern "system" {
        fn SendMessageTimeoutW(
            hwnd: isize,
            msg: u32,
            wparam: usize,
            lparam: *const u16,
            flags: u32,
            timeout: u32,
            result: *mut usize,
        ) -> isize;
    }

    const HWND_BROADCAST: isize = 0xFFFF;
    const WM_SETTINGCHANGE: u32 = 0x001A;
    const SMTO_ABORTIFHUNG: u32 = 0x0002;

    let env: Vec<u16> = OsStr::new("Environment")
        .encode_wide()
        .chain(std::iter::once(0))
        .collect();

    let mut result: usize = 0;
    unsafe {
        SendMessageTimeoutW(
            HWND_BROADCAST,
            WM_SETTINGCHANGE,
            0,
            env.as_ptr(),
            SMTO_ABORTIFHUNG,
            5000,
            &mut result,
        );
    }
}
