# Shiboroom セットアップガイド（簡易版）

## 前提条件
- サーバー: 162.43.74.38
- ユーザー: grik
- ドメイン: shiboroom.com
- アプリケーションファイルは既にデプロイ済み

## セットアップ手順（4ステップ）

サーバーにSSHして、以下のスクリプトを順番に実行してください：

```bash
ssh grik@162.43.74.38
```

### ステップ1: systemdサービスとNginx基本設定
```bash
bash /tmp/setup-step1-services.sh
```

実行内容：
- systemdサービスファイルの配置と有効化
- Nginx HTTP設定の作成（証明書取得前）
- sudoers設定

### ステップ2: MySQLデータベースセットアップ
```bash
bash /tmp/setup-step2-database.sh
```

実行内容：
- MySQLデータベース（shiboroom）の作成
- データベースユーザー（shiboroom_user）の作成
- 設定ファイルへのパスワード反映

**注意**:
- MySQLのrootパスワードが必要です
- 新しいshiboroom_userのパスワードを入力する必要があります

### ステップ3: SSL証明書取得とHTTPS設定
```bash
bash /tmp/setup-step3-ssl.sh
```

実行内容：
- Let's Encrypt SSL証明書の取得
- Nginx HTTPS設定への更新
- 証明書の自動更新設定

**注意**:
- certbotがない場合は自動インストールされます
- メールアドレスの入力が必要です
- 利用規約への同意が必要です

### ステップ4: サービス起動と動作確認
```bash
bash /tmp/setup-step4-start.sh
```

実行内容：
- バックエンドサービスの起動
- フロントエンドサービスの起動
- 動作確認とステータス表示

## セットアップ完了後

### アクセス確認
- https://shiboroom.com
- https://shiboroom.com/api/health

### サービス管理コマンド

**ステータス確認:**
```bash
sudo systemctl status shiboroom-backend
sudo systemctl status shiboroom-frontend
```

**ログ確認:**
```bash
sudo journalctl -u shiboroom-backend -f
sudo journalctl -u shiboroom-frontend -f
sudo tail -f /var/log/nginx/shiboroom-error.log
```

**サービス再起動:**
```bash
/var/www/shiboroom/restart-shiboroom-services.sh
```

または個別に：
```bash
sudo systemctl restart shiboroom-backend
sudo systemctl restart shiboroom-frontend
sudo systemctl reload nginx
```

## トラブルシューティング

### バックエンドが起動しない
```bash
# ログ確認
sudo journalctl -u shiboroom-backend -n 100

# 手動起動してエラー確認
cd /var/www/shiboroom/backend
CONFIG_PATH=/var/www/shiboroom/config/scraper_config.yaml ./shiboroom-api
```

### フロントエンドが起動しない
```bash
# ログ確認
sudo journalctl -u shiboroom-frontend -n 100

# 手動起動してエラー確認
cd /var/www/shiboroom/frontend/.next/standalone
NODE_ENV=production PORT=5177 node server.js
```

### データベース接続エラー
```bash
# 設定ファイル確認
cat /var/www/shiboroom/config/scraper_config.yaml

# MySQL接続テスト
mysql -u shiboroom_user -p shiboroom
```

### SSL証明書エラー
```bash
# 証明書確認
sudo certbot certificates

# 証明書の手動更新
sudo certbot renew
```

## 再デプロイ方法

ローカルマシンから：
```bash
cd /Users/shu/Documents/dev/real-estate-portal
./deploy.sh
```

サービスは自動的に再起動されます。

## ポート一覧

| サービス | ポート | 説明 |
|---------|--------|------|
| Frontend | 5177 | Next.js (内部) |
| Backend | 8085 | Go API (内部) |
| MySQL | 3306 | データベース (内部) |
| Meilisearch | 7700 | 検索エンジン (内部) |
| Nginx | 80, 443 | リバースプロキシ (外部) |

※ grikアプリとは完全に独立（別ポート、別ディレクトリ）
