package dto

// UserAcquisitionCreateRequest は登録直後に webapp が送る流入元。
// 中身は着地時に proxy.ts が発行した first-party Cookie(vsr_attr)そのままで、
// キー名も Cookie の JSON に合わせてある。
//
// 値の検証・丸めはすべて usecase 側(entity.NormalizeAcquisition*)で行う。
// ここで弾くとフィールドが1つ壊れているだけで流入元がまるごと失われるため、
// 「不正なら弾く」ではなく「読める項目だけ残す」方針にしている。
type UserAcquisitionCreateRequest struct {
	Source      string `json:"source"`
	Medium      string `json:"medium"`
	Campaign    string `json:"campaign"`
	Content     string `json:"content"`
	Referrer    string `json:"referrer"`
	LandingPath string `json:"landing_path"`
	LandingAt   string `json:"landing_at"`
}
