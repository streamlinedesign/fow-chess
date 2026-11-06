//! Eat the King for the desktop: one window that opens the game site.
//!
//! The game itself (rules, fog of war, accounts) lives on the server, so the
//! app holds no chess logic. It starts on a bundled loading page (`ui/`),
//! which checks the server and then moves the window over to it. Links that
//! leave the game open in the system browser.

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri::{Manager, WebviewUrl, WebviewWindowBuilder};
use tauri_plugin_opener::OpenerExt;
use url::Url;

/// The game server, set at build time with EATTHEKING_URL. A local
/// `make dev` server when unset. EATTHEKING_URL at run time overrides it, for
/// trying a build against another server.
fn server() -> Url {
    let built = option_env!("EATTHEKING_URL").unwrap_or("http://localhost:4100");
    let url = std::env::var("EATTHEKING_URL").unwrap_or_else(|_| built.to_string());
    Url::parse(&url).unwrap_or_else(|_| Url::parse("http://localhost:4100").unwrap())
}

/// Same scheme, host and port.
fn same_origin(a: &Url, b: &Url) -> bool {
    a.origin() == b.origin()
}

/// The bundled loading page: tauri://localhost on Mac and Linux,
/// http(s)://tauri.localhost on Windows.
fn is_bundled(url: &Url) -> bool {
    url.scheme() == "tauri" || url.host_str() == Some("tauri.localhost")
}

fn main() {
    let server = server();
    let version = env!("CARGO_PKG_VERSION");

    tauri::Builder::default()
        // A second launch (or a game link opened from elsewhere) brings the
        // open window forward instead of starting another app.
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            if let Some(w) = app.get_webview_window("main") {
                let _ = w.unminimize();
                let _ = w.set_focus();
            }
        }))
        .plugin(tauri_plugin_window_state::Builder::default().build())
        .plugin(tauri_plugin_opener::init())
        .setup(move |app| {
            // Every page the window loads, the game site included, learns it
            // runs in the app, which version, and which server to open.
            let init = format!(
                "window.__ETK_DESKTOP__ = Object.freeze({{ version: {}, server: {} }});",
                serde_json::to_string(version)?,
                serde_json::to_string(server.as_str())?,
            );
            let handle = app.handle().clone();
            let home = server.clone();
            WebviewWindowBuilder::new(app, "main", WebviewUrl::App("index.html".into()))
                .title("Eat the King")
                .inner_size(1280.0, 860.0)
                .min_inner_size(400.0, 600.0)
                .background_color(tauri::window::Color(0xff, 0xfa, 0xbd, 0xff))
                .initialization_script(&init)
                .on_navigation(move |url| {
                    if is_bundled(url) || same_origin(url, &home) {
                        return true;
                    }
                    // Anything else (Streamline's site, GitHub) belongs in the browser.
                    let _ = handle.opener().open_url(url.as_str(), None::<&str>);
                    false
                })
                .build()?;
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("couldn't start Eat the King");
}
