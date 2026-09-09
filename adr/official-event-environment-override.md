# 公式イベントの環境の例外(official_event_environments)

大型大会だけ「開催日から引いた環境」と「実際に対戦する環境」がズレる問題への対処。
**なぜそう決めたか**と**どう動くか**の両方をまとめる。

## 背景

環境(`environments`)は `from_date`〜`to_date` の期間で定義され、対戦や公式イベントの日付が
どの期間に入るかで決まる。この前提はほとんどのイベントで正しい。

しかし大型大会(チャンピオンズリーグ・PJCS)は、開催日時点で発売済みの最新弾がカードプールに
入らないことがある。例えばチャンピオンズリーグ2027横浜(2026-09-20〜22)は、30th CELEBRATION
(`m6a`, 2026-09-16〜)の発売後に開催されるが、使えるカードはストームエメラルダ(`m6`)まで。
同じ日のジムバトルは `m6a` なので、「日付 → 環境」の対応そのものが同日内で分岐する。

期間の定義を変えて解決することはできない。同じ 9/20 に `m6` のCLと `m6a` のジムバトルが
並存するため、日付だけでは区別できないからである。

## 決定

ズレるイベントだけを列挙する例外テーブル `official_event_environments` を置き、
環境の判定を通る全経路(イベント表示・統計・環境バッジ)をその例外を見る実装に統一する。

```sql
CREATE TABLE official_event_environments (
    official_event_id INT        NOT NULL PRIMARY KEY REFERENCES official_events (id),
    environment_id    VARCHAR(8) NOT NULL REFERENCES environments (id)
);
```

判断の理由:

- **`official_events` に列を足さない。** `import-officialevent-bat` が gorm の `Save` で
  行を丸ごと上書きするため、次回のインポートで消えてしまう。
- **ルール化(「大型大会は開催日のN日前の環境」など)はしない。** 実際のカードプールは
  大会ごとに公式が指定するもので、規則から導くと誤判定する。手で登録するほうが確実で、
  対象は年に数件しかない。
- **`records` に環境を焼き込まない。** `environments` は次の弾が出るまで `to_date` が
  暫定値のことがあり(`usecase/deck_code_post.go` の resolveEnvironment 参照)、焼き込むと
  期間の修正のたびに全記録の再計算が要る。判定は都度行い、例外だけを持つ。

## どう動く

### 環境の判定(単発) — 環境バッジ

`usecase.ResolveEnvironmentForOfficialEvent` が唯一の入口。例外登録があればその環境、
無ければ従来どおり基準日時(`RecordBasisTime`)から引く。

環境バッジは公式イベントに紐づく記録だけが対象なので(`usecase/match.go`)、
判定に必要な `official_event_id` は親recordから常に得られる。

例外を後から追加・修正したときは `cmd/backfill-user-environment-badges` を再実行すると
付与済みのバッジを新しい判定で付け直せる(判定はこの関数に寄せてある)。

### 環境の判定(集合) — 統計

統計は「環境 → 期間」に潰して `records.event_date` で絞っていたが、上記のとおり環境は
期間だけでは表せない。`repository.StatPeriod` に例外を持たせ、`applyStatPeriod` が
次の条件を組み立てる。

```sql
(event_date >= From AND event_date < To AND official_event_id NOT IN (Exclude))
OR (official_event_id IN (Include) AND event_date >= BaseFrom AND event_date < BaseTo)
```

- `Include`: 開催日が期間外でも、その環境として集計するイベント
- `Exclude`: 開催日が期間内でも、別の環境として登録されているイベント
- `BaseFrom`/`BaseTo`: **環境以外の条件**(week / year_month / season / standard_regulation)
  だけで決まる期間。環境と他の条件は交差を取る仕様なので、拾い直す例外イベントも他の条件は
  満たしている必要がある(`year_month=2026-10` と `environment=m6` を同時に指定したとき、
  9月開催のCLを拾ってはいけない)。このため `buildStatPeriod` は環境を**最後に**適用する。

実装上の注意:

- gorm の `IN ?` に空スライスを渡すと `IN (NULL)` になり常に偽になる。Include/Exclude が
  空のときは節ごと落とす(`statPeriodIncludeClause` が空文字を返す)。
- `NOT IN` は NULL に対して不定になるため、公式イベントに紐づかない記録を落とさないよう
  `official_event_id IS NULL` を明示的に許可する。
- 期間の指定が無い(全期間)ときは例外も当然含まれるので、条件を一切付けない。ここでORを
  足すと「例外イベントだけ」に絞られてしまう。

### イベント表示

`internal/infrastructure/official_event.go` のJOINで解決する。

```sql
LEFT JOIN official_event_environments AS oee ON oee.official_event_id = official_events.id
LEFT JOIN environments ON environments.id = COALESCE(
    oee.environment_id,
    (SELECT e.id FROM environments AS e WHERE e.from_date <= official_events.date AND e.to_date >= official_events.date)
)
```

例外の有無をORで分岐させず相関サブクエリにしているのは、`official_events` が100万行を
超えており、ORのJOIN条件だとプランが崩れて `FindByShopIds`(ホームのパネルが全ログイン
ユーザ分叩く)の走査量が戻ってしまうため。`environments` は数十行なのでサブクエリ自体の
コストは無視できる。

シティリーグ結果(`internal/infrastructure/cityleague_result.go`)も同じ形にしてある。
こちらは `official_events` 側の行が欠けていても環境名を出すため日付の基準が
`cityleague_results.event_date` だが、例外の引き方(`official_event_id` で登録を探し、
無ければ日付)は同じ。今はシティリーグにズレる大会は無いが、発売日当日開催などで
同じ状況が起きたときに登録だけで対応できる。

### 直近N戦の環境ラベル

`usecase/user_stat_recent.go` は対戦ごとに環境ラベルを付ける。ここも例外を優先し、
無ければ対戦日から引く。例外テーブルは極小なので全件を1回引き(`FindAll`)、
該当する対戦があったときだけ環境を引き当てる(環境IDごとに1回)。

### OpenSearch への焼き込み(バトラボ)

`import-cityleague-result-opensearch` は `cityleague_results` インデックスに
`environment_id` を焼き込み、バトラボ(vslab)がそれで絞り込む。ここは以前
`/environments?date=` を叩いて開催日から引いていたが、同じバッチが公式イベントAPIも
呼んでいるため、**その応答の `environment_id` をそのまま使う**ように変えた
(APIは例外を反映済み)。API呼び出しも1つ減る。

一度焼き込んだドキュメントは日付では直らないので、例外を後から登録したときは
該当開催回を取り込み直す。

## 環境の判定を持つ箇所の棚卸し

例外を効かせる必要があるのは「公式イベントに紐づく環境」だけである。日付から引くのが
正しい箇所まで変えないよう、棚卸しの結果を残す。

例外を見る(対応済み):

| 箇所 | 用途 |
| --- | --- |
| `infrastructure/official_event.go` | イベント一覧・詳細・Myジム・カレンダーの環境 |
| `infrastructure/cityleague_result.go` | シティリーグ結果の環境 |
| `usecase/environment_badge_evaluation.go` | 環境バッジ(記録時) |
| `cmd/backfill-user-environment-badges` | 環境バッジ(遡り) |
| `usecase/{deck,opponent_deck}_usage_stat.go`・`user_stat.go` | 統計の環境フィルタ |
| `usecase/season.go` の `StatPeriodFor` | バッチの環境フィルタ |
| `usecase/user_stat_recent.go` | 直近N戦の環境ラベル |
| `import-cityleague-result-opensearch` | OpenSearchへの焼き込み |

日付から引くのが正しい(変えない):

| 箇所 | 理由 |
| --- | --- |
| `GET /environments?date=`(webappのヘッダー「現在の対戦環境」) | 今日の環境であって、イベントの環境ではない |
| webapp の自由形式イベント(`UnofficialEventInfo.tsx`) | 公式イベントに紐づかない |
| `usecase/deck_code_post.go` の `resolveEnvironment` | 投稿の公開日で決まる。イベントと無関係 |
| `usecase/environment_badge.go`(バッジ一覧) | 保存済みの `user_environment_badges` を読むだけ |

未対応(現時点では実害なし):

- webapp の `/cityleague_results/environments/[id]` は環境の `from_date`〜`to_date` で
  シティリーグ大会を引く。シティリーグに例外を登録したときだけズレる(今は0件)。
  対応するなら core-apiserver 側に環境指定の絞り込みを足すことになる。

## 運用

新しい大型大会が登録されたら、カードプールが開催日の環境と異なる場合だけ
`db/schema.sql` に INSERT を追記し、本番へ同じSQLを流す(マイグレーションツールは無い)。

対戦記録が作られる前に登録できれば、バッジ・統計・表示のすべてが最初から正しくなる。
記録が作られた後に登録した場合、統計と表示は都度クエリなので即座に追従するが、
環境バッジだけは記録時に確定しているため `cmd/backfill-user-environment-badges` の
再実行が要る。

同一大会でも日程ごとに別イベントとして登録されるため、**イベントIDを1件ずつ列挙する**。
公式の登録が後から増えること(決勝日程の追加など)があるので、大会終了までは追加分がないか
確認する。
