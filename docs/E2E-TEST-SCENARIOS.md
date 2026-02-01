# AkiGura E2E テストシナリオ

本番環境の動作確認用エンドツーエンドテストシナリオ集

---

## 概要

AkiGura の主要機能をエンドツーエンドで検証するためのテストシナリオです。
新規デプロイ後や機能追加後に実行し、システム全体が正常に動作することを確認します。

---

## テスト環境

| 項目 | 値 |
|------|----|
| Control Plane URL | https://akigura.exe.xyz:8000 |
| 管理画面 | https://akigura.exe.xyz:8000/admin/ |
| ユーザー画面 | https://akigura.exe.xyz:8000/user |

---

## シナリオ一覧

### 1. ユーザー登録フロー

**目的**: 新規ユーザーがマジックリンクでアカウントを作成できることを確認

**前提条件**: SMTP 設定済み（または debug_link でテスト）

**手順**:

1. `/user` にアクセス
2. 「ログイン / 新規登録」セクションでメールアドレスを入力
3. 「マジックリンクを送信」をクリック
4. メールを確認し、リンクをクリック
5. ユーザーダッシュボードにリダイレクトされることを確認

**期待結果**:
- [ ] マジックリンクメールが届く
- [ ] リンククリック後、ダッシュボードに遷移
- [ ] localStorage に `akigura_team` が保存される
- [ ] DB の `teams` テーブルに新規レコードが作成される

**確認コマンド**:
```bash
sqlite3 /home/exedev/AkiGura/control-plane/db.sqlite3 "SELECT id, name, email, plan FROM teams ORDER BY created_at DESC LIMIT 5;"
```

---

### 2. 監視条件（Watch Condition）登録フロー

**目的**: ユーザーが特定の施設・時間帯の監視条件を設定できることを確認

**前提条件**: ユーザーがログイン済み

**手順**:

1. ユーザーダッシュボードにアクセス
2. 「監視条件を追加」をクリック
3. 以下を設定:
   - 自治体を選択（例: 平塚市）
   - 施設を選択（例: 総合公園テニスコート）
   - 曜日を選択（例: 土曜、日曜）
   - 時間帯を設定（例: 09:00〜18:00）
4. 「保存」をクリック

**期待結果**:
- [ ] 監視条件が一覧に表示される
- [ ] DB の `watch_conditions` テーブルに新規レコードが作成される
- [ ] 条件の有効/無効を切り替えられる

**確認コマンド**:
```bash
sqlite3 /home/exedev/AkiGura/control-plane/db.sqlite3 "SELECT wc.id, t.email, wc.days_of_week, wc.time_from, wc.time_to, wc.enabled FROM watch_conditions wc JOIN teams t ON wc.team_id = t.id ORDER BY wc.created_at DESC LIMIT 5;"
```

---

### 3. スクレイピング実行フロー

**目的**: Worker が施設の空き状況を正しくスクレイピングできることを確認

**前提条件**: Worker サービスが稼働中

**手順**:

1. 管理画面にアクセス（`/admin/`）
2. 「スクレイピング実行」ボタンをクリック
3. またはコマンドラインで実行:
   ```bash
   /home/exedev/AkiGura/worker/akigura-worker -once -db /home/exedev/AkiGura/control-plane/db.sqlite3 -scraper /home/exedev/AkiGura/worker/scraper_wrapper.py
   ```

**期待結果**:
- [ ] `scrape_jobs` テーブルにジョブが記録される
- [ ] `slots` テーブルに空き枠データが追加される
- [ ] エラーがあれば `scrape_diagnostics` に記録される

**確認コマンド**:
```bash
# ジョブ履歴
sqlite3 /home/exedev/AkiGura/control-plane/db.sqlite3 "SELECT id, municipality_id, status, slots_found, started_at FROM scrape_jobs ORDER BY started_at DESC LIMIT 10;"

# 空き枠データ
sqlite3 /home/exedev/AkiGura/control-plane/db.sqlite3 "SELECT id, slot_date, time_from, time_to, court_name, municipality FROM slots ORDER BY scraped_at DESC LIMIT 10;"
```

---

### 4. 空き枠マッチング＆通知フロー

**目的**: 監視条件に一致する空き枠が見つかった場合に通知が送られることを確認

**前提条件**:
- ユーザーが監視条件を登録済み
- スクレイピングが実行済みで空き枠データがある
- SMTP または SendGrid が設定済み

**手順**:

1. Worker の通知処理を実行（自動で 1 分ごとに実行）
2. またはログで確認:
   ```bash
   journalctl -u akigura-worker -f
   ```

**期待結果**:
- [ ] 条件にマッチする空き枠がメールで通知される
- [ ] `notifications` テーブルにレコードが作成される
- [ ] 同じ空き枠は重複通知されない

**確認コマンド**:
```bash
sqlite3 /home/exedev/AkiGura/control-plane/db.sqlite3 "SELECT n.id, t.email, n.channel, n.status, n.sent_at FROM notifications n JOIN teams t ON n.team_id = t.id ORDER BY n.created_at DESC LIMIT 10;"
```

---

### 5. カレンダー表示フロー

**目的**: ユーザーが空き状況をカレンダービューで確認できることを確認

**前提条件**: スクレイピング済みの空き枠データがある

**手順**:

1. ユーザーダッシュボードにアクセス
2. カレンダータブを選択
3. 自治体・施設でフィルタリング
4. 日付をクリックして詳細を確認

**期待結果**:
- [ ] 空き枠がカレンダー上に表示される
- [ ] フィルタリングが機能する
- [ ] 予約サイトへのリンクが正しい

---

### 6. 管理画面操作フロー

**目的**: 管理者がシステム全体を管理できることを確認

**前提条件**: 管理者認証（ADMIN_USER/ADMIN_PASS が設定されている場合）

**手順**:

1. `/admin/` にアクセス
2. 各メニューを確認:
   - ダッシュボード: 統計情報
   - チーム: ユーザー一覧
   - 施設: 登録施設一覧
   - 空き状況: スロット一覧
   - ワーカー: ジョブ履歴
   - AI: サポートチケット

**期待結果**:
- [ ] 各画面が正しく表示される
- [ ] CRUD 操作が機能する
- [ ] スクレイピング手動実行が機能する

---

## 環境変数チェックリスト

本番環境で設定すべき環境変数:

### 必須

| 変数 | 説明 | 設定状況 |
|------|------|----------|
| `DATABASE_PATH` | SQLite パス | ✅ デフォルト使用 |
| `BASE_URL` | 公開 URL（マジックリンク用） | ⚠️ 要設定 |

### 認証（いずれか必須）

| 変数 | 説明 | 設定状況 |
|------|------|----------|
| `SMTP_USER` | Gmail アドレス | ⚠️ 要設定 |
| `SMTP_PASSWORD` | Gmail アプリパスワード | ⚠️ 要設定 |
| `SENDGRID_API_KEY` | SendGrid API キー | ⚠️ または SMTP |

### 管理画面保護

| 変数 | 説明 | 設定状況 |
|------|------|----------|
| `ADMIN_USER` | Basic 認証ユーザー名 | ⚠️ 要設定 |
| `ADMIN_PASS` | Basic 認証パスワード | ⚠️ 要設定 |

### オプション

| 変数 | 説明 |
|------|------|
| `GOOGLE_CLIENT_ID` | Google OAuth クライアント ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth シークレット |
| `OPENAI_API_KEY` | AI チャット用 |
| `ANTHROPIC_API_KEY` | AI チャット用（Claude） |
| `LINE_CHANNEL_TOKEN` | LINE 通知用 |
| `SLACK_WEBHOOK_URL` | Slack 通知用 |

---

## トラブルシューティング

### サービスが起動しない

```bash
# ログ確認
journalctl -u akigura -n 50
journalctl -u akigura-worker -n 50

# サービス状態
sudo systemctl status akigura akigura-worker
```

### マジックリンクが届かない

1. SMTP 設定を確認
2. ログで debug_link を確認（SMTP 未設定時）:
   ```bash
   journalctl -u akigura | grep "Magic link"
   ```

### スクレイピングが失敗する

```bash
# Python 環境確認
/home/exedev/AkiGura/worker/.venv/bin/python -c "import bs4; print('OK')"

# 手動実行でエラー確認
cd /home/exedev/AkiGura/worker
./.venv/bin/python scraper_wrapper.py hiratsuka
```

---

## 定期チェック項目

毎日確認すべき項目:

- [ ] サービス稼働状態: `systemctl status akigura akigura-worker`
- [ ] 最新のスクレイピングジョブ: 成功/失敗
- [ ] 未送信の通知がないか
- [ ] ディスク使用量: `df -h`

---

## 更新履歴

| 日付 | 変更内容 |
|------|----------|
| 2026-02-01 | 初版作成 |
