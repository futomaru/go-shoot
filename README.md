# Go Shoot

Go Shoot は、Go 言語のみを用いて P2P ブラウザシューティングゲームを構築するためのプロジェクトです。本リポジトリには、開発計画書で定義されたアーキテクチャを元にした初期実装が含まれており、ローカル環境で動作可能なバックエンド API とロビー WebSocket を提供します。

## 主な機能

- ゲームロビーの作成・一覧・参加 API (`/api/v1/rooms`)
- WebSocket を用いたロビー内チャットブリッジ (`/ws/lobby/{roomID}`)
- `/healthz` によるヘルスチェック
- 静的フロントエンド (プロトタイプ UI) の配信
- 開発計画書や今後のタスクを参照できるドキュメント配信

## 必要要件

- Go 1.22 以上

## 使い方

```bash
# 依存パッケージを取得
go mod tidy

# サーバーを起動
GO_SHOOT_HTTP_PORT=8080 go run ./cmd/server
```

サーバーが起動すると、以下の機能を試すことができます。

### ヘルスチェック

```bash
curl localhost:8080/healthz
```

### ロビーの作成と参加

```bash
# 部屋を作成
curl -X POST localhost:8080/api/v1/rooms

# レスポンス例
# {"id":"<uuid>","name":"Quick Match","createdAt":"2024-05-21T00:00:00Z","players":[]}

# プレイヤーを参加させる
curl -X POST localhost:8080/api/v1/rooms/<uuid>/join \
  -H 'Content-Type: application/json' \
  -d '{"player":"alice"}'
```

### WebSocket ロビー

任意の WebSocket クライアントで `ws://localhost:8080/ws/lobby/<uuid>` に接続すると、同じ部屋に接続したクライアント同士でメッセージをブロードキャストできます。

### フロントエンド

ブラウザで `http://localhost:8080/` にアクセスすると、プロトタイプのランディングページが表示されます。開発計画書は `http://localhost:8080/docs/development_plan.md` から参照できます。

## プロジェクト構成

```
.
├── cmd/server          # アプリケーションエントリポイント
├── internal
│   ├── config          # 環境変数の読み込み
│   ├── lobby           # WebSocket ロビーハブ
│   ├── rooms           # ルーム管理サービス (インメモリ)
│   └── server          # HTTP サーバーとルーティング
├── web                 # 静的フロントエンドアセット
└── docs                # 開発計画書などのドキュメント
```

## 今後の展開

- TinyGo + WebAssembly によるゲームクライアントの実装
- WebRTC DataChannel を利用した P2P 同期
- PostgreSQL / Redis との接続や永続化レイヤの実装
- OAuth2 を用いた認証/認可
- CI/CD、監視、ログ整備

詳細は [`docs/development_plan.md`](docs/development_plan.md) を参照してください。
