#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/config.sh"

usage() {
  echo "Usage: $0 [options]"
  echo ""
  echo "Options:"
  echo "  -s <status>   Filter by status: received (default), read, or all"
  echo "  -a            Include messages from numbers NOT in the allowlist"
  echo "  -h            Show this help"
  exit 1
}

STATUS="received"
INCLUDE_NON_ALLOWLIST=false

while getopts "s:ah" opt; do
  case "${opt}" in
    s) STATUS="${OPTARG}" ;;
    a) INCLUDE_NON_ALLOWLIST=true ;;
    h) usage ;;
    *) usage ;;
  esac
done

# Build query parameters
QUERY=""
if [[ "${STATUS}" == "all" ]]; then
  QUERY="?all=true"
else
  QUERY="?status=${STATUS}"
fi

# Fetch inbox messages
RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X GET "${SMS_GATEWAY_URL}/api/v1/sms/inbox${QUERY}" \
  -H "X-API-Key: ${SMS_GATEWAY_API_KEY}" \
  -H "Content-Type: application/json")

HTTP_CODE=$(echo "${RESPONSE}" | tail -1)
BODY_RESPONSE=$(echo "${RESPONSE}" | sed '$d')

if [[ "${HTTP_CODE}" -ge 200 && "${HTTP_CODE}" -lt 300 ]]; then
  # By default, only show messages from allowlisted numbers
  # Use -a to also include messages from non-allowlisted numbers
  if [[ "${INCLUDE_NON_ALLOWLIST}" == false ]]; then
    if [[ ! -f "${ALLOWLIST_FILE}" ]]; then
      echo "Error: Allowlist file not found at ${ALLOWLIST_FILE}" >&2
      exit 1
    fi
    ALLOWED_PHONES=$(jq -r '[.users[].phone]' "${ALLOWLIST_FILE}")
    BODY_RESPONSE=$(echo "${BODY_RESPONSE}" | jq --argjson allowed "${ALLOWED_PHONES}" '[.[] | select(.phone_number as $p | $allowed | index($p))]')
  fi

  COUNT=$(echo "${BODY_RESPONSE}" | jq 'length')
  echo "Inbox messages (${STATUS}): ${COUNT}"
  echo ""
  echo "${BODY_RESPONSE}" | jq -r '.[] | "[\(.created_at)] From: \(.phone_number)\n  Status: \(.status)\n  Body: \(.body)\n  ID: \(.id)\n"'

  # Mark unread messages as read
  UNREAD_IDS=$(echo "${BODY_RESPONSE}" | jq -r '.[] | select(.status == "received") | .id')
  for ID in ${UNREAD_IDS}; do
    curl -s -X PUT "${SMS_GATEWAY_URL}/api/v1/sms/${ID}/read" \
      -H "X-API-Key: ${SMS_GATEWAY_API_KEY}" \
      -H "Content-Type: application/json" > /dev/null
    echo "Marked message ${ID} as read."
  done
else
  echo "Error: Failed to fetch inbox (HTTP ${HTTP_CODE})." >&2
  echo "${BODY_RESPONSE}" | jq . 2>/dev/null || echo "${BODY_RESPONSE}" >&2
  exit 1
fi
