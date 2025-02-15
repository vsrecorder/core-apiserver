package infrastructure

import (
	"context"
	"strings"

	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type User struct {
	firebaseAuthClient *firebaseAuth.Client
}

func NewUser(
	firebaseAuthClient *firebaseAuth.Client,
) repository.UserInterface {
	return &User{firebaseAuthClient}
}

func (i *User) FindById(
	ctx context.Context,
	id string,
) (*entity.User, error) {
	userRecord, err := i.firebaseAuthClient.GetUser(context.Background(), id)
	if err != nil {
		return nil, err
	}

	entity := entity.NewUser(
		userRecord.UID,
		userRecord.DisplayName,
		// Twitter/Googleのアイコン画像を大きくする
		strings.Replace(strings.Replace(userRecord.PhotoURL, "_normal", "", -1), "=s96-c", "", -1),
	)

	return entity, nil
}
