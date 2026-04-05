# install.ps1 — download and install VibeMenu binaries from GitHub Releases
#
# Usage (run in an elevated PowerShell prompt):
#   irm https://raw.githubusercontent.com/Maelwalser/vibemenu/main/install.ps1 | iex
#
# Override defaults:
#   $env:VIBEMENU_VERSION = "v1.2.3"
#   $env:INSTALL_DIR = "$env:LOCALAPPDATA\Programs\vibemenu"
#   irm https://raw.githubusercontent.com/Maelwalser/vibemenu/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo       = "Maelwalser/vibemenu"
$Version    = if ($env:VIBEMENU_VERSION) { $env:VIBEMENU_VERSION } else { "" }
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\vibemenu" }

function Write-Info  { Write-Host "[info]  $args" -ForegroundColor Cyan }
function Write-Ok    { Write-Host "[ok]    $args" -ForegroundColor Green }
function Write-Err   { Write-Host "[error] $args" -ForegroundColor Red; exit 1 }

# ---------------------------------------------------------------------------
# Detect arch
# ---------------------------------------------------------------------------
$Arch = if ([System.Environment]::Is64BitOperatingSystem) { "amd64" } else {
    Write-Err "Only x86-64 (amd64) Windows is supported. Download manually from https://github.com/$Repo/releases"
}

# ---------------------------------------------------------------------------
# Resolve version
# ---------------------------------------------------------------------------
if (-not $Version) {
    Write-Info "Fetching latest release version..."
    try {
        $Release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
        $Version = $Release.tag_name
    } catch {
        Write-Err "Could not determine latest version. Set `$env:VIBEMENU_VERSION` explicitly."
    }
}
if (-not $Version) { Write-Err "Empty version returned from GitHub API." }

Write-Info "Installing VibeMenu $Version (windows/$Arch) -> $InstallDir"

# ---------------------------------------------------------------------------
# Download and extract
# ---------------------------------------------------------------------------
$ZipName    = "vibemenu-$Version-windows-$Arch.zip"
$Url        = "https://github.com/$Repo/releases/download/$Version/$ZipName"
$ChecksumUrl = "https://github.com/$Repo/releases/download/$Version/checksums.txt"
$TmpDir     = Join-Path $env:TEMP "vibemenu-install-$([System.IO.Path]::GetRandomFileName())"
New-Item -ItemType Directory -Path $TmpDir | Out-Null

try {
    $ZipPath = Join-Path $TmpDir $ZipName
    Write-Info "Downloading $Url"
    Invoke-WebRequest -Uri $Url -OutFile $ZipPath -UseBasicParsing

    # Verify checksum if available.
    try {
        $ChecksumPath = Join-Path $TmpDir "checksums.txt"
        Invoke-WebRequest -Uri $ChecksumUrl -OutFile $ChecksumPath -UseBasicParsing -ErrorAction SilentlyContinue
        if (Test-Path $ChecksumPath) {
            $Expected = (Get-Content $ChecksumPath | Where-Object { $_ -match [regex]::Escape($ZipName) }) -split '\s+' | Select-Object -First 1
            if ($Expected) {
                $Actual = (Get-FileHash $ZipPath -Algorithm SHA256).Hash.ToLower()
                if ($Actual -ne $Expected.ToLower()) {
                    Write-Err "Checksum mismatch — download may be corrupted. Expected: $Expected  Got: $Actual"
                }
                Write-Ok "Checksum verified"
            }
        }
    } catch {
        Write-Info "Skipping checksum verification (checksums.txt unavailable)"
    }

    Write-Info "Extracting archive..."
    Expand-Archive -Path $ZipPath -DestinationPath $TmpDir -Force

    # ---------------------------------------------------------------------------
    # Install
    # ---------------------------------------------------------------------------
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir | Out-Null
    }

    foreach ($Bin in @("vibemenu.exe", "realize.exe")) {
        $Src = Join-Path $TmpDir $Bin
        if (-not (Test-Path $Src)) { Write-Err "Binary '$Bin' not found in release archive." }
        Copy-Item $Src -Destination (Join-Path $InstallDir $Bin) -Force
        Write-Ok "Installed $InstallDir\$Bin"
    }

    # Add InstallDir to the user PATH if not already present.
    $UserPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
    if ($UserPath -notlike "*$InstallDir*") {
        [System.Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstallDir", "User")
        Write-Info "Added $InstallDir to your user PATH (restart your terminal to apply)"
    }

} finally {
    Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue
}

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------
Write-Host ""
Write-Host "  VibeMenu $Version installed successfully." -ForegroundColor Green
Write-Host ""
Write-Host "  Quick start:"
Write-Host "    vibemenu          # open the TUI editor"
Write-Host "    realize --help    # run code generation (skills auto-extracted on first run)"
Write-Host ""
Write-Host "  Skills are embedded in the realize binary and extracted to .vibemenu\skills\"
Write-Host "  on first run. Existing files are never overwritten — customise freely."
Write-Host ""
