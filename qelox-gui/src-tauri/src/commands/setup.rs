use serde::Serialize;
use std::fs;


#[derive(Serialize)]
pub struct SetupResponse {
    pub is_first_run: bool,
    pub config_dir: String,
}

#[tauri::command]
pub fn setup_first_run() -> Result<SetupResponse, String> {
    let mut config_dir = dirs::config_dir().ok_or("Could not find config directory")?;
    config_dir.push("qelox");

    let is_first_run = !config_dir.exists();

    if is_first_run {
        fs::create_dir_all(&config_dir).map_err(|e| format!("Failed to create config dir: {}", e))?;
        // In a real scenario, we might want to copy a default config.toml here.
        // For now, we just ensure the directory exists.
    }

    Ok(SetupResponse {
        is_first_run,
        config_dir: config_dir.to_string_lossy().into_owned(),
    })
}
