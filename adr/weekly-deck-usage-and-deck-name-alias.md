# 週次デッキ使用率の算出と、デッキ名からのスプライト推測

## ステータス

採用 (Accepted) — 2026-07-24

## Context

「対戦環境分析(週次デッキ使用率)」は、プラットフォーム全体のデッキ使用率ランキングを非会員にも公開するレポートである([webapp `/deck_meta`](webapp/src/app/deck_meta/page.tsx))。本ADRは以下の3つを扱う。

1. **使用率ランキングの算出方法** — 何を母数に、どの単位でデッキを同一視して数えるか
2. **スプライト未設定票の救済** — スプライトが登録されていない対戦を、デッキ名から推測して集計に含める仕組み
3. **エイリアス辞書の自動生成** — 推測に使う「デッキ名→スプライト」辞書を、実データの共起から自動で育てる仕組み

### 解決したい問題

デッキの同一視は**スプライト(ポケモンアイコン)の集合**で行っている。デッキ名はフリーテキストで表記ゆれが大きく、集計キーには使えないためである。

しかしこの方式には穴があった。**スプライトが未設定の対戦は指紋が空になり、`total_votes` にも勝敗にも「その他」にも入らず、完全に集計から除外されていた**。相手デッキ名(`matches.opponents_deck_info`)や自分のデッキ名(`decks.name`)は記録されているのに、それらを一切使っていなかったためである。

記録の absolute な量が増えるほど、この取りこぼしはランキングの正確性を損なう。

### 既存コードベースの前提(踏襲する規約)

- **スプライトの識別子は `pokemon_sprites.id`(`VARCHAR(128)`)** で、`'0006'`(リザードン)、`'0006_mega_x'`(メガリザードンX)のような padded 形式。`national_pokedex_no` のような数値IDではない。名前は同テーブルの `name` に持つ([schema.sql](core-apiserver/db/schema.sql))。
- **中間テーブルは `(親ID, position, pokemon_sprite_id)` + `PRIMARY KEY(親ID, position)`** の形。`match_pokemon_sprites` / `deck_pokemon_sprites` が前例。`position` は UI の表示スロット(1枠目/2枠目)に対応する。
- レイヤードアーキテクチャ(gin + gorm): `entity` → `repository`(interface) → `infrastructure` → `usecase` → `controller`。
- バッチは `cmd/<name>/main.go` に置き、`-dry-run` を既定 true にする(`cmd/backfill-tonamel-events` 等)。
- テストは go-sqlmock(`internal/infrastructure/sqlmock_helper_test.go`)。実DB・testcontainers は使わない。
- webapp は Next.js の BFF(`src/app/api/*/route.ts`)経由で中継する。この機能は**非会員も閲覧できるため認証を通さない**。

---

## Decision

### D1. 使用率の母数は「のべ票数」— 1対戦につき自分・相手の2票

1マッチから**2票**を独立に生成する([weekly_deck_usage_stat.go](core-apiserver/internal/infrastructure/weekly_deck_usage_stat.go))。

- **相手側の票**: 相手デッキの指紋に1票。勝敗は `!victory_flg`(記録者が負けた = その指紋が勝った)
- **自分側の票**: 自分デッキの指紋に1票。勝敗は `victory_flg` そのまま

母数 `total_votes` はこの票の総数であり、**対戦数より多くなる**。UIにもその旨を注記している。貢献者数 `contributor_count` は票を投じた実ユーザー数(重複なし)。

集計対象の絞り込みは `records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL`。期間は `records.event_date` の半開区間 `[週の月曜, 翌週の月曜)`。

**`private_flg` はフィルタしない。** 現状すべて true の予約フラグであり、条件に入れると全件が消えるため。

### D2. デッキの同一視は「スプライト指紋」— 名前は使わない

指紋は `NormalizeFingerprint`([fingerprint.go](core-apiserver/internal/infrastructure/fingerprint.go))が決める。

- スプライトIDの**重複を排除**する
- 集計キーは**並び順を無視**する(ソートしてカンマ連結)。登録順の違いで変種が分裂しないため
- 表示用の並びはソートせず、元データの `position` 順を保つ

フリーテキスト(デッキ名)は**指紋の計算に一切使わない**。表記ゆれで同じデッキが無数の変種に割れるのを避けるためである。

### D3. 出現数が閾値未満の変種は「その他」に集約する

`minVariantCount = 5` 未満の変種は、指紋 `""`・スプライト空の1件にまとめる。少数の変種をそのまま公開すると個人が特定されうるため(匿名化・希薄化対策、[DATA_STRATEGY.md](vsrecorder/DATA_STRATEGY.md) 第5章)。

**この「その他」は `total_votes` に含まれる。** 後述する「集計不能で除外される票」とは別物なので混同しないこと。

### D4. 並び順は使用率の降順、同数は勝率の降順

「その他」は常に末尾。サーバ側で確定した順序を webapp 側でも同じ規則で安定ソートし直す([WeeklyDeckUsagePanel.tsx](webapp/src/app/components/organisms/DeckMeta/WeeklyDeckUsagePanel.tsx))。

使用率の分母は UI で切り替えられる。「全体」は `total_votes`、「その他を除く」は `total_votes - その他の件数` をフロント側で再計算する(この表示では「その他」自身は分母から外れるため「集計対象外」と表示する)。

### D5. スプライト未設定の票は、デッキ名から推測して救済する

指紋が空(スプライト0件)の票は、**集計時に動的に**デッキ名から推測する。DBのスプライトテーブルには書き込まない。

```
相手票: match_pokemon_sprites が空 → matches.opponents_deck_info から推測
自分票: deck_pokemon_sprites が空 → decks.name から推測
```

推測は `addVote` の**手前**で解決し、`addVote` 本体と `NormalizeFingerprint` は変更していない。推測できなければ従来どおり除外される(指紋が空のまま)。

**推測対象が1件も無い週では、デッキ名・辞書のクエリを一切発行しない**(遅延ロード)。

`NewWeeklyDeckUsageStat(db)` のシグネチャは変えず、辞書は同じ `*gorm.DB` からリクエスト時にロードする。したがって DI 配線(`cmd/core-apiserver/main.go`)は無変更。

### D6. 名前の正規化は表記ゆれを吸収する

`NormalizeDeckName`([deck_name.go](core-apiserver/internal/infrastructure/deck_name.go))が突合の直前に次を行う。辞書側・デッキ名側の**両方に等しくかかる**。

1. 全角英数→半角、半角カナ→全角カナ(`width.Fold`)。半角濁点は結合文字になるため **NFC で合成**する(`ﾊﾞ`→`バ`)
2. **ひらがな→カタカナ**(U+3041〜U+3096 に +0x60)
3. **漢字とアルファベットを除去**する。「改」「型」「〜版」「ex」のような修飾語のことが多く、残すと「バシャドラ改」と「バシャドラ」が別キーに割れて教師データが薄まるため
4. 上記以外も文字(カナ)と数字だけ残し、空白・記号を除去。長音「ー」は文字扱いで残る

これにより `ロスバレ` / `ろすばれ` / `ﾛｽﾊﾞﾚ` / `ロス バレ` に加え、`バシャドラ改` / `バシャドラ`、`リザードンex` / `リザードン` もそれぞれ同一キーになる。

既知の割り切り:
- **アルファベットのみのデッキ名(`Lost Box` 等)は正規化後に空になり、突合・集計の対象外**になる(英字エイリアスも登録できない)
- 「不明」のような漢字のみの名前も空になり対象外(無意味な名前が落ちるのはむしろ望ましい)
- 繰り返し記号 `ゝゞ` ⇔ `ヽヾ` は非対応。デッキ名での使用はまず無いため許容

### D7. 突合は「部分一致・最長一致1件」

辞書エントリを**文字数の降順**に並べ、正規化したデッキ名に対して最初に部分一致したエントリを採用する。同長はエイリアス昇順で決定的にする。

これにより `リザード` と `リザードン` のような包含関係は長い方が自然に勝つ。正規化後 **2文字未満のエントリは無視**する(`x` 等が全デッキ名に誤爆するため)。

優先順位は **手動エイリアス > 自動エイリアス > `pokemon_sprites.name`(正式名)**。同一の正規化キーは先勝ち。正式名は辞書に無いキーだけ取り込む。

### D8. 辞書は `deck_name_aliases` テーブル、`source` で手動と自動を分ける

```sql
CREATE TABLE deck_name_aliases (
    alias             VARCHAR(256) NOT NULL,
    position          SMALLINT NOT NULL CHECK (position > 0),
    pokemon_sprite_id VARCHAR(128) NOT NULL,
    source            VARCHAR(16) NOT NULL DEFAULT 'manual',  -- 'manual' | 'auto'
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (alias, position),
    FOREIGN KEY (pokemon_sprite_id) REFERENCES pokemon_sprites(id)
);
```

既存の中間テーブルと同型。`alias` は**人間可読のまま**登録してよい(ロード時に正規化される)。1エイリアスにつき代表スプライトは position 1/2 の複数行で表す。

**エントリの追加・修正は INSERT/UPDATE のみで、次のリクエストから反映される(デプロイ不要)。**

### D9. 辞書は実データの共起マイニングで自動生成する

**デッキ名とスプライトを両方登録している記録が、そのまま名前→スプライトの教師データになる。** 外部データも LLM も不要。

処理は4段階([deck_name_alias_generator.go](core-apiserver/internal/infrastructure/deck_name_alias_generator.go))。

```
供給側(教師データ)                        需要側(救済対象)
decks.name × deck_pokemon_sprites    スプライト未設定の票の
opponents_deck_info × match_sprites       デッキ名(頻度順)
        │                                    │
        ▼                                    ▼
「ロスバレ」→ {0487_origin,0225} が 87%  「ロスバレ」×42票が現在除外中
        └────────────┬─────────────────┘
                     ▼
        しきい値を満たせば source='auto' で INSERT
```

1. **供給側の集計** — 名前とスプライトが両方ある**マッチ**を1件として、`正規化名 × 指紋`で数える。集計時と同じ `NormalizeDeckName` / `NormalizeFingerprint` を通すため、正規化が二重管理にならない
2. **需要側の集計** — スプライト未設定で現在除外されている票を、正規化名ごとに数える
3. **候補生成としきい値判定** — 需要の多い名前から順に評価する
4. **書き込み** — `source='auto'` の行だけをトランザクション内で**全削除→再生成**(冪等)

**完全一致しない需要名は、名前の包含関係で教師データへ繋ぐ。** エイリアスは突合時に「エイリアスを含む名前」へヒットするため、教師データの評価範囲も突合の範囲と一致させる。

- **プール評価**: 需要名を含む供給キーを**すべて束ねて**しきい値を判定する(完全一致はこの特殊形)。略称「オロチン」は「オロチンサナ」等の実登録を継ぐ。プールに複数構成が混ざって割れれば、従来どおり占有率不足で保留される(「リザードン」がピジョット型とビーダル型に割れる等)
- **核へのフォールバック**: プールが空なら、需要名に**含まれる**最長の供給キー(核)をエイリアスにする(「マリィノオーロンゲシクボ」→「オーロンゲ」)。複数の需要名が同じ核に合流したら救済見込み票を合算する

**判定は3軸のAND。** それぞれ別のものを測っている。

| 軸 | 既定 | 測っているもの | 落とすケース |
|---|---|---|---|
| 支持(最頻構成の件数) | 10件以上 | データ量が足りているか | 3件だけ同じ構成 = 偶然かもしれない |
| 占有率(最頻構成 ÷ その名前の全件) | 60%以上 | 名前が構成を一意に決めているか | 「ルギア」が2構成に半々 = 名前が曖昧 |
| 実ユーザー数(重複なし) | 3人以上 | 1人の命名癖でないか | 1人が100試合記録して支持を作る |

マッチ単位で数えるため1人が支持を膨らませられる。これを人数の下限で防ぐ考え方は、D3の `minVariantCount` と同じ匿名化の思想である。

代表スプライトは最頻指紋の構成をそのまま採る(`position` の割り当ても最頻のものを採用)。**これにより推測票が実スプライトの変種と同じ指紋に合流し、後述の分裂が緩和される。**

**手動エントリで既に解決できる名前は候補にしない**(人の意図を尊重する)。一方、**正式名でしか解決できない名前は対象に含める** — 正式名の解決は1体止まりで実構成と指紋が分裂しがちなため、代表構成で上書きする。

### D10. 生成バッチは dry-run 既定、しきい値は全てオプション

`cmd/generate-deck-name-aliases`。

```bash
# 候補の確認のみ(既定。DBは変更しない)
go run ./cmd/generate-deck-name-aliases

# 反映する
go run ./cmd/generate-deck-name-aliases -dry-run=false

# しきい値・集計期間を変える
go run ./cmd/generate-deck-name-aliases -min-support=20 -min-ratio=0.7 -supply-weeks=24
```

| オプション | 既定 | 内容 |
|---|---|---|
| `-dry-run` | true | 書き込みせず候補一覧のみ出力 |
| `-supply-weeks` | 12 | 教師データを遡る週数(代表構成を安定させるため長め) |
| `-demand-weeks` | 4 | 救済対象を遡る週数(いま効くものを優先するため短め) |
| `-min-support` | 10 | 代表構成の支持件数の下限 |
| `-min-ratio` | 0.6 | 代表構成の占有率の下限(**割合で指定**) |
| `-min-contributors` | 3 | 代表構成を使った実ユーザー数の下限 |
| `-min-alias-runes` | 4 | 生成するエイリアスの最小文字数(手動運用の下限より保守的に) |

しきい値の指定ミスは「候補0件」という**正常終了に紛れて気づけない**ため、起動時に検証して終了コード1で弾く。とくに占有率はログに `%` で出す一方で指定は割合(`0.6`)のため `60` と書かれやすく、専用のメッセージを出す。

---

## 1. スキーマ (core-apiserver/db/schema.sql)

- `deck_name_aliases` を新設(D8)。`deck_pokemon_sprites` の直後に配置
- `CREATE INDEX idx_deck_name_aliases_source` — 自動生成の全削除→再生成が `source` で絞るため

マイグレーションツールは無いため本番へは手動適用する。**DDL を先に適用してからコードをデプロイすること**(テーブルが無い状態でコードが動くと、推測対象が存在する週の weekly_usage がエラーになる)。

## 2. バックエンド (core-apiserver)

| ファイル | 役割 |
|---|---|
| `internal/infrastructure/weekly_deck_usage_stat.go` | 使用率集計。D1〜D5 |
| `internal/infrastructure/fingerprint.go` | 指紋の正規化。D2 |
| `internal/infrastructure/deck_name.go` | 名前の正規化・辞書ロード・突合。D6/D7 |
| `internal/infrastructure/deck_name_alias_generator.go` | 共起マイニングと書き込み。D9 |
| `cmd/generate-deck-name-aliases/main.go` | 生成バッチ。D10 |
| `internal/infrastructure/model/pokemon_sprite.go` | `PokemonSprite` / `DeckNameAlias` モデル |

API は `GET /api/v1beta/deck_meta/weekly_usage?week=<月曜日YYYY-MM-DD>`(認証なし)。レスポンス形式は本変更で**不変**。

## 3. フロントエンド (webapp)

レスポンス形式が変わらないため、変更は注記1行のみ([WeeklyDeckUsagePanel.tsx](webapp/src/app/components/organisms/DeckMeta/WeeklyDeckUsagePanel.tsx))。

> ※ポケモン未設定の対戦はデッキ名から推測して集計しています

週の既定値は**先週**(今週は記録が途中経過で使用率が変動しやすいため)。週は月曜始まりで、`week` パラメータはその週の月曜日を `YYYY-MM-DD` で表す([week.ts](webapp/src/app/utils/week.ts))。

## 4. Consequences

### 良くなること

- スプライト未設定でデッキ名だけある対戦が集計に入り、ランキングの母数と正確性が上がる
- 辞書が実データから育つため、環境の変化(新アーキタイプの登場、通称の変化)に人手を介さず追従できる
- 代表構成を実データの主流構成に合わせるため、推測票が実スプライトの変種と同じ指紋に合流する

### 受け入れるトレードオフ

- **指紋の分裂は完全には無くならない。** 推測が1体しか特定できない場合(例 `0006`)、実登録の2体構成(`0006,0018`)とは別変種になる。緩和策は辞書に代表2体を持たせることで、D9の自動生成はこれを自動で行う
- **過去週の数字が後から変わる。** weekly_usage はリクエスト時に現在の辞書で全期間を推測するため、辞書が育つと過去週も再計算される。手動運用でも同じ性質だが、自動化で変動頻度が上がる
- **しきい値付近の名前は週によって採否が振れうる。** 気になる場合は「2回連続で同じ結論のときだけ反映」といったヒステリシスを後から足せる
- **辞書は毎リクエスト全ロード。** エイリアス数百＋マスタ数千、名前は週あたり高々数千ユニークを想定しており、メモリ内の部分一致＋キャッシュで足りる。重くなったら Aho-Corasick か起動時キャッシュへ切り替える(その場合のみ `NewWeeklyDeckUsageStat` のシグネチャ変更と DI 1行の修正が必要)

### 運用

- **段階導入を推奨する。** まず `-dry-run` を cron で数週間回して候補レポートだけ観察し、しきい値に納得してから `-dry-run=false` に切り替える
- 誤った推測が混じるなら**占有率**を、母数の薄い候補が気になるなら**支持**を上げる
- 個別に直したいものは `source='manual'` で登録する。自動生成は手動エントリを上書きしない
