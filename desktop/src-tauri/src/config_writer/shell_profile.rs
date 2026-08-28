use std::env;
use std::fs;
use std::path::PathBuf;

pub fn get_shell_profile_path() -> Result<PathBuf, String> {
    let home = dirs::home_dir().ok_or("无法获取 HOME 目录")?;

    if cfg!(windows) {
        return Err("Windows 不使用 Shell Profile".to_string());
    }

    let shell = env::var("SHELL").unwrap_or_default();
    if shell.contains("zsh") {
        Ok(home.join(".zshrc"))
    } else {
        Ok(home.join(".bashrc"))
    }
}

pub fn backup_file(path: &PathBuf) -> Result<(), String> {
    if path.exists() {
        let backup = path.with_extension("bak.llm-gateway");
        fs::copy(path, &backup).map_err(|e| format!("备份失败: {}", e))?;
    }
    Ok(())
}

pub fn write_env_block(path: &PathBuf, tool_name: &str, env_vars: &[(&str, &str)]) -> Result<(), String> {
    let content = fs::read_to_string(path).unwrap_or_default();
    let marker_start = format!("# --- LLM Gateway Config [{}] ---", tool_name);
    let marker_end = format!("# --- End LLM Gateway Config [{}] ---", tool_name);

    let mut block = String::new();
    block.push_str(&marker_start);
    block.push('\n');
    for (key, value) in env_vars {
        block.push_str(&format!("export {}=\"{}\"\n", key, value));
    }
    block.push_str(&marker_end);

    let new_content = if content.contains(&marker_start) {
        let start = content.find(&marker_start).unwrap();
        let end = content.find(&marker_end).map(|i| i + marker_end.len()).unwrap_or(content.len());
        format!("{}{}{}", &content[..start], block, &content[end..])
    } else {
        if content.ends_with('\n') || content.is_empty() {
            format!("{}\n{}\n", content, block)
        } else {
            format!("{}\n\n{}\n", content, block)
        }
    };

    fs::write(path, new_content).map_err(|e| format!("写入失败: {}", e))?;

    // macOS: set env vars immediately for new processes via launchctl
    #[cfg(target_os = "macos")]
    for (key, value) in env_vars {
        let _ = std::process::Command::new("launchctl")
            .args(["setenv", key, value])
            .output();
    }

    Ok(())
}

pub fn remove_env_block(path: &PathBuf, tool_name: &str) -> Result<(), String> {
    if !path.exists() {
        return Ok(());
    }
    let content = fs::read_to_string(path).map_err(|e| e.to_string())?;
    let marker_start = format!("# --- LLM Gateway Config [{}] ---", tool_name);
    let marker_end = format!("# --- End LLM Gateway Config [{}] ---", tool_name);
    if !content.contains(&marker_start) {
        return Ok(());
    }

    let start = content.find(&marker_start).unwrap();
    let end = content.find(&marker_end).map(|i| i + marker_end.len()).unwrap_or(content.len());

    let before = content[..start].trim_end_matches('\n');
    let after = content[end..].trim_start_matches('\n');
    let new_content = format!("{}\n{}", before, after);

    fs::write(path, new_content.trim_end().to_owned() + "\n").map_err(|e| format!("写入失败: {}", e))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[test]
    fn test_write_and_remove_env_block() {
        let mut f = NamedTempFile::new().unwrap();
        writeln!(f, "# existing config").unwrap();
        let path = f.path().to_path_buf();

        write_env_block(&path, "test_tool", &[("FOO", "bar"), ("BAZ", "qux")]).unwrap();
        let content = fs::read_to_string(&path).unwrap();
        assert!(content.contains("export FOO=\"bar\""));
        assert!(content.contains("LLM Gateway Config [test_tool]"));

        remove_env_block(&path, "test_tool").unwrap();
        let content = fs::read_to_string(&path).unwrap();
        assert!(!content.contains("LLM Gateway Config"));
        assert!(content.contains("# existing config"));
    }

    #[test]
    fn test_replace_existing_block() {
        let mut f = NamedTempFile::new().unwrap();
        writeln!(f, "before").unwrap();
        let path = f.path().to_path_buf();

        write_env_block(&path, "my_tool", &[("A", "1")]).unwrap();
        write_env_block(&path, "my_tool", &[("B", "2")]).unwrap();

        let content = fs::read_to_string(&path).unwrap();
        assert!(!content.contains("export A="));
        assert!(content.contains("export B=\"2\""));
        assert_eq!(content.matches("# --- LLM Gateway Config [my_tool] ---").count(), 1);
    }

    #[test]
    fn test_multiple_tools_coexist() {
        let mut f = NamedTempFile::new().unwrap();
        writeln!(f, "# shell config").unwrap();
        let path = f.path().to_path_buf();

        write_env_block(&path, "claude_code", &[("ANTHROPIC_BASE_URL", "https://gw.com")]).unwrap();
        write_env_block(&path, "codex_cli", &[("OPENAI_BASE_URL", "https://gw.com/v1")]).unwrap();

        let content = fs::read_to_string(&path).unwrap();
        assert!(content.contains("LLM Gateway Config [claude_code]"));
        assert!(content.contains("LLM Gateway Config [codex_cli]"));

        remove_env_block(&path, "claude_code").unwrap();
        let content = fs::read_to_string(&path).unwrap();
        assert!(!content.contains("claude_code"));
        assert!(content.contains("codex_cli"));
    }
}
