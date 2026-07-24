# 対戦環境分析(週次デッキ使用率)のアルゴリズム

週次デッキ使用率の集計・デッキ名からのスプライト推測・エイリアス辞書の自動生成の
各アルゴリズムをまとめる。**なぜそう決めたか**は ADR
([adr/weekly-deck-usage-and-deck-name-alias.md](../adr/weekly-deck-usage-and-deck-name-alias.md))を参照。
本書は**どう動くか**の仕様書である。

## 全体像

```
[記録データ]                          [エイリアス辞書]
matches / records                     deck_name_aliases (manual/auto)
match_pokemon_sprites                 pokemon_sprites (正式名)
deck_pokemon_sprites / decks.name          ▲
        │                                  │ 週次バッチが共起マイニングで
        ▼                                  │ source='auto' を再生成
┌─────────────────────────┐          ┌────┴─────────────────────┐
│ 週次集計 (リクエスト時)   │◀─────────│ cmd/generate-deck-name-  │
│ FindWeeklyDeckUsageStat  │  推測に使用│ aliases (週次バッチ)      │
└─────────────────────────┘          └──────────────────────────┘
        │
        ▼
GET /api/v1beta/deck_meta/weekly_usage?week=YYYY-MM-DD (公開・認証なし)
```

---

## 1. 週次デッキ使用率の集計

実装: `internal/infrastructure/weekly_deck_usage_stat.go`

### 1.1 対象データ

- `matches JOIN records` で対象週の全マッチ(1マッチ=1行)
- 除外: `records.deleted_at IS NULL` / `records.ignore_stats_flg = false` / `matches.deleted_at IS NULL`
- 期間: `records.event_date` の半開区間 `[週の月曜, 翌週の月曜)`。`week` パラメータは月曜日の `YYYY-MM-DD`
- `private_flg` はフィルタしない(全件 true の予約フラグのため)

### 1.2 票の生成 — 1マッチ = 最大2票

| 票 | スプライト源 | 勝敗 |
|---|---|---|
| 相手側 | `match_pokemon_sprites` | `!victory_flg`(記録者が負けた = 相手の勝ち) |
| 自分側(deck_id があるときのみ) | `deck_pokemon_sprites` | `victory_flg` |

母数 `total_votes` は有効票の総数(**対戦数より多くなる**)。`contributor_count` は
票を投じた実ユーザー数(重複なし)。

### 1.3 スプライト未設定票の推測フォールバック

スプライトが空の票は、デッキ名から推測して救済する(**集計時に動的、DBには書かない**)。

```
相手票: match_pokemon_sprites が空 → matches.opponents_deck_info を突合(§3)
自分票: deck_pokemon_sprites が空 → decks.name を突合(§3)
```

推測できなければその票は除外される(total_votes にも勝敗にも入らない)。
推測対象が1件も無い週では、デッキ名・辞書のクエリを一切発行しない(遅延ロード)。

### 1.4 指紋(fingerprint) — 変種の同一視キー

実装: `NormalizeFingerprint` (`fingerprint.go`) + `addVote` の前処理

1. **position 1/2 のスプライトのみ使う**(3体目以降は無視)。表示が2枠のため、
   見えないスプライトが指紋を分けると「見た目が同じ行」が複数並んでしまう
2. スプライトIDの重複を排除する
3. IDをソートしてカンマ連結 → これが指紋(**並び順に依存しない**)
4. 指紋が空(スプライト0体)の票は集計から除外する

表示用のスプライト列は position ASC 順を保ち、ID重複だけ排除する。

### 1.5 「その他」への集約

出現数が `minVariantCount`(現在 **3**)未満の変種は、指紋 `""`・スプライト空の
「その他」1件に集約する(匿名化・希薄化対策)。

- 「その他」の count/wins は **total_votes に含まれる**(除外票とは別物)
- 集約した個別変種は `Members` に元の順序のまま保持し、UI のアコーディオンで内訳表示できる

### 1.6 並び順と使用率

- 並び: count 降順 → 同数は勝率降順(安定ソート)。「その他」は常に末尾
- `usage_rate = count / total_votes`
- webapp 側は同じ規則で再ソートし、分母を「全体 / その他を除く」で切替表示する
  (`WeeklyDeckUsagePanel.tsx`)

---

## 2. デッキ名の正規化 — NormalizeDeckName

実装: `internal/infrastructure/deck_name.go`

辞書側・デッキ名側の**両方に等しく**かかる(教師データ集計も同じ関数を使う)。

| # | 処理 | 例 |
|---|---|---|
| 1 | 全角英数→半角、半角カナ→全角カナ(`width.Fold`)、NFC合成 | `ﾛｽﾊﾞﾚ` → `ロスバレ` |
| 2 | ひらがな→カタカナ(U+3041〜U+3096 に +0x60) | `ろすばれ` → `ロスバレ` |
| 3 | **漢字・アルファベットを除去**(修飾語のことが多いため) | `バシャドラ改` → `バシャドラ`、`リザードンex` → `リザードン` |
| 4 | 残りも文字(カナ)と数字のみ残し、空白・記号を除去。長音「ー」は残る | `ロス バレ!` → `ロスバレ` |

既知の割り切り:

- アルファベットのみの名前(`Lost Box` 等)は空になり集計対象外(英字エイリアスも不可)
- 漢字のみの名前(`不明` 等)も空になり対象外
- 繰り返し記号 `ゝゞ`⇔`ヽヾ` は非対応

---

## 3. 突合(マッチャ) — デッキ名 → 代表スプライト

実装: `loadDeckNameMatcher` / `guess` (`deck_name.go`)

### 3.1 辞書の構築(リクエスト時に全ロード)

1. `deck_name_aliases` を全件読み、生エイリアス単位で position ASC のスプライト列に束ねる。
   **position > 2 の行は読み込まない**(表示スロットは2枠)
2. エイリアスを正規化(§2)してキー化。正規化後2文字未満は無視(誤爆防止)。
   同一キーの衝突は **alias 昇順の先勝ち**
3. `pokemon_sprites.name`(正式名)を、辞書に無いキーだけ1体エントリ(position 1)として自動追加

優先順位: **手動エイリアス > 自動エイリアス > 正式名**(同一キーは先勝ち。
手動と自動は同じテーブルで、生成バッチが手動と同名の自動行を作らないことで保証)

### 3.2 突合

1. デッキ名を正規化(§2)
2. エントリを**文字数降順**(同長はエイリアス昇順)に並べ、
   `strings.Contains(正規化名, エイリアス)` が最初に成立した1件を採用(**部分一致・最長一致**)
3. ヒットしなければ nil(その票は除外)
4. 結果は名前単位でキャッシュ(週内は同名が多いため)

辞書の追加・修正(INSERT/UPDATE)は**次のリクエストから反映**(デプロイ不要)。

---

## 4. エイリアス辞書の自動生成(共起マイニング)

実装: `internal/infrastructure/deck_name_alias_generator.go` / `cmd/generate-deck-name-aliases`

**「デッキ名とスプライトを両方登録している記録」を教師データとして、
名前ごとの代表スプライト構成を求め、スプライト未設定で除外されている名前ぶんだけ
エイリアスを生成する。** 外部データ・LLM は不要。

### 4.1 教師データ(供給側)の集計

対象(期間: `-supply-weeks`、既定12週):

| 経路 | 名前 | スプライト |
|---|---|---|
| 相手デッキ | `matches.opponents_deck_info` | `match_pokemon_sprites` |
| 自分デッキ | `decks.name` | `deck_pokemon_sprites` |

- 1件 = **1マッチ**(JOIN で名前・スプライトが揃うものだけ)
- `string_agg` で `"1:0006,2:0018"` 形式(layout)に畳み、Go 側で正規化名(§2)×指紋(§1.4)に集計。
  **position > 2 は指紋にも代表構成にも含めない**
- 集計値: `stats[正規化名][指紋] = {件数, 実ユーザー集合, layoutの内訳}`

### 4.2 需要(救済対象)の集計

スプライト未設定で現在除外されている票を、正規化名ごとに数える(期間: `-demand-weeks`、既定4週)。

### 4.3 候補の評価 — 需要の多い名前から順に

各需要名 D について:

```
1. 前処理での落選:
   - 正規化後の文字数 < MinAliasRunes(既定4) → too_short
   - 手動辞書で既に解決できる            → manual_exists
2. エイリアス A と教師データのプールを決める:
   a. A = D として、「D を含む供給キー」をすべて束ねたプールで評価する
      (完全一致はこの特殊形。略称「オロチン」は「オロチンサナ」等の実登録を継ぐ)
   b. プールが空なら、D に含まれる最長の供給キー(核)へフォールバック:
      A = 核(「マリィノオーロンゲシクボ」→「オーロンゲ」)
   c. どちらも無ければ → no_supply
3. 同じ A へ合流した需要名は救済見込み票を合算する(候補は1つ)
4. プールの最頻指紋(best)に対して3軸のしきい値を判定(AND):
   - best.count      < MinSupport(既定10)      → low_support
   - best.count/total < MinRatio(既定0.6)      → low_ratio
   - bestの実ユーザー数 < MinContributors(既定3) → few_contributors
5. 代表スプライト = best の最頻 layout(position 1/2 のみ、最大2体)
```

プールの範囲(「A を含む名前」)は突合(§3.2)がヒットする範囲と一致しており、
「そのエイリアスが救済する名前群の実登録」がそのまま代表構成の根拠になる。

落選した名前は理由・診断値つきで返し、`-show-rejected` で表示できる
(「教師データなし」で票の多い名前 = 手動登録の最優先候補)。

### 4.4 書き込み(冪等)

1トランザクションで `source='auto'` の行を**全削除→候補で再生成**。
`source='manual'` は読むだけで書き換えない(alias 完全一致の候補は取り込まない)。

### 4.5 バッチの使い方

```bash
go run ./cmd/generate-deck-name-aliases                  # dry-run(既定): 候補確認のみ
go run ./cmd/generate-deck-name-aliases -dry-run=false   # 反映
go run ./cmd/generate-deck-name-aliases -show-rejected   # 落選名も理由つきで表示
```

| オプション | 既定 | 内容 |
|---|---|---|
| `-dry-run` | true | 書き込みせず候補一覧のみ |
| `-supply-weeks` / `-demand-weeks` | 12 / 4 | 教師データ / 救済対象を遡る週数 |
| `-min-support` | 10 | 代表構成の支持件数の下限 |
| `-min-ratio` | 0.6 | 代表構成の占有率の下限(**割合**。60%なら0.6) |
| `-min-contributors` | 3 | 代表構成の実ユーザー数の下限 |
| `-min-alias-runes` | 4 | 生成エイリアスの最小文字数 |
| `-show-rejected` / `-rejected-limit` | false / 30 | 落選名の表示 / 上限(0で全件) |

しきい値の指定ミス(例: `-min-ratio=60`)は起動時に検証して弾く。

---

## 5. パラメータ一覧

| 定数/設定 | 値 | 場所 | 意味 |
|---|---|---|---|
| `minVariantCount` | 3 | weekly_deck_usage_stat.go | 個別表示する変種の最小出現数(未満は「その他」) |
| `minAliasRunes` | 2 | deck_name.go | 辞書エントリの正規化後最小文字数(突合側) |
| 表示スロット | position 1/2 | 全箇所 | 指紋・代表構成・表示すべてこの範囲 |
| 生成しきい値 | 10 / 0.6 / 3 / 4 | generator | §4.3 参照(バッチオプションで変更可) |

## 6. 関連ファイル

| 役割 | パス |
|---|---|
| 週次集計 | `internal/infrastructure/weekly_deck_usage_stat.go` |
| 指紋の正規化 | `internal/infrastructure/fingerprint.go` |
| 名前の正規化・突合 | `internal/infrastructure/deck_name.go` |
| エイリアス自動生成 | `internal/infrastructure/deck_name_alias_generator.go` |
| 生成バッチ | `cmd/generate-deck-name-aliases/main.go` |
| スキーマ | `db/schema.sql`(`deck_name_aliases`, `pokemon_sprites` ほか) |
| フロント表示 | `webapp/src/app/components/organisms/DeckMeta/WeeklyDeckUsagePanel.tsx` |
| 意思決定の記録 | `adr/weekly-deck-usage-and-deck-name-alias.md` |
