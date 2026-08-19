#!/usr/bin/env bash
#
# Installer for KzLogViewer (https://github.com/karozadev/KzLogViewer).
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/karozadev/KzLogViewer/main/install.sh | bash
#   wget -qO- https://raw.githubusercontent.com/karozadev/KzLogViewer/main/install.sh | bash
#
# Environment variables:
#   KZLOGVIEWER_INSTALL_DIR   Target directory for the binary (default: /usr/local/bin, falling
#                             back to $HOME/.local/bin when not writable).

set -euo pipefail

REPO="karozadev/KzLogViewer"
BINARY_NAME="kzlogviewer"

log() { printf '==> %s\n' "$*"; }
fail() { printf 'error: %s\n' "$*" >&2; exit 1; }

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *) fail "unsupported operating system: $(uname -s). Download a release manually from https://github.com/${REPO}/releases" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac
}

detect_install_dir() {
  if [ -n "${KZLOGVIEWER_INSTALL_DIR:-}" ]; then
    echo "$KZLOGVIEWER_INSTALL_DIR"
    return
  fi
  if [ -w "/usr/local/bin" ]; then
    echo "/usr/local/bin"
    return
  fi
  echo "${HOME}/.local/bin"
}

download() {
  url="$1"
  dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$dest"
  else
    fail "curl or wget is required to install KzLogViewer"
  fi
}

main() {
  os="$(detect_os)"
  arch="$(detect_arch)"
  install_dir="$(detect_install_dir)"
  archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
  download_url="https://github.com/${REPO}/releases/latest/download/${archive_name}"

  work_dir="$(mktemp -d)"
  trap 'rm -rf "$work_dir"' EXIT

  log "Downloading ${download_url}"
  download "$download_url" "${work_dir}/${archive_name}"

  log "Extracting archive"
  tar -xzf "${work_dir}/${archive_name}" -C "$work_dir" "$BINARY_NAME"

  mkdir -p "$install_dir"
  install -m 0755 "${work_dir}/${BINARY_NAME}" "${install_dir}/${BINARY_NAME}"

  log "Installed ${BINARY_NAME} to ${install_dir}/${BINARY_NAME}"

  case ":$PATH:" in
    *":${install_dir}:"*) ;;
    *) log "Note: ${install_dir} is not on your PATH. Add it with: export PATH=\"${install_dir}:\$PATH\"" ;;
  esac

  log "Run '${BINARY_NAME}' to start, or '${BINARY_NAME} update' to upgrade later."
}

main "$@"
