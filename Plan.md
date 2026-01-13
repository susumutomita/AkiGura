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
