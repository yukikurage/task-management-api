# Task Management API

簡易タスク管理アプリケーションのバックエンド API

## 技術スタック

- **言語**: Go 1.21+
- **Web フレームワーク**: Gin
- **データベース**: MySQL 8.0
- **ORM**: GORM
- **認証**: Session + Cookie (gorilla/sessions)
- **コンテナ**: Docker + Docker Compose

## 機能

- ✅ ユーザー認証（サインアップ・ログイン・ログアウト）
- ✅ タスクの CRUD 操作（作成・取得・更新・削除）
- ✅ タスクへのユーザーアサイン機能
- ✅ 認可機能（作成者またはアサイン者のみアクセス可能）
- ✅ データベース設計とマイグレーション
- ✅ エラーハンドリング
- ✅ **AI タスク自動生成機能**（議事録・メモから自動でタスクを抽出）

## プロジェクト構造

```
task-management-api/
├── cmd/
│   └── server/
│       └── main.go              # エントリーポイント
├── internal/
│   ├── models/                  # データモデル
│   │   ├── user.go
│   │   ├── task.go
│   │   └── task_assignment.go
│   ├── handlers/                # HTTPハンドラ
│   │   ├── auth.go
│   │   └── task.go
│   ├── middleware/              # ミドルウェア
│   │   ├── auth.go
│   │   └── task_auth.go
│   ├── database/                # DB接続・初期化
│   │   └── database.go
│   └── config/                  # 設定管理
│       └── config.go
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

## データベース設計

### users テーブル

| カラム名      | 型        | 説明                       |
| ------------- | --------- | -------------------------- |
| id            | UINT      | 主キー                     |
| email         | VARCHAR   | メールアドレス（ユニーク） |
| password_hash | VARCHAR   | ハッシュ化されたパスワード |
| created_at    | TIMESTAMP | 作成日時                   |
| updated_at    | TIMESTAMP | 更新日時                   |
| deleted_at    | TIMESTAMP | 削除日時（ソフトデリート） |

### tasks テーブル

| カラム名    | 型        | 説明                            |
| ----------- | --------- | ------------------------------- |
| id          | UINT      | 主キー                          |
| title       | VARCHAR   | タスクのタイトル                |
| description | TEXT      | タスクの詳細                    |
| due_date    | TIMESTAMP | 期限日                          |
| creator_id  | UINT      | 作成者のユーザー ID（外部キー） |
| created_at  | TIMESTAMP | 作成日時                        |
| updated_at  | TIMESTAMP | 更新日時                        |
| deleted_at  | TIMESTAMP | 削除日時（ソフトデリート）      |

### task_assignments テーブル

| カラム名   | 型        | 説明                      |
| ---------- | --------- | ------------------------- |
| task_id    | UINT      | タスク ID（複合主キー）   |
| user_id    | UINT      | ユーザー ID（複合主キー） |
| created_at | TIMESTAMP | 作成日時                  |

## セットアップ・起動方法

### 前提条件

- Docker と Docker Compose がインストールされていること
- （ローカル実行の場合）Go 1.21+ と MySQL 8.0

### Docker Compose を使った起動（推奨）

1. **リポジトリをクローン**

```bash
git clone <repository-url>
cd task-management-api
```

2. **Docker Compose で起動**

```bash
make docker-up
# または
docker-compose up -d
```

3. **ログを確認**

```bash
make docker-logs
# または
docker-compose logs -f
```

4. **動作確認**

```bash
curl http://localhost:8080/health
```

5. **停止**

```bash
make docker-down
# または
docker-compose down
```

### ローカル環境での起動

1. **MySQL を起動**（既存の MySQL または Docker で起動）

```bash
docker run -d \
  --name mysql \
  -e MYSQL_ROOT_PASSWORD=rootpassword \
  -e MYSQL_DATABASE=task_management \
  -e MYSQL_USER=taskuser \
  -e MYSQL_PASSWORD=taskpassword \
  -p 3306:3306 \
  mysql:8.0
```

2. **環境変数を設定**

`.env.example` をコピーして `.env` ファイルを作成し、必要な値を設定してください。

```bash
cp .env.example .env
```

または、直接環境変数を設定：

```bash
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=taskuser
export DB_PASSWORD=taskpassword
export DB_NAME=task_management
export REDIS_HOST=localhost
export REDIS_PORT=6379
export SESSION_SECRET=your-secret-key-change-in-production
export GIN_MODE=debug
export OPENAI_API_KEY=sk-your-openai-api-key-here  # AI機能を使う場合のみ必須
```

3. **依存関係をインストール**

```bash
make deps
# または
go mod download
```

4. **アプリケーションを起動**

```bash
make run
# または
go run ./cmd/server/main.go
```

## API エンドポイント仕様

### ベース URL

```
http://localhost:8080/api
```

### 認証エンドポイント

#### 1. サインアップ（ユーザー登録）

```http
POST /api/auth/signup
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

**レスポンス例:**

```json
{
  "id": 1,
  "email": "user@example.com",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### 2. ログイン

```http
POST /api/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

**レスポンス例:**

```json
{
  "message": "Login successful",
  "user": {
    "id": 1,
    "email": "user@example.com"
  }
}
```

#### 3. ログアウト

```http
POST /api/auth/logout
```

**レスポンス例:**

```json
{
  "message": "Logged out successfully"
}
```

#### 4. 現在のユーザー情報取得

```http
GET /api/auth/me
```

**レスポンス例:**

```json
{
  "id": 1,
  "email": "user@example.com",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### タスクエンドポイント（要認証）

#### 5. タスク一覧取得

```http
GET /api/tasks?page=1&limit=10
```

**レスポンス例:**

```json
{
  "tasks": [
    {
      "id": 1,
      "title": "タスク1",
      "description": "説明文",
      "due_date": "2024-12-31T23:59:59Z",
      "creator_id": 1,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "limit": 10
}
```

#### 6. タスク作成

```http
POST /api/tasks
Content-Type: application/json

{
  "title": "新しいタスク",
  "description": "タスクの詳細",
  "due_date": "2024-12-31T23:59:59Z"
}
```

**レスポンス例:**

```json
{
  "id": 1,
  "title": "新しいタスク",
  "description": "タスクの詳細",
  "due_date": "2024-12-31T23:59:59Z",
  "creator_id": 1,
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### 7. タスク詳細取得

```http
GET /api/tasks/:id
```

**レスポンス例:**

```json
{
  "id": 1,
  "title": "タスク1",
  "description": "説明文",
  "due_date": "2024-12-31T23:59:59Z",
  "creator_id": 1,
  "creator": {
    "id": 1,
    "email": "user@example.com"
  },
  "assignments": [
    {
      "user_id": 2,
      "user": {
        "id": 2,
        "email": "user2@example.com"
      }
    }
  ]
}
```

#### 8. タスク更新

```http
PUT /api/tasks/:id
Content-Type: application/json

{
  "title": "更新されたタスク",
  "description": "更新された説明",
  "due_date": "2025-01-01T00:00:00Z"
}
```

**レスポンス例:**

```json
{
  "id": 1,
  "title": "更新されたタスク",
  "description": "更新された説明",
  "due_date": "2025-01-01T00:00:00Z",
  "updated_at": "2024-01-02T00:00:00Z"
}
```

#### 9. タスク削除

```http
DELETE /api/tasks/:id
```

**レスポンス例:**

```json
{
  "message": "Task deleted successfully"
}
```

#### 10. タスクにユーザーをアサイン

```http
POST /api/tasks/:id/assign
Content-Type: application/json

{
  "user_ids": [2, 3, 4]
}
```

**レスポンス例:**

```json
{
  "message": "Users assigned successfully",
  "assigned_users": [2, 3, 4]
}
```

#### 11. タスクからユーザーをアンアサイン

```http
POST /api/tasks/:id/unassign
Content-Type: application/json

{
  "user_ids": [2, 3]
}
```

**レスポンス例:**

```json
{
  "message": "Users unassigned successfully",
  "unassigned_users": [2, 3]
}
```

#### 12. AI によるタスク自動生成

```http
POST /api/tasks/generate
Content-Type: application/json

{
  "text": "明日までにレポートを作成。来週火曜日に打ち合わせ。月末までに請求書を送付する。",
  "organization_id": 1
}
```

**レスポンス例:**

```json
{
  "tasks": [
    {
      "title": "レポート作成",
      "description": "レポートを作成する",
      "due_date": "2025-10-28T23:59:59Z"
    },
    {
      "title": "打ち合わせ",
      "description": "打ち合わせに参加する",
      "due_date": "2025-10-29T10:00:00Z"
    },
    {
      "title": "請求書送付",
      "description": "請求書を送付する",
      "due_date": "2025-10-31T23:59:59Z"
    }
  ]
}
```

**注意:**
- `OPENAI_API_KEY` 環境変数が設定されていない場合、503 エラーが返されます
- 生成されたタスクは自動的に DB に保存されません（フロントエンドで確認後、`POST /api/tasks` で保存）
- 相対的な期限表現（「明日」「来週」など）は自動的に具体的な日時に変換されます

## エラーレスポンス

すべてのエラーは以下の形式で返されます：

```json
{
  "error": "エラーメッセージ"
}
```

### HTTP ステータスコード

- `200 OK`: 成功
- `201 Created`: リソース作成成功
- `400 Bad Request`: リクエストが不正
- `401 Unauthorized`: 認証が必要
- `403 Forbidden`: アクセス権限なし
- `404 Not Found`: リソースが見つからない
- `500 Internal Server Error`: サーバーエラー
- `501 Not Implemented`: 未実装（スタブエンドポイント）

## 環境変数

| 変数名          | デフォルト値                 | 説明                                                    |
| --------------- | ---------------------------- | ------------------------------------------------------- |
| DB_HOST         | localhost                    | MySQL ホスト                                            |
| DB_PORT         | 3306                         | MySQL ポート                                            |
| DB_USER         | taskuser                     | MySQL ユーザー名                                        |
| DB_PASSWORD     | taskpassword                 | MySQL パスワード                                        |
| DB_NAME         | task_management              | データベース名                                          |
| REDIS_HOST      | localhost                    | Redis ホスト                                            |
| REDIS_PORT      | 6379                         | Redis ポート                                            |
| SESSION_SECRET  | default-secret-key-change-me | セッション秘密鍵                                        |
| GIN_MODE        | debug                        | Gin モード（debug/release）                             |
| OPENAI_API_KEY  | (なし)                       | OpenAI API キー（AI タスク生成機能を使用する場合必須） |

## Makefile コマンド

```bash
make help              # ヘルプを表示
make build             # アプリケーションをビルド
make run               # ローカルで実行
make test              # テストを実行
make docker-up         # Dockerで起動
make docker-down       # Dockerを停止
make docker-logs       # Dockerログを表示
make deps              # 依存関係をダウンロード
make fmt               # コードをフォーマット
make vet               # go vetを実行
```

## 実装状況

### 完全実装済み

- ✅ プロジェクト構造
- ✅ データベースモデル（User, Task, TaskAssignment）
- ✅ データベース接続とマイグレーション
- ✅ セッション認証ミドルウェア
- ✅ タスクアクセス認可ミドルウェア
- ✅ ルーティング設定
- ✅ Docker 環境

### スタブ実装（要実装）

以下のエンドポイントはスタブ実装となっており、実際のロジックの実装が必要です。各ハンドラファイルに TODO コメントと実装ガイドがあります。

- ⚠️ POST /api/auth/signup（ユーザー登録）
- ⚠️ POST /api/auth/login（ログイン）
- ⚠️ GET /api/auth/me（現在のユーザー取得）
- ⚠️ GET /api/tasks（タスク一覧）
- ⚠️ POST /api/tasks（タスク作成）
- ⚠️ GET /api/tasks/:id（タスク詳細）
- ⚠️ PUT /api/tasks/:id（タスク更新）
- ⚠️ DELETE /api/tasks/:id（タスク削除）
- ⚠️ POST /api/tasks/:id/assign（ユーザーアサイン）
- ⚠️ POST /api/tasks/:id/unassign（ユーザーアンアサイン）

実装ガイドは各ハンドラファイル内に記載されています：

- `internal/handlers/auth.go`
- `internal/handlers/task.go`

## 開発のヒント

### パスワードハッシュ化

`golang.org/x/crypto/bcrypt` を使用してパスワードをハッシュ化してください。

```go
import "golang.org/x/crypto/bcrypt"

// ハッシュ化
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// 検証
err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
```

### セッション管理

セッションは `github.com/gin-contrib/sessions` で管理されています。

```go
session := sessions.Default(c)
session.Set("user_id", userID)
session.Save()

userID := session.Get("user_id")
```

### データベース操作

GORM を使用してデータベース操作を行います。

```go
import "github.com/yukikurage/task-management-api/internal/database"

db := database.GetDB()

// 作成
db.Create(&user)

// 取得
db.First(&user, id)

// 更新
db.Save(&user)

// 削除（ソフトデリート）
db.Delete(&user, id)
```

## テスト方法

### curl を使用した動作確認

```bash
# ヘルスチェック
curl http://localhost:8080/health

# サインアップ（実装後）
curl -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' \
  -c cookies.txt

# ログイン（実装後）
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' \
  -c cookies.txt

# タスク作成（実装後、セッション付き）
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"title":"My Task","description":"Task description","due_date":"2024-12-31T23:59:59Z"}'
```

### Postman でのテスト

1. Postman をインストール
2. Collection を作成
3. Cookie/Session 管理を有効化
4. 各エンドポイントをテスト

## トラブルシューティング

### データベース接続エラー

```bash
# MySQLコンテナのステータスを確認
docker-compose ps

# MySQLログを確認
docker-compose logs mysql

# MySQLに手動接続して確認
docker exec -it task-management-mysql mysql -u taskuser -ptaskpassword task_management
```

### アプリケーションログ確認

```bash
docker-compose logs app
```

### コンテナの再起動

```bash
docker-compose restart
```

### データベースのリセット

```bash
# ボリュームを含めて削除
docker-compose down -v

# 再起動
docker-compose up -d
```

## 今後の拡張案

- [ ] JWT 認証への移行
- [ ] ユニットテストの実装
- [ ] バリデーションの強化
- [ ] ページネーションの改善
- [ ] タスクのフィルタリング・検索機能
- [ ] タスクの優先度管理
- [ ] タスクのステータス管理（TODO, IN_PROGRESS, DONE）
- [ ] API レート制限
- [ ] HTTPS 対応
- [ ] CI/CD パイプライン

## ライセンス

MIT License

## 作成者

Your Name

---

**注意**: 本プロジェクトは課題提出用のテンプレートです。実際のビジネスロジックは各ハンドラファイル内の TODO コメントを参考に実装してください。
