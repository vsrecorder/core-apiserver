package model

type ResData struct {
	Result      bool   `json:"result"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"og:image"`
}

type TonamelEvent struct {
	Result  bool `json:"result"`
	ResData `json:"resData"`
}
