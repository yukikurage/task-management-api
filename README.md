# Task Management API

## セットアップ・起動

1. `.env.example` をコピーして環境変数を設定する（OPENAI_API_KEY などを設定）
   `cp .env.example .env`
2. `make docker-up` で起動する

ベース URL: `http://localhost:8080/api`

## エンドポイント一覧

詳細な仕様は [OpenAPI ドキュメント](./openapi.yaml) を参照してください。

### 認証

- `POST /auth/signup` — 新しいユーザーを登録し、セッションを開始する
- `POST /auth/login` — ユーザー名とパスワードでログインする
- `POST /auth/logout` — 現在のセッションを終了する
- `GET /auth/me` — 現在ログイン中のユーザー情報を取得する

### タスク

- `GET /tasks` — フィルタやページネーション付きでタスク一覧を取得する
- `POST /tasks` — タスクを作成し、作成者を自動でアサインする
- `GET /tasks/:id` — 単一タスクの詳細を取得する
- `PUT /tasks/:id` — タスクの内容や期限を更新する
- `DELETE /tasks/:id` — タスクを削除する（作成者のみ）
- `POST /tasks/:id/assign` — タスクにユーザーを追加でアサインする
- `POST /tasks/:id/unassign` — タスクからユーザーのアサインを解除する
- `POST /tasks/:id/toggle-status` — TODO/DONE のステータスを切り替える
- `POST /tasks/generate` — AI でタスク候補を生成する（保存はフロントエンド側で実行する必要がある）
