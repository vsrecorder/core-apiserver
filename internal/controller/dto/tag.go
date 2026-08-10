package dto

import "time"

type TagCreateRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type TagUpdateRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type TagResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	// PresetFlg=true は全ユーザー共通のプリセットタグ(例: ACE SPEC)。
	// フロントはこれで「編集不可・別セクション表示」を判断する。
	PresetFlg bool `json:"preset_flg"`
}

type TagGetResponse []TagResponse

type TagCreateResponse struct {
	TagResponse
}

type TagUpdateResponse struct {
	TagResponse
}
