package entity

import "regexp"

// tonamelEventIdPattern は Tonamel の大会ID(URL https://tonamel.com/competition/<id> の
// <id>)として受け付ける形。実際のIDは英数字5文字だが、DBの列(VARCHAR(8))に収まる範囲で
// 少し余裕を持たせる。
var tonamelEventIdPattern = regexp.MustCompile(`^[0-9A-Za-z]{1,8}$`)

// IsValidTonamelEventId は Tonamel の大会IDの形式かを返す。
//
// IDはそのまま tonamel.com のURLに連結して大会情報を取得するため、"/" や "?" を
// 通すと任意のページを取得させられる。また records.tonamel_event_id は VARCHAR(8) で、
// 長い値は保存時にDBエラー(500)になるため、入口で 400 にする。
func IsValidTonamelEventId(id string) bool {
	return tonamelEventIdPattern.MatchString(id)
}

type TonamelEvent struct {
	ID          string
	Title       string
	Description string
	Image       string
}

func NewTonamelEvent(
	id string,
	title string,
	description string,
	image string,
) *TonamelEvent {
	return &TonamelEvent{
		ID:          id,
		Title:       title,
		Description: description,
		Image:       image,
	}
}
