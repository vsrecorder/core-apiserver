package model

// Shop は shops テーブルに対応する。
//
// PrefectureName は shops の列ではなく prefectures を結合して受けるため、
// gorm:"-" は付けずに Select のエイリアスで埋める(official_events と同じ扱い)。
type Shop struct {
	ID             uint
	Name           string
	ZipCode        string
	PrefectureId   uint
	PrefectureName string
	Address        string
	Tel            string
	BusinessHours  string
	URL            string
}
