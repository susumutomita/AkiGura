# Refer to [CLAUDE.md](./CLAUDE.md)

## 開発ルール（重要）

- すべてのコミット前に必ず `make before-commit` を実行し、textlint / go test / go build / worker build をローカルでグリーンにすること。
- textlint の指摘は Plan.md や README などの表記ゆれを手動または `textlint --fix` で解消してから push すること。
- データベースは「本番: Turso(libSQL)」「ローカル開発: SQLite (`control-plane/db.sqlite3`)」の 2 層構成で運用し、マイグレーションと初期データ投入で整合性を保つこと。
- 上記が守れない場合は push せず、修正してから再実行すること。
