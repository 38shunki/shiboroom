# デプロイガイド

## デプロイスクリプト

本番環境へのデプロイを自動化するスクリプトを用意しました。

### フロントエンドのデプロイ

```bash
./deploy-frontend.sh
```

**何が起きるか:**
1. ソースコードをサーバーに同期（rsync）
2. サーバー上で `npm install`（初回のみ）
3. サーバー上でクリーンビルド（`.next` 削除 → 再ビルド）
4. 環境変数 `NEXT_PUBLIC_API_URL=https://shiboroom.com` で本番ビルド
5. 静的ファイルをコピー
6. フロントエンドサービスを再起動

**所要時間:** 約2-3分

**確認方法:**
- https://shiboroom.com にアクセス
- 物件が表示されることを確認

---

### バックエンドのデプロイ

```bash
./deploy-backend.sh
```

**何が起きるか:**
1. ローカルでLinux用バイナリをビルド
2. バイナリをサーバーにコピー
3. 現在のバイナリをバックアップ
4. 新しいバイナリをインストール
5. バックエンドサービスを再起動
6. サービス起動を確認

**所要時間:** 約30秒

**確認方法:**
```bash
curl https://shiboroom.com/api/properties?limit=1
```

---

## なぜこの方法が良いのか

### 従来の問題点:
- ❌ ローカルでビルド → 環境変数が正しく反映されないことがある
- ❌ ビルドキャッシュが残る → 古いURLが混在
- ❌ Mac → Linux転送 → メタデータの警告が大量に出る

### デプロイスクリプトの利点:
- ✅ サーバー上で常にクリーンビルド
- ✅ 環境変数が確実に反映される
- ✅ 一貫性のあるデプロイプロセス
- ✅ 自動でサービス再起動
- ✅ エラー時は自動ロールバック（バックエンド）

---

## トラブルシューティング

### フロントエンドが起動しない

```bash
ssh grik@162.43.74.38
sudo journalctl -u shiboroom-frontend -n 50
```

### バックエンドが起動しない

```bash
ssh grik@162.43.74.38
sudo journalctl -u shiboroom-backend -n 50
```

### 物件が0件表示される

1. バックエンドAPIを確認:
```bash
curl https://shiboroom.com/api/properties?limit=1
```

2. ブラウザの開発者ツール (F12) でネットワークタブを確認
3. `/api/properties` へのリクエストとレスポンスを確認

---

## 手動デプロイ（トラブル時）

### フロントエンド:
```bash
ssh grik@162.43.74.38
cd /var/www/shiboroom/frontend-next
rm -rf .next
NEXT_PUBLIC_API_URL=https://shiboroom.com npm run build
cp -r .next/static .next/standalone/.next/
cp -r public .next/standalone/
sudo systemctl restart shiboroom-frontend
```

### バックエンド:
```bash
# ローカル
cd backend
GOOS=linux GOARCH=amd64 go build -o shiboroom-api ./cmd/api
scp shiboroom-api grik@162.43.74.38:/tmp/

# サーバー
ssh grik@162.43.74.38
sudo mv /tmp/shiboroom-api /var/www/shiboroom/
sudo chmod +x /var/www/shiboroom/shiboroom-api
sudo systemctl restart shiboroom-backend
```

---

## 今後の改善案

- [ ] GitHub Actionsで自動デプロイ
- [ ] デプロイ前の自動テスト
- [ ] Blue-Green デプロイメント
- [ ] ロールバック機能の強化
