# 週次デッキ使用率の集計単位 — 1体目のスプライトでまとめる

## ステータス

採用 (Accepted) — 2026-09-14

## Context

「対戦環境分析(週次デッキ使用率)」は、デッキの同一視を**スプライト(ポケモンアイコン)の集合**で
行っている(指紋。[adr/weekly-deck-usage-and-deck-name-alias.md](weekly-deck-usage-and-deck-name-alias.md) D2)。
表示スロットが2枠のため、指紋に使うのは position 1/2 の2体である。

この単位は「何と何を組み合わせた構築が使われたか」を正確に映すが、**同じ軸のデッキが
2体目の違いで別の行に割れる**。1体目が同じで2体目だけ違う派生が3つあれば、環境での
実際の存在感は3つ分なのにランキング上は3行に分散し、それぞれが `minVariantCount`(3件)を
下回れば「その他」へ落ちて一覧から消える。「このデッキは今どれくらい使われているのか」を
読み取りたい見方には、この分裂が邪魔になる。

## Decision

### D1. 集計単位を選べるようにする(既定は従来どおり)

リクエストごとに集計単位(`entity.DeckUsageGrouping`)を選べるようにした。

| 値 | 同じデッキとみなす条件 |
|---|---|
| `exact`(既定) | position 1/2 のスプライトの集合が一致する |
| `first_sprite` | **1体目のスプライトが同じ** |

**既定を `exact` のままにしたのは、どちらか一方が正しいわけではないため。**
組み合わせ単位は「何を組んだか」を、1体目単位は「どの軸が多いか」を映す。
片方へ寄せると失われる情報があるので、既定を変えずに切り替えを足した。

### D2. `first_sprite` では表示も1体目だけにする

指紋を1体目だけで作るなら、行の代表として出すスプライトも1体目だけにする。
グループ内の誰か1人の2体目を代表として描くと、その2体目を使っていない票まで
含んだ数字がその構築のものに見える。**表示と集計の単位は一致させる**
(3体目以降を指紋に含めない既存の判断と同じ理由)。

表示用の `position` は 1 に揃える。webapp は position で表示スロットを決めるため
([spriteSlot.ts](webapp/src/app/utils/spriteSlot.ts))、元の枠のまま返すと
2枠目に描かれてしまう。

### D3. 先頭のスプライトを「1体目」とする(position==1 ではない)

`position == 1` で抜き出すのではなく、**position ASC で並べた先頭**を1体目として扱う。
1枠目が欠けて2枠目にだけスプライトが入っている票(旧データ)を、指紋を作れない票として
丸ごと捨ててしまわないため。

### D4. 未知の値はコントローラで 400、下層では既定へ寄せる

`grouping` クエリの未知の値は 400 で弾く。タイプミスを黙って既定へ寄せると、
**まとめたつもりの数字を見て環境を読み違える**ため。一方 infrastructure / usecase は
未知・空の値を `exact` として扱う(バッチなど HTTP を経由しない呼び出しのため)。

webapp の proxy も受け付ける値へ正規化してから core-api へ渡す
([deckUsageGrouping.ts](webapp/src/app/utils/deckUsageGrouping.ts))。
URL を手で書き換えられてもページがエラーにならないようにするため。

### D5. 週次レポートの通知は `exact` のまま

`cmd/notify-weekly-report` の環境ニュースは「○○＋××が伸びた」と具体的な構築を
名指しする文面なので、1体目でまとめると何が動いたのか分からなくなる。従来どおり
組み合わせ単位で集計する。

## Consequences

### 良くなること

- 2体目が違うだけの派生が1行にまとまり、軸ごとの使用率が読める
- 派生に分散して「その他」へ落ちていたデッキが一覧に現れる
- 前週比較も同じ単位で集計するため、▲▼・pt差はその単位の中で整合する

### 受け入れるトレードオフ

- **1体目が同じでも別物のデッキは混ざる。** 同じポケモンを起点にした別コンセプトの
  構築は区別できない。区別したいときは `exact` へ切り替える
- **2つの単位の数字が並ぶ。** 同じ週でも単位が違えば使用率も順位も変わる。
  どちらを見ているかが分かるよう、レスポンスに `grouping` を返し、UI にも注記を出す
- **集計は単位ごとに走る。** 切り替えるたびに当週＋前週の集計が動く(キャッシュはしない)。
  現在のデータ量では実測で問題になっていないが、重くなったら単位込みでキャッシュする

## 実装

| 層 | 変更 |
|---|---|
| entity | `DeckUsageGrouping`(`exact` / `first_sprite`)と `IsValid`。`WeeklyDeckUsageStat.Grouping` |
| repository / infrastructure | `FindWeeklyDeckUsageStat` に `grouping`。`addVote` が指紋と表示スプライトを単位に応じて作る |
| usecase | `GetWeeklyDeckUsageStat` に `grouping`。空・未知は `exact` |
| controller | `grouping` クエリ(`helper.ParseQueryDeckUsageGrouping`)。未知なら 400。応答に `grouping` |
| webapp | proxy が `grouping` を中継。パネルにタブ(組み合わせ別 / 1体目でまとめる)と注記。URL の `grouping` を初期値に引き継ぐ |

API は `GET /api/v1beta/deck_meta/weekly_usage?week=<月曜日YYYY-MM-DD>&grouping=<exact|first_sprite>`(認証なし)。
