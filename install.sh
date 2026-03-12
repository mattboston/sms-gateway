#!/usr/bin/env bash
set -euo pipefail

DEBUG="${DEBUG:-0}"

REPO="mattboston/sms-gateway"
INSTALL_DIR="/opt/sms-gateway"
SERVICE_USER="sms-gateway"
REGISTRY="ghcr.io"

# --- Helpers ---

info()  { printf '\033[1;34m::\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33m::\033[0m %s\n' "$*"; }
error() { printf '\033[1;31m::\033[0m %s\n' "$*" >&2; }
fatal() { error "$@"; exit 1; }

usage() {
  cat <<'EOF'
Usage: install.sh [--debug]

Options:
  -d, --debug  Enable verbose debug output (set -x)
  -h, --help   Show this help message
EOF
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      -d|--debug)
        DEBUG=1
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        fatal "Unknown option: $1"
        ;;
    esac
    shift
  done

  if [ "$DEBUG" = "1" ]; then
    info "Debug mode enabled"
    set -x
  fi
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fatal "Required command not found: $1"
}

need_root() {
  [ "$(id -u)" -eq 0 ] || fatal "This script must be run as root (use sudo)"
}

prompt() {
  local var="$1" msg="$2" default="$3"
  printf '%s [%s]: ' "$msg" "$default"
  read -r input
  eval "$var=\${input:-$default}"
}

# --- Detect architecture ---

detect_arch() {
  local arch
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)    echo "linux-amd64" ;;
    armv7*|armhf)    echo "linux-arm7" ;;
    aarch64|arm64)   echo "linux-arm64" ;;
    *)               fatal "Unsupported architecture: $arch" ;;
  esac
}

# --- Fetch latest release tag ---

get_latest_version() {
  need_cmd curl
  local url="https://api.github.com/repos/${REPO}/releases/latest"
  curl -fsSL "$url" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
}

# --- Systemd install ---

install_systemd() {
  need_root
  need_cmd curl

  local arch version binary_url mode

  mode="install"
  if [ -f "${INSTALL_DIR}/sms-gateway" ] || systemctl list-unit-files | grep -q '^sms-gateway.service'; then
    mode="upgrade"
  fi
  info "Mode: ${mode}"

  arch="$(detect_arch)"
  info "Detected architecture: $arch"

  info "Fetching latest release..."
  version="$(get_latest_version)"
  [ -n "$version" ] || fatal "Could not determine latest version"
  info "Latest version: $version"

  binary_url="https://github.com/${REPO}/releases/download/${version}/sms-gateway-${arch}"

  # Configuration values used only when creating a new config file.
  local device_path="/dev/ttyUSB2" jwt_secret="" port="5174"
  if [ "$mode" = "install" ]; then
    prompt device_path "Serial device path" "$device_path"
    prompt port "HTTP port" "$port"
    prompt jwt_secret "JWT secret (leave blank to auto-generate)" ""
  else
    info "Upgrade mode: skipping config prompts"
  fi

  if [ -z "$jwt_secret" ]; then
    if command -v openssl >/dev/null 2>&1; then
      jwt_secret="$(openssl rand -base64 32)"
    else
      jwt_secret="$(head -c 32 /dev/urandom | base64)"
    fi
    if [ "$mode" = "install" ]; then
      info "Generated JWT secret"
    fi
  fi

  # Create service user
  if ! id "$SERVICE_USER" >/dev/null 2>&1; then
    info "Creating service user: $SERVICE_USER"
    useradd -r -s /usr/sbin/nologin "$SERVICE_USER"
  else
    info "Service user already exists, skipping: $SERVICE_USER"
  fi

  # Add to dialout group for modem access
  if getent group dialout >/dev/null 2>&1; then
    if id -nG "$SERVICE_USER" | grep -qw dialout; then
      info "$SERVICE_USER is already in dialout group, skipping"
    else
      usermod -aG dialout "$SERVICE_USER"
      info "Added $SERVICE_USER to dialout group"
    fi
  fi

  # Create install directory
  if [ -d "$INSTALL_DIR" ]; then
    info "Install directory already exists, reusing: $INSTALL_DIR"
  else
    info "Creating install directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
  fi

  # Download binary
  if [ -f "${INSTALL_DIR}/sms-gateway" ]; then
    info "Replacing existing binary"
  fi
  info "Downloading sms-gateway ${version} (${arch})..."
  curl -fsSL -o "${INSTALL_DIR}/sms-gateway" "$binary_url"
  chmod 755 "${INSTALL_DIR}/sms-gateway"

  # Write config file (don't overwrite existing)
  local config_file="${INSTALL_DIR}/sms-gateway.conf"
  if [ -f "$config_file" ]; then
    info "Config file already exists, preserving: $config_file"
  else
    if [ "$mode" = "upgrade" ]; then
      warn "Config file missing during upgrade; creating default config: $config_file"
    fi
    cat > "$config_file" <<EOF
DB_DRIVER=sqlite
DB_DSN=${INSTALL_DIR}/sms-gateway.db
DEVICE_PATH=${device_path}
BAUD_RATE=9600
PORT=${port}
JWT_SECRET=${jwt_secret}
EOF
    chmod 600 "$config_file"
    info "Wrote configuration to $config_file"
  fi

  # Set ownership
  chown -R "$SERVICE_USER":"$SERVICE_USER" "$INSTALL_DIR"

  # Install systemd unit
  cat > /etc/systemd/system/sms-gateway.service <<EOF
[Unit]
Description=SMS Gateway
Documentation=https://github.com/${REPO}
After=network.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/sms-gateway serve --config-file ${INSTALL_DIR}/sms-gateway.conf
Restart=on-failure
RestartSec=5

NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${INSTALL_DIR}
PrivateTmp=true

SupplementaryGroups=dialout

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable sms-gateway
  if [ "$mode" = "upgrade" ]; then
    systemctl restart sms-gateway
  else
    systemctl start sms-gateway
  fi

  local service_state service_enabled
  service_state="$(systemctl is-active sms-gateway 2>/dev/null || true)"
  service_enabled="$(systemctl is-enabled sms-gateway 2>/dev/null || true)"

  info "Installation complete!"
  info ""
  info "  Binary:  ${INSTALL_DIR}/sms-gateway"
  info "  Config:  ${INSTALL_DIR}/sms-gateway.conf"
  info "  Service: sms-gateway.service"
  info "  Status:  ${service_state:-unknown} (enabled: ${service_enabled:-unknown})"
  info ""
  info "Start with: sudo systemctl start sms-gateway"
  info "Logs:       sudo journalctl -u sms-gateway -f"
}

# --- Docker install ---

install_docker() {
  info "Docker installation coming soon!"
}

# --- Main ---

main() {
  parse_args "$@"

  echo ""
  echo "  SMS Gateway Installer"
  echo "  ====================="
  echo ""
  echo "  1) Systemd  - Install as a native Linux service"
  echo "  2) Docker   - Run as a Docker container"
  echo ""

  local choice
  prompt choice "Install method" "1"

  case "$choice" in
    1|systemd)  install_systemd ;;
    2|docker)   install_docker ;;
    *)          fatal "Invalid choice: $choice" ;;
  esac
}

main "$@"
