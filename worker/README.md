# AkiGura Worker

スクレイピングと通知マッチングを行うWorker

## セットアップ

### 1. ground-reservationのセットアップ

```bash
# ground-reservationをクローン
git clone https://github.com/susumutomita/ground-reservation.git ../ground-reservation

# Python依存関係をインストール
cd ../ground-reservation
pip install -r requirements.txt
```

### 2. Workerのビルド

```bash
cd worker
go build -o akigura-worker ./cmd/worker
```

### 3. 実行

```bash
# 一回だけ実行
./akigura-worker -once

# スケジューラーとして実行 (15分間隔)
./akigura-worker -interval 15m
```

## オプション

| フラグ | デフォルト | 説明 |
|------|----------|------|
| `-db` | `../control-plane/db.sqlite3` | データベースパス |
| `-scraper` | `./scraper_wrapper.py` | スクレイパーラッパー |
| `-interval` | `15m` | スクレイプ間隔 |
| `-once` | `false` | 一回だけ実行 |

## アーキテクチャ

```
Worker
  │
  ├─ scraper_wrapper.py  # Pythonスクレイパーを呼び出しJSON出力
  │     │
  │     └─ ground-reservation/  # 実際のスクレイピングロジック
  │
  ├─ worker.go  # スクレイピング実行・スロット保存
  │
  └─ matcher.go # 監視条件とのマッチング・通知作成
```

## 対応施設

- yokohama: 横浜市
- ayase: 綾瀬市
- hiratsuka: 平塚市
- kanagawa: 神奈川県
- kamakura: 鎌倉市
- fujisawa: 藤沢市
