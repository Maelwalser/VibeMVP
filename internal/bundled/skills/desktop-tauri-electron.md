# Tauri + Electron Skill Guide

## Tauri

### Project Layout

```
app/
├── src-tauri/
│   ├── Cargo.toml
│   ├── tauri.conf.json
│   ├── build.rs
│   └── src/
│       ├── main.rs             # Entry point
│       ├── lib.rs              # Command registration
│       └── commands/
│           ├── mod.rs
│           ├── fs_commands.rs
│           └── system_commands.rs
├── src/                        # Frontend (React/Vue/Svelte)
│   ├── App.tsx
│   └── lib/
│       └── tauri.ts            # Typed invoke wrappers
├── package.json
└── index.html
```

### Key Dependencies

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never hardcode version strings.

```toml
# src-tauri/Cargo.toml
[package]
name = "app"
version = "0.1.0"
edition = "2021"

[dependencies]
tauri = { version = "<version>", features = [] }
tauri-plugin-fs = "<version>"
tauri-plugin-dialog = "<version>"
tauri-plugin-notification = "<version>"
tauri-plugin-shell = "<version>"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
tokio = { version = "1", features = ["full"] }
```

### #[tauri::command] Functions

```rust
// src-tauri/src/commands/system_commands.rs
use serde::{Deserialize, Serialize};
use tauri::State;

#[derive(Debug, Serialize, Deserialize)]
pub struct FileInfo {
    pub path: String,
    pub size: u64,
    pub modified: u64,
}

// Basic command
#[tauri::command]
pub fn get_platform() -> String {
    std::env::consts::OS.to_string()
}

// Async command with error handling
#[tauri::command]
pub async fn read_config(path: String) -> Result<String, String> {
    tokio::fs::read_to_string(&path)
        .await
        .map_err(|e| e.to_string())
}

// Command with application state
#[tauri::command]
pub fn get_app_version(app_handle: tauri::AppHandle) -> String {
    app_handle.package_info().version.to_string()
}

// Command with managed state
#[tauri::command]
pub fn increment_counter(counter: State<'_, std::sync::Mutex<u32>>) -> u32 {
    let mut count = counter.lock().unwrap();
    *count += 1;
    *count
}
```

### lib.rs — Command Registration

```rust
// src-tauri/src/lib.rs
mod commands;

use commands::system_commands::*;

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_shell::init())
        .manage(std::sync::Mutex::new(0u32))   // managed state
        .invoke_handler(tauri::generate_handler![
            get_platform,
            read_config,
            get_app_version,
            increment_counter,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
```

### invoke() from Frontend

```typescript
// src/lib/tauri.ts
import { invoke } from '@tauri-apps/api/core';
import { open, save } from '@tauri-apps/plugin-dialog';
import { readTextFile, writeTextFile } from '@tauri-apps/plugin-fs';
import { sendNotification } from '@tauri-apps/plugin-notification';

// Typed wrapper around invoke
export async function getPlatform(): Promise<string> {
  return invoke('get_platform');
}

export async function readConfig(path: string): Promise<string> {
  return invoke('read_config', { path });
}

// Dialog
export async function openFile(): Promise<string | null> {
  const path = await open({
    multiple: false,
    filters: [{ name: 'JSON', extensions: ['json'] }],
  });
  return typeof path === 'string' ? path : null;
}

// Filesystem
export async function loadFile(path: string): Promise<string> {
  return readTextFile(path);
}

export async function saveFile(path: string, content: string): Promise<void> {
  return writeTextFile(path, content);
}

// Notification
export function notify(title: string, body: string) {
  sendNotification({ title, body });
}
```

### tauri.conf.json (Security)

```json
{
  "productName": "MyApp",
  "version": "0.1.0",
  "identifier": "com.example.myapp",
  "build": {
    "frontendDist": "../dist",
    "devUrl": "http://localhost:5173"
  },
  "app": {
    "security": {
      "csp": "default-src 'self'; script-src 'self'"
    },
    "windows": [
      {
        "title": "MyApp",
        "width": 1200,
        "height": 800,
        "minWidth": 800,
        "minHeight": 600
      }
    ]
  }
}
```

### Emit / Listen (Events)

```rust
// Rust → Frontend
use tauri::Emitter;

app_handle.emit("download-progress", serde_json::json!({ "percent": 42 })).unwrap();
```

```typescript
// Frontend — listen to events
import { listen } from '@tauri-apps/api/event';

const unlisten = await listen<{ percent: number }>('download-progress', event => {
  console.log('Progress:', event.payload.percent);
});

// Cleanup
unlisten();
```

### Tauri Key Rules

- Every command exposed to the frontend must be registered in `generate_handler![]`.
- Always return `Result<T, String>` from fallible commands — never panic.
- Use `serde::Serialize` on all return types — structs must derive it.
- Restrict permissions in `tauri.conf.json` — only enable plugins you use.
- Use managed state (`app.manage()`) for shared app state accessed across commands.
- Frontend `invoke` calls must match command name and argument names exactly.

---

## Electron

### Project Layout

```
app/
├── package.json
├── electron-builder.yml
├── src/
│   ├── main/                   # Main process (Node.js)
│   │   ├── index.ts            # BrowserWindow creation + IPC handlers
│   │   └── ipc/
│   │       └── handlers.ts
│   ├── preload/
│   │   └── index.ts            # contextBridge exposure
│   └── renderer/               # Frontend (React/Vue/etc.)
│       ├── index.html
│       └── src/
│           └── App.tsx
└── resources/
```

### Main Process

```typescript
// src/main/index.ts
import { app, BrowserWindow, ipcMain } from 'electron';
import { join } from 'path';
import { registerHandlers } from './ipc/handlers';

let mainWindow: BrowserWindow | null = null;

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    webPreferences: {
      preload: join(__dirname, '../preload/index.js'),
      contextIsolation: true,   // REQUIRED for security
      nodeIntegration: false,   // REQUIRED for security
    },
  });

  if (process.env.NODE_ENV === 'development') {
    mainWindow.loadURL('http://localhost:5173');
    mainWindow.webContents.openDevTools();
  } else {
    mainWindow.loadFile(join(__dirname, '../renderer/index.html'));
  }
}

app.whenReady().then(() => {
  createWindow();
  registerHandlers();
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});
```

### IPC Handlers (Main Process)

```typescript
// src/main/ipc/handlers.ts
import { ipcMain, dialog } from 'electron';
import { readFile, writeFile } from 'fs/promises';

export function registerHandlers() {
  // Handle typed IPC calls from renderer
  ipcMain.handle('fs:read-file', async (_, path: string) => {
    return readFile(path, 'utf-8');
  });

  ipcMain.handle('fs:write-file', async (_, path: string, content: string) => {
    await writeFile(path, content, 'utf-8');
  });

  ipcMain.handle('dialog:open', async (event) => {
    const window = BrowserWindow.fromWebContents(event.sender);
    const result = await dialog.showOpenDialog(window!, {
      properties: ['openFile'],
      filters: [{ name: 'JSON', extensions: ['json'] }],
    });
    return result.canceled ? null : result.filePaths[0];
  });
}
```

### Preload Script (contextBridge)

```typescript
// src/preload/index.ts
import { contextBridge, ipcRenderer } from 'electron';

// Expose a safe API — renderer has NO direct access to Node/Electron APIs
contextBridge.exposeInMainWorld('electronAPI', {
  readFile: (path: string) => ipcRenderer.invoke('fs:read-file', path),
  writeFile: (path: string, content: string) =>
    ipcRenderer.invoke('fs:write-file', path, content),
  openDialog: () => ipcRenderer.invoke('dialog:open'),

  // Event listener with cleanup
  onDownloadProgress: (callback: (percent: number) => void) => {
    const handler = (_: Electron.IpcRendererEvent, percent: number) => callback(percent);
    ipcRenderer.on('download-progress', handler);
    return () => ipcRenderer.removeListener('download-progress', handler);  // cleanup
  },
});
```

```typescript
// src/renderer/src/global.d.ts — type declarations
interface ElectronAPI {
  readFile: (path: string) => Promise<string>;
  writeFile: (path: string, content: string) => Promise<void>;
  openDialog: () => Promise<string | null>;
  onDownloadProgress: (cb: (percent: number) => void) => () => void;
}
declare interface Window {
  electronAPI: ElectronAPI;
}
```

### Renderer Usage

```typescript
// Any React component
const content = await window.electronAPI.readFile('/path/to/file.json');
const path = await window.electronAPI.openDialog();

// Cleanup event listener
useEffect(() => {
  const cleanup = window.electronAPI.onDownloadProgress(pct => setProgress(pct));
  return cleanup;
}, []);
```

### electron-builder Config

```yaml
# electron-builder.yml
appId: com.example.myapp
productName: MyApp
directories:
  output: dist-electron
files:
  - dist/
  - dist-electron/
mac:
  category: public.app-category.developer-tools
  target: [dmg, zip]
win:
  target: [nsis, portable]
linux:
  target: [AppImage, deb]
  category: Development
nsis:
  oneClick: false
  allowToChangeInstallationDirectory: true
```

### Electron Key Rules

- ALWAYS set `contextIsolation: true` and `nodeIntegration: false` — non-negotiable security.
- ALL IPC must go through `contextBridge` — never expose `ipcRenderer` directly.
- Use `ipcMain.handle` + `ipcRenderer.invoke` for request/response patterns.
- Use `ipcRenderer.on` + `removeListener` for event streams — always clean up listeners.
- Type the `electronAPI` in `global.d.ts` so the renderer is fully type-safe.
- Never send sensitive data through IPC without validation in the main process handler.
- Use `BrowserWindow.fromWebContents(event.sender)` to get the window for dialogs.
