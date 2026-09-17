package entity

import "testing"

func TestIsValidDeckCodeFormat(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{"公式サイトの形式", "5dbFbk-uBwjqP-VVk5Vv", true},
		{"英数字のみ", "abc123", true},

		{"空文字", "", false},
		{"スラッシュを含む(パス操作)", "abc/../x", false},
		{"クエリ文字を含む", "abc?x=1", false},
		{"ドットを含む", "abc.png", false},
		{"空白を含む", "abc def", false},
		{"パーセントエンコードを含む", "abc%2F", false},
		{"日本語を含む", "デッキ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidDeckCodeFormat(tt.code); got != tt.want {
				t.Errorf("IsValidDeckCodeFormat(%q) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}
