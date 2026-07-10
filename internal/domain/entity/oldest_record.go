package entity

import "time"

// OldestRecord はユーザー（および任意でデッキ）が持つ対戦記録のうち、
// 最も古いevent_dateを表す。該当する記録が1件も無い場合はEventDateがnilになる。
type OldestRecord struct {
	EventDate *time.Time
}

func NewOldestRecord(
	eventDate *time.Time,
) *OldestRecord {
	return &OldestRecord{
		EventDate: eventDate,
	}
}
