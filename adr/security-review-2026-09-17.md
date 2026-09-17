# セキュリティ精査レポート（第2回） — core-apiserver

- **対象**: vsrecorder/core-apiserver (main)
- **作成日**: 2026-09-17
- **範囲**: 前回（[security-review-2026-07-18.md](security-review-2026-07-18.md)）以降に増えた機能
  （push通知・みんなの公開デッキ・通知・活動ログ・きずな 等）と、既存エンドポイントの認可・
  参照整合性・DoS耐性（Goファイル 635）

## サマリ

| 区分 | 件数 | 内訳 |
| --- | --- | --- |
| 修正済み | 11 | 高 2 / 中 4 / 低 5（第2回精査の4件を含む） |
| 未対応（要判断） | 2 | 期間指定一覧の上限 / デッキコード作成の頻度制限（第2回精査） |
| 仕様として確認したい（未対応） | 2 | 公開統計の集計範囲 / CORSのlocalhost |
| 精査して問題なし | 8 | SQLi・タグ所有者・並び替え・通知/push計測の所有者条件 ほか |

前回入れた防御（JWT鍵検証・exp必須・タイムアウト・ボディ上限・CORS・SQLパラメータ化）は
維持されていることを確認した。`go vet` 警告なし、`govulncheck` は到達可能な既知脆弱性なし。

---

## 修正済み（7件）

### 1. [高] 記録一覧の `deck_id` 絞り込みで、他人の非公開記録まで読めた

認証済みユーザーが `GET /records?deck_id=<他人のデッキID>` を叩くと、そのデッキに紐づく記録が
所有者・`private_flg` を問わず返っていた。ハンドラは `deck_id` があると自分の `uid` を使わず、
リポジトリも `deck_id = ?` だけで絞っていた。デッキIDは公開デッキ一覧やみんなの公開デッキの
応答（`deck_id`）から誰でも取得できるため、memo や tcg_meister_url を含む非公開記録が漏れる。

- **対処**: `RecordInterface.FindByDeckId` / `FindByDeckIdOnCursor` / `FindByDeckCodeId` に
  `uid` を追加し、クエリで `user_id = ?` を必ず含める（認可はクエリで完結させる）。
  ハンドラは認証済みの `uid` を渡す。デッキ／デッキコードの削除可否（「記録に使用中」の判定）も
  所有者自身の記録に限った（他人の記録参照で削除を妨害できないようにするため。3も参照）。
- **箇所**: `domain/repository/record.go`, `infrastructure/record.go`, `usecase/record.go`,
  `controller/record.go`, `auth/authorization/deck.go`, `auth/authorization/deck_code.go`

### 2. [高] `GET /matches` が全ユーザーの対戦をメモ付きで返していた

認証必須だが `uid` を使わず、`deleted_at IS NULL` だけで全ユーザーの最新対戦を返していた。
非公開記録に属する対戦、対戦メモ・ゲームメモ・対戦相手ID・タグまで含まれていた。

webapp はこの応答を「相手デッキの入力候補（自分の対戦がまだ無い人向けのダミー候補）」にだけ
使っており（`buildDeckHistories` が見るのは `opponents_deck_info`・スプライト・不戦勝/不戦敗フラグ）、
横断の一覧自体は必要な機能だった。

- **対処**: リポジトリ側で `records.private_flg = false AND records.deleted_at IS NULL` の記録に
  属する対戦だけを対象にし、usecase 側で本人向けの項目（memo・games.memo・opponents_user_id・
  deck_id・deck_code_id・tags）を落として返す（`sanitizeMatchForPublicFeed`）。
- **箇所**: `infrastructure/match.go`, `usecase/match.go`
- **その後**: 候補の用途に対戦結果を返す必要は無いため、「表記 × スプライト」の出現回数だけを
  返す専用のエンドポイント `GET /matches/opponent_deck_candidates` を追加した（記録の公開・
  非公開を問わず集計する。[opponent-deck-candidates.md](opponent-deck-candidates.md)）。
  `GET /matches` は webapp が切り替えるまで残し、その後に削除する。

### 3. [中] 作成・更新時に他人のデッキ・デッキコード・記録を参照できた

リクエストの `deck_id` / `deck_code_id` / `record_id` は認可ミドルウェアを通らない
（ミドルウェアが見るのはパスの `:id` だけ）ため、次の書き込みが可能だった。

- `POST /deck_codes`：他人のデッキにデッキコードを追加。公開デッキ一覧の「最新デッキコード」は
  `user_id` を問わず `created_at` 最新を採るため、他人のデッキの表示（コード・メモ）を差し替えられた。
- `POST/PUT /records`：他人のデッキ・デッキコードを参照。「記録に使用中」として相手がデッキを
  削除できなくなる。
- `POST/PUT /matches`：他人の記録に対戦結果を混入。`GET /records/:id/matches` や対戦サマリに反映される。

- **対処**: `usecase/ownership.go` に所有者検証を集約し、DeckCode.Create / Record.Create・Update /
  Match.Create・Update で保存前に検証する。他人のもの・存在しないものはどちらも
  `apperror.ErrRecordNotFound`（IDの存在を教えない。`DeckCodePost.Publish` と同じ方針）とし、
  コントローラは 404 を返す。Match.Create は親 record を冒頭で1度だけ取得し、所有者検証と
  環境バッジ判定の両方に使う。
- **箇所**: `usecase/ownership.go`, `usecase/deck_code.go`, `usecase/record.go`, `usecase/match.go`,
  `controller/deck_code.go`, `controller/record.go`, `controller/match.go`, `cmd/core-apiserver/main.go`

### 4. [中] `limit` に上限が無く、未認証で OOM を誘発できた

`helper.ParseQueryLimit` に上限が無く（みんなの公開デッキ系の50件だけ独自に上限あり）、
未認証の `GET /records` や `GET /decks` に巨大な `limit` を渡すと全件をメモリに載せていた。
コンテナは `mem_limit: 128m` なので OOM Kill と再起動を繰り返させられる。

- **対処**: `helper.MaxLimit = 100` を設け、超過は上限へ丸める（0以下を既定値へ寄せるのと同じ扱い）。
  100 は webapp が使う最大値（相手デッキの入力候補の取得 `?limit=100`）。
- **箇所**: `controller/helper/parse.go`

### 5. [低] デッキコードと Tonamel の大会IDの文字種を検証していなかった

デッキコードは長さ（21文字以下）だけの検証で、公式サイトのURLとオブジェクトストレージのキーに
そのまま埋め込まれていた。`../` や `?` を含む値で公式サイトの別ページを取得し、公開読み取りの
バケットへ保存させられる。Tonamel の大会IDもパスパラメータをそのままURLへ連結しており、
`records.tonamel_event_id` は VARCHAR(8) のため長い値は保存時に 500 になっていた。
どちらも接続先ホストは固定で、影響は限定的。

- **対処**: `entity.IsValidDeckCodeFormat`（英数字とハイフンのみ）と `entity.IsValidTonamelEventId`
  （英数字1〜8文字）を追加し、デッキ／デッキコード作成・記録作成/更新（controller と usecase の両方）・
  `GET /tonamel_events/:id`（新設の `validation.TonamelEventGetByIdMiddleware`）で検証する。
  併せて URL への埋め込みを `url.PathEscape` 経由にした（検証済みの値では変化しない多層防御）。
- **箇所**: `domain/entity/deck_code.go`, `domain/entity/tonamel_event.go`, `validation/deck.go`,
  `validation/deck_code.go`, `validation/record.go`, `validation/tonamel_event.go`,
  `controller/tonamel_event.go`, `infrastructure/deck_asset.go`, `infrastructure/tonamel_event.go`

### 6. [低] レートリミッタのキーが解放されなかった

`ratelimit.Limiter` は一度使われた UID のエントリを削除せず、二度と来ないキーが残り続けた。
キーは認証済み UID なので悪用は難しいが、128MiB のコンテナで長期稼働すると効いてくる。

- **対処**: ウィンドウごとに1回、ウィンドウ内に試行の無いキーをまとめて削除する
  （保持するキーは「直近2ウィンドウで試行があったもの」に収まる）。時計を差し替えられるようにして
  テストで検証。
- **箇所**: `ratelimit/ratelimit.go`

### 7. [低] push 購読の endpoint を、鍵を知らない第三者が奪えた

`push_subscriptions.endpoint` は全体で一意で、`Upsert` が `user_id` と鍵を無条件に上書きしていた。
同じ端末でのアカウント切替（webapp はログアウト時に購読を解除せず、ブラウザに残っている購読を
そのまま次のアカウントで登録し直す）を成立させるための動作だが、endpoint を知る第三者が
自分のトークンで同じ endpoint を登録すると、元の持ち主のその端末への通知を止められた
（通知の中身は鍵が違うため読めない）。endpoint は API が返さない高エントロピーの秘密URLで、
漏れるのは端末や DB に触れる状況に限られるため現実的なリスクは低い。

- **対処**: 持ち主の変更は「鍵（p256dh / auth）が一致するとき」だけ許す
  （`usecase.canTakeOverPushSubscription`）。ブラウザの購読が同じなら鍵も同じなので、同じ端末での
  切替は通り、endpoint しか知らない第三者は 409 で弾かれる。本人による登録し直しは常に許す。
- **箇所**: `usecase/push_subscription.go`, `domain/apperror`, `controller/apierror`,
  `controller/push_subscription.go`

---

## 第2回精査（同日）で修正した4件

1〜7 の修正後に、残りの面（非公開データの派生・外部通信の乱用・入力の上限・認証の前提）を
精査して見つかったもの。

### 8. [中] 非公開デッキのデッキコードが誰でも読めた

`GET /decks/:id/deck_codes` と `GET /deck_codes/:id` は任意認証のみで、親デッキの `private_flg` を
見ていなかった。非公開デッキのIDは公開記録の応答（`deck_id` / `deck_code_id`）から誰でも取得でき、
コード非公開フラグの無いデッキコード本体・メモ・user_id が返っていた。デッキ本体
（`GET /decks/:id`）は 403 なのに、その派生データだけ読める不整合。

- **対処**: デッキ別一覧に既存の `DeckGetByIdAuthorizationMiddleware` を付け、個別取得には
  親デッキを引いて公開範囲を判定する `DeckCodeGetByIdAuthorizationMiddleware` を新設した
  （対戦→記録の `MatchGetByIdAuthorizationMiddleware` と同じ形）。非公開デッキは他人・未認証に 403。
- **箇所**: `auth/authorization/deck_code.go`, `controller/deck_code.go`

### 9. [中〜低] `GET /tonamel_events/:id` が未認証の外部取得プロキシになっていた

usecase は HTTP 取得しか行わず、記録作成時に保存している `tonamel_events` を参照しなかった。
毎回 tonamel.com へ取りに行き（最大10秒）、応答本文を上限なしで `html.Parse` に渡していた。
大量に叩かれると外向き接続と goroutine が滞留し、Tonamel 側から遮断されれば記録作成の連携も止まる。
nginx 越しで送信元IPは1つなので API 側の IP 制限は効かない。

- **対処**: 保存済み（`tonamel_events`）を先に引き、無いときだけ取得して保存する。本文は
  `io.LimitReader`（2MiB）で打ち切り、tonamel.com への同時取得数をプロセス全体で 4 に制限して、
  超えたぶんは待たせずに 503 で断る（`apperror.ErrExternalFetchBusy`）。記録作成側の取得は
  失敗を許容しているので影響しない。
- **箇所**: `usecase/tonamel_event.go`, `infrastructure/tonamel_event.go`, `controller/tonamel_event.go`

### 10. [低] 認証ミドルウェアがユーザーの存在・退会を見ていなかった

署名と exp だけを検証し、退会（論理削除）済み uid のトークンでも `POST /records` などの書き込みが
通っていた（410 を返すのは `POST /users` だけ）。webapp は Firebase 側の消し損ねを前提に 30 分間隔の
退会チェックを持つが、その間や API を直接叩く経路では書き込め、「退会したユーザのデータを残さない」が崩れる。

- **対処**: 認証ミドルウェアが `authentication.UserVerifier` に uid の状態を問い合わせ、未登録・
  退会済みは 401（`apierror.ErrUnregisteredUser`）にする。実装は `usecase.ActiveUserVerifier`
  （有効と確認した結果を1分だけ保持、否定は保持しない）で、main が `SetUserVerifier` で注入する。
  未設定なら 500（fail closed）。登録前の uid を通す必要がある `POST /users` だけ
  `RegistrationAuthenticationMiddleware` を使う。
- **箇所**: `auth/authentication/authentication.go`, `usecase/active_user_verifier.go`,
  `controller/user.go`, `cmd/core-apiserver/main.go`

### 11. [低] 文字列IDの長さ未検証で 500 になっていた

`friend_id`（32）、`unofficial_event_id`・`deck_id`・`deck_code_id`・`record_id`（26）、
`opponents_user_id`（32）の超過、存在しない `pokemon_sprite_id`（外部キー）、スプライトの
`position` 重複（主キー）は、400 ではなく DB エラーの 500 になっていた。

- **対処**: validation で列幅と形式を検証する（`isValidOptionalId` / `validatePokemonSprites`）。
  形式は正しいが存在しないスプライトIDは、保存時の外部キー違反を `wrapForeignKeyViolation` で
  `apperror.ErrInvalidReference` に変換し、コントローラが 400 にする。
- **箇所**: `validation/util.go`, `validation/pokemon_sprite.go`, `validation/record.go`,
  `validation/match.go`, `validation/deck.go`, `infrastructure/util.go`, `infrastructure/match.go`,
  `infrastructure/deck.go`, `controller/deck.go`, `controller/match.go`

---

## 未対応（要判断）（2件）

### B'. [中] 期間指定の公開一覧に件数上限も期間上限も無い

`GET /cityleague_results?from_date&to_date` は範囲内の全行を LIMIT 無しで返す
（`infrastructure/cityleague_result.go`）。大会×入賞者の行なので、数年分を指定すると数万〜数十万行に
なり得る。`GET /official_events?start_date&end_date` も同様。どちらも未認証で叩けるため、
4 の `limit` と同じ理屈でコンテナのメモリ上限を超えさせられる。

- **推奨**: validation で期間の上限（例: 92日）を設ける。

### D'. [低] デッキコード作成の外部取得とストレージ書き込みに頻度制限が無い

1回の作成で公式サイトへ2回取得し、S3 へ公開読み取りのオブジェクトを2つ書く。認証済みなら無制限に
呼べ、他人の公開デッキコードは無数にあるので、外向き通信とストレージ費用を膨らませられる。

- **推奨**: uid ごとの `ratelimit`（例: 60回/時）を `DeckCodeCreateMiddleware` / `DeckCreateMiddleware`
  （コード付き）に入れる。

---

## 仕様として確認したい（未対応）（2件）

### A. 公開統計は非公開記録も集計に含む

`GET /users/:id/stats` などは認証不要で、`ignore_stats_flg` は見るが `private_flg` は見ない。
集計値なので実害は小さいが、「非公開」の意味と一致しているかは要確認。

### B. CORS の許可オリジンに `http://localhost:3000`

`AllowCredentials: false` かつ Bearer 認証なので実害はない。

---

## 精査して問題なしと確認（8件）

- **変更系エンドポイントの認証**: すべて `RequiredAuthenticationMiddleware` 付き。
- **JWT**: 空鍵拒否・HMAC限定・issuer・exp 必須を維持。
- **SQLインジェクション**: `Raw` / `Exec` / `gorm.Expr` はすべてプレースホルダかコード内定数のみ。
  店舗検索の LIKE はワイルドカードをエスケープ済み。
- **タグの所有者**: `FindAttachableByIds` で本人のタグとプリセットだけに絞る。
- **対戦の並び替え**: `record_id` で範囲を固定し、他人の対戦の position は変えられない。
- **通知・push 到達計測**: 更新条件に `user_id` を含め、他人の行は 404 になる。
- **みんなの公開デッキ**: 公開時に本人のコードとデッキであることを確認。取り下げ・非表示投稿の
  いいね一覧を返さない。push endpoint の検証で IP 直指定・localhost・http を拒否（SSRF 抑止）。
- **秘密情報**: `.env` と `bin/` は `.gitignore` と `.dockerignore` の両方に登録済み。
  Slack webhook は環境変数のみ。前回の未対応3件（sslmode / 鍵共有 / インメモリ制限）は状況に変化なし。

---

## 検証

- `make test`（`go mod tidy` → `make lint-tx` → UTC と Asia/Tokyo で `go test -race ./...`）通過。
- `go vet ./...` 警告なし。`govulncheck ./...` 到達可能な既知脆弱性なし。
- 追加したテスト: `usecase/ownership_test.go`（所有者検証・公開フィードの秘匿）、
  `ratelimit/ratelimit_test.go`、`entity/deck_code_test.go`・`entity/tonamel_event_test.go`、
  `validation/deck_code_format_test.go`・`validation/tonamel_event_test.go`、
  `controller/*_ownership_test.go`、`infrastructure/record_by_deck_test.go`。
- 第2回精査で追加したテスト: `auth/authorization/deck_code_get_test.go`、
  `auth/authentication/user_verifier_test.go`、`usecase/active_user_verifier_test.go`、
  `usecase/tonamel_event_test.go`（保存優先）、`infrastructure/tonamel_event_test.go`（上限）、
  `validation/pokemon_sprite_test.go`・`validation/reference_id_test.go`、
  `infrastructure/util_test.go`、`controller/invalid_reference_test.go`。
  コントローラのテストは `controller/main_test.go` の `TestMain` で全 uid を有効として通す。
