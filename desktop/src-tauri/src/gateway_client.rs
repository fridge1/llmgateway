use futures_util::StreamExt;
use reqwest::{Client, StatusCode};
use serde::{Deserialize, Serialize};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;

#[derive(Debug, Serialize, Deserialize)]
pub struct LoginRequest {
    pub phone: String,
    pub password: String,
    pub remember: bool,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct LoginResponse {
    pub token: String,
    pub phone: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserInfo {
    pub user_id: Option<String>,
    pub phone: String,
    pub role: String,
}

#[derive(Debug)]
pub enum GatewayError {
    Network(String),
    Unauthorized,
    ApiError(String),
}

impl std::fmt::Display for GatewayError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            GatewayError::Network(e) => write!(f, "网络错误: {}", e),
            GatewayError::Unauthorized => write!(f, "登录已过期"),
            GatewayError::ApiError(e) => write!(f, "API 错误: {}", e),
        }
    }
}

pub struct GatewayClient {
    client: Client,
    pub base_url: String,
}

impl GatewayClient {
    pub fn new(base_url: &str) -> Self {
        Self {
            client: Client::new(),
            base_url: base_url.trim_end_matches('/').to_string(),
        }
    }

    pub async fn login(&self, phone: &str, password: &str, remember: bool) -> Result<LoginResponse, GatewayError> {
        let resp = self.client
            .post(format!("{}/api/login", self.base_url))
            .json(&LoginRequest { phone: phone.to_string(), password: password.to_string(), remember })
            .send()
            .await
            .map_err(|e| GatewayError::Network(e.to_string()))?;

        match resp.status() {
            StatusCode::OK => resp.json().await.map_err(|e| GatewayError::ApiError(e.to_string())),
            StatusCode::UNAUTHORIZED => Err(GatewayError::Unauthorized),
            s => Err(GatewayError::ApiError(format!("HTTP {}", s))),
        }
    }

    pub async fn get_me(&self, token: &str) -> Result<UserInfo, GatewayError> {
        let resp = self.client
            .get(format!("{}/api/me", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .send()
            .await
            .map_err(|e| GatewayError::Network(e.to_string()))?;

        match resp.status() {
            StatusCode::OK => resp.json().await.map_err(|e| GatewayError::ApiError(e.to_string())),
            StatusCode::UNAUTHORIZED => Err(GatewayError::Unauthorized),
            s => Err(GatewayError::ApiError(format!("HTTP {}", s))),
        }
    }

    async fn get_json(&self, path: &str, token: &str) -> Result<serde_json::Value, GatewayError> {
        let resp = self.client
            .get(format!("{}{}", self.base_url, path))
            .header("Authorization", format!("Bearer {}", token))
            .send().await.map_err(|e| GatewayError::Network(e.to_string()))?;
        self.handle_response(resp).await
    }

    async fn handle_response(&self, resp: reqwest::Response) -> Result<serde_json::Value, GatewayError> {
        match resp.status() {
            StatusCode::OK | StatusCode::CREATED => resp.json().await.map_err(|e| GatewayError::ApiError(e.to_string())),
            StatusCode::UNAUTHORIZED => Err(GatewayError::Unauthorized),
            s => Err(GatewayError::ApiError(format!("HTTP {}", s))),
        }
    }

    pub async fn list_keys(&self, token: &str) -> Result<serde_json::Value, GatewayError> {
        self.get_json("/api/keys", token).await
    }

    pub async fn create_key(&self, token: &str) -> Result<serde_json::Value, GatewayError> {
        let resp = self.client.post(format!("{}/api/keys", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .json(&serde_json::json!({"name": "desktop"}))
            .send().await.map_err(|e| GatewayError::Network(e.to_string()))?;
        self.handle_response(resp).await
    }

    pub async fn get_balance(&self, token: &str) -> Result<serde_json::Value, GatewayError> {
        self.get_json("/api/billing/balance", token).await
    }

    pub async fn get_stats(&self, token: &str) -> Result<serde_json::Value, GatewayError> {
        self.get_json("/api/billing/stats?days=30", token).await
    }

    pub async fn list_models(&self, token: &str) -> Result<serde_json::Value, GatewayError> {
        self.get_json("/api/v1/models", token).await
    }

    pub async fn get_or_create_key(&self, token: &str) -> Result<String, GatewayError> {
        let new_key = self.create_key(token).await?;
        new_key.get("key").and_then(|k| k.as_str()).map(|s| s.to_string())
            .ok_or(GatewayError::ApiError("创建 Key 失败".to_string()))
    }

    pub async fn create_key_for_tool(&self, token: &str, tool_name: &str) -> Result<serde_json::Value, GatewayError> {
        let key_name = format!("desktop-{}", tool_name);
        let resp = self.client.post(format!("{}/api/keys", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .json(&serde_json::json!({"name": key_name}))
            .send().await.map_err(|e| GatewayError::Network(e.to_string()))?;
        self.handle_response(resp).await
    }

    pub async fn get_or_create_key_for_tool(&self, token: &str, tool_name: &str) -> Result<String, GatewayError> {
        let new_key = self.create_key_for_tool(token, tool_name).await?;
        new_key.get("key").and_then(|k| k.as_str()).map(|s| s.to_string())
            .ok_or(GatewayError::ApiError("创建 Key 失败".to_string()))
    }

    pub async fn request(
        &self,
        method: &str,
        path: &str,
        body: Option<serde_json::Value>,
        token: Option<&str>,
    ) -> Result<serde_json::Value, GatewayError> {
        let url = format!("{}{}", self.base_url, path);
        let mut req = match method.to_uppercase().as_str() {
            "GET" => self.client.get(&url),
            "POST" => self.client.post(&url),
            "PUT" => self.client.put(&url),
            "DELETE" => self.client.delete(&url),
            "PATCH" => self.client.patch(&url),
            _ => return Err(GatewayError::ApiError(format!("不支持的 HTTP 方法: {}", method))),
        };
        if let Some(t) = token {
            req = req.header("Authorization", format!("Bearer {}", t));
        }
        if let Some(b) = body {
            req = req.json(&b);
        }
        let resp = req.send().await.map_err(|e| GatewayError::Network(e.to_string()))?;
        if resp.status() == StatusCode::NO_CONTENT {
            return Ok(serde_json::Value::Null);
        }
        self.handle_response(resp).await
    }

    /// POSTs a multipart/form-data request. `text_fields` are plain form fields;
    /// `file_fields` are (field_name, filename, mime, bytes) tuples. Used for the
    /// image-edit endpoint, which the JSON IPC path can't carry.
    pub async fn request_multipart(
        &self,
        path: &str,
        text_fields: Vec<(String, String)>,
        file_fields: Vec<(String, String, String, Vec<u8>)>,
        token: &str,
    ) -> Result<serde_json::Value, GatewayError> {
        let url = format!("{}{}", self.base_url, path);
        let mut form = reqwest::multipart::Form::new();
        for (name, value) in text_fields {
            form = form.text(name, value);
        }
        for (name, filename, mime, bytes) in file_fields {
            let part = reqwest::multipart::Part::bytes(bytes)
                .file_name(filename)
                .mime_str(&mime)
                .map_err(|e| GatewayError::ApiError(e.to_string()))?;
            form = form.part(name, part);
        }
        let resp = self
            .client
            .post(&url)
            .header("Authorization", format!("Bearer {}", token))
            .multipart(form)
            .send()
            .await
            .map_err(|e| GatewayError::Network(e.to_string()))?;
        if resp.status() == StatusCode::NO_CONTENT {
            return Ok(serde_json::Value::Null);
        }
        self.handle_response(resp).await
    }
}

/// A single parsed SSE data line from an OpenAI-style streaming response.
#[derive(Debug, PartialEq)]
pub enum SseEvent {
    /// A content delta to append to the assistant message.
    Delta(String),
    /// Usage stats (may arrive on a chunk without a content delta).
    Usage(serde_json::Value),
    /// The terminal `[DONE]` marker.
    Done,
    /// A line that carries no actionable payload (blank, comment, unparseable).
    Ignore,
}

/// Parses one line of an SSE stream into an [`SseEvent`].
///
/// Mirrors the web client's parsing: only `data: ` lines are meaningful,
/// `[DONE]` ends the stream, and each JSON chunk may carry a content delta
/// and/or a usage object.
pub fn parse_sse_line(line: &str) -> SseEvent {
    let trimmed = line.trim();
    if trimmed.is_empty() || !trimmed.starts_with("data:") {
        return SseEvent::Ignore;
    }
    let data = trimmed["data:".len()..].trim();
    if data == "[DONE]" {
        return SseEvent::Done;
    }
    let parsed: serde_json::Value = match serde_json::from_str(data) {
        Ok(v) => v,
        Err(_) => return SseEvent::Ignore,
    };
    if let Some(usage) = parsed.get("usage") {
        if !usage.is_null() {
            return SseEvent::Usage(usage.clone());
        }
    }
    if let Some(delta) = parsed
        .get("choices")
        .and_then(|c| c.get(0))
        .and_then(|c| c.get("delta"))
        .and_then(|d| d.get("content"))
        .and_then(|c| c.as_str())
    {
        if !delta.is_empty() {
            return SseEvent::Delta(delta.to_string());
        }
    }
    SseEvent::Ignore
}

impl GatewayClient {
    /// POSTs a streaming chat-completion request and invokes `on_event` for each
    /// parsed SSE event. Returns when the stream ends, errors, or `cancel` is set.
    pub async fn stream_completions<F>(
        &self,
        body: serde_json::Value,
        token: &str,
        cancel: Arc<AtomicBool>,
        mut on_event: F,
    ) -> Result<(), GatewayError>
    where
        F: FnMut(SseEvent),
    {
        let resp = self
            .client
            .post(format!("{}/api/playground/completions", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .json(&body)
            .send()
            .await
            .map_err(|e| GatewayError::Network(e.to_string()))?;

        match resp.status() {
            StatusCode::OK => {}
            StatusCode::UNAUTHORIZED => return Err(GatewayError::Unauthorized),
            s => {
                let detail = resp.text().await.unwrap_or_default();
                let msg = serde_json::from_str::<serde_json::Value>(&detail)
                    .ok()
                    .and_then(|v| {
                        v.get("error")
                            .and_then(|e| e.get("message"))
                            .and_then(|m| m.as_str())
                            .map(|s| s.to_string())
                    })
                    .unwrap_or_else(|| format!("HTTP {}", s));
                return Err(GatewayError::ApiError(msg));
            }
        }

        let mut stream = resp.bytes_stream();
        let mut buffer = String::new();

        while let Some(chunk) = stream.next().await {
            if cancel.load(Ordering::Relaxed) {
                break;
            }
            let bytes = chunk.map_err(|e| GatewayError::Network(e.to_string()))?;
            buffer.push_str(&String::from_utf8_lossy(&bytes));

            // Process complete lines; keep the trailing partial line in the buffer.
            while let Some(pos) = buffer.find('\n') {
                let line: String = buffer.drain(..=pos).collect();
                match parse_sse_line(&line) {
                    SseEvent::Ignore => {}
                    SseEvent::Done => return Ok(()),
                    ev => on_event(ev),
                }
            }
        }

        // Flush any remaining buffered line.
        if !buffer.trim().is_empty() {
            match parse_sse_line(&buffer) {
                SseEvent::Ignore | SseEvent::Done => {}
                ev => on_event(ev),
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_new_client() {
        let client = GatewayClient::new("https://example.com");
        assert_eq!(client.base_url, "https://example.com");
    }

    #[test]
    fn test_new_client_trailing_slash() {
        let client = GatewayClient::new("https://example.com/");
        assert_eq!(client.base_url, "https://example.com");
    }

    #[test]
    fn test_parse_sse_delta() {
        let line = r#"data: {"choices":[{"delta":{"content":"你好"}}]}"#;
        assert_eq!(parse_sse_line(line), SseEvent::Delta("你好".to_string()));
    }

    #[test]
    fn test_parse_sse_done() {
        assert_eq!(parse_sse_line("data: [DONE]"), SseEvent::Done);
    }

    #[test]
    fn test_parse_sse_usage() {
        let line = r#"data: {"choices":[],"usage":{"total_tokens":42}}"#;
        match parse_sse_line(line) {
            SseEvent::Usage(v) => assert_eq!(v["total_tokens"], 42),
            other => panic!("expected Usage, got {:?}", other),
        }
    }

    #[test]
    fn test_parse_sse_ignores_blank_and_comments() {
        assert_eq!(parse_sse_line(""), SseEvent::Ignore);
        assert_eq!(parse_sse_line("\n"), SseEvent::Ignore);
        assert_eq!(parse_sse_line(": keep-alive"), SseEvent::Ignore);
    }

    #[test]
    fn test_parse_sse_ignores_empty_delta() {
        let line = r#"data: {"choices":[{"delta":{}}]}"#;
        assert_eq!(parse_sse_line(line), SseEvent::Ignore);
    }

    #[test]
    fn test_parse_sse_ignores_unparseable() {
        assert_eq!(parse_sse_line("data: not-json"), SseEvent::Ignore);
    }
}
