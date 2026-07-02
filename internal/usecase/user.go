package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type UserCreateParam struct {
	ID       string
	Name     string
	ImageURL string
}

type UserUpdateParam struct {
	Name     string
	ImageURL string
}

func NewUserCreateParam(
	id string,
	name string,
	imageURL string,
) *UserCreateParam {
	return &UserCreateParam{
		ID:       id,
		Name:     name,
		ImageURL: imageURL,
	}
}

func NewUserUpdateParam(
	name string,
	imageURL string,
) *UserUpdateParam {
	return &UserUpdateParam{
		Name:     name,
		ImageURL: imageURL,
	}
}

type UserInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.User, error)

	Create(
		ctx context.Context,
		param *UserCreateParam,
	) (*entity.User, error)

	Update(
		ctx context.Context,
		id string,
		param *UserUpdateParam,
	) (*entity.User, error)

	Delete(
		ctx context.Context,
		id string,
	) error
}

type User struct {
	repository         repository.UserInterface
	recordRepository   repository.RecordInterface
	deckRepository     repository.DeckInterface
	deckCodeRepository repository.DeckCodeInterface
	transactionManager repository.TransactionManager
}

func NewUser(
	repository repository.UserInterface,
	recordRepository repository.RecordInterface,
	deckRepository repository.DeckInterface,
	deckCodeRepository repository.DeckCodeInterface,
	transactionManager repository.TransactionManager,
) UserInterface {
	return &User{repository, recordRepository, deckRepository, deckCodeRepository, transactionManager}
}

func (u *User) FindById(
	ctx context.Context,
	id string,
) (*entity.User, error) {
	user, err := u.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *User) Create(
	ctx context.Context,
	param *UserCreateParam,
) (*entity.User, error) {
	createdAt := time.Now().Local()

	user := entity.NewUser(
		param.ID,
		createdAt,
		param.Name,
		param.ImageURL,
	)

	_, err := u.repository.FindById(ctx, user.ID)
	if err == nil {
		return nil, apperror.ErrAlreadyExists
	} else if err != apperror.ErrRecordNotFound {
		return nil, err
	}

	if err := u.repository.Save(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *User) Update(
	ctx context.Context,
	id string,
	param *UserUpdateParam,
) (*entity.User, error) {
	ret, err := u.repository.FindById(ctx, id)
	if err == apperror.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		return nil, err
	}

	user := entity.NewUser(
		id,
		ret.CreatedAt,
		param.Name,
		param.ImageURL,
	)

	if err := u.repository.Save(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *User) Delete(
	ctx context.Context,
	id string,
) error {
	// 退会にあたり、ユーザ本体を消す前に対戦記録・デッキ・デッキコードを連鎖削除する。
	// Record.Delete / Deck.Delete は Match・Game・そのデッキ自身のDeckCode等の
	// 関連レコードもあわせて削除するため、ここでは ID を洗い出して順に呼び出すだけでよい。
	// 全体を1つのDBトランザクションにまとめており、途中で失敗した場合はここまでの
	// 削除もすべてロールバックされる。
	return u.transactionManager.Do(ctx, func(ctx context.Context) error {
		recordIds, err := u.recordRepository.FindIdsByUserId(ctx, id)
		if err != nil {
			return err
		}

		for _, recordId := range recordIds {
			if err := u.recordRepository.Delete(ctx, recordId); err != nil {
				return err
			}
		}

		deckIds, err := u.deckRepository.FindIdsByUserId(ctx, id)
		if err != nil {
			return err
		}

		for _, deckId := range deckIds {
			if err := u.deckRepository.Delete(ctx, deckId); err != nil {
				return err
			}
		}

		// DeckCode.DeckId は必ずしも本人が所有するデッキとは限らない(他人のデッキに
		// 対して作成できてしまう)ため、上記のデッキ連鎖削除だけでは削除しきれない
		// ケースがある。user_id で直接洗い出して個別に削除する。
		deckCodeIds, err := u.deckCodeRepository.FindIdsByUserId(ctx, id)
		if err != nil {
			return err
		}

		for _, deckCodeId := range deckCodeIds {
			if err := u.deckCodeRepository.Delete(ctx, deckCodeId); err != nil {
				return err
			}
		}

		return u.repository.Delete(ctx, id)
	})
}
