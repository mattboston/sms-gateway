#!/usr/bin/env bash
# Configuration for SMS Gateway scripts
# Override these via environment variables or a .env file

SMS_GATEWAY_URL="${SMS_GATEWAY_URL:-http://localhost:5174}"
SMS_GATEWAY_API_KEY="${SMS_GATEWAY_API_KEY:-}"

# Path to the allowlist file (relative to this script's directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ALLOWLIST_FILE="${ALLOWLIST_FILE:-${SCRIPT_DIR}/allowlist.json}"

# Load .env file if it exists alongside these scripts
if [[ -f "${SCRIPT_DIR}/.env" ]]; then
  # shellcheck source=/dev/null
  source "${SCRIPT_DIR}/.env"
fi

# Validate required settings
if [[ -z "${SMS_GATEWAY_API_KEY}" ]]; then
  echo "Error: SMS_GATEWAY_API_KEY is not set." >&2
  echo "Set it via environment variable or in ${SCRIPT_DIR}/.env" >&2
  exit 1
fi
