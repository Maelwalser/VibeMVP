#!/usr/bin/env bash
# install.sh — download and install VibeMenu binaries from GitHub Releases
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Maelwalser/vibemenu/main/install.sh | bash
#   VIBEMENU_VERSION=v1.2.3 bash install.sh
#   INSTALL_DIR=~/.local/bin bash install.sh

set -euo pipefail

REPO="Maelwalser/vibemenu"
VERSION="${VIBEMENU_VERSION:-}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()  { printf '\033[1;34m[info]\033[0m  %s\n' "$*"; }
ok()    { printf '\033[1;32m[ok]\033[0m    %s\n' "$*"; }
err()   { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; exit 1; }

need() {
  command -v "$1" >/dev/null 2>&1 || err "required tool not found: $1"
}

need curl
need tar

# ---------------------------------------------------------------------------
# Detect OS / arch
# ---------------------------------------------------------------------------
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  linux)  ;;
  darwin) ;;
  *)      err "Unsupported OS: $OS. On Windows use: irm https://raw.githubusercontent.com/${REPO}/main/install.ps1 | iex" ;;
esac

case "$ARCH" in
  x86_64)          ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *)               err "Unsupported architecture: $ARCH" ;;
esac

# ---------------------------------------------------------------------------
# Resolve version
# ---------------------------------------------------------------------------
if [ -z "$VERSION" ]; then
  info "Fetching latest release version…"
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
  [ -n "$VERSION" ] || err "Could not determine latest version. Set VIBEMENU_VERSION explicitly."
fi

info "Installing VibeMenu ${VERSION} (${OS}/${ARCH}) → ${INSTALL_DIR}"

# ---------------------------------------------------------------------------
# Download and extract
# ---------------------------------------------------------------------------
TARBALL="vibemenu-${VERSION}-${OS}-${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading ${URL}"
curl -fsSL "$URL" -o "${TMPDIR}/${TARBALL}" || err "Download failed. Check that ${VERSION} exists for ${OS}/${ARCH}."

# Verify checksum if sha256sum or shasum is available.
if command -v sha256sum >/dev/null 2>&1 || command -v shasum >/dev/null 2>&1; then
  info "Verifying checksum…"
  curl -fsSL "$CHECKSUM_URL" -o "${TMPDIR}/checksums.txt" 2>/dev/null || true
  if [ -s "${TMPDIR}/checksums.txt" ]; then
    (cd "$TMPDIR" && grep "${TARBALL}" checksums.txt | \
      (command -v sha256sum >/dev/null 2>&1 && sha256sum --check --status || shasum -a 256 --check --status)) \
      || err "Checksum verification failed — download may be corrupted."
    ok "Checksum verified"
  fi
fi

tar -xzf "${TMPDIR}/${TARBALL}" -C "${TMPDIR}"

# ---------------------------------------------------------------------------
# Install
# ---------------------------------------------------------------------------
install_bin() {
  local name="$1"
  local src="${TMPDIR}/${name}"
  [ -f "$src" ] || err "Binary '${name}' not found in release archive."

  if [ -w "$INSTALL_DIR" ]; then
    install -m 755 "$src" "${INSTALL_DIR}/${name}"
  else
    info "Requesting sudo to install to ${INSTALL_DIR}"
    sudo install -m 755 "$src" "${INSTALL_DIR}/${name}"
  fi
  ok "Installed ${INSTALL_DIR}/${name}"
}

mkdir -p "$INSTALL_DIR"
install_bin vibemenu
install_bin realize

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------
cat <<EOF

  VibeMenu ${VERSION} installed successfully.

  Quick start:
    vibemenu          # open the TUI editor
    realize --help    # run code generation (skills auto-extracted on first run)

  Skills are embedded in the realize binary and extracted to .vibemenu/skills/
  on first run. Existing files are never overwritten — customise freely.

EOF
