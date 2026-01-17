# Stripe 課金設計書

## 目的

- AkiGura ユーザーが自分でプランを選択し、Stripe Checkout で決済できるようにする
- 決済後は Webhook で teams テーブルの `plan` や `billing_interval`、`current_period_end` を自動更新する
- Billing Portal からユーザー自身がプラン変更や解約を行えるようにする

## 主要コンポーネント

| 項目 | 内容 |
|------|------|
| Checkout | `/api/billing/checkout` → Stripe Checkout → success/cancel URL で `/user` に戻る |
| Billing Portal | `/api/billing/portal` → Stripe Billing Portal を開き、カード変更や解約をセルフサービス化 |
| Webhook | `/api/billing/webhook`。Stripe から `checkout.session.completed` などを受信し DB 更新 |
| Plans | `billing.Plans` でプラン ID と Price ID を管理。環境変数から注入 |

## 必要な環境変数

| 変数 | 用途 |
|------|------|
| `STRIPE_SECRET_KEY` | Stripe ダッシュボードのテスト/本番 Secret Key |
| `STRIPE_WEBHOOK_SECRET` | Webhook エンドポイント作成時に発行される Signing secret |
| `STRIPE_PRICE_PERSONAL_MONTHLY` | Product「Personal」月額の Price ID |
| `STRIPE_PRICE_PERSONAL_YEARLY` | Personal 年額 Price ID |
| `STRIPE_PRICE_PRO_MONTHLY` | Pro 月額 Price ID |
| `STRIPE_PRICE_PRO_YEARLY` | Pro 年額 Price ID |
| `STRIPE_PRICE_ORG_MONTHLY` | Organization 月額 Price ID |
| `STRIPE_PRICE_ORG_YEARLY` | Organization 年額 Price ID |
| `STRIPE_SUCCESS_URL` | Checkout 成功時に戻る URL（例: `https://compass-patch.exe.xyz:8001/user?success=true`） |
| `STRIPE_CANCEL_URL` | Checkout キャンセル時の URL |

`.env` 例:

```
STRIPE_SECRET_KEY=sk_test_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx
STRIPE_PRICE_PERSONAL_MONTHLY=price_...
STRIPE_PRICE_PERSONAL_YEARLY=price_...
STRIPE_PRICE_PRO_MONTHLY=price_...
STRIPE_PRICE_PRO_YEARLY=price_...
STRIPE_PRICE_ORG_MONTHLY=price_...
STRIPE_PRICE_ORG_YEARLY=price_...
STRIPE_SUCCESS_URL=https://compass-patch.exe.xyz:8001/user?success=true
STRIPE_CANCEL_URL=https://compass-patch.exe.xyz:8001/user?canceled=true
```

## フロー概要

### 1. セルフサインアップ + Checkout

```
ユーザー → /user でプラン選択
      → /api/billing/checkout
      → Stripe Checkout (カード入力)
      → success_url で /user に戻る
```

### 2. Webhook 同期

```
Stripe → /api/billing/webhook (checkout.session.completed)
      → Subscription ID / Customer ID を受信
      → billing.WebhookHandler が teams テーブルを更新
      → plan / billing_interval / current_period_end を保存
```

受信イベントと処理:

| イベント | 処理内容 |
|----------|----------|
| `checkout.session.completed` | `metadata.team_id` からチームを特定し、`stripe_subscription_id` を保存 |
| `customer.subscription.updated` | 価格 ID を `billing.Plans` と突き合わせ、plan/interval を更新 |
| `customer.subscription.deleted` | `CancelTeamSubscription` で plan を free に戻す |
| `invoice.paid` | ログのみ（将来の領収書メール用） |
| `invoice.payment_failed` | ログのみ（将来のリトライ通知用） |

### 3. Billing Portal

```
ユーザー → 「請求設定を開く」
      → /api/billing/portal
      → Stripe Billing Portal (カード変更、解約)
      → return_url で /user に戻る
```

## Stripe ダッシュボードでの準備手順

1. アカウント作成後、「製品」→「+ 商品」で Personal / Pro / Organization を追加
2. 各商品で「定期課金」「JPY」「毎月 / 毎年」の価格を作成し Price ID を控える
3. 「開発者」→「Webhook」→「+ エンドポイント」で `/api/billing/webhook` を登録
   - テスト: `stripe listen --forward-to localhost:8001/api/billing/webhook`
   - 本番: `https://compass-patch.exe.xyz:8001/api/billing/webhook`
4. 受信イベントに `checkout.session.completed` と `customer.subscription.*` を追加
5. Signing secret を `.env` の `STRIPE_WEBHOOK_SECRET` に設定

## データモデル連携

teams テーブルの関連カラム:

| カラム | 用途 |
|--------|------|
| `plan` | free/personal/pro/org |
| `billing_interval` | monthly/yearly |
| `stripe_customer_id` | Stripe Customer ID。Checkout 前に生成 |
| `stripe_subscription_id` | Webhook で更新 |
| `current_period_end` | Stripe Subscription `current_period_end` を Unix 秒から変換 |

## テスト手順

1. `source .env` で Stripe 変数を読み込む
2. `go test ./...` / `go build ./cmd/srv` を通す
3. サーバーを `./akigura-srv -listen :8001` で起動
4. Stripe CLI で Webhook を転送
   ```
   stripe listen --forward-to localhost:8001/api/billing/webhook
   ```
5. `/user` → プラン選択 → Checkout でテストカード `4242 4242 4242 4242` を使用
6. 決済後、`sqlite3 control-plane/db.sqlite3 "SELECT plan,billing_interval,current_period_end FROM teams"` で反映を確認

## 今後の課題

- 税込/税抜表示や請求書 PDF の添付方法を決定する
- 個人利用と自治体請求（請求書払い）を切り替える機能
- Stripe Customer Portal のドメイン固有化（`billing.portal` の UI ブランディング）
- 失敗決済時の自動メール送信
