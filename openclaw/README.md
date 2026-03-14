# SMS Gateway — OpenClaw Skill

An [OpenClaw](https://openclaw.ai) skill for sending and receiving SMS through a self-hosted [SMS Gateway](https://github.com/mattboston/sms-gateway) running on a USB GSM modem.

## Requirements

- A USB GSM modem (e.g., [SIM7600G-H 4G LTE USB Dongle](https://www.amazon.com/dp/B0BHQFTFPH?tag=mattboston-20))
- Any SIM card with an active SMS plan will work.
  - We use [Tello](https://tello.com/account/register?_referral=P30KX3Z2), which offers affordable pay-as-you-go plans on the T-Mobile network. I think I'm paying $8/month for unlimited SMS.

> **Note:** The links above are referral links. Using them helps support the development of this project and is greatly appreciated!
>
> If you'd like to support the project further, check out my [Amazon Wish List](https://www.amazon.com/hz/wishlist/ls/T3L6QCKZJ4Q4?ref_=wl_share).

## Install the SMS Gateway Service

The SMS Gateway is a self-hosted Go binary that serves both a REST API and a WebUI. Run the automated install script on your server or Raspberry Pi:

```bash
curl -fsSL https://raw.githubusercontent.com/mattboston/sms-gateway/main/install.sh | sudo bash
```

Or download and run manually:

```bash
curl -fsSL -o install.sh https://raw.githubusercontent.com/mattboston/sms-gateway/main/install.sh
chmod +x install.sh
sudo ./install.sh
```

The script installs SMS Gateway as a systemd service, prompts for your device path (e.g., `/dev/ttyUSB0`), port, and JWT secret, then starts the service automatically.

### Upgrading

Re-run the install script to upgrade to the latest release — it will update the binary and restart the service.

### Default Login

On first start, a default admin account is created:

- **Username:** `admin`
- **Password:** `admin123`

You will be required to change the password on first login.

## Install the OpenClaw Skill

Copy the skill files into your OpenClaw workspace:

```bash
mkdir -p ~/.openclaw/workspace/skills/sms-gateway
cp -R path/to/openclaw/* ~/.openclaw/workspace/skills/sms-gateway/
```

Or clone and symlink from the repo:

```bash
git clone https://github.com/mattboston/sms-gateway.git ~/sms-gateway
mkdir -p ~/.openclaw/workspace/skills
ln -s ~/sms-gateway/openclaw ~/.openclaw/workspace/skills/sms-gateway
```

## Setup

1. Create your environment file:

```bash
cp ~/.openclaw/workspace/skills/sms-gateway/scripts/.env.example \
   ~/.openclaw/workspace/skills/sms-gateway/scripts/.env
```

2. Log in to the SMS Gateway WebUI at `http://localhost:5174`.
3. Go to **API Keys** and create a new API key.
4. Set `SMS_GATEWAY_API_KEY` in your `.env` file to that key:

```
SMS_GATEWAY_URL=http://127.0.0.1:5174
SMS_GATEWAY_API_KEY=your-api-key-here
```

5. Edit `~/.openclaw/workspace/skills/sms-gateway/scripts/allowlist.json` with the names and phone numbers permitted to send and receive messages:

```json
{
  "users": [
    {
      "name": "Alice Example",
      "phone": "+15551234567"
    }
  ]
}
```

6. Restart OpenClaw so it picks up the new skill.

## Usage

### Send an SMS

```bash
~/.openclaw/workspace/skills/sms-gateway/scripts/send_sms.sh -t "+15551234567" -m "Hello from OpenClaw"
```

### Receive SMS

```bash
~/.openclaw/workspace/skills/sms-gateway/scripts/receive_sms.sh
```

See [SKILL.md](SKILL.md) for full usage, options, and OpenClaw integration guidelines.

## Scripts

| Script | Arguments | Description |
|--------|-----------|-------------|
| `send_sms.sh` | `-t <phone> -m <message>` | Send an SMS to an allowlisted number |
| `receive_sms.sh` | `[-s status] [-a]` | Check inbox for received messages |

## Packaging

```bash
./package.sh
# Creates dist/sms-gateway-<version>.tar.gz
```

## License

GPL-3.0
