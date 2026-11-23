package model

import "time"

// Player は players テーブルの1行。大会結果に出てくるプレイヤーの、最新の表示名を持つ。
//
// 書き込むのは取り込みバッチ(import-cityleague-result-job 等)で、core-apiserver は
// 大会結果の読み出しで結合するだけ。LatestEventDate はこの名前を取った大会の開催日で、
// バッチはこれより古い大会の名前では上書きしない(db/schema.sql の players のコメント参照)。
type Player struct {
	Id              string `gorm:"primaryKey"`
	Name            string
	LatestEventDate time.Time
}
