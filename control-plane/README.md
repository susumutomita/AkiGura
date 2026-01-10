# AkiGura Control Plane

Go製の管理コンソール

## セットアップ

```bash
cd control-plane
go build -o akigura-srv ./cmd/srv
./akigura-srv -listen :8001
```

## 機能

- ダッシュボード
- チーム管理
- 施設管理
- AIサポートチャット

## API

| Method | Endpoint | 説明 |
|--------|----------|------|
| GET | /api/dashboard | ダッシュボード |
| GET | /api/teams | チーム一覧 |
| POST | /api/teams | チーム作成 |
| GET | /api/facilities | 施設一覧 |
| POST | /api/facilities | 施設作成 |
| POST | /api/chat | AIチャット |
