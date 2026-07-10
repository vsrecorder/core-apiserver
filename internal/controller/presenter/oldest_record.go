package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewOldestRecordResponse(
	record *entity.OldestRecord,
	userId string,
	deckId string,
) *dto.OldestRecordResponse {
	var eventDate *string
	if record.EventDate != nil {
		s := record.EventDate.Format("2006-01-02")
		eventDate = &s
	}

	return &dto.OldestRecordResponse{
		UserId:    userId,
		DeckId:    deckId,
		EventDate: eventDate,
	}
}
