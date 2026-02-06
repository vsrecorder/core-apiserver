package model

type Metadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"og:image"`
}

type Result struct {
	URL      string `json:"url"`
	Success  bool   `json:"success"`
	Error    string `json:"error"`
	Metadata `json:"metadata"`
}

type TonamelEvent struct {
	Success bool `json:"success"`
	Result  `json:"result"`
}
