package entity

import (
	"time"
)

// UnofficialEvent は非公式イベント(公式イベントやTonamel以外でユーザが任意に管理するイベント)を表す。
// records テーブルとは疎結合とし、records からは ID で参照する。
type UnofficialEvent struct {
	ID     string
	UserId string
	Title  string
	Date   time.Time
}

func NewUnofficialEvent(
	id string,
	userId string,
	title string,
	date time.Time,
) *UnofficialEvent {
	return &UnofficialEvent{
		ID:     id,
		UserId: userId,
		Title:  title,
		Date:   date,
	}
}
