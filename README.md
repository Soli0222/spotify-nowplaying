# Spotify Now Playing Sharing App

このアプリケーションは、Spotifyで現在再生中の曲やエピソードをMisskeyサーバー・Polestarに共有するためのWebアプリケーションです。

## アプリケーションの概要

このアプリケーションは、SpotifyのAPIを使用して、現在再生中の曲やエピソードの情報を取得し、簡単に共有できるようにシェアリンクを生成し、Misskeyにリダイレクトさせます。

## セットアップ

アプリケーションをローカルで実行するために、以下の手順に従ってください。

1. このリポジトリをクローンします。

   ```bash
   git clone https://github.com/your-username/spotify-now-playing.git
   ```

2. 必要なモジュールをインストールします。

   ```bash
   npm install
   ```

## 実行

開発環境でアプリケーションを実行するには、以下のコマンドを使用します。

```bash
npm run dev
```

アプリケーションはローカルでポート3000番で実行されます。  
ポート番号を変更して実行したい場合は以下のようにします。

```bash
PORT=3500 npm run dev
```

## ビルド

アプリケーションを本番環境用にビルドするには、以下のコマンドを使用します。

```bash
npm run build
```
