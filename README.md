# Task Management API エンドポイント一覧

ベース URL: `http://localhost:8080/api`

## 認証

- `POST /auth/signup` — 新しいユーザーを登録し、セッションを開始する
- `POST /auth/login` — メールアドレスとパスワードでログインする
- `POST /auth/logout` — 現在のセッションを終了する
- `GET /auth/me` — 現在ログイン中のユーザー情報を取得する

## タスク

- `GET /tasks` — フィルタやページネーション付きでタスク一覧を取得する
- `POST /tasks` — タスクを作成
- `GET /tasks/:id` — 単一タスクの詳細を取得する
- `PUT /tasks/:id` — タスクの内容や期限を更新する
- `DELETE /tasks/:id` — タスクを削除する（作成者のみ）
- `POST /tasks/:id/assign` — タスクにユーザーを追加でアサインする（作成者のみ）
- `POST /tasks/:id/unassign` — タスクからユーザーのアサインを解除する
- `POST /tasks/:id/toggle-status` — TODO/DONE のステータスを切り替える
- `POST /tasks/generate` — AI でタスク候補を生成する（保存はフロントエンド側で実行）
