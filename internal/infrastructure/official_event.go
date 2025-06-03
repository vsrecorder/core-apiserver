package infrastructure

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
)

type OfficialEvent struct {
	db *gorm.DB
}

func NewOfficialEvent(
	db *gorm.DB,
) repository.OfficialEventInterface {
	return &OfficialEvent{db}
}

func (i *OfficialEvent) Find(
	ctx context.Context,
	typeId uint,
	leagueType uint,
	startDate time.Time,
	endDate time.Time,
) ([]*entity.OfficialEvent, error) {
	var models []*model.OfficialEvent

	if typeId == 0 { // 大会の種類に指定がない場合
		if 1 <= leagueType && leagueType <= 4 { // リーグの種類に指定がある場合
			leagueTitle := ""

			switch leagueType {
			case 1:
				leagueTitle = "オープン"
			case 2:
				leagueTitle = "ジュニア"
			case 3:
				leagueTitle = "シニア"
			case 4:
				leagueTitle = "マスター"
			}

			if tx := i.db.Where("league_title = ? AND date BETWEEN ? AND ?", leagueTitle, startDate, endDate).Find(&models); tx.Error != nil {
				return nil, tx.Error
			}
		} else {
			if tx := i.db.Where("date BETWEEN ? AND ?", startDate, endDate).Find(&models); tx.Error != nil {
				return nil, tx.Error
			}
		}
	} else { // 大型大会(typeId: 1) / シティ(typeId: 2) / トレリ(typeId: 3) / ジムイベント(typeId: 4) / オーガナイザーイベント(typeId: 6) / その他(typeId: 7)の場合
		if 1 <= leagueType && leagueType <= 4 { // リーグの種類に指定がある場合
			leagueTitle := ""

			switch leagueType {
			case 1:
				leagueTitle = "オープン"
			case 2:
				leagueTitle = "ジュニア"
			case 3:
				leagueTitle = "シニア"
			case 4:
				leagueTitle = "マスター"
			}

			if tx := i.db.Where("type_id = ? AND league_title = ? AND date BETWEEN ? AND ?", typeId, leagueTitle, startDate, endDate).Find(&models); tx.Error != nil {
				return nil, tx.Error
			}
		} else {
			if tx := i.db.Where("type_id = ? AND date BETWEEN ? AND ?", typeId, startDate, endDate).Find(&models); tx.Error != nil {
				return nil, tx.Error
			}
		}
	}

	var entities []*entity.OfficialEvent
	for _, m := range models {
		entities = append(
			entities,
			entity.NewOfficialEvent(
				m.ID,
				m.Title,
				m.Address,
				m.Venue,
				m.Date,
				m.StartedAt,
				m.EndedAt,
				m.DeckCount,
				m.TypeId,
				m.TypeName,
				m.CSPFlg,
				m.LeagueId,
				m.LeagueTitle,
				m.RegulationId,
				m.RegulationTitle,
				m.Capacity,
				m.AttrId,
				m.ShopId,
				m.ShopName,
			),
		)
	}

	return entities, nil
}

func (i *OfficialEvent) FindById(
	ctx context.Context,
	id uint,
) (*entity.OfficialEvent, error) {
	var m model.OfficialEvent

	if tx := i.db.Where("id = ? ", id).First(&m); tx.Error != nil {
		return nil, tx.Error
	}

	e := entity.NewOfficialEvent(
		m.ID,
		m.Title,
		m.Address,
		m.Venue,
		m.Date,
		m.StartedAt,
		m.EndedAt,
		m.DeckCount,
		m.TypeId,
		m.TypeName,
		m.CSPFlg,
		m.LeagueId,
		m.LeagueTitle,
		m.RegulationId,
		m.RegulationTitle,
		m.Capacity,
		m.AttrId,
		m.ShopId,
		m.ShopName,
	)

	return e, nil
}
