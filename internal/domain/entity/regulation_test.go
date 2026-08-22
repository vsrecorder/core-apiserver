package entity

import "testing"

func TestIsValidRegulationId(t *testing.T) {
	tests := []struct {
		name string
		id   uint
		want bool
	}{
		{"スタンダード", RegulationIdStandard, true},
		{"エクストラ", RegulationIdExtra, true},
		{"殿堂", RegulationIdHallOfFame, true},
		{"その他", RegulationIdOther, true},

		{"未指定", 0, false},
		{"存在しないID", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidRegulationId(tt.id); got != tt.want {
				t.Errorf("IsValidRegulationId() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeRegulationId(t *testing.T) {
	tests := []struct {
		name string
		id   uint
		want uint
	}{
		{"未指定はスタンダードへ寄せる", 0, RegulationIdStandard},
		{"指定済みはそのまま", RegulationIdExtra, RegulationIdExtra},
		{"存在しないIDは寄せずにそのまま(検証側で弾く)", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeRegulationId(tt.id); got != tt.want {
				t.Errorf("NormalizeRegulationId() = %v, want %v", got, tt.want)
			}
		})
	}
}
