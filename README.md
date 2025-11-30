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

## デモ

Spotifyで音楽を再生した状態で、[ここ](https://spn.soli0222.com/tweet)にアクセスしてみてください。  
曲名、アーティスト名が入った状態で、簡単に共有することができます。

## プロジェクト構成

```
spotify-nowplaying/
├── cmd/
│   └── server/
│       └── main.go          # アプリケーションエントリーポイント
├── internal/
│   ├── handler/             # HTTPハンドラー
│   │   ├── handler.go       # 共通ハンドラーロジック
│   │   ├── note.go          # Misskey関連
│   │   ├── tweet.go         # Twitter関連
│   │   └── status.go        # ステータスエンドポイント
│   ├── metrics/             # Prometheusメトリクス
│   │   ├── metrics.go
│   │   └── middleware.go
│   └── spotify/             # Spotify API関連
│       ├── auth.go          # OAuth認証（型定義）
│       ├── client.go        # APIクライアント
│       └── player.go        # 再生情報取得・シェアURL生成
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── README.md
```

## セットアップ

アプリケーションをローカルで実行するためには、以下の手順に従ってください。

1. このリポジトリをクローンします。

   ```bash
   git clone https://github.com/Soli0222/spotify-now-playing.git
   ```

2. 必要なモジュールをインストールします。(Dockerを使用する場合は、スキップしてください。)

   ```bash
   go mod download
   ```

3. ``.env``ファイルを編集し、環境変数を設定します

   ```bash
   cp .env.example .env
   ```

   ```.env
   PORT=8080                    # 起動するポート番号
   METRICS_PORT=9090            # メトリクスポート番号（デフォルト: 9090）
   SERVER_URI=example.tld       # リダイレクト先サーバーのドメイン
   SPOTIFY_CLIENT_ID=xxxxxxxx   # クライアントID
   SPOTIFY_CLIENT_SECRET=xxxxx  # クライアントシークレット
   SPOTIFY_REDIRECT_URI_NOTE=https://example.tld/note/callback   # noteのリダイレクトURL
   SPOTIFY_REDIRECT_URI_TWEET=https://example.tld/tweet/callback # tweetのリダイレクトURL
   ```

## 実行

開発環境でアプリケーションを実行するには、以下のコマンドを使用します。

```bash
go run ./cmd/server
```

Dockerを用いてアプリケーションを実行するには、以下のコマンドを使用します。

```bash
docker compose up -d
```

## ビルド

アプリケーションを本番環境用にビルドし、実行するには、以下のコマンドを使用します。

```bash
go build -o server ./cmd/server

./server
```

マルチアーキテクチャでコンテナイメージをビルドするには、以下のコマンドを使用します。

```bash
docker buildx build --platform linux/amd64,linux/arm64 -t your-registry/spotify-nowplaying:latest --push .
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
