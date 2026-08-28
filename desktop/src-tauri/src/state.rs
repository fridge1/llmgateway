use std::collections::HashMap;
use std::sync::atomic::AtomicBool;
use std::sync::{Arc, Mutex};

pub struct AppState {
    pub gateway_url: Mutex<String>,
    pub token: Mutex<Option<String>>,
    /// Cancellation flags for in-flight streaming requests, keyed by request_id.
    pub stream_cancels: Mutex<HashMap<String, Arc<AtomicBool>>>,
}

impl AppState {
    pub fn new(gateway_url: String) -> Self {
        Self {
            gateway_url: Mutex::new(gateway_url),
            token: Mutex::new(None),
            stream_cancels: Mutex::new(HashMap::new()),
        }
    }
}
