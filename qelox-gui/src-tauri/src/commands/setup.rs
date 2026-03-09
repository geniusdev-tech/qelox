use serde::Serialize;
use std::fs;
use std::path::PathBuf;

#[derive(Serialize)]
pub struct SetupResponse {
    pub is_first_run: bool,
    pub config_dir: String,
}

#[tauri::command]
pub fn setup_first_run() -> Result<SetupResponse, String> {
    let home_dir = dirs::home_dir().ok_or("Could not find home directory")?;
    let config_dir = PathBuf::from(&home_dir).join("qelox");
    let config_path = config_dir.join("config.toml");

    let is_first_run = !config_path.exists();

    if is_first_run {
        fs::create_dir_all(&config_dir)
            .map_err(|e| format!("Failed to create config dir: {}", e))?;
    }

    Ok(SetupResponse {
        is_first_run,
        config_dir: config_dir.to_string_lossy().into_owned(),
    })
}
