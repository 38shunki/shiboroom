# フロントエンドデプロイメント問題の解決策まとめ

## 問題の概要

フロントエンドをデプロイした際に「物件一覧 (0件)」と表示され、バックエンドには99件のデータがあるにも関わらず、フロントエンドで表示されない問題が発生しました。

## 発見された問題と解決策

### 1. 環境変数の設定ミス (CORS エラーの原因)

**問題:**
- 本番環境で `.env.local` (ローカル開発用) が使用されていた
- `NEXT_PUBLIC_API_URL=http://localhost:8084` が本番ビルドに含まれていた
- Next.js では `NEXT_PUBLIC_*` 変数はビルド時に JavaScript に埋め込まれる
- 本番サイト (`https://shiboroom.com`) から `http://localhost:8084` へのリクエストで CORS エラー発生

**解決策:**
1. `.env.production` ファイルを作成し、本番用 API URL を設定:
   ```bash
   NEXT_PUBLIC_API_URL=https://shiboroom.com
   ```

2. デプロイスクリプトで `.env.local` を除外:
   ```bash
   rsync --exclude '.env.local' ...
   ```

3. サーバー上でビルド前に `.env.local` を削除:
   ```bash
   rm -f .env.local
   NODE_ENV=production npm run build
   ```

### 2. Next.js ルーティングの競合 (404 エラーの原因)

**問題:**
- プロジェクトルートに空の `app/` ディレクトリが作成されていた
- Next.js は `src/app/` より root の `app/` を優先する
- 空の `app/` ディレクトリのため Pages Router にフォールバック
- App Router で作成したページが認識されず 404 エラー

**解決策:**
1. デプロイ前にローカルの `app/` ディレクトリを削除
2. サーバー上でビルド前に `app/` ディレクトリを削除:
   ```bash
   rm -rf app
   ```

### 3. API レスポンス形式の不一致

**問題:**
- コードは `data.properties` を期待していた
- 実際の API は配列を直接返していた `[{...}, {...}]`
- `data.properties` が `undefined` になり、プロパティが表示されない

**解決策:**
両方の形式に対応するように修正:
```typescript
const propertiesArray = Array.isArray(data) ? data : (data.properties || [])
```

## デプロイスクリプトの改善内容

`deploy.sh` に以下の改善を追加しました:

### デプロイ前のクリーンアップ
```bash
# ローカルの競合するディレクトリを削除
if [ -d "$PROJECT_ROOT/frontend/app" ]; then
  rm -rf "$PROJECT_ROOT/frontend/app"
fi
```

### rsync で .env.local を除外
```bash
rsync -avz --delete \
  --exclude 'node_modules' \
  --exclude '.next' \
  --exclude '.git' \
  --exclude '.env.local' \  # 追加
  --quiet \
  "$PROJECT_ROOT/frontend/" "$SERVER:/var/www/shiboroom/frontend/"
```

### サーバー上での重要なクリーンアップ
```bash
# 競合するディレクトリとファイルを削除
rm -rf app  # Pages Router へのフォールバックを防ぐ
rm -f .env.local  # 本番環境設定を上書きしないように
```

### ビルド検証
```bash
# App Router が使用されたか確認
if [ -d ".next/server/app" ]; then
  echo "✅ App Router build verified"
fi

# 本番 API URL がビルドに含まれているか確認
if grep -q "shiboroom.com" .next/static/chunks/app/page*.js 2>/dev/null; then
  echo "✅ Production API URL verified in build"
fi
```

## 使用方法

### フロントエンドのみデプロイ
```bash
./deploy.sh frontend
```

### バックエンドのみデプロイ
```bash
./deploy.sh backend
```

### 両方デプロイ
```bash
./deploy.sh
# または
./deploy.sh all
```

## ローカル開発環境の設定

### frontend/.env.local (ローカル開発用)
```bash
NEXT_PUBLIC_API_URL=http://localhost:8084
```

### frontend/.env.production (本番用)
```bash
NEXT_PUBLIC_API_URL=https://shiboroom.com
```

これらのファイルは既に作成済みです。今後は `./deploy.sh frontend` を実行するだけで、正しい環境変数で本番デプロイされます。

## 検証方法

デプロイ後、以下を確認してください:

1. **サービスが起動しているか:**
   ```bash
   ssh grik@162.43.74.38 'systemctl status shiboroom-frontend'
   ```

2. **本番サイトで物件が表示されるか:**
   - https://shiboroom.com にアクセス
   - 「物件一覧 (99件)」のように正しい件数が表示されることを確認

3. **ブラウザコンソールにエラーがないか:**
   - F12 でデベロッパーツールを開く
   - CORS エラーや 404 エラーがないことを確認

## トラブルシューティング

### まだ 0 件と表示される場合

1. ブラウザのキャッシュをクリア (Ctrl+Shift+R / Cmd+Shift+R)
2. サーバーで API URL を確認:
   ```bash
   ssh grik@162.43.74.38 'grep -r "API_URL" /var/www/shiboroom/frontend/.next/server/app/page.js'
   ```
   `https://shiboroom.com` が含まれていることを確認

### 404 エラーが発生する場合

1. App Router が使用されているか確認:
   ```bash
   ssh grik@162.43.74.38 'ls -la /var/www/shiboroom/frontend/.next/server/'
   ```
   `app/` ディレクトリが存在することを確認

2. root に app/ ディレクトリがないか確認:
   ```bash
   ssh grik@162.43.74.38 'ls -la /var/www/shiboroom/frontend/ | grep app'
   ```
   `app/` ディレクトリが存在する場合は削除して再デプロイ

## まとめ

全ての問題は **デプロイスクリプトの改善** で解決されました。今後は:

1. `./deploy.sh frontend` を実行するだけで正しくデプロイされます
2. 環境変数は自動的に本番用が使用されます
3. 競合するディレクトリは自動的に削除されます
4. ビルドの検証が自動的に行われます

手動での作業は不要になりました。
