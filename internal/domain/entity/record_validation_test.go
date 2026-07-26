package entity

import "testing"

func TestIsValidRecordEventSource(t *testing.T) {
	tests := []struct {
		name string
		src  RecordEventSource
		want bool
	}{
		{"公式イベントのみ", RecordEventSource{OfficialEventId: 1}, true},
		{"Tonamelのみ", RecordEventSource{TonamelEventId: "abc"}, true},
		{"フレンド対戦のみ", RecordEventSource{FriendId: "f1"}, true},
		{"自由形式のみ", RecordEventSource{UnofficialEventId: "u1"}, true},

		{"1つも指定なし", RecordEventSource{}, false},
		{"公式とTonamelの2つ", RecordEventSource{OfficialEventId: 1, TonamelEventId: "abc"}, false},
		{"フレンドと自由形式の2つ", RecordEventSource{FriendId: "f1", UnofficialEventId: "u1"}, false},
		{"4つすべて指定", RecordEventSource{OfficialEventId: 1, TonamelEventId: "abc", FriendId: "f1", UnofficialEventId: "u1"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidRecordEventSource(tt.src); got != tt.want {
				t.Errorf("IsValidRecordEventSource() = %v, want %v", got, tt.want)
			}
		})
	}
}
