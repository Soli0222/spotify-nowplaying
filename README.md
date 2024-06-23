# Spotify NowPlaying Sharing App

このアプリケーションは、Spotifyで現在再生中の曲やエピソードを、任意のMisskeyサーバーもしくはTwitter(自称:𝕏)に共有するためのWebアプリケーションです。

## 概要

SpotifyのAPIを使用して、現在再生中の曲やエピソードの情報を取得します。  
そして、簡単に共有できるようにシェアリンクを生成し、MisskeyもしくはTwitter(自称:𝕏)にリダイレクトさせます。  

シェアリンクには、以下のようなフォーマットのテキストが含まれており、曲名, アーティスト名, #NowPlayingを自ら入力する必要はありません。

```
放課後マーメイド / しぐれうい
#NowPlaying #PsrPlaying
https://open.spotify.com/track/4EwRjHviKXIlmEUfIItkqP
```

## デモ

Spotifyで音楽を再生した状態で、[ここ](https://spn.soli0222.com/tweet)にアクセスしてみてください。  
曲名、アーティスト名が入った状態で、簡単に共有することができます。

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