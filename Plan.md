# Development Plan

## プロジェクト概要

AkiGura は、グラウンドや体育館などの空き枠を自動で監視し、希望する条件に合う枠が出た際に通知するサービス。ターゲットは草野球やサッカーなどのチーム、学校や自治体の施設管理者など、コードに詳しくない層を含むため、運用しやすく安定した SaaS を提供することが目的。

### 目的とゴール

- ユーザーごとの監視条件の実現: 曜日・時間帯・期間を自由に組み合わせ、ユーザーが登録した条件にマッチする枠だけを通知する
- 拡張性と安定性: 利用者が増えても安定稼働できるよう、スクレイピング部分とユーザー管理・通知部分を分離して開発する
- OSS としての公開: コア部分を OSS (Apache License 2.0) で公開し、地域コミュニティや自治体のコントリビューションを受け入れやすくする
- 商用サポート: SaaS として提供し、インフラ運用やアップデート、課金処理を提供することでビジネスを成立させる

---

## アーキテクチャ

### コンポーネント

1. **Scraping Core** (既存リポジトリ由来)
   - ground-reservation リポジトリで実装されている各自治体向けのスクレイピングエンジンを再利用
   - HTML パーサーやリクエスト処理などのコアロジックを残しつつ、結果を構造化データ (Slot) として返すように整理

2. **Subscription API**
   - ユーザー管理、課金状態、監視条件を管理するバックエンド
   - Supabase や FastAPI 等で実装し、サブスクプラン (free/personal/pro/org) や請求書発行を担う
   - API は facility_id ごとにアクティブな購読者と条件を返す

3. **Worker** (Monitoring)
   - Cron やキュー駆動でスクレイピングコアを呼び出し、返ってきた空き枠一覧を各購読者の条件でフィルタリング
   - マッチした枠をメールや LINE 等で通知する
   - 通知先は Subscription API から取得した購読者ごとに個別に送る

4. **Web/UI**
   - ユーザーが自身の監視条件を登録・編集したり、プラン変更をしたりできる画面
   - 管理者用ダッシュボードを用意し、自治体向けに導入状況を可視化する

5. **Notification Service**
   - メール送信は SendGrid などのトランザクションメール API を利用
   - LINE、Slack など他チャネルへの拡張もできるように抽象化しておく

### データモデル

| モデル | 説明 |
|--------|------|
| Subscriber | ユーザーを表し、メールアドレスや LINE ID、ステータス (active/paused/cancelled) などを保持 |
| SubscriptionPlan | プラン名、月額料金、登録可能施設数・条件数などのプラン仕様を保持 |
| Subscription | Subscriber がどのプランに加入しているか、課金状態や契約期間を保持。SaaS 利用者であれば is_active=true。自治体/法人向けには請求書支払いを想定 |
| WatchCondition | 監視条件。facility_id、days_of_week (例: [5,6,7] = 金・土・日)、time_from・time_to、期間 (date_from・date_to) など。複数条件を登録可能 |
| Slot | スクレイピング結果を表す構造体。施設 ID、施設名、日付、開始時刻、終了時刻、コート名、原文テキストなどを保持。Worker では Slot と WatchCondition を突き合わせて通知対象を決める |

### プロジェクト構成 (初期案)

```text
akigura/
  README.md             … プロジェクト概要とインストール手順
  LICENSE               … Apache License 2.0
  docs/                 … 設計資料や仕様書
  scraping_core/        … 各自治体のスクレイピングロジック (submodule またはコピー)
  worker/               … 定期実行スクリプトと通知処理
  backend/              … Subscription API (FastAPI/Supabase Functions など)
  web/                  … ユーザー向け UI (将来的に追加)
  tests/                … ユニットテスト
```

### OSS ライセンスの選択

Apache License 2.0 を採用。MIT に比べ特許保護があり、他者がコードをそのまま再販することが難しくなる一方、商用利用や派生開発を許容するため、コミュニティへの貢献も期待できる。また GPL のように派生物へのソース公開義務がないため、SaaS 部分をクローズドなまま開発することが可能。

---

## 実行計画 (Exec Plans)

### Phase 1: スクレイピングコアの整理

**目的 (Objective)**:

- 既存 ground-reservation リポジトリをライブラリ化し、Slot 構造体を返す統一インタフェースを作成する

**制約 (Guardrails)**:

- 既存のスクレイピングロジックを壊さない
- テストカバレッジ 100％ を維持する

**タスク (TODOs)**:

- [ ] 既存 ground-reservation リポジトリをサブモジュールとして追加
- [ ] Slot データ構造を定義
- [ ] 各施設クラスに `search_slots()` メソッドを追加
- [ ] ユニットテストを作成

**検証手順 (Validation)**:

- `nr test` でテスト通過
- `nr typecheck` で型エラーなし

**未解決の質問 (Open Questions)**:

- サブモジュールとして利用するか、コードをコピーするか

**進捗ログ (Progress Log)**:

- (未着手)

---

### Phase 2: Subscription API の MVP

**目的 (Objective)**:

- ユーザー管理と監視条件を管理する API を構築する

**制約 (Guardrails)**:

- プロダクションレディな実装 (モック禁止)
- 認証・認可を実装する
- OWASP Top 10 対策を実施する

**タスク (TODOs)**:

- [ ] PostgreSQL テーブル設計 (Subscriber, SubscriptionPlan, Subscription, WatchCondition)
- [ ] マイグレーションファイル作成
- [ ] `GET /api/subscribers?facility_id=xxx` エンドポイント実装
- [ ] Stripe Billing と Webhook 設定
- [ ] 請求書発行機能 (自治体向け)
- [ ] 認証・認可の実装
- [ ] API ドキュメント作成

**検証手順 (Validation)**:

- `nr lint && nr typecheck && nr test && nr build`
- API エンドポイントの動作確認

**未解決の質問 (Open Questions)**:

- バックエンドフレームワークの選定 (FastAPI vs Supabase Functions)
- 認証方式の選定 (Supabase Auth vs Auth0)

**進捗ログ (Progress Log)**:

- (未着手)

---

### Phase 3: Worker の作成

**目的 (Objective)**:

- 定期実行でスクレイピングを行い、条件にマッチする枠を通知する

**制約 (Guardrails)**:

- エラーハンドリングとリトライ処理を実装する
- 通知履歴をログに残す

**タスク (TODOs)**:

- [ ] Worker 実行環境の選定 (Cloudflare Workers / AWS Lambda / Supabase Edge Functions)
- [ ] Cron 設定
- [ ] Subscription API から購読者と条件を取得する処理
- [ ] Slot と WatchCondition のマッチング処理
- [ ] 通知送信処理
- [ ] エラーハンドリングとリトライ処理
- [ ] 通知履歴のログ記録

**検証手順 (Validation)**:

- ローカルでの動作確認
- E2E テストの作成

**未解決の質問 (Open Questions)**:

- 実行頻度の設定 (5 分ごと? 15 分ごと?)

**進捗ログ (Progress Log)**:

- (未着手)

---

### Phase 4: 通知機能の実装

**目的 (Objective)**:

- メール送信機能を実装し、将来的に LINE/Slack にも対応できる抽象化を行う

**制約 (Guardrails)**:

- SendGrid API を使用する
- 通知チャネルを抽象化する

**タスク (TODOs)**:

- [ ] Mailer クラスの作成
- [ ] SendGrid API 連携
- [ ] メール文面のテンプレート作成
- [ ] 複数件をまとめて送るか枠ごとに送るかの検討と実装
- [ ] LINE/Slack 対応の抽象化

**検証手順 (Validation)**:

- テスト環境でのメール送信確認

**未解決の質問 (Open Questions)**:

- メール送信頻度の制限

**進捗ログ (Progress Log)**:

- (未着手)

---

### Phase 5: UI/管理画面の構築

**目的 (Objective)**:

- ユーザー向けの監視条件登録画面と管理者用ダッシュボードを構築する

**制約 (Guardrails)**:

- モバイルフレンドリーなデザイン
- アクセシビリティ対応

**タスク (TODOs)**:

- [ ] フレームワーク選定 (Next.js / Vue.js)
- [ ] ユーザー登録・ログイン画面
- [ ] 監視条件登録・編集画面
- [ ] プラン変更・解約画面
- [ ] 管理者用ダッシュボード
- [ ] 通知ログ閲覧画面

**検証手順 (Validation)**:

- E2E テスト
- アクセシビリティチェック

**未解決の質問 (Open Questions)**:

- デザインシステムの選定

**進捗ログ (Progress Log)**:

- (未着手)

---

### Phase 6: OSS 公開準備

**目的 (Objective)**:

- コア部分を Apache License 2.0 で公開し、コミュニティからのコントリビューションを受け入れる準備をする

**制約 (Guardrails)**:

- SaaS 部分は別リポジトリでクローズドに管理する

**タスク (TODOs)**:

- [ ] README.md 作成
- [ ] CONTRIBUTING.md 作成
- [ ] LICENSE ファイル追加
- [ ] GitHub Actions で CI 設定
- [ ] Issue テンプレート作成
- [ ] PR テンプレート作成

**検証手順 (Validation)**:

- ドキュメントのレビュー
- CI の動作確認

**未解決の質問 (Open Questions)**:

- なし

**進捗ログ (Progress Log)**:

- (未着手)

---

## 振り返り (Retrospective)

(今後問題が発生した際に記録する)

### Phase 2.5: Turso への移行 - 2026-01-17

**目的 (Objective)**:
- ローカル SQLite を Turso (libSQL) に移行し、本番想定のクラウドデータベースで稼働させる

**制約 (Guardrails)**:
- 既存機能を停止させずにシームレスに移行する
- インフラ秘密情報は `.env` などに格納し、リポジトリには含めない
- `Plan.md` のルールと `CLAUDE.md` の実装原則を順守する

**タスク (TODOs)**:
- [ ] Turso CLI をインストールし、アカウントへログインする
- [ ] 新しい Turso データベースを作成し、地域名を設定する
- [ ] スキーママイグレーションを Turso に適用する
- [ ] アプリの接続設定(`TURSO_DATABASE_URL` / `TURSO_AUTH_TOKEN`)を`.env`と systemd に追加する
- [ ] コード側で libsql ドライバ接続を本番設定に切り替える
- [ ] マイグレーションとアプリ起動テストを行う
- [ ] Plan.md に進捗と検証結果を記録する

**検証手順 (Validation)**:
- `go test ./...`と`go build ./cmd/srv`が成功する
- Turso 上の`teams`テーブルに実データが存在することを確認する
- `/api/dashboard`など主要エンドポイントが動作し、データが Turso から返ることを確認する

**未解決の質問 (Open Questions)**:
- Turso のレプリカ構成は必要か(後続で検討)

**進捗ログ (Progress Log)**:
- [2026-01-17 07:05]Turso への移行準備を開始
- [2026-01-17 07:08]npm が入っておらず Turso CLI インストールに失敗。別手段で CLI を導入する方針に変更
- [2026-01-17 07:10]Turso 公式インストーラで CLI 導入に成功
- [2026-01-17 07:11]CLI パスを確認し`/home/exedev/.turso/turso`が利用可能なことを確認
- [2026-01-17 07:12]Turso CLI で auth login 済み
- [2026-01-17 07:13]既存の Turso DB 一覧を確認(akigura 等)
- [2026-01-17 07:14]既存 DB`akigura`のメタ情報を確認(73MB / us-west-2)
- [2026-01-17 07:15]新規 DB`akigura-app`を作成
- [2026-01-17 07:16]`akigura-app`の URL を確認(libsql://akigura-app-susumutomita.aws-us-west-2.turso.io)
- [2026-01-17 07:17]Turso auth token を生成(安全のため`.env`にのみ保存予定)
- [2026-01-17 07:19]`.env`を新規 Turso URL/Token に更新
- [2026-01-17 07:20]`akigura-app` DB の初期テーブルが空であることを確認
- [2026-01-17 07:23]`sqlite3 db.sqlite3 .dump | turso db shell`で Turso にスキーマとデータを移行中
- [2026-01-17 07:24]Turso への dump import が完了
- [2026-01-17 07:26]`go test ./...`を実行し全て通過
- [2026-01-17 07:27]`go build -o akigura-srv ./cmd/srv`成功
- [2026-01-17 07:28]Turso 接続状態で`./akigura-srv`を起動し`/api/dashboard`が正常レスポンスを返すことを確認
- [2026-01-17 07:29]ローカルテスト終了のためサーバープロセスを停止

**振り返り (Retrospective)**:
- 問題: npm が入っておらず Turso CLI インストールに失敗した
- 根本原因: グローバル npm が環境に未導入だった
- 予防策: CLI インストール時には公式スクリプトやバイナリを直接使用する

- [2026-01-17 07:32]Turso 上で`slots`件数(1480)など主要テーブルを確認

### Phase 3.5: Stripe 課金統合ブラッシュアップ - 2026-01-17

**目的 (Objective)**:
- Stripe Checkout / Billing Portal / Webhook を本番レベルで動かし、プラン変更が完結する体験を提供する

**制約 (Guardrails)**:
- 実際の Stripe API を使用し、モックやスタブを置き換える
- 環境変数 (STRIPE_SECRET_KEY など) は `.env` や systemd で安全に設定する
- UI からの操作で Checkout → Webhook → DB 更新まで全て確認する

**タスク (TODOs)**:
- [ ] Stripe API キーと Price ID を取得し `.env` に設定する
- [ ] `billing.Plans` の Price ID を最新のものに更新する
- [ ] `StripeClient.VerifyWebhookSignature` を stripe-go で本番仕様にする
- [ ] `HandleCreateCheckout` で SuccessURL / CancelURL を環境変数ベースに切り替える
- [ ] Webhook エンドポイントを本番 URL で受けられるよう systemd/ngrok 設定を適用する
- [ ] `/user` UI で Checkout → Stripe へ遷移できることをブラウザ確認する
- [ ] Webhook 受信後に teams.plan などが更新されることを DB で確認する
- [ ] Plan.md にログと振り返りを記録する

**検証手順 (Validation)**:
- `go test ./...` と `go build ./cmd/srv` を通す
- テスト用チームで Checkout 〜 Webhook まで通し、`teams` テーブルの plan/billing_interval/current_period_end を確認する
- `/api/dashboard` で課金ステータスが反映されることを確認する

**未解決の質問 (Open Questions)**:
- 日本円での税込表示・消費税処理が必要か

**進捗ログ (Progress Log)**:
- [2026-01-17 07:55]Stripe 統合作業の計画を追加
