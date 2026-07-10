package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type RecordParam struct {
	officialEventId   uint
	tonamelEventId    string
	friendId          string
	unofficialEventId string
	userId            string
	deckId            string
	deckCodeId        string
	eventDate         time.Time
	privateFlg        bool
	ignoreStatsFlg    bool
	tcgMeisterURL     string
	memo              string
}

func NewRecordParam(
	officialEventId uint,
	tonamelEventId string,
	friendId string,
	unofficialEventId string,
	userId string,
	deckId string,
	deckCodeId string,
	eventDate time.Time,
	privateFlg bool,
	ignoreStatsFlg bool,
	tcgMeisterURL string,
	memo string,
) *RecordParam {
	return &RecordParam{
		officialEventId:   officialEventId,
		tonamelEventId:    tonamelEventId,
		friendId:          friendId,
		unofficialEventId: unofficialEventId,
		userId:            userId,
		deckId:            deckId,
		deckCodeId:        deckCodeId,
		eventDate:         eventDate,
		privateFlg:        privateFlg,
		ignoreStatsFlg:    ignoreStatsFlg,
		tcgMeisterURL:     tcgMeisterURL,
		memo:              memo,
	}
}

type RecordInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.Record, error)

	Find(
		ctx context.Context,
		limit int,
		offset int,
		eventType string,
	) ([]*entity.Record, error)

	FindOnCursor(
		ctx context.Context,
		limit int,
		cursorEventDate time.Time,
		cursorCreatedAt time.Time,
		eventType string,
	) ([]*entity.Record, error)

	FindByUserId(
		ctx context.Context,
		uid string,
		limit int,
		offset int,
		eventType string,
	) ([]*entity.Record, error)

	FindByUserIdOnCursor(
		ctx context.Context,
		uid string,
		limit int,
		cursorEventDate time.Time,
		cursorCreatedAt time.Time,
		eventType string,
	) ([]*entity.Record, error)

	FindByOfficialEventId(
		ctx context.Context,
		officialEventId uint,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindByTonamelEventId(
		ctx context.Context,
		tonamelEventId string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindByDeckId(
		ctx context.Context,
		deckId string,
		limit int,
		offset int,
		eventType string,
	) ([]*entity.Record, error)

	FindByDeckIdOnCursor(
		ctx context.Context,
		deckId string,
		limit int,
		cursorEventDate time.Time,
		cursorCreatedAt time.Time,
		eventType string,
	) ([]*entity.Record, error)

	FindByDeckCodeId(
		ctx context.Context,
		deckCodeId string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	Create(
		ctx context.Context,
		param *RecordParam,
	) (*entity.Record, error)

	Update(
		ctx context.Context,
		id string,
		param *RecordParam,
	) (*entity.Record, error)

	Delete(
		ctx context.Context,
		id string,
	) error
}

type Record struct {
	repository            repository.RecordInterface
	badgeEvaluation       BadgeEvaluationInterface
	designationEvaluation DesignationEvaluationInterface
}

func NewRecord(
	repository repository.RecordInterface,
	badgeEvaluation BadgeEvaluationInterface,
	designationEvaluation DesignationEvaluationInterface,
) RecordInterface {
	return &Record{repository, badgeEvaluation, designationEvaluation}
}

func (u *Record) FindById(
	ctx context.Context,
	id string,
) (*entity.Record, error) {
	record, err := u.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return record, nil
}

func (u *Record) Find(
	ctx context.Context,
	limit int,
	offset int,
	eventType string,
) ([]*entity.Record, error) {
	records, err := u.repository.Find(ctx, limit, offset, eventType)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindOnCursor(
	ctx context.Context,
	limit int,
	cursorEventDate time.Time,
	cursorCreatedAt time.Time,
	eventType string,
) ([]*entity.Record, error) {
	records, err := u.repository.FindOnCursor(ctx, limit, cursorEventDate, cursorCreatedAt, eventType)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByUserId(
	ctx context.Context,
	uid string,
	limit int,
	offset int,
	eventType string,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByUserId(ctx, uid, limit, offset, eventType)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByUserIdOnCursor(
	ctx context.Context,
	uid string,
	limit int,
	cursorEventDate time.Time,
	cursorCreatedAt time.Time,
	eventType string,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByUserIdOnCursor(ctx, uid, limit, cursorEventDate, cursorCreatedAt, eventType)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByOfficialEventId(
	ctx context.Context,
	officialEventId uint,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByOfficialEventId(ctx, officialEventId, limit, offset)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByTonamelEventId(
	ctx context.Context,
	tonamelEventId string,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByTonamelEventId(ctx, tonamelEventId, limit, offset)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByDeckId(
	ctx context.Context,
	deckId string,
	limit int,
	offset int,
	eventType string,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByDeckId(ctx, deckId, limit, offset, eventType)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByDeckIdOnCursor(
	ctx context.Context,
	deckId string,
	limit int,
	cursorEventDate time.Time,
	cursorCreatedAt time.Time,
	eventType string,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByDeckIdOnCursor(ctx, deckId, limit, cursorEventDate, cursorCreatedAt, eventType)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByDeckCodeId(
	ctx context.Context,
	deckCodeId string,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByDeckCodeId(ctx, deckCodeId, limit, offset)
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) Create(
	ctx context.Context,
	param *RecordParam,
) (*entity.Record, error) {
	id, err := generateId()
	if err != nil {
		return nil, err
	}

	createdAt := time.Now().Local()

	// 称号のtier変化を記録の前後で比較するため、保存前の時点で取得しておく。
	// シーズン範囲が定まらない等でエラーになった場合は、この記録作成では
	// 称号/ランクの通知判定自体を行わない(記録作成そのものは失敗させない)。
	beforeTier, tierErr := u.designationEvaluation.CurrentTier(ctx, param.userId)

	record := entity.NewRecord(
		id,
		createdAt,
		param.officialEventId,
		param.tonamelEventId,
		param.friendId,
		param.unofficialEventId,
		param.userId,
		param.deckId,
		param.deckCodeId,
		param.eventDate,
		param.privateFlg,
		param.ignoreStatsFlg,
		param.tcgMeisterURL,
		param.memo,
	)

	if err := u.repository.Save(ctx, record); err != nil {
		return nil, err
	}

	// 通知一覧はcreated_at DESC(新しい順、同値時はid DESC)で表示されるため、後から
	// 生成した通知ほど上に表示される。作成順序を「ユーザバッジ→称号/ランクアップ」に
	// することで、表示順序は下から「ユーザバッジ→称号/ランクアップ」(=上から称号/
	// ランクアップ→ユーザバッジ)になる。
	if _, err := u.badgeEvaluation.EvaluateOnRecordCreated(ctx, param.userId, record); err != nil {
		return nil, err
	}

	if tierErr == nil {
		// 通知のcreated_atは対戦日(event_date)ではなく実際の処理時刻を使う。
		// event_dateはユーザ登録直後に過去日を入力されると、登録バッジ通知より
		// 過去のcreated_atになり通知の並び順が崩れるため使わない。
		u.designationEvaluation.NotifyIfTierChanged(ctx, param.userId, beforeTier, record.CreatedAt)
	}

	return record, nil
}

func (u *Record) Update(
	ctx context.Context,
	id string,
	param *RecordParam,
) (*entity.Record, error) {
	// 指定されたidのRecordが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == apperror.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		return nil, err
	}

	// 称号のtier変化を更新の前後で比較するため、保存前の時点で取得しておく。
	// デッキ未登録のまま作成した記録に、後からデッキを登録するケースでは、この
	// Updateで初めて称号のrecordカウント対象になりtierが変化しうる(Createと同様)。
	beforeTier, tierErr := u.designationEvaluation.CurrentTier(ctx, param.userId)

	record := entity.NewRecord(
		id,
		ret.CreatedAt,
		param.officialEventId,
		param.tonamelEventId,
		param.friendId,
		param.unofficialEventId,
		param.userId,
		param.deckId,
		param.deckCodeId,
		param.eventDate,
		param.privateFlg,
		param.ignoreStatsFlg,
		param.tcgMeisterURL,
		param.memo,
	)

	if err := u.repository.Save(ctx, record); err != nil {
		return nil, err
	}

	if tierErr == nil {
		u.designationEvaluation.NotifyIfTierChanged(ctx, param.userId, beforeTier, time.Now().Local())
		u.designationEvaluation.NotifyIfTierLost(ctx, param.userId, beforeTier)
	}

	return record, nil
}

func (u *Record) Delete(
	ctx context.Context,
	id string,
) error {
	record, err := u.repository.FindById(ctx, id)
	if err != nil {
		return err
	}

	// 称号のtier変化を削除の前後で比較するため、削除前の時点で取得しておく。
	beforeTier, tierErr := u.designationEvaluation.CurrentTier(ctx, record.UserId)

	if err := u.repository.Delete(ctx, id); err != nil {
		return err
	}

	if err := u.badgeEvaluation.EvaluateOnRecordDeleted(ctx, record.UserId); err != nil {
		return err
	}

	if tierErr == nil {
		u.designationEvaluation.NotifyIfTierLost(ctx, record.UserId, beforeTier)
	}

	return nil
}
