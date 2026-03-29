#!/usr/bin/env bash
# mole installer — https://getkaze.dev/mole/install.sh
# Usage: curl -fsSL https://getkaze.dev/mole/install.sh | sudo bash
set -euo pipefail

BINARY_NAME="mole"
RELEASES_BASE="https://github.com/getkaze/mole/releases"

# Install directory: always ~/.local/bin (user-writable, enables self-update without sudo).
REAL_USER="${SUDO_USER:-$(whoami)}"
REAL_HOME=$(eval echo "~${REAL_USER}")
if [ "$(id -u)" = "0" ]; then
  INSTALL_DIR="${REAL_HOME}/.local/bin"
  mkdir -p "$INSTALL_DIR"
  chown "$REAL_USER" "$INSTALL_DIR"
else
  INSTALL_DIR="${HOME}/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

# ── color helpers ──────────────────────────────────────────────────────────────
bold=$(tput bold 2>/dev/null || true)
reset=$(tput sgr0 2>/dev/null || true)
green=$(tput setaf 2 2>/dev/null || true)
red=$(tput setaf 1 2>/dev/null || true)

info()  { echo "${bold}==>${reset} $*"; }
ok()    { echo "${green}  ✓${reset} $*"; }
fail()  { echo "${red}error:${reset} $*" >&2; exit 1; }

# ── sanity checks ─────────────────────────────────────────────────────────────
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux)  ;;
  darwin) ;;
  *)      fail "unsupported OS: $OS (supported: linux, darwin)" ;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       fail "unsupported architecture: $ARCH" ;;
esac

# Config directory
CONFIG_DIR="/etc/mole"

# Detect MOLE_VERSION (optional — defaults to latest)
VERSION="${MOLE_VERSION:-}"
if [ -z "$VERSION" ]; then
  info "fetching latest version"
  VERSION="$(curl -fsSL -o /dev/null -w '%{url_effective}' "${RELEASES_BASE}/latest" 2>/dev/null || true)"
  VERSION="${VERSION##*/}"
  [ -n "$VERSION" ] && [ "$VERSION" != "latest" ] || fail "could not fetch latest version from ${RELEASES_BASE}/latest"
  VERSION="$(echo "$VERSION" | tr -d '[:space:]')"
fi

DOWNLOAD_URL="${RELEASES_BASE}/download/${VERSION}/${BINARY_NAME}-${OS}-${ARCH}"

# ── download binary ────────────────────────────────────────────────────────────
info "installing mole ${VERSION} (${OS}/${ARCH})"

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

if command -v curl &>/dev/null; then
  curl -fsSL "$DOWNLOAD_URL" -o "$TMP"
elif command -v wget &>/dev/null; then
  wget -qO "$TMP" "$DOWNLOAD_URL"
else
  fail "curl or wget is required"
fi

chmod +x "$TMP"
mv "$TMP" "${INSTALL_DIR}/${BINARY_NAME}"
ok "binary installed to ${INSTALL_DIR}/${BINARY_NAME}"

# ── create config directory ───────────────────────────────────────────────────
info "setting up config directory (${CONFIG_DIR})"

mkdir -p "${CONFIG_DIR}"

if [ ! -f "${CONFIG_DIR}/mole.yaml" ]; then
  cat > "${CONFIG_DIR}/mole.yaml" <<'YAML'
# Mole — AI-powered PR reviewer
# Docs: https://getkaze.dev/mole/docs

github:
  app_id: 0
  private_key_path: /etc/mole/github-app.pem
  webhook_secret: ""

llm:
  api_key: ""
  review_model: "claude-sonnet-4-6"
  deep_review_model: "claude-opus-4-6"
  pricing:
    claude-sonnet-4-6: [3.00, 15.00]
    claude-opus-4-6: [15.00, 75.00]

mysql:
  host: localhost
  port: 3306
  database: mole
  user: mole
  password: ""

valkey:
  host: localhost
  port: 6379

server:
  port: 8080

worker:
  count: 3

log:
  level: info

dashboard:
  github_client_id: ""
  github_client_secret: ""
  session_secret: ""
  base_url: "http://localhost:8080"
  allowed_org: ""
YAML
  ok "created default mole.yaml"
else
  ok "mole.yaml already exists, skipping"
fi

# ── ownership ──────────────────────────────────────────────────────────────────
# Give the calling (non-root) user ownership so mole can self-update without sudo.
if [ "$(id -u)" = "0" ] && [ -n "${SUDO_USER:-}" ]; then
  chown -R "${SUDO_USER}" "${CONFIG_DIR}"
  chown "${SUDO_USER}" "${INSTALL_DIR}/${BINARY_NAME}"
  ok "ownership set to ${SUDO_USER} (self-update enabled)"
fi

# ── done ───────────────────────────────────────────────────────────────────────
echo ""
echo "${bold}mole ${VERSION} installed successfully!${reset}"

# Check if install dir is in PATH
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "${bold}Note:${reset} ${INSTALL_DIR} is not in your PATH."
    echo "  Add it to your shell profile:"
    echo ""
    echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""
    ;;
esac

echo ""
echo "Next steps:"
echo "  1. Edit ${CONFIG_DIR}/mole.yaml with your credentials"
echo "  2. Copy your GitHub App private key to ${CONFIG_DIR}/github-app.pem"
echo "  3. Ensure MySQL and Valkey are running"
echo ""
echo "Quick start:"
echo "  mole migrate             # run database migrations"
echo "  mole health              # check MySQL, Valkey, GitHub connectivity"
echo "  mole serve               # start the server"
echo ""
echo "Docs: https://getkaze.dev/mole/docs"
