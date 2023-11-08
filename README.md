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

2. 必要なモジュールをインストールします。

   ```bash
   go mod download
   ```

3. ``.env``ファイルを編集し、環境変数を設定します

   ```bash
   cp .env.example
   ```

   ```.env
   PORT=8080  //起動するポート番号
   SPOTIFY_CLIENT_ID= //クライアントID
   SPOTIFY_CLIENT_SECRET= //クライアントシークレット
   SPOTIFY_REDIRECT_URI_NOTE= //noteのリダイレクトURL
   SPOTIFY_REDIRECT_URI_TWEET= //tweetのリダイレクトURL
   ```

## 実行

開発環境でアプリケーションを実行するには、以下のコマンドを使用します。

```bash
go run main.go
```

## ビルド

アプリケーションを本番環境用にビルドするには、以下のコマンドを使用します。

```bash
go build
```
