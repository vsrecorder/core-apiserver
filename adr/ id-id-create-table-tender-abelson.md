# バトレコユーザーIDとポケモンカードゲーム プレイヤーズクラブIDの紐付け機能

## ステータス

採用 (Accepted) — 実装済み

**改訂履歴**

- 2026-07-30 (第5版): **チャンピオンシップポイントの表示を廃止**(D10)。プレイヤーIDが本人のものである保証が無くなったため。`GET /usersplayers` はランキング検索を行わず、`champion_ship_point` / `ranking_date` も返さない。
- 2026-07-29 (第4版): **プレイヤーIDの検証を全廃**。実在確認・所有権確認(アバターチャレンジ)をどちらも行わず、利用者の自己申告として受け入れる方針に転換(D3'/D5')。あわせて `player_id` の重複登録を許容する(D1')。外部サイトへのリクエストは本機能から完全に無くなった。
- 2026-07-29 (第3版): 実在確認・所有権確認の責務を core-apiserver から webapp(BFF) へ移動。
- 2026-07-2x (第2版): なりすまし対策としてアバター変更チャレンジを追加。レート制限・キルスイッチを追加。`player_rankings` 連携によるチャンピオンシップポイント表示を追加。
- 初版: 実在確認のみで紐付ける計画。

---

## Context

バトレコ(本サービス)のユーザーアカウントと、公式のポケモンカードゲーム プレイヤーズクラブが発行する `player_id` を紐付けたい。公式サイト側のチャンピオンシップポイントや、シティリーグの公式結果と対戦記録を連携させるための土台となる機能。

紐付けは誤登録防止のため一度行うと1ヶ月は変更できない仕様とする。誤ったIDを紐付けてしまった際は、**追加のコードは実装せず運用者(開発者自身)が手動でDBを修正する**ことで合意している(本サービスは小規模な個人運用サービスのため、管理用エンドポイントを新設するコストに見合わない)。

### 検証を全廃するに至った経緯

第2〜3版では、プレイヤーズクラブの以下のAPIで実在確認を行い、さらにアバター画像の変更を課すことで所有権を確認していた。

```
curl -s -X POST -d 'player_id=XXXXXXXXXXXXXXXX' "https://players.pokemon-card.com/get_player_account_other"
```

しかしこの方式は次の理由で維持できなくなった。

1. **`players.pokemon-card.com` が Cloudflare でブロックされた。** トップページを含む同ホストの全URLが 403(`Attention Required!` のHTML)を返す。User-Agent(空/Go既定/自己申告名)を変えても、HTTPクライアント(Go / curl / Node)を変えても結果は同じで、当方サーバーからの通信そのものが遮断されている。
2. **ブラウザから直接叩く回避策も成立しない。** 利用者自身のブラウザであれば Cloudflare は通る可能性が高いが、このAPIは `Access-Control-Allow-Origin` を返さないため JavaScript から応答を読めない。Chromium で実測したところ、通常の `fetch` は `TypeError: Failed to fetch`、`no-cors` では `type=opaque` / `status=0` となりボディを取得できなかった。ブロックされていない `www.pokemon-card.com` の 200 応答にも CORS ヘッダは無く、公式サイトが自サイト内から呼ぶ前提のAPIであることが確認できた。
3. 仮にブラウザで読めたとしても、クライアントの自己申告を信じることになり所有権確認として成立しない。

つまり**検証を継続する手段が無い**。検証できないまま機能を止め続けるより、検証を行わない前提で機能を提供する方が利用者の利益になると判断した。

---

## Decision

### D1'. 有効な紐付けは1ユーザーにつき1件。`player_id` の重複は許容する

**第4版で変更。** `user_id` 側の部分ユニークインデックスは維持するが、`player_id` 側は一意にしない。

所有権を確認しない以上、**先に登録した人が正しい持ち主とは限らない**。重複を禁止すると、他人に先に登録された正当な利用者が締め出され、しかもそれを救済する手段が運用者の手動介入しか無くなる。重複を許容する方が実害が小さい。

`player_id` は `cityleague_results` との結合に使うため、索引自体は非ユニークとして残す。

### D2. 1ヶ月ロックはアプリ層で判定する

DB制約ではなく usecase 層で「有効行の `created_at` + 1ヶ月」を見る。ビジネスルールはアプリ層に置く既存方針(`deck_codes` の所有者チェック等)に揃える。

検証が無くなった今、**これが唯一残る誤登録の抑止**であり、入力前の注意喚起(モーダルの警告)とあわせて機能する。

### D3'. 実在確認・所有権確認は行わない

**第4版で変更(第2版のD3を廃止)。** `player_id` はプレイヤーズクラブに実在するかを確認せず、利用者が入力した値をそのまま保存する。存在しないIDでも登録できる。

この紐付けは**「本人であることの証明」ではなく、利用者の自己申告**である。表示・集計にあたってはその前提で扱うこと。

### D5'. 外部サイトへのリクエストを行わない

**第4版で変更(第3版のD5を廃止)。** 本機能から `players.pokemon-card.com` へのリクエストは、core-apiserver・webapp のどちらからも無くなった。webapp のBFFは、セッションを確認して core-apiserver へ中継するだけの単純な構成に戻る。

これに伴い次を削除した。

- webapp: `utils/players_club.ts` / `utils/user_player_challenge.ts` / `utils/ratelimit.ts` / `utils/user_player_upstream.ts` / `api/usersplayers/verify/route.ts`
- core-apiserver: `usecase/player_account.go` / `usecase/user_player_verification.go`、`POST /usersplayers/challenge` エンドポイント、`PokemonAvatar` のリポジトリ・エンティティ・インフラ・モック

`pokemon_avatars` テーブルと `cmd/sync-pokemon-avatars` は残しているが、本機能からは参照しない。

### D6'. レート制限は uid 単位のみ残す

**第4版で縮小。** 外部サイトへの問い合わせが無くなり、他人の `player_id` を総当たりで探索する余地も無くなったため、`player_id` 単位の制限は廃止した。uid 単位(10回/1時間)は書き込みの連打を防ぐために残す。通常は1ヶ月ロックがあるため、ここに到達するのは失敗を繰り返した場合に限られる。

### D10. チャンピオンシップポイントは表示しない

**第5版で追加。** 検証を廃止した結果、連携されている `player_id` が本人のものである保証が無くなった。他人のポイントを本人の実績のように見せてしまうため、プロフィールパネルからCSPの表示(アイコン・ポイント数・「〜現在」の日付)を削除した。

表示しないものを取得し続ける意味も無いため、`GET /usersplayers` はランキング検索そのものを行わない。レスポンスから `champion_ship_point` / `ranking_date` を削除し、usecase から `FindLatestPlayerRanking` を廃止した。プロフィール表示ごとに発生していた `player_rankings` への1クエリも無くなる。

これに伴い `PlayerRanking` の読み取り層(entity / repository / infrastructure / model / モック)は参照が無くなったため削除した。`player_rankings` テーブルと、別リポジトリの `import-player-ranking-job` による日次取込は残置している。

### D7. 環境変数によるキルスイッチを設ける

悪用が多発した場合にデプロイなしで機能全体を止められるよう、`USERS_PLAYERS_LINKING_ENABLED=false` で全エンドポイントが 503 を返す。

### D9. 誤紐付けは運用者が手動でDBを修正する

管理用エンドポイントは作らない(Context 記載の合意事項)。

---

## 1. スキーマ (core-apiserver/db/schema.sql)

```sql
CREATE TABLE users_players (
    id          VARCHAR(26) PRIMARY KEY,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    deleted_at  TIMESTAMP DEFAULT NULL,
    user_id     VARCHAR(32) NOT NULL,
    player_id   VARCHAR(16) NOT NULL
);

CREATE INDEX idx_users_players_created_at ON users_players(created_at);
CREATE INDEX idx_users_players_deleted_at ON users_players(deleted_at);

CREATE UNIQUE INDEX unique_users_players_user_id ON users_players (user_id)   WHERE deleted_at IS NULL;
CREATE INDEX idx_users_players_player_id         ON users_players (player_id) WHERE deleted_at IS NULL;
```

`id` は `deck_codes` と同じ `VARCHAR(26)`(ULID)。

**既存環境への適用が必要。** `schema.sql` は新規構築用でマイグレーション機構は無いため、稼働中のDBには手動で次を実行する。

```sql
DROP INDEX unique_users_players_player_id;
CREATE INDEX idx_users_players_player_id ON users_players (player_id) WHERE deleted_at IS NULL;
```

関連テーブル:

- `player_rankings` — 主キーは `(ranking_date, league_id, player_id)`。別リポジトリの `import-player-ranking-job` が日次で取り込む。第5版で本機能からは未参照になった(D10)。
- `pokemon_avatars` — アバターチャレンジ用に導入したが、第4版で本機能からは未使用になった。テーブルと `cmd/sync-pokemon-avatars` は残置。

---

## 2. バックエンド (core-apiserver)

### 2.1 エンドポイント

いずれも `linkingEnabledMiddleware`(D7) → `RequiredAuthenticationMiddleware` を通る。操作対象は URL パラメータではなく **JWT の `uid` のみを信頼**し、他人の紐付けを操作できないようにする。

| メソッド | パス | 役割 |
| --- | --- | --- |
| GET | `/usersplayers` | 紐付けを返す |
| POST | `/usersplayers` | 紐付けを保存する |

`GET` が返すのは紐付けそのもの(`id` / `created_at` / `user_id` / `player_id` / `locked_until`)のみ。ランキング関連のフィールドは返さない(D10)。

### 2.2 ドメイン層

- `entity.UserPlayer{ID, CreatedAt, UserId, PlayerId}` + `LockedUntil()`(= `CreatedAt.AddDate(0, 1, 0)`)
- `repository.UserPlayerInterface`: `FindByUserId` / `Save` / `Delete`(soft delete)
- `apperror.ErrLocked` — 1ヶ月ロック中 (409)

### 2.3 ユースケース層 (`internal/usecase/user_player.go`)

`Create` が「新規紐付け」と「ロック解除後の変更(旧レコードを soft delete + 新規作成)」の両方を担う。PUT/Update・Delete エンドポイントは作らない。

処理順:

1. `FindByUserId` で既存を取得
2. **同じ `player_id` なら変更不要**として既存をそのまま返す
3. 既存があり `now < LockedUntil()` → `ErrLocked`
4. トランザクション内で、既存があれば `Delete` → `Save`

実在確認・所有権確認・重複チェックはいずれも行わない。

### 2.4 DI登録

[main.go:289-297](core-apiserver/cmd/core-apiserver/main.go#L289-L297) で `usecase.NewUserPlayer` に `UserPlayer` リポジトリと `TransactionManager` を渡し、第4引数にキルスイッチの評価結果を渡す。

---

## 3. フロントエンド (webapp)

### 3.1 BFF route

| メソッド | パス | 処理 |
| --- | --- | --- |
| GET | `/api/usersplayers` | core-apiserver へ中継(404は未紐付けとして透過) |
| POST | `/api/usersplayers` | core-apiserver へ中継 |

### 3.2 ステータスと画面の対応

| ステータス | 意味 | モーダルの挙動 |
| --- | --- | --- |
| 400 | player_id が空/16文字超 | エラートースト |
| 409 | ロック中 | エラートースト |
| 429 | レート制限 | 時間をおくよう案内 |
| 503 | 連携機能が停止中 | 時間をおくよう案内 |

### 3.3 UI

- `Modal/LinkPlayerIdModal.tsx` — **入力1ステップ**。確認画面とアバター表示は第4版で廃止した。「入力されたプレイヤーIDが正しいかどうかの確認は行いません」「一度連携すると1ヶ月間は変更できません」を送信前に表示する。
- `User/PlayerLinkCard.tsx` — 紐付け状態を表示。未紐付け/ロック中(解除日を表示しボタン非活性)/変更可能の3状態。`?link_player=1` 付きで遷移してきた場合はモーダルを自動で開く。
- `User/UserProfileCard.tsx` — 連携済みなら「プレイヤーズクラブ連携済み」とだけ表示し、未連携なら連携導線を出す。チャンピオンシップポイントは第5版で表示をやめた(D10)。
- `Designation/DesignationPanel.tsx` — 称号判定に連携が必要なため、連携状態を参照する。

---

## 4. 連携後に有効になるもの

- **称号(デジグネーション)判定** — `cityleague_results.player_id = users_players.player_id` で結合し、シティリーグの公式結果とユーザーの記録を突き合わせる([designation_stats.go](core-apiserver/internal/infrastructure/designation_stats.go))。

これは**入力された `player_id` を正しいものとして扱う**。第4版以降、この前提は保証されない(後述)。

チャンピオンシップポイント表示は第2版で追加したが、同じ理由により第5版で廃止した(D10)。**現時点で `player_id` を実際に使っているのは称号判定のみ**である。

---

## 5. 誤紐付け時の対応(コード実装なし)

1. 対象ユーザーの `users_players` の有効行(`deleted_at IS NULL`)を確認する。
2. `UPDATE users_players SET deleted_at = now(), updated_at = now() WHERE id = '...'` で soft delete する。
3. 正しい `player_id` で `INSERT INTO users_players (id, created_at, updated_at, user_id, player_id) VALUES (...)` を実行する(`id` は ULID 採番)。

アプリ側のロック判定は有効行の `created_at` のみを見るため、この操作で当該ユーザーは新しい1ヶ月ロックのもとで正しい状態に戻る。

---

## 6. Consequences

**Pros**

- **外部サイトの可用性に依存しなくなった。** Cloudflare のブロックが続いていても連携機能は動作する。
- 実装が大幅に単純になった。webapp のBFFは中継のみ、core-apiserver は自分のDBだけを見る。トークンは認証用の1種類だけになり、チャレンジトークンの取り違え(iss の共用)といった事故の余地も消えた。
- 利用者の手間が減った。アバターを一時的に変更してもらう必要がなくなり、入力1ステップで完了する。
- 正当な利用者が「他人に先に登録された」という理由で締め出されることが無くなった(D1')。

**Cons / トレードオフ**

- **なりすましを防げない。** 他人のプレイヤーIDを自分のアカウントに登録できる。プレイヤーIDは公式サイトで公開されているため、知っていれば誰でも登録できる。
- **称号(デジグネーション)判定で他人のシティリーグ成績が自分のものとして計上される。** `player_id` の重複を許容したため、同じIDを登録した複数のユーザーが同時に同じ成績で判定される。**第5版時点で未対応の唯一の実害**であり、CSP表示(D10)と違って単純に消せない(称号機能そのものが公式結果との突き合わせで成立しているため)。
- **実在しないプレイヤーIDも登録できる。** 公式結果と一致する行が無いだけで、エラーにはならない。
- 誤登録に気づいても1ヶ月は自力で直せず、運用者の手動対応に頼ることになる(D2/D9)。
- CSP表示を失った(D10)。連携して得られる利点が称号判定だけになり、**利用者から見た連携の動機が薄くなった**。

**未決事項**

- 称号判定における成績の誤計上をどこまで許容するか。公式結果との突き合わせを外す、表示側で「自己申告」であることを明示する、といった緩和策は未検討。
- プレイヤーズクラブのブロックが解消した場合に検証を復活させるか。復活させる場合、`player_id` のユニーク制約とCSP表示(D10)も戻すかを併せて判断する必要がある。
- 連携の動機が称号判定のみになったことを踏まえ、機能自体を維持するか。

---

## 7. 既知の問題: プレイヤーズクラブがWAFでブロックされている

本機能は外部サイトを叩かなくなったため影響を受けないが、**同ホストを利用する他のバッチは停止したままである**。

- `import-player-ranking-job` — ランキングの日次取込。第5版でCSP表示を廃止した(D10)ため、`player_rankings` の内容が古くなっても本サービスの表示に影響しなくなった。
- `cmd/sync-pokemon-avatars` — アバター一覧の同期。本機能からは未使用のため実害は無い。

`www.pokemon-card.com`(デッキコード確認・デッキ画像)は正常であり、影響は `players.pokemon-card.com` に限られる。

---

## 8. 検証方法

- **core-apiserver**: `internal/usecase/user_player_test.go` で、①紐付けの作成、②**別ユーザーに登録済みの `player_id` でも作成できること**、③**実在しない `player_id` でも作成できること**、④1ヶ月経過後は旧行を soft delete して作成、⑤同じ `player_id` なら変更不要として既存を返す、⑥ロック期間中は `ErrLocked`、を確認する。`internal/controller/user_player_test.go` でステータスの対応と、**`GET` のレスポンスに `champion_ship_point` / `ranking_date` が含まれないこと**を確認する。
- `go build ./...` / `go vet ./...` / `go test ./...`、`npx tsc --noEmit` / `npx next lint` / `npx next build` を通す。
