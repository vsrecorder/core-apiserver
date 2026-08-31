package dto

// UserAcquisitionCreateRequest は登録直後に webapp が送る流入元。
// 中身は着地時に proxy.ts が発行した first-party Cookie(vsr_attr)そのままで、
// キー名も Cookie の JSON に合わせてある。
//
// 値の検証・丸め(小文字化・allowlist・長さ切り詰め)はすべて usecase 側
// (entity.NormalizeAcquisition*)で行う。ここは JSON の形だけを見る。
//
// 全項目を string にしてあるのは、型が1つでも合わないと ShouldBindJSON が失敗し、
// 流入元がまるごと 400 で失われるため。値の中身がおかしいだけなら該当項目が
// 落ちるだけで済むよう、判定を下流に寄せている。
type UserAcquisitionCreateRequest struct {
	Source      string `json:"source"`
	Medium      string `json:"medium"`
	Campaign    string `json:"campaign"`
	Content     string `json:"content"`
	Referrer    string `json:"referrer"`
	LandingPath string `json:"landing_path"`
	LandingAt   string `json:"landing_at"`
}
