package infrastructure

import (
	"context"
	"strings"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
)

type User struct {
	db *gorm.DB
}

func NewUser(
	db *gorm.DB,
) repository.UserInterface {
	return &User{db}
}

func (i *User) FindById(
	ctx context.Context,
	id string,
) (*entity.User, error) {
	var model *model.User
	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		return nil, tx.Error
	}

	// Twitter/Googleのアイコン画像を大きくする
	model.ImageURL = strings.Replace(strings.Replace(model.ImageURL, "_normal", "", -1), "=s96-c", "", -1)

	user := entity.NewUser(
		model.ID,
		model.CreatedAt,
		model.Name,
		model.ImageURL,
	)

	return user, nil
}

func (i *User) Save(
	ctx context.Context,
	user *entity.User,
) error {
	model := model.NewUser(
		user.ID,
		user.CreatedAt,
		user.Name,
		user.ImageURL,
	)

	if tx := i.db.Save(model); tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (i *User) Delete(
	ctx context.Context,
	id string,
) error {
	if tx := i.db.Where("id = ?", id).Delete(&model.User{}); tx.Error != nil {
		return tx.Error
	}

	return nil
}
