---
name: sms-gateway
description: Send and receive SMS messages through a self-hosted SMS Gateway running on a USB GSM modem. Use when the user needs to send text messages, check for incoming messages, or manage SMS communications via a local gateway.
metadata:
  openclaw:
    requires:
      env:
        - SMS_GATEWAY_API_KEY
      bins:
        - curl
        - jq
    primaryEnv: SMS_GATEWAY_API_KEY
---

# SMS Gateway Skill

You can send and receive SMS messages through the SMS Gateway using shell scripts located in `~/.openclaw/workspace/skills/sms-gateway/scripts`.

## Prerequisites

Before using the SMS Gateway skill with OpenClaw, you need to install the SMS Gateway service. Read the [README.md](https://github.com/mattboston/sms-gateway/blob/main/openclaw/README.md) for more details.

## Setup

Before using this skill, create a `.env` file in the `~/.openclaw/workspace/skills/sms-gateway/scripts` directory with the following variables:

```text
SMS_GATEWAY_URL=http://127.0.0.1:5174
SMS_GATEWAY_API_KEY=your-api-key-here
```

If the `.env` file is missing or `SMS_GATEWAY_API_KEY` is not set, the scripts will exit with an error.

### Allowlist

The file `~/.openclaw/workspace/skills/sms-gateway/scripts/allowlist.json` contains a list of users and phone numbers that are permitted to send and receive messages. The send script will refuse to send to numbers not in this list. The receive script only shows messages from allowlisted numbers by default, filtering out unknown senders. To add or remove allowed contacts, edit the `allowlist.json` file directly.

## Capabilities

### Send an SMS

Send a text message to an allowed phone number.

```bash
~/.openclaw/workspace/skills/sms-gateway/scripts/send_sms.sh -t "+15551234567" -m "Hello from SMS Gateway"
```

Options:

- `-t` - Recipient phone number (must be in `allowlist.json`)
- `-m` - Message body

The script validates the recipient against `allowlist.json` before sending. If the number is not in the allowlist, it prints the list of allowed numbers and exits with an error.

On success, the script prints "Message sent successfully." followed by the JSON response containing the message ID and status.

On failure, the script prints the HTTP status code and error response.

### Receive SMS Messages

Check the inbox for received messages.

```bash
~/.openclaw/workspace/skills/sms-gateway/scripts/receive_sms.sh
```

Options:

- `-s <status>` - Filter by status: `received` (default), `read`, or `all`
- `-a` - Include messages from numbers not in the allowlist (by default, only allowlisted numbers are shown)

Examples:

```bash
~/.openclaw/workspace/skills/sms-gateway/scripts/receive_sms.sh              # Unread messages from allowlisted numbers
~/.openclaw/workspace/skills/sms-gateway/scripts/receive_sms.sh -s all       # All messages from allowlisted numbers
~/.openclaw/workspace/skills/sms-gateway/scripts/receive_sms.sh -s read      # Previously read messages from allowlisted numbers
~/.openclaw/workspace/skills/sms-gateway/scripts/receive_sms.sh -a           # Unread messages from all numbers
~/.openclaw/workspace/skills/sms-gateway/scripts/receive_sms.sh -a -s all    # All messages from all numbers
```

The script displays each message with its timestamp, sender number, status, body, and ID. Any unread messages (`status=received`) are automatically marked as read after being displayed. By default, messages from numbers not in `allowlist.json` are filtered out.

## Output Format

### Send output

```text
Message sent successfully.
{
  "id": "uuid",
  "status": "sent",
  "message": "message sent"
}
```

### Receive output

```text
Inbox messages (received): 2

[2026-03-08T12:00:00Z] From: +15551234567
  Status: received
  Body: Hey there
  ID: some-uuid

Marked message some-uuid as read.
```

## Usage Guidelines

- Only send SMS messages to numbers listed in `allowlist.json`
- Only accept SMS messages from numbers listed in `allowlist.json`
- Always include the country code in phone numbers (e.g., `+1` for US)
- When the user asks to "text" or "message" someone, use `send_sms.sh`
- When the user asks about new messages, run `receive_sms.sh` with no flags
- When the user asks to see all messages, run `receive_sms.sh -s all`
- If a send fails, relay the error message to the user
- Only use the `send_sms.sh` and `receive_sms.sh` scripts to send and receive, do not communicate directly with the `SMS_GATEWAY_API_KEY`
- When you receive an SMS from a contact, check whether you previously sent an SMS to that same number in the current session or a recent workflow — if so, treat the incoming message as a reply to that conversation rather than an unrelated inbound message

## Troubleshooting

### "Phone number is not in the allowlist"

The recipient number must exactly match an entry in `allowlist.json`, including the country code prefix (e.g., `+1`). Check the allowlist and add the number if appropriate.

### "SMS_GATEWAY_API_KEY is not set"

Create a `.env` file in `~/.openclaw/workspace/skills/sms-gateway/scripts/` with your API key. See the Setup section above.

### Send fails with error

Check that the SMS Gateway is running and accessible at the URL configured in `.env`. You can verify connectivity with:

```bash
curl -s http://127.0.0.1:5174/api/v1/health
```
