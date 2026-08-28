use std::process::Command;
use std::path::PathBuf;
use std::env;
use std::sync::OnceLock;

fn shell_path() -> &'static str {
    static CACHED: OnceLock<String> = OnceLock::new();
    CACHED.get_or_init(|| {
        if cfg!(windows) {
            return String::new();
        }
        let shell = env::var("SHELL").unwrap_or_else(|_| "/bin/zsh".to_string());
        Command::new(&shell)
            .args(["-l", "-c", "echo $PATH"])
            .output()
            .ok()
            .filter(|o| o.status.success())
            .and_then(|o| {
                let s = String::from_utf8_lossy(&o.stdout).trim().to_string();
                if s.is_empty() { None } else { Some(s) }
            })
            .unwrap_or_default()
    })
}

pub fn which_command(cmd: &str) -> Option<PathBuf> {
    if cfg!(windows) {
        let output = Command::new("where").arg(cmd).output().ok()?;
        if output.status.success() {
            let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
            let lines: Vec<&str> = stdout.lines().collect();
            // Prefer .cmd/.exe over extensionless POSIX shell scripts
            let preferred = lines.iter().find(|l| {
                let lower = l.to_lowercase();
                lower.ends_with(".cmd") || lower.ends_with(".exe")
            });
            let path = preferred.or(lines.first())?;
            return Some(PathBuf::from(path));
        }
        return None;
    }

    let full_path = shell_path();
    let output = Command::new("/usr/bin/env")
        .arg("which")
        .arg(cmd)
        .env("PATH", if full_path.is_empty() { env::var("PATH").unwrap_or_default() } else { full_path.to_string() })
        .output()
        .ok()?;
    if output.status.success() {
        let path = String::from_utf8_lossy(&output.stdout).trim().lines().next()?.to_string();
        Some(PathBuf::from(path))
    } else {
        None
    }
}

pub fn fallback_paths(cmd: &str) -> Vec<PathBuf> {
    let home = dirs::home_dir().unwrap_or_default();
    let mut paths = Vec::new();

    if cfg!(target_os = "macos") || cfg!(target_os = "linux") {
        paths.push(home.join(".npm/bin").join(cmd));
        paths.push(PathBuf::from(format!("/usr/local/bin/{}", cmd)));
        // Homebrew (Apple Silicon / Intel)
        paths.push(PathBuf::from(format!("/opt/homebrew/bin/{}", cmd)));
        // nvm default
        if let Ok(entries) = std::fs::read_dir(home.join(".nvm/versions/node")) {
            let mut versions: Vec<_> = entries.filter_map(|e| e.ok()).collect();
            versions.sort_by(|a, b| b.file_name().cmp(&a.file_name()));
            if let Some(latest) = versions.first() {
                paths.push(latest.path().join("bin").join(cmd));
            }
        }
        // volta
        paths.push(home.join(".volta/bin").join(cmd));
        // fnm
        paths.push(home.join(".local/share/fnm/aliases/default/bin").join(cmd));
        // pnpm
        paths.push(home.join(".local/share/pnpm").join(cmd));
        // common local bin
        paths.push(home.join(".local/bin").join(cmd));
    }

    if cfg!(windows) {
        if let Ok(appdata) = env::var("APPDATA") {
            paths.push(PathBuf::from(&appdata).join("npm").join(format!("{}.cmd", cmd)));
        }
        if let Ok(local) = env::var("LOCALAPPDATA") {
            paths.push(PathBuf::from(&local).join("Yarn").join("bin").join(format!("{}.cmd", cmd)));
            paths.push(PathBuf::from(&local).join("pnpm").join(format!("{}.cmd", cmd)));
        }
        paths.push(home.join(".volta").join("bin").join(format!("{}.exe", cmd)));
        if let Ok(userprofile) = env::var("USERPROFILE") {
            paths.push(PathBuf::from(userprofile).join("scoop").join("shims").join(format!("{}.cmd", cmd)));
        }
        if let Ok(nvm_symlink) = env::var("NVM_SYMLINK") {
            paths.push(PathBuf::from(nvm_symlink).join(format!("{}.cmd", cmd)));
        }
        if let Ok(fnm_path) = env::var("FNM_MULTISHELL_PATH") {
            paths.push(PathBuf::from(fnm_path).join(format!("{}.cmd", cmd)));
        }
    }

    paths
}

pub fn find_command(cmd: &str) -> Option<PathBuf> {
    if let Some(p) = which_command(cmd) {
        return Some(p);
    }
    fallback_paths(cmd).into_iter().find(|p| p.exists())
}

pub fn run_version(path: &PathBuf) -> Result<String, String> {
    let mut cmd = Command::new(path);
    cmd.arg("--version");
    let full_path = shell_path();
    if !full_path.is_empty() {
        cmd.env("PATH", full_path);
    }
    let output = cmd.output().map_err(|e| format!("执行失败: {}", e))?;

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
    } else {
        let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
        Err(format!("版本检查失败: {}", stderr))
    }
}

pub fn read_env_var_from_shell_profile(var_name: &str) -> Option<String> {
    #[cfg(windows)]
    {
        crate::config_writer::windows_env::read_env_var(var_name)
    }
    #[cfg(not(windows))]
    {
        let home = dirs::home_dir()?;
        let shell = env::var("SHELL").unwrap_or_default();
        let profile = if shell.contains("zsh") {
            home.join(".zshrc")
        } else {
            home.join(".bashrc")
        };

        let content = std::fs::read_to_string(profile).ok()?;
        for line in content.lines() {
            let trimmed = line.trim();
            if trimmed.starts_with(&format!("export {}=", var_name)) {
                let val = trimmed
                    .splitn(2, '=')
                    .nth(1)?
                    .trim_matches('"')
                    .trim_matches('\'');
                return Some(val.to_string());
            }
        }
        None
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_which_nonexistent_command() {
        assert!(which_command("definitely_not_a_real_command_xyz").is_none());
    }

    #[test]
    fn test_fallback_paths_contains_expected() {
        let paths = fallback_paths("claude");
        assert!(!paths.is_empty());
    }
}
