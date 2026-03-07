mod commands;

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_updater::Builder::new().build())
        .invoke_handler(tauri::generate_handler![
            commands::setup::setup_first_run,
            commands::hardware::detect_hardware,
            commands::installer::check_node_installed,
            commands::installer::install_node,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

