# Spotify NowPlaying Sharing App

このアプリケーションは、Spotifyで現在再生中の曲やエピソードを、任意のMisskeyサーバーもしくはTwitter(自称:𝕏)に共有するためのWebアプリケーションです。

## 概要

SpotifyのAPIを使用して、現在再生中の曲やエピソードの情報を取得します。  
そして、簡単に共有できるようにシェアリンクを生成し、MisskeyもしくはTwitter(自称:𝕏)にリダイレクトさせます。  

シェアリンクには、以下のようなフォーマットのテキストが含まれており、曲名, アーティスト名, #NowPlayingを自ら入力する必要はありません。

```
あとがき / 来栖夏芽
#NowPlaying #PsrPlaying
https://open.spotify.com/track/5WehEFiES0ebVqgXpYQ8Fi
```

## 機能

### リダイレクト共有（従来機能）

- `/note` - Misskey向けシェアリンクへリダイレクト
- `/tweet` - Twitter(𝕏)向けシェアリンクへリダイレクト

### API直接投稿（新機能）

ダッシュボードでMisskey/Twitterアカウントを連携し、APIトークンを使って直接投稿が可能です。

- **MiAuth** - Misskeyアカウント連携
- **Twitter OAuth 2.0 PKCE** - Twitterアカウント連携
- **ヘッダートークン認証** - オプションでAPIにセキュリティ層を追加

## デモ

Spotifyで音楽を再生した状態で、[ここ](https://spn.soli0222.com/tweet)にアクセスしてみてください。  
曲名、アーティスト名が入った状態で、簡単に共有することができます。

## プロジェクト構成

```
spotify-nowplaying/
├── .github/
│   └── workflows/
│       └── test.yaml        # CI/CDワークフロー
├── cmd/
│   └── server/
│       └── main.go          # アプリケーションエントリーポイント
├── frontend/                # React フロントエンド
│   ├── src/
│   │   ├── pages/           # ページコンポーネント
│   │   │   ├── Login.tsx    # ログインページ
│   │   │   └── Dashboard.tsx# ダッシュボード
│   │   ├── test/            # テストユーティリティ
│   │   │   ├── setup.ts     # テストセットアップ
│   │   │   └── test-utils.tsx # カスタムrender
│   │   ├── api.ts           # API クライアント
│   │   ├── api.test.ts      # APIテスト
│   │   ├── App.tsx          # ルーティング
│   │   ├── App.test.tsx     # ルーティングテスト
│   │   └── main.tsx         # エントリーポイント
│   ├── package.json
│   └── vite.config.ts
├── internal/
│   ├── auth/                # 認証ユーティリティ
│   │   ├── auth.go          # JWT, PKCE, トークンハッシュ
│   │   └── auth_test.go     # 認証テスト
│   ├── handler/             # HTTPハンドラー
│   │   ├── handler.go       # 共通ハンドラーロジック
│   │   ├── handler_test.go  # ハンドラーテスト
│   │   ├── note.go          # Misskey リダイレクト
│   │   ├── tweet.go         # Twitter リダイレクト
│   │   ├── status.go        # ステータスエンドポイント
│   │   ├── spotify_auth.go  # Spotify OAuth（フロントエンド用）
│   │   ├── miauth.go        # MiAuth フロー
│   │   ├── twitter_auth.go  # Twitter OAuth 2.0 PKCE
│   │   ├── api_post.go      # API 直接投稿
│   │   └── settings.go      # ユーザー設定
│   ├── metrics/             # Prometheusメトリクス
│   │   ├── metrics.go
│   │   ├── metrics_test.go  # メトリクステスト
│   │   └── middleware.go
│   ├── spotify/             # Spotify API関連
│   │   ├── auth.go          # OAuth認証（型定義）
│   │   ├── auth_test.go     # 認証テスト
│   │   ├── client.go        # APIクライアント
│   │   ├── client_test.go   # クライアントテスト
│   │   ├── player.go        # 再生情報取得・シェアURL生成
│   │   └── player_test.go   # プレイヤーテスト
│   └── store/               # データベース
│       └── store.go         # PostgreSQL 操作
├── migrations/              # DBマイグレーション
│   ├── 001_create_users.up.sql
│   └── 001_create_users.down.sql
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── README.md
```

## セットアップ

アプリケーションをローカルで実行するためには、以下の手順に従ってください。

### 1. リポジトリをクローン

```bash
git clone https://github.com/Soli0222/spotify-now-playing.git
cd spotify-now-playing
```

### 2. 環境変数を設定

```bash
cp .env.example .env
```

```.env
# 基本設定
PORT=8080                        # 起動するポート番号
METRICS_PORT=9090                # メトリクスポート番号（デフォルト: 9090）
SERVER_URI=example.tld           # リダイレクト先サーバーのドメイン
BASE_URL=https://example.tld     # アプリケーションのベースURL

# Spotify API
SPOTIFY_CLIENT_ID=xxxxxxxx       # Spotify クライアントID
SPOTIFY_CLIENT_SECRET=xxxxx      # Spotify クライアントシークレット

# データベース（API直接投稿機能を使用する場合）
DATABASE_URL=postgres://user:password@localhost:5432/spotify_nowplaying?sslmode=disable

# JWT（API直接投稿機能を使用する場合）
JWT_SECRET=your-secret-key       # JWT署名用シークレット

# Twitter API（Twitter連携を使用する場合）
TWITTER_CLIENT_ID=xxxxxxxx       # Twitter クライアントID
TWITTER_CLIENT_SECRET=xxxxx      # Twitter クライアントシークレット
```

### 3. OAuth設定

#### Spotify Developer Dashboard

以下のCallback URIを登録してください：

- `{BASE_URL}/note/callback`
- `{BASE_URL}/tweet/callback`
- `{BASE_URL}/api/auth/spotify/callback`

#### X (Twitter) Developer Portal

以下のCallback URLを登録してください：

- `{BASE_URL}/api/twitter/callback`

### 4. フロントエンドのビルド（開発時）

```bash
cd frontend
pnpm install
pnpm build
cd ..
```

### 5. データベースのセットアップ（API直接投稿機能を使用する場合）

PostgreSQLを起動し、マイグレーションを実行してください。

Docker Composeを使用する場合は自動的にセットアップされます。

## 実行

### 開発環境

```bash
# バックエンドのみ（リダイレクト機能のみ）
go run ./cmd/server

# フロントエンド開発サーバー
cd frontend
pnpm dev
```

### Docker Compose（推奨）

```bash
docker compose up -d
```

これにより以下のサービスが起動します：
- アプリケーション（ポート8080）
- PostgreSQL（ポート5432）
- メトリクス（ポート9090）

## ビルド

### ローカルビルド

```bash
# フロントエンド
cd frontend
pnpm install
pnpm build
cd ..

# バックエンド
go build -o server ./cmd/server
./server
```

### マルチアーキテクチャDockerイメージ

```bash
docker buildx build --platform linux/amd64,linux/arm64 -t your-registry/spotify-nowplaying:latest --push .
```

## テスト

### バックエンドテスト

```bash
# すべてのテストを実行
go test -v ./...

# カバレッジレポート付きで実行
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### フロントエンドテスト

```bash
cd frontend

# テストを実行
pnpm test:run

# ウォッチモードでテストを実行
pnpm test

# カバレッジレポート付きで実行
pnpm test:coverage
```

### Lint

```bash
# バックエンド
go vet ./...
golangci-lint run

# フロントエンド
cd frontend
pnpm lint
pnpm tsc -b  # 型チェック
```

## CI/CD

GitHub Actionsを使用して、すべてのプッシュとプルリクエストで自動テストが実行されます。

### ワークフロー

| ジョブ | 説明 |
|---|---|
| `test` | Goのユニットテスト |
| `lint` | golangci-lintによる静的解析 |
| `frontend-test` | Vitestによるフロントエンドテスト |
| `frontend-lint` | ESLint + TypeScript型チェック |
| `build` | バックエンド・フロントエンドのビルド確認 |

## API

### リダイレクトエンドポイント

| エンドポイント | 説明 |
|---|---|
| `GET /note` | Spotify認証 → Misskeyシェアへリダイレクト |
| `GET /tweet` | Spotify認証 → Twitterシェアへリダイレクト |

### 直接投稿API

| エンドポイント | 説明 |
|---|---|
| `GET /api/post/:token` | APIトークンを使って直接投稿 |

#### クエリパラメータ

| パラメータ | 値 | 説明 |
|---|---|---|
| `target` | `misskey`, `twitter`, `both` | 投稿先（デフォルト: `both`） |

#### ヘッダートークン認証（オプション）

ダッシュボードでヘッダートークンを設定した場合、リクエストヘッダーに含める必要があります：

```
X-API-Token: your-header-token
```

#### 使用例

```bash
# Misskeyのみに投稿
curl "https://example.tld/api/post/your-api-token?target=misskey"

# 両方に投稿（ヘッダートークン認証あり）
curl -H "X-API-Token: your-header-token" "https://example.tld/api/post/your-api-token"
```

## メトリクス

Prometheusメトリクスは別ポート（デフォルト: 9090）の `/metrics` エンドポイントで公開されます。

### 利用可能なメトリクス

| メトリクス名 | タイプ | ラベル | 説明 |
|---|---|---|---|
| `http_requests_total` | Counter | method, path, status | HTTPリクエスト総数 |
| `http_request_duration_seconds` | Histogram | method, path | HTTPリクエスト処理時間 |
| `spotify_api_requests_total` | Counter | endpoint, status | Spotify APIリクエスト総数 |
| `spotify_api_request_duration_seconds` | Histogram | endpoint | Spotify API応答時間 |
| `share_redirects_total` | Counter | platform, content_type | シェアリダイレクト数 |
| `oauth_callbacks_total` | Counter | platform, status | OAuthコールバック数 |

## ライセンス

このプロジェクトはMITライセンスの下で公開されています。詳細はLICENSEファイルをご覧ください。
