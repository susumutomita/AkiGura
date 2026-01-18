# AkiGura アーキテクチャ概要（2026-01 現在）

最新の構成・実装を前提に再整理したドキュメントです。以前の Clerk・SES 依存構成とは異なり、現バージョンでは **Go 製コントロールプレーン + Python ワーカー + Turso(DB)** を中核とし、UI も同サーバーで提供しています。

---

## 1. サービス全体像

```
┌──────────┐         ┌────────────┐
│ AkiGura UI │<───────>│ Go Control │───┐
│ (管理/ユーザー)│  REST  │ Plane/API  │   │
└──────────┘         │ (./control-plane)│   │
                       └──────┬─────┘   │
                              │         │
                              ▼         │
                       ┌────────────┐  │
                       │ Turso(libSQL)│◄┘  永続 DB
                       └────────────┘
                              ▲
                              │Queue/Jobs
                              ▼
                       ┌────────────┐
                       │Python Worker│（スクレイパー）
                       └────────────┘
```

- **Control Plane**: Go 1.25.5。API・管理画面・ユーザー画面を提供し、Turso に接続してデータを保存。Stripe 課金や通知ルール管理を担う。
- **Worker**: 既存の ground-reservation 由来の Python スクレイパーを呼び出して空き枠を取得。結果は `slots` テーブルに書き込む。
- **DB**: Turso(libSQL)。ローカル開発では `control-plane/db.sqlite3` を生成し、同一 schema を使用。

### 認証・課金
- 現在のローカル実装は **Magic Link / Clerk** ではなく、独自 UI + エミュレータ的なログインフロー（メールアドレスを入力→Magic Link風に擬似ログイン）を使用しています。
- 課金は Stripe Checkout + Billing Portal + Webhook を実装済み。`billing.Plans` に PriceID を環境変数で注入。
- Webhook 署名検証は `stripe-go/v79` を使用し、環境変数 `STRIPE_WEBHOOK_SECRET` に依存。

### 通知
- まだ Amazon SES ではなく、通知送信は今後 SendGrid 等で実装予定（Plan.md Phase 4 参照）。現バージョンのドキュメントからは SES の記述を削除。

---

## 2. データモデル

### 主要テーブル（抜粋）

`teams`
- plan (`free/personal/pro/org`)
- stripe_customer_id, stripe_subscription_id
- billing_interval, current_period_end

`grounds`
- municipality_id, name, court_pattern 等。UI の監視条件選択で使用。

`watch_conditions`
- team_id, municipality_id, ground_id, days_of_week, time_from/to
- UI から作成・削除可能。`/user` ページの “Add Watch Rule” モーダルに紐づく。

`slots`
- municipality_id + slot_date + time_from + court_name で UNIQUE 制約
- Python Worker がスクレイピング結果を挿入し、Go UI が参照。

---

## 3. Stripe フロー（最新版）

1. `/user` 画面でプランを選択し「Upgrade」→ `/api/billing/checkout` を呼び出す。
2. Go サーバーが Stripe Checkout Session を作成し、成功/キャンセル URL を環境変数で決定。`metadata[team_id]` にチームIDを付与。
3. ユーザーが Checkout を完了すると、`/api/billing/webhook` へ以下イベントが届く：
   - `checkout.session.completed`
   - `customer.subscription.updated`
   - `customer.subscription.deleted`
4. Webhook Handler (`billing/webhook.go`) が `teams` テーブルの plan/billing_interval 等を更新。
5. `/user` UI の Danger Zone からアカウント削除 (`DELETE /api/teams/{id}`) も可能。

【環境変数】
```
STRIPE_SECRET_KEY=
STRIPE_WEBHOOK_SECRET=
STRIPE_PRICE_PERSONAL_MONTHLY=
STRIPE_PRICE_PERSONAL_YEARLY=
...（Pro/Org）
STRIPE_SUCCESS_URL=https://.../user?success=true
STRIPE_CANCEL_URL=https://.../user?canceled=true
STRIPE_RETURN_URL=https://.../user
```

詳しくは `docs/STRIPE.md` を参照。

---

## 4. UI / UX のポイント

- `/user` 画面は Alpine.js + Tailwind で実装。ローカルストレージに `akigura_team` を保存して疑似ログイン状態を再現。
- 監視条件作成モーダルでは、`municipalities` → `grounds` の依存を以下の API で取得：
  - `GET /api/municipalities`
  - `GET /api/grounds`
- 「グラウンドが選べない」場合は DB の `grounds` が空・または `municipality_id` が null の可能性が高い。Turso/SQLite を初期化した際には最新マイグレーションを適用し、`grounds` データを投入すること。

---

## 5. 開発フロー / CI

- すべてのコミット前に `make before-commit` を必須化（textlint + go test/build + worker build）。`AGENTS.md` にも明記し、`.git/hooks/pre-commit` でも自動実行。
- GitHub Actions: lint/text / go test / CodeQL / 依存分析 / CLAUDEレビュー などを自動実行。PR は main への auto-squash merge ルール。

---

## 6. 未完了タスクと今後の方針

Plan.md Phase 3.5 〜 Phase 4 参照。
- Webhook endpoint の本番配備（systemd + HTTPS）が未着手。
- `watch_conditions` UI で ground セレクトが空になる問題 → DB 初期データと API フィールドを再確認。
- E2E テスト（UI 操作を含む）や SendGrid 通知、Stripe 自動請求などを段階的に実装予定。

---

## 参照
- `docs/STRIPE.md`: Stripe 環境変数とフロー詳細
- `Plan.md`: 日次の進捗ログとタスク一覧
- `AGENTS.md`: 開発ルール（コミット前チェック必須）
