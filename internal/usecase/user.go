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

// User は退会でユーザに紐づく全テーブルを消すため、そのぶんリポジトリを抱える。
// 「user_id を持つテーブル」と「それらへFKで繋がる中間テーブル」を足したものが
// 削除対象で、対応は Delete のコメントに一覧してある。
type User struct {
	repository                     repository.UserInterface
	recordRepository               repository.RecordInterface
	deckRepository                 repository.DeckInterface
	deckCodeRepository             repository.DeckCodeInterface
	tagRepository                  repository.TagInterface
	userFavoriteDeckRepository     repository.UserFavoriteDeckInterface
	userStreakRepository           repository.UserStreakInterface
	userDailyActivityRepository    repository.UserDailyActivityInterface
	userBadgeRepository            repository.UserBadgeInterface
	userEnvironmentBadgeRepository repository.UserEnvironmentBadgeInterface
	notificationRepository         repository.NotificationInterface
	userPlayerRepository           repository.UserPlayerInterface
	pushSubscriptionRepository     repository.PushSubscriptionInterface
	pushDeliveryRepository         repository.PushDeliveryInterface
	userAcquisitionRepository      repository.UserAcquisitionInterface
	transactionManager             repository.TransactionManager
	badgeEvaluation                BadgeEvaluationInterface
}

func NewUser(
	repository repository.UserInterface,
	recordRepository repository.RecordInterface,
	deckRepository repository.DeckInterface,
	deckCodeRepository repository.DeckCodeInterface,
	tagRepository repository.TagInterface,
	userFavoriteDeckRepository repository.UserFavoriteDeckInterface,
	userStreakRepository repository.UserStreakInterface,
	userDailyActivityRepository repository.UserDailyActivityInterface,
	userBadgeRepository repository.UserBadgeInterface,
	userEnvironmentBadgeRepository repository.UserEnvironmentBadgeInterface,
	notificationRepository repository.NotificationInterface,
	userPlayerRepository repository.UserPlayerInterface,
	pushSubscriptionRepository repository.PushSubscriptionInterface,
	pushDeliveryRepository repository.PushDeliveryInterface,
	userAcquisitionRepository repository.UserAcquisitionInterface,
	transactionManager repository.TransactionManager,
	badgeEvaluation BadgeEvaluationInterface,
) UserInterface {
	// フィールドが多いので、並び順の取り違えを避けるため名前付きで組み立てる。
	return &User{
		repository:                     repository,
		recordRepository:               recordRepository,
		deckRepository:                 deckRepository,
		deckCodeRepository:             deckCodeRepository,
		tagRepository:                  tagRepository,
		userFavoriteDeckRepository:     userFavoriteDeckRepository,
		userStreakRepository:           userStreakRepository,
		userDailyActivityRepository:    userDailyActivityRepository,
		userBadgeRepository:            userBadgeRepository,
		userEnvironmentBadgeRepository: userEnvironmentBadgeRepository,
		notificationRepository:         notificationRepository,
		userPlayerRepository:           userPlayerRepository,
		pushSubscriptionRepository:     pushSubscriptionRepository,
		pushDeliveryRepository:         pushDeliveryRepository,
		userAcquisitionRepository:      userAcquisitionRepository,
		transactionManager:             transactionManager,
		badgeEvaluation:                badgeEvaluation,
	}
}

func (u *User) FindById(
	ctx context.Context,
	id string,
) (*entity.User, error) {
	user, err := u.repository.FindById(ctx, id)

	if err != nil {
		logError(ctx, err)
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

	// 退会済みのユーザーはFindByIdからは見えないため、ここまで到達してしまう。
	// そのままSaveするとUPDATEにdeleted_at IS NULLが付いて0件更新になり、
	// 実体が無いまま作成に成功したことになってしまうので、明示的に弾く。
	withdrawn, err := u.repository.IsWithdrawn(ctx, user.ID)
	if err != nil {
		logError(ctx, err)
		return nil, err
	} else if withdrawn {
		return nil, apperror.ErrWithdrawn
	}

	if err := u.repository.Save(ctx, user); err != nil {
		logError(ctx, err)
		return nil, err
	}

	if _, err := u.badgeEvaluation.EvaluateOnUserCreated(ctx, user.ID, user.CreatedAt); err != nil {
		logError(ctx, err)
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
		logError(ctx, err)
		return nil, err
	}

	user := entity.NewUser(
		id,
		ret.CreatedAt,
		param.Name,
		param.ImageURL,
	)

	if err := u.repository.Save(ctx, user); err != nil {
		logError(ctx, err)
		return nil, err
	}

	return user, nil
}

func (u *User) Delete(
	ctx context.Context,
	id string,
) error {
	// 退会にあたり、ユーザ本体を消す前に、そのユーザに紐づくデータをすべて削除する。
	// 全体を1つのDBトランザクションにまとめており、途中で失敗した場合はここまでの
	// 削除もすべてロールバックされる。
	//
	// 削除対象は「user_id を持つテーブル」と「それらへFKで繋がる中間テーブル」の全部:
	//
	//   Record.DeleteByUserId    records / matches / games / unofficial_events /
	//                            record_tags / match_tags / match_pokemon_sprites
	//   Deck.DeleteByUserId      decks / deck_codes(自分のデッキに紐づくもの) /
	//                            deck_tags / deck_code_tags / deck_pokemon_sprites /
	//                            user_favorite_decks(自分のデッキに付いたもの)
	//   DeckCode.DeleteByUserId  deck_codes(自分が作ったもの) / deck_code_tags
	//   Tag.DeleteByUserId       tags と、そのタグの中間テーブルの行
	//   以下は1テーブルずつ       user_favorite_decks(他人のデッキに付けたもの) /
	//                            user_streaks / user_daily_activities / user_badges /
	//                            user_environment_badges / notifications / users_players /
	//                            push_subscriptions / push_deliveries / user_acquisitions
	//
	// 論理削除(deleted_at)を持つテーブルは論理削除、持たないテーブルは行ごと物理削除する
	// (ユーザ本体 users も論理削除のため、それに揃えている)。
	// 唯一残るのは matches.opponents_user_id で、これは他のユーザが作った対戦記録の中で
	// 対戦相手として参照されているもの。他人のデータなので消さない。
	//
	// 記録やデッキを1件ずつ削除すると、記録数の多いユーザほどクエリ数と
	// トランザクションの保持時間が線形に伸びるため、まとめて削除する。
	return u.transactionManager.Do(ctx, func(ctx context.Context) error {
		if err := u.recordRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		if err := u.deckRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		// DeckCode.DeckId は必ずしも本人が所有するデッキとは限らない(他人のデッキに
		// 対して作成できてしまう)ため、上記のデッキ連鎖削除だけでは削除しきれない
		// ケースがある。user_id でも直接削除する。
		if err := u.deckCodeRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		// タグは付与先(デッキ/デッキコード/記録/対戦結果)より後に消す。
		// 付与先の削除で中間テーブルの行はすでに消えているが、付与先を持たない
		// タグもあるため、タグ側からも中間テーブルごと消す。
		if err := u.tagRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		// 他人のデッキに付けたお気に入り(自分のデッキのぶんは Deck 側で消えている)。
		if err := u.userFavoriteDeckRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		// 以下は論理削除を持たないため、行ごと物理削除される。
		if err := u.userStreakRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		if err := u.userDailyActivityRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		if err := u.userBadgeRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		if err := u.userEnvironmentBadgeRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		if err := u.notificationRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		// プレイヤーIDの紐付けは1ユーザーにつき有効な行が最大1件の想定だが、
		// 万一2件以上あっても消し残さないよう user_id でまとめて消す。
		if err := u.userPlayerRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		// Web Push の購読と配達ログ。購読を残すと退会後も push が届き続ける。
		if err := u.pushSubscriptionRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		if err := u.pushDeliveryRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		// 流入元(施策0-4)。退会後も残すと、退会者に紐づく行だけが取り残される。
		// 集計は生存ユーザー(deleted_at IS NULL)を分母にしているため、消しても
		// 流入元別の登録数・定着率の読みは変わらない。
		if err := u.userAcquisitionRepository.DeleteByUserId(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		return u.repository.Delete(ctx, id)
	})
}
