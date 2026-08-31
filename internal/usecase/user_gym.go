package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// MaxUserGymsPerUser は1ユーザが登録できるMyジムの数。
//
// ホームのパネルは登録店舗のイベントを1つのリストにまとめて出すが、既定では
// 先頭の数件だけを見せて残りは畳むため、枠を増やしてもパネルが伸び続けることはない。
// それでも上限を置くのは、1回の取得で引く店舗数(FindByShopIds の IN 句)を抑えるためと、
// 「よく行く店を選ぶ」という機能の性格を保つため。
const MaxUserGymsPerUser = 5

type UserGymInterface interface {
	Find(
		ctx context.Context,
		uid string,
	) ([]*entity.UserGymView, error)

	// Create は shopId の店舗をMyジムに登録する。
	//
	// 店舗が実在しない場合は apperror.ErrRecordNotFound、
	// 既に登録済みの場合は apperror.ErrAlreadyExists、
	// 上限に達している場合は apperror.ErrTooManyUserGyms を返す。
	Create(
		ctx context.Context,
		uid string,
		shopId uint,
	) (*entity.UserGymView, error)

	Delete(
		ctx context.Context,
		uid string,
		shopId uint,
	) error

	// FindOfficialEvents は uid のMyジムと、その店舗で期間内に開催される
	// 公式イベントを合わせて返す。
	//
	// パネルは「登録している店」と「そこでの予定」を同時に描くため、
	// 2回に分けず1度の呼び出しで揃える。Myジムが0件ならイベントも空で返す。
	FindOfficialEvents(
		ctx context.Context,
		uid string,
		startDate time.Time,
		endDate time.Time,
	) ([]*entity.UserGymView, []*entity.OfficialEvent, error)
}

type UserGym struct {
	userGymRepository       repository.UserGymInterface
	shopRepository          repository.ShopInterface
	officialEventRepository repository.OfficialEventInterface
}

func NewUserGym(
	userGymRepository repository.UserGymInterface,
	shopRepository repository.ShopInterface,
	officialEventRepository repository.OfficialEventInterface,
) UserGymInterface {
	return &UserGym{
		userGymRepository:       userGymRepository,
		shopRepository:          shopRepository,
		officialEventRepository: officialEventRepository,
	}
}

func (u *UserGym) Find(
	ctx context.Context,
	uid string,
) ([]*entity.UserGymView, error) {
	views, err := u.userGymRepository.FindByUserId(ctx, uid)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return views, nil
}

func (u *UserGym) Create(
	ctx context.Context,
	uid string,
	shopId uint,
) (*entity.UserGymView, error) {
	// 実在しない店舗IDを登録すると外部キー違反で500になるため、先に存在を確かめる。
	// 「消えた店を登録しようとした」を 404 として返せるようにもなる。
	shop, err := u.shopRepository.FindById(ctx, shopId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	views, err := u.userGymRepository.FindByUserId(ctx, uid)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	for _, view := range views {
		if view.Shop.ID == shopId {
			return nil, apperror.ErrAlreadyExists
		}
	}

	// 上限に達していたら、どれを外すかはユーザーに選ばせる(自動で押し出さない)。
	if len(views) >= MaxUserGymsPerUser {
		return nil, apperror.ErrTooManyUserGyms
	}

	createdAt := timeNow()

	if err := u.userGymRepository.Create(
		ctx,
		entity.NewUserGym(uid, shopId, createdAt),
	); err != nil {
		logError(ctx, err)
		return nil, err
	}

	return entity.NewUserGymView(shop, createdAt), nil
}

func (u *UserGym) Delete(
	ctx context.Context,
	uid string,
	shopId uint,
) error {
	if err := u.userGymRepository.Delete(ctx, uid, shopId); err != nil {
		logError(ctx, err)
		return err
	}

	return nil
}

func (u *UserGym) FindOfficialEvents(
	ctx context.Context,
	uid string,
	startDate time.Time,
	endDate time.Time,
) ([]*entity.UserGymView, []*entity.OfficialEvent, error) {
	views, err := u.userGymRepository.FindByUserId(ctx, uid)
	if err != nil {
		logError(ctx, err)
		return nil, nil, err
	}

	if len(views) == 0 {
		return []*entity.UserGymView{}, []*entity.OfficialEvent{}, nil
	}

	shopIds := make([]uint, 0, len(views))
	for _, view := range views {
		shopIds = append(shopIds, view.Shop.ID)
	}

	officialEvents, err := u.officialEventRepository.FindByShopIds(ctx, shopIds, startDate, endDate)
	if err != nil {
		logError(ctx, err)
		return nil, nil, err
	}

	return views, officialEvents, nil
}
