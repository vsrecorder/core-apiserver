package model

type Regulation struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func NewRegulation(
	id uint,
	name string,
) *Regulation {
	return &Regulation{
		ID:   id,
		Name: name,
	}
}
