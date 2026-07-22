package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewKizunaResponse(kizuna *entity.Kizuna) *dto.KizunaResponse {
	// 空スライスで初期化して、JSON が null にならないようにする
	decks := []*dto.KizunaDeckResponse{}

	for _, deck := range kizuna.Decks {
		metrics := []*dto.KizunaMetricResponse{}
		for _, m := range deck.Metrics {
			metrics = append(metrics, &dto.KizunaMetricResponse{
				Key:       string(m.Key),
				Weight:    m.Weight,
				Value:     m.Value,
				Points:    m.Points,
				MaxPoints: m.MaxPoints,
			})
		}

		decks = append(decks, &dto.KizunaDeckResponse{
			DeckId:  deck.DeckId,
			Level:   deck.Level,
			Metrics: metrics,
		})
	}

	return &dto.KizunaResponse{
		UserId:   kizuna.UserId,
		MaxLevel: entity.KizunaMaxLevel,
		Decks:    decks,
	}
}
