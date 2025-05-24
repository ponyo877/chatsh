# chatsh gRPC-Web Integration

このドキュメントでは、chatshのWebクライアントで実際のgRPC通信を使用する方法について説明します。

## アーキテクチャ

```
Browser (Vite Dev Server :3000)
    ↓ gRPC-Web requests
Envoy Proxy (:8080)
    ↓ gRPC requests
chatsh Server (:50051)
```

## セットアップ手順

### 1. 前提条件

- Docker & Docker Compose
- Node.js & npm
- Go (サーバー開発用)

### 2. gRPC-Webクライアントの生成

```bash
# プロトコルバッファからTypeScriptコードを生成
cd browser
./generate-proto.sh
```

### 3. Webクライアントの起動

```bash
# 依存関係のインストール
cd browser
npm install

# 開発サーバーの起動
npm run dev
```

ブラウザで `http://localhost:3000` にアクセス

### 4. gRPCサーバーとEnvoyプロキシの起動

```bash
# Docker Composeでサーバーとプロキシを起動
docker-compose up --build
```

これにより以下が起動されます：
- chatsh gRPCサーバー (ポート50051)
- Envoyプロキシ (ポート8080) - gRPC-Web変換
- Envoy管理インターフェース (ポート9901)

## 利用可能な機能

### 基本コマンド
- `pwd` - 現在のパス表示
- `ls` - ディレクトリ/ルーム一覧
- `cd <path>` - ディレクトリ移動
- `touch <name>` - チャットルーム作成
- `mkdir <name>` - ディレクトリ作成
- `clear` - 画面クリア
- `help` - ヘルプ表示

### チャット機能
- `vim <room_name>` - チャットルームに入室
- チャットモードでメッセージ送受信
- `Ctrl+C` でチャットモード終了

## gRPC-Web通信の詳細

### 実装されたgRPCメソッド
- `ListNodes` - ディレクトリ/ルーム一覧取得
- `CreateRoom` - チャットルーム作成
- `CreateDirectory` - ディレクトリ作成
- `CheckDirectoryExists` - ディレクトリ存在確認
- `WriteMessage` - メッセージ送信
- `ListMessages` - メッセージ履歴取得

### フォールバック機能
gRPCサーバーが利用できない場合、自動的にモックデータにフォールバックします。

## トラブルシューティング

### gRPC接続エラー
1. Envoyプロキシが起動しているか確認: `http://localhost:9901`
2. chatshサーバーが起動しているか確認
3. ブラウザの開発者ツールでネットワークエラーを確認

### CORS エラー
Envoyプロキシの設定でCORSが有効になっています。設定を変更する場合は `envoy.yaml` を編集してください。

### プロトコルバッファの更新
プロトファイルを変更した場合：
```bash
cd browser
./generate-proto.sh
npm run dev  # 開発サーバーを再起動
```

## ファイル構成

```
chatsh/
├── browser/                    # Webクライアント
│   ├── src/
│   │   ├── main.ts            # メインアプリケーション
│   │   ├── grpc-client.ts     # gRPC-Webクライアント
│   │   ├── chat-ui.ts         # チャットUI
│   │   ├── grpc/              # プロトファイル
│   │   └── generated/         # 生成されたgRPCコード
│   ├── package.json
│   └── generate-proto.sh      # コード生成スクリプト
├── envoy.yaml                 # Envoyプロキシ設定
├── docker-compose.yml         # Docker Compose設定
└── server/                    # Goサーバー
```

## 開発者向け情報

### gRPC-Webクライアントの拡張
新しいgRPCメソッドを追加する場合：
1. `grpc/chatsh.proto` を更新
2. `./generate-proto.sh` を実行
3. `grpc-client.ts` にメソッドを追加
4. UIコンポーネントで使用

### デバッグ
- ブラウザの開発者ツールでgRPC-Webリクエストを確認
- Envoy管理インターフェース: `http://localhost:9901`
- サーバーログ: `docker-compose logs chatsh-server`

## 次のステップ

1. **リアルタイムストリーミング**: gRPCストリーミングを使用したリアルタイムメッセージ配信
2. **認証**: JWTトークンベースの認証システム
3. **ファイルアップロード**: バイナリファイルの送受信
4. **通知**: ブラウザ通知API統合
