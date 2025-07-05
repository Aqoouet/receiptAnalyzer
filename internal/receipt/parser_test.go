package receipt

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// templatesUnderTest is the list supplied to ParseReceiptAuto.
var templatesUnderTest = []Template{
	DefaultBeelineTemplate,
	DefaultTaxcomTemplate,
}

// receiptCase ties a sample HTML with the index of the template that should be
// selected by ParseReceiptAuto.  If wantIdx < 0 we only assert that at least
// one template matches (useful when layout may change but still detectable).
type receiptCase struct {
	name    string
	path    string
	wantIdx int // expected template index in templatesUnderTest, -1 = any
	wantErr bool
}

func TestParseReceiptAuto_TableDriven(t *testing.T) {
	cases := []receiptCase{
		{
			name:    "Beeline чек валидный",
			path:    "../../output/msg_html/20250702_115846_ofdreceipt@beeline.ru_Чек_на_119.99_₽_от_02.07.2025,_АО__ТОРГОВЫЙ_ДОМ__ПЕРЕКРЕСТОК_.html",
			wantIdx: 0,
			wantErr: false,
		},
		{
			name:    "Taxcom чек валидный",
			path:    "../../output/msg_html/20231026_043113_noreply@taxcom.ru_Кассовый_чек_от_ООО__Спар_Миддл_Волга__за_26.10.2023.html",
			wantIdx: 1,
			wantErr: false,
		},
		{
			name:    "Пустой файл",
			path:    "testdata/empty.html",
			wantIdx: -1,
			wantErr: true,
		},
		{
			name:    "Невалидный HTML",
			path:    "testdata/invalid.html",
			wantIdx: -1,
			wantErr: true,
		},
		{
			name:    "Неизвестный формат",
			path:    "testdata/unknown_format.html",
			wantIdx: -1,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := os.Stat(tc.path)
			if err != nil {
				if tc.wantErr {
					t.Skipf("test file %s not found, skipping negative case", tc.path)
				}
				t.Fatalf("failed to read test file: %v", err)
			}
			js, idx, err := ParseReceiptAuto(tc.path, templatesUnderTest)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tc.wantIdx >= 0 {
				assert.Equal(t, tc.wantIdx, idx)
			}
			var items []Item
			assert.NoError(t, json.Unmarshal([]byte(js), &items))
			assert.NotEmpty(t, items)
		})
	}
}

// TestParseFloat validates helper that converts Russian decimal strings.
func TestParseFloat_TableDriven(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		in     string
		want   float64
		wantOk bool
	}{
		{"Обычное число", "123.45", 123.45, true},
		{"Запятая как разделитель", "123,45", 123.45, true},
		{"Пустая строка", "", 0, false},
		{"Невалидное число", "abc", 0, false},
		{"Только точка", ".", 0, false},
		{"Только запятая", ",", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseFloat(tt.in)
			if tt.wantOk {
				assert.True(t, ok)
				assert.InDelta(t, tt.want, got, 0.001)
			} else {
				assert.False(t, ok)
			}
		})
	}
}
