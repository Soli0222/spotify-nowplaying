# Spotify Now Playing Sharing App

このアプリケーションは、Spotifyで現在再生中の曲やエピソードをMisskeyサーバー・Polestarに共有するためのWebアプリケーションです。

## アプリケーションの概要

このアプリケーションは、SpotifyのAPIを使用して、現在再生中の曲やエピソードの情報を取得し、簡単に共有できるようにシェアリンクを生成し、Misskeyにリダイレクトさせます。

## セットアップ

アプリケーションをローカルで実行するために、以下の手順に従ってください。

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
   PORT=8080 //起動するポート番号
   SERVER_URI=example.tld //リダイレクト先サーバーのドメイン
   SPOTIFY_CLIENT_ID=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx //クライアントID
   SPOTIFY_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx //クライアントシークレット
   SPOTIFY_REDIRECT_URI_NOTE=https://example.tld/note/callback //noteのリダイレクトURL
   SPOTIFY_REDIRECT_URI_TWEET=https://example.tld/tweet/callback //tweetのリダイレクトURL
   ```

## 実行

開発環境でアプリケーションを実行するには、以下のコマンドを使用します。

```bash
go run main.go
```

Dockerを用いてアプリケーションを実行するには、以下のコマンドを使用します。

```bash
docker compose up -d
```

## ビルド

アプリケーションを本番環境用にビルドし、実行するには、以下のコマンドを使用します。

```bash
go build

./spotify-nowplaying
```

コンテナイメージを自らビルドするには、``docker-compose.yml``に対して、以下の変更を行います。

```docker-compose.yml
version: '3'
services:
  app:
    build: . //ここを変更
    platform: linux/amd64
    volumes:
      - './.env:/app/.env:ro'
    ports:
      - 8080:8080
    restart: always
```