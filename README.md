# AkiGura SaaS - グラウンド監視サービス

草野球チーム向けのグラウンド空き枠監視・通知SaaSのコントロールプレーン

## 機能

### ダッシュボード
- チーム数・施設数・監視条件数のリアルタイム表示
- プラン別チーム分布
- 未対応サポートチケット数

### チーム管理
- チームの登録・編集・削除
- プラン管理 (Free/Personal/Pro/Org)
- ステータス管理 (active/paused/cancelled)

### 施設管理
- 施設の登録・編集
- 自治体別管理
- スクレイパー設定

### AIサポート
- FAQ自動応答チャット
- サポートチケット管理
- 自動エスカレーション

## 技術スタック

- **Backend**: Go 1.24
- **Database**: SQLite (WAL mode)
- **ORM**: sqlc
- **Frontend**: Alpine.js + Tailwind CSS
- **Deployment**: systemd

## セットアップ

```bash
# ビルド
go build -o akigura-srv ./cmd/srv

# 起動
./akigura-srv -listen :8001

# systemdサービスとしてインストール
sudo cp srv.service /etc/systemd/system/akigura.service
sudo systemctl daemon-reload
sudo systemctl enable akigura.service
sudo systemctl start akigura.service
```

## APIエンドポイント

| Method | Endpoint | 説明 |
|--------|----------|------|
| GET | /api/dashboard | ダッシュボードデータ |
| GET | /api/teams | チーム一覧 |
| POST | /api/teams | チーム作成 |
| GET | /api/facilities | 施設一覧 |
| POST | /api/facilities | 施設作成 |
| GET | /api/tickets | チケット一覧 |
| POST | /api/tickets | チケット作成 |
| POST | /api/chat | AIチャット |

## データモデル

- **teams**: チーム情報
- **facilities**: 施設情報
- **watch_conditions**: 監視条件
- **slots**: 空き枠情報
- **notifications**: 通知履歴
- **support_tickets**: サポートチケット
- **support_messages**: チケットメッセージ

## ライセンス

Apache License 2.0
