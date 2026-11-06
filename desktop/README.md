# Eat the King for the desktop

A small Rust app ([Tauri 2](https://tauri.app)) for Windows, Mac and Linux. It
opens the game server in a window of its own. It holds no chess rules: the
server decides every move and what each player may see, as it does for the
website.

- `ui/` is the bundled loading screen: it checks the server, then moves the
  window over to it. If the server can't be reached it says so and retries.
- `src/main.rs` makes the window, keeps one copy of the app running, remembers
  the window's size and place, and sends links that leave the game to the
  system browser.
- Every page the window loads gets `window.__ETK_DESKTOP__ = { version, server }`.
  The web app reads it (`web/src/desktop.ts`) to show the app's home with the
  three ways to play, and to hide the Download link and the footer.
- The site gets no access to the app: there are no permissions in
  `capabilities/`.

## Run and build

    make dev            # the server, in one terminal
    make desktop        # the app against it, in another

    GAME_URL=https://example.com make desktop-build   # installers for this system

`GAME_URL` is the server the app opens, fixed at build time. Running a build
with `EATTHEKING_URL` set opens another server instead, for testing.

On Linux, install WebKitGTK first:

    sudo apt install libwebkit2gtk-4.1-dev libayatana-appindicator3-dev librsvg2-dev libxdo-dev libssl-dev

## Releases

`make release` pushes a version tag; `.github/workflows/desktop.yml` then
builds the installers on Windows, Mac and Linux and attaches them to that
tag's GitHub release as `EatTheKing-windows-x64.msi`,
`EatTheKing-mac-universal.dmg`, `EatTheKing-linux-x86_64.AppImage` and
`EatTheKing-linux-amd64.deb`. The site's /download page links to
`releases/latest/download/<name>`. The workflow needs the `GAME_URL`
repository variable.

## Open

- The builds aren't code-signed: Windows SmartScreen and macOS Gatekeeper
  warn on first launch (the download page says how to get past it). Signing
  needs an Apple Developer account and a Windows certificate.
- No notifications when it's your move, no `eattheking://` links and no
  auto-update yet.
