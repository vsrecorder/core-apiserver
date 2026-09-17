package entity

import "testing"

func TestIsValidTonamelEventId(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"実際の大会IDの形(英数字5文字)", "OakZc", true},
		{"数字混じり", "61ozP", true},
		{"上限の8文字", "abcdefgh", true},

		{"空文字", "", false},
		{"9文字(DBの列幅を超える)", "abcdefghi", false},
		{"スラッシュを含む(パス操作)", "../x", false},
		{"クエリ文字を含む", "ab?c", false},
		{"ハイフンを含む", "ab-c", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidTonamelEventId(tt.id); got != tt.want {
				t.Errorf("IsValidTonamelEventId(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}
