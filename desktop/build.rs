fn main() {
    // The game server the app opens, fixed at build time (see README).
    println!("cargo:rerun-if-env-changed=EATTHEKING_URL");
    tauri_build::build()
}
