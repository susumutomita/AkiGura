# AkiGura

野球場の空き枠監視・通知システム

## アーキテクチャ

```
┌─────────────────────────────────────────────────────────┐
│                    AkiGura System                        │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────────┐      ┌──────────────────────────┐ │
│  │  Control Plane   │      │        Worker            │ │
│  │  (Go Server)     │      │   (Go + Python)          │ │
│  │                  │      │                          │ │
│  │  - Dashboard     │◄────►│  - Scraping              │ │
│  │  - Team管理      │      │  - Matching              │ │
│  │  - 施設管理      │  DB  │  - 通知送信              │ │
│  │  - AIチャット    │      │                          │ │
│  │  - Billing       │      │  ┌────────────────────┐  │ │
│  └──────────────────┘      │  │ ground-reservation │  │ │
│          │                 │  │ (Python scraper)   │  │ │
│          ▼                 │  └────────────────────┘  │ │
│   ┌──────────────┐         └──────────────────────────┘ │
│   │   SQLite     │                                      │
│   │   Database   │                                      │
│   └──────────────┘                                      │
└─────────────────────────────────────────────────────────┘
```

## クイックスタート

### 前提条件

- Go 1.21+
- Python 3.10+
- SQLite3

### 1. リポジトリのクローン

```bash
git clone https://github.com/susumutomita/AkiGura.git
cd AkiGura
```

### 2. ground-reservation (Pythonスクレイパー) のセットアップ

```bash
# 別リポジトリをクローン
git clone https://github.com/susumutomita/ground-reservation.git ../ground-reservation

# Python仮想環境を作成・有効化
cd ../ground-reservation
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

cd ../AkiGura
```

### 3. Control Plane (管理サーバー) の起動

```bash
cd control-plane
go build -o akigura-srv ./cmd/srv
./akigura-srv -listen :8000
```

ブラウザで http://localhost:8000 にアクセス

### 4. Worker の起動 (別ターミナル)

```bash
cd worker
go build -o akigura-worker ./cmd/worker

# 一回だけ実行
./akigura-worker -once

# または定期実行 (15分間隔)
./akigura-worker -interval 15m
```

## 起動スクリプト

両方を一度に起動するには:

```bash
./start.sh
```

## ディレクトリ構成

```
AkiGura/
├── control-plane/      # Go製管理サーバー
│   ├── cmd/srv/        # メインエントリーポイント
│   ├── db/             # データベース・マイグレーション
│   ├── srv/            # HTTPハンドラー・テンプレート
│   └── billing/        # Stripe課金
│
├── worker/             # スクレイピングWorker
│   ├── cmd/worker/     # メインエントリーポイント
│   ├── notifier/       # 通知送信 (Email/LINE/Slack)
│   └── scraper_wrapper.py
│
└── packages/           # (レガシー: TypeScript版)
```

## 環境変数

### Control Plane

| 変数 | 説明 | デフォルト |
|------|------|------------|
| `PORT` | サーバーポート | 8000 |
| `DATABASE_PATH` | SQLiteパス | `./db.sqlite3` |
| `OPENAI_API_KEY` | OpenAI APIキー | (AIチャット用) |
| `ANTHROPIC_API_KEY` | Claude APIキー | (AIチャット用) |
| `STRIPE_SECRET_KEY` | Stripe秘密鍵 | (課金用) |
| `STRIPE_WEBHOOK_SECRET` | Stripeウェブフック秘密 | (課金用) |

### Worker

| 変数 | 説明 | デフォルト |
|------|------|------------|
| `RESEND_API_KEY` | Resend APIキー | (メール通知用) |
| `LINE_CHANNEL_TOKEN` | LINE Messaging APIトークン | (LINE通知用) |
| `SLACK_WEBHOOK_URL` | Slack Webhook URL | (Slack通知用) |

## 対応施設

- 横浜市
- 綾瀬市
- 平塚市
- 神奈川県
- 鎌倉市
- 藤沢市

## 開発

詳細は [CLAUDE.md](./CLAUDE.md) を参照。

## ライセンス

MIT
