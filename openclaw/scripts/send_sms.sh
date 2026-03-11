#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/config.sh"

usage() {
  echo "Usage: $0 -t <phone_number> -m <message>"
  echo ""
  echo "Options:"
  echo "  -t    Recipient phone number (must be in allowlist)"
  echo "  -m    Message body"
  echo "  -h    Show this help"
  exit 1
}

TO=""
BODY=""

while getopts "t:m:h" opt; do
  case "${opt}" in
    t) TO="${OPTARG}" ;;
    m) BODY="${OPTARG}" ;;
    h) usage ;;
    *) usage ;;
  esac
done

if [[ -z "${TO}" || -z "${BODY}" ]]; then
  usage
fi

# Verify the recipient is in the allowlist
if [[ ! -f "${ALLOWLIST_FILE}" ]]; then
  echo "Error: Allowlist file not found at ${ALLOWLIST_FILE}" >&2
  exit 1
fi

ALLOWED=$(jq -r --arg phone "${TO}" '.users[] | select(.phone == $phone) | .phone' "${ALLOWLIST_FILE}")

if [[ -z "${ALLOWED}" ]]; then
  echo "Error: Phone number ${TO} is not in the allowlist." >&2
  echo "Allowed numbers:" >&2
  jq -r '.users[] | "  \(.name): \(.phone)"' "${ALLOWLIST_FILE}" >&2
  exit 1
fi

# Send the SMS
RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X POST "${SMS_GATEWAY_URL}/api/v1/sms/send" \
  -H "X-API-Key: ${SMS_GATEWAY_API_KEY}" \
  -H "Content-Type: application/json" \
  -d "$(jq -n --arg to "${TO}" --arg body "${BODY}" '{to: $to, body: $body}')")

HTTP_CODE=$(echo "${RESPONSE}" | tail -1)
BODY_RESPONSE=$(echo "${RESPONSE}" | sed '$d')

if [[ "${HTTP_CODE}" -ge 200 && "${HTTP_CODE}" -lt 300 ]]; then
  echo "Message sent successfully."
  echo "${BODY_RESPONSE}" | jq .
else
  echo "Error: Failed to send message (HTTP ${HTTP_CODE})." >&2
  echo "${BODY_RESPONSE}" | jq . 2>/dev/null || echo "${BODY_RESPONSE}" >&2
  exit 1
fi
