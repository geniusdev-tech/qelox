use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;
use tauri::{Emitter, Window};

#[derive(Serialize, Clone)]
pub struct InstallProgress {
    pub percentage: u8,
    pub status: String,
}

#[derive(Deserialize)]
struct GitHubRelease {
    assets: Vec<GitHubAsset>,
}

#[derive(Deserialize)]
struct GitHubAsset {
    name: String,
    browser_download_url: String,
}

#[tauri::command]
pub async fn check_node_installed(path: String) -> bool {
    fs::metadata(path).is_ok()
}

#[tauri::command]
pub async fn install_node(window: Window, target_dir: String) -> Result<String, String> {
    // 1. Query GitHub for latest release
    let client = reqwest::Client::builder()
        .user_agent("qelox-gui")
        .build()
        .map_err(|e| e.to_string())?;

    window
        .emit(
            "install-progress",
            InstallProgress {
                percentage: 5,
                status: "Fetching latest release...".into(),
            },
        )
        .unwrap();

    let release: GitHubRelease = client
        .get("https://api.github.com/repos/dominant-strategies/go-quai/releases/latest")
        .send()
        .await
        .map_err(|e| e.to_string())?
        .json()
        .await
        .map_err(|e| e.to_string())?;

    // 2. Select asset based on OS/Arch
    let (os, arch) = match (std::env::consts::OS, std::env::consts::ARCH) {
        ("linux", "x86_64") => ("linux", "amd64"),
        ("windows", "x86_64") => ("windows", "amd64"),
        ("macos", "x86_64") => ("darwin", "amd64"),
        ("macos", "aarch64") => ("darwin", "arm64"),
        _ => return Err("Unsupported platform".into()),
    };

    let asset = release
        .assets
        .iter()
        .find(|a| a.name.contains(os) && a.name.contains(arch))
        .ok_or_else(|| format!("No asset found for {}/{}", os, arch))?;

    window
        .emit(
            "install-progress",
            InstallProgress {
                percentage: 10,
                status: format!("Downloading {}...", asset.name),
            },
        )
        .unwrap();

    // 3. Download
    let response = client
        .get(&asset.browser_download_url)
        .send()
        .await
        .map_err(|e| e.to_string())?;

    let total_size = response.content_length().unwrap_or(0);

    let mut dest_path = PathBuf::from(&target_dir);
    fs::create_dir_all(&dest_path).map_err(|e| e.to_string())?;
    dest_path.push(if os == "windows" {
        "go-quai.exe"
    } else {
        "go-quai"
    });

    let mut out = fs::File::create(&dest_path).map_err(|e| e.to_string())?;
    let mut downloaded: u64 = 0;

    use futures_util::StreamExt;
    let mut stream = response.bytes_stream();

    while let Some(item) = stream.next().await {
        let chunk = item.map_err(|e| e.to_string())?;
        std::io::copy(&mut &*chunk, &mut out).map_err(|e| e.to_string())?;
        downloaded += chunk.len() as u64;

        if total_size > 0 {
            let percentage = (downloaded as f64 / total_size as f64 * 80.0) as u8 + 10;
            window
                .emit(
                    "install-progress",
                    InstallProgress {
                        percentage,
                        status: "Downloading...".into(),
                    },
                )
                .unwrap();
        }
    }

    // 4. Finalize
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        let mut perms = fs::metadata(&dest_path)
            .map_err(|e| e.to_string())?
            .permissions();
        perms.set_mode(0o755);
        fs::set_permissions(&dest_path, perms).map_err(|e| e.to_string())?;
    }

    window
        .emit(
            "install-progress",
            InstallProgress {
                percentage: 100,
                status: "Installation complete!".into(),
            },
        )
        .unwrap();

    Ok(dest_path.to_string_lossy().into_owned())
}
