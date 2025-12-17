# Phase 0: PoC検証ガイド

このガイドでは、Phase 0の4つのPoC検証項目を実行する方法を説明します。

## 前提条件

- Dockerとdocker-composeがインストールされている
- すべてのコンテナが起動している

## 検証項目

1. **スクレイピング安定性**: 同じURLで3回連続成功（タイトル/画像URL/詳細URLが取れる）
2. **検索機能**: Meilisearchで検索が正しく動作する
3. **画像外部参照**: Yahoo不動産の画像URLが表示できる
4. **Yahoo不動産リンク**: 詳細ページへのリンクが正しく動作する

## 実行手順

### 1. Dockerコンテナの起動確認

```bash
# コンテナの起動状態を確認
docker ps

# 以下のコンテナが起動していることを確認
# - realestate-backend
# - realestate-mysql
# - realestate-meilisearch
# - realestate-frontend-next
```

### 2. PoC検証スクリプトの実行

```bash
# バックエンドコンテナに入る
docker exec -it realestate-backend /bin/sh

# PoC検証スクリプトを実行
cd /app
go run cmd/test-poc/main.go

# 結果はJSON形式で保存されます
# poc-results-YYYYMMDD-HHMMSS.json
```

### 3. 手動検証（推奨）

自動テストに加えて、以下の手動検証も実施してください：

#### 検証1: スクレイピング安定性

```bash
# バックエンドコンテナ内で実行
docker exec -it realestate-backend /bin/sh

# テスト用のスクレイピング実行（3回）
# 実際のYahoo不動産URLを指定してください
export TEST_URL="https://realestate.yahoo.co.jp/rent/search/0123/list/"

# 手動で3回実行して、毎回成功することを確認
curl -X POST http://localhost:8084/api/scrape/list \
  -H "Content-Type: application/json" \
  -d "{\"url\": \"$TEST_URL\"}"
```

**期待結果**:
- 3回とも200 OKが返る
- 物件URLのリストが返る（空でないこと）
- エラーが発生しない

#### 検証2: 検索機能（Meilisearch）

```bash
# ブラウザまたはcurlで以下のエンドポイントにアクセス

# 1. 全物件を取得
curl "http://localhost:8084/api/properties?limit=10"

# 2. キーワード検索（新宿）
curl "http://localhost:8084/api/search?q=新宿"

# 3. 高度な検索（フィルタ付き）
curl -X POST http://localhost:8084/api/search/advanced \
  -H "Content-Type: application/json" \
  -d '{
    "query": "新宿",
    "min_rent": 80000,
    "max_rent": 150000,
    "floor_plans": ["1K", "1DK"],
    "sort": "rent_asc",
    "limit": 20
  }'
```

**期待結果**:
- 検索結果が返る
- フィルタが正しく適用される（min_rent/max_rentで絞り込まれている）
- ソート順が正しい

#### 検証3: 画像外部参照

```bash
# フロントエンドで物件一覧を表示
# http://localhost:5176 にアクセス

# 1. 一覧画面で物件の画像が表示されることを確認
# 2. 画像が表示されない場合、「画像なし」プレースホルダが表示されることを確認
# 3. ブラウザの開発者ツールで画像URLを確認
#    - yimg.jp ドメインの画像URLが使われているか確認
```

**期待結果**:
- 物件画像が表示される（Yahoo不動産の外部画像URLが使える）
- 画像が取得できない場合は「画像なし」と表示される
- 404エラーやCORSエラーが発生しない

#### 検証4: Yahoo不動産リンク

```bash
# フロントエンドで物件詳細リンクをクリック
# http://localhost:5176 にアクセス

# 1. 物件カードの「Yahoo不動産で詳細を見る」をクリック
# 2. Yahoo不動産の詳細ページが新しいタブで開くことを確認
# 3. URLが正しい形式であることを確認
#    https://realestate.yahoo.co.jp/rent/detail/[48文字のID]
```

**期待結果**:
- Yahoo不動産の詳細ページが正しく開く
- target="_blank" で新しいタブが開く
- リファラーが送信されない（rel="noreferrer"）

## PoC合格基準

以下の4項目すべてがクリアできれば **PoC合格** です：

- ✅ **項目1**: 同一URLで3回中3回成功（物件URLが取得できる）
- ✅ **項目2**: Meilisearchで検索・フィルタが正しく動作する
- ✅ **項目3**: 画像URLが表示できる（または適切にフォールバックする）
- ✅ **項目4**: Yahoo不動産へのリンクが正しく動作する

## PoC中止ライン（いずれか該当したら中止）

- ❌ 同一URLで3回中2回以上スクレイピング失敗
- ❌ 画像URLが24時間以内に頻繁に失効
- ❌ 一覧ページ取得で明確なブロック挙動（403/429が連続）
- ❌ Meilisearchのインデックスが再構築不能

## トラブルシューティング

### スクレイピングが失敗する場合

```bash
# バックエンドのログを確認
docker logs realestate-backend --tail 100

# 403/429エラーが出る場合
# → レート制限に引っかかっている可能性
# → backend/internal/ratelimit/yahoo_limiter.go の設定を確認

# circuit breaker が open になっている場合
# → 連続で失敗が多い
# → 1時間待ってから再試行
```

### Meilisearchで検索結果が返らない場合

```bash
# Meilisearchの状態を確認
curl http://localhost:7700/health

# インデックスの確認
curl -H "Authorization: Bearer masterKey123" \
  http://localhost:7700/indexes/properties/stats

# インデックスが空の場合、データを再登録
docker exec -it realestate-backend /bin/sh
# バックエンドのAPI経由でスクレイピングを実行
```

### 画像が表示されない場合

1. ブラウザの開発者ツールでネットワークタブを確認
2. 画像URLに直接アクセスして確認
3. CORSエラーの場合は正常（外部画像なので発生する可能性あり）
4. 404エラーの場合は、画像URLが変更された可能性あり

## 次のステップ

PoC合格後は、以下の順で進めます：

1. **Phase 1**: 技術スタック修正（PostgreSQL→MySQL、ORM導入）
2. **Phase 2**: データ品質改善（URL正規化、画像検証、バリデーション）
3. **Phase 3**: 差分検出・毎日更新機能
4. **Phase 4**: 負荷最小化・安全運用設計

詳細は `docs/TODO.md` を参照してください。
