use serde::Serialize;
use sysinfo::System;

#[derive(Serialize)]
pub struct HardwareInfo {
    pub cpu_cores: usize,
    pub cpu_model: String,
    pub total_ram_gb: u64,
    pub os_name: String,
    pub os_version: String,
}

#[tauri::command]
pub fn detect_hardware() -> HardwareInfo {
    let mut sys = System::new_all();
    sys.refresh_all();

    HardwareInfo {
        cpu_cores: sys.cpus().len(),
        cpu_model: sys
            .cpus()
            .get(0)
            .map(|c| c.brand().to_string())
            .unwrap_or_else(|| "Unknown".to_string()),
        total_ram_gb: sys.total_memory() / 1024 / 1024 / 1024,
        os_name: System::name().unwrap_or_else(|| "Unknown".to_string()),
        os_version: System::os_version().unwrap_or_else(|| "Unknown".to_string()),
    }
}
