# AkiGura

Sports facility availability monitoring and notification system.

## Features

- **Automatic Monitoring**: Continuously scrapes public sports facility booking sites
- **Instant Notifications**: Get notified via email when slots become available
- **Calendar View**: Visual calendar interface showing all available slots
- **Watch Rules**: Set up custom monitoring rules for specific facilities and time slots
- **Multi-facility Support**: Monitor multiple facilities across different municipalities

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                       AkiGura System                       │
├─────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────┐      ┌──────────────────────────┐    │
│  │  Control Plane   │      │         Worker           │    │
│  │   (Go Server)    │      │     (Go + Python)        │    │
│  │                  │      │                          │    │
│  │  - Web UI        │◄────►│  - Scraping              │    │
│  │  - Auth (Magic   │  DB  │  - Matching              │    │
│  │    Link/Google)  │      │  - Notifications         │    │
│  │  - REST API      │      │                          │    │
│  │  - Billing       │      │  ┌────────────────────┐  │    │
│  └──────────────────┘      │  │ ground-reservation │  │    │
│          │                 │  │ (Python scraper)   │  │    │
│          ▼                 │  └────────────────────┘  │    │
│   ┌──────────────┐         └──────────────────────────┘    │
│   │ Turso/SQLite │                                       │
│   │   Database   │                                       │
│   └──────────────┘                                       │
└─────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21+
- Python 3.10+
- SQLite3

### 1. Clone the Repository

```bash
git clone https://github.com/susumutomita/AkiGura.git
cd AkiGura
```

### 2. Set Up ground-reservation (Python Scraper)

```bash
git clone https://github.com/susumutomita/ground-reservation.git ../ground-reservation
cd ../ground-reservation
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
cd ../AkiGura
```

### 3. Start the Control Plane

```bash
cd control-plane
go build -o akigura-srv ./cmd/srv
./akigura-srv -listen :8001
```

Open http://localhost:8001 in your browser.

### 4. Start the Worker (separate terminal)

```bash
cd worker
go build -o akigura-worker ./cmd/worker

# Run once
./akigura-worker -once

# Or run periodically (every 15 minutes)
./akigura-worker -interval 15m
```

## Directory Structure

```
AkiGura/
├── control-plane/      # Go web server
│   ├── cmd/srv/        # Main entry point
│   ├── db/             # Database & migrations
│   ├── srv/            # HTTP handlers & templates
│   └── billing/        # Stripe integration
│
├── worker/             # Scraping worker
│   ├── cmd/worker/     # Main entry point
│   ├── notifier/       # Notifications (Email/LINE/Slack)
│   └── scraper_wrapper.py
│
└── docs/               # Documentation
```

## Environment Variables

### Database (Turso / SQLite)

| Variable | Description | Default |
|----------|-------------|----------|
| `TURSO_DATABASE_URL` | Turso database URL | (uses local SQLite if not set) |
| `TURSO_AUTH_TOKEN` | Turso auth token | (required for Turso) |
| `DATABASE_PATH` | Local SQLite path | `./db.sqlite3` |

### Authentication

| Variable | Description | Default |
|----------|-------------|----------|
| `BASE_URL` | Public URL for magic links | `http://localhost:8001` |
| `GOOGLE_CLIENT_ID` | Google OAuth client ID | (optional) |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret | (optional) |
| `SMTP_USER` | Gmail address for sending emails | (optional) |
| `SMTP_PASSWORD` | Gmail app password | (optional) |

### Control Plane

| Variable | Description | Default |
|----------|-------------|----------|
| `OPENAI_API_KEY` | OpenAI API key | (for AI chat) |
| `ANTHROPIC_API_KEY` | Claude API key | (for AI chat) |
| `STRIPE_SECRET_KEY` | Stripe secret key | (for billing) |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook secret | (for billing) |

### Worker

| Variable | Description | Default |
|----------|-------------|----------|
| `SMTP_USER` | Gmail address | (for email notifications) |
| `SMTP_PASSWORD` | Gmail app password | (for email notifications) |
| `SENDGRID_API_KEY` | SendGrid API key | (alternative to SMTP) |
| `LINE_CHANNEL_TOKEN` | LINE Messaging API token | (for LINE notifications) |
| `SLACK_WEBHOOK_URL` | Slack Webhook URL | (for Slack notifications) |

## Supported Facilities

- Yokohama City
- Ayase City
- Hiratsuka City
- Kanagawa Prefecture
- Kamakura City
- Fujisawa City

## Authentication

AkiGura supports two authentication methods:

1. **Magic Link (Email)**: Enter your email to receive a sign-in link
2. **Google OAuth**: Sign in with your Google account (requires configuration)

## Development

See [CLAUDE.md](./CLAUDE.md) for development guidelines.

## License

MIT
