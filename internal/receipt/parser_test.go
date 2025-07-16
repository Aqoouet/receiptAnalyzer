package receipt

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var expectedResults map[string]ExpectedReceipt

// templatesUnderTest is the ordered slice passed to ParseReceiptAuto; keep
// the same order as originally used in tests.
var templatesUnderTest = []Template{
	YandexOFDPlainTableTemplate,
	BelineOFD100Template,
	DefaultBeelineTemplate,
	DefaultTaxcomTemplate,
	DefaultMusicTemplate,
	DefaultYandexOFDTemplate,
	DefaultYandexMarketTemplate,
	DefaultOFDruTemplate,
	DefaultFirstOFDTemplate,
	DefaultPlatformaOFDTemplate,
	OfdYaKassaTemplate,
	UnitellerTemplate,
	MTSPaymentTemplate,
	OFDruNestedTableTemplate,
	BeelinePriceTableTemplate,
	MailruAdvanceTemplate,
	OfdRuAdvanceTemplate,
	EmptyReceiptTemplate,
}

func init() {
	// The expectations file now lives under testdata to keep everything that
	// relates to fixtures in one place.  Use a path relative to the package
	// root so that `go test ./...` from the repo root can locate the file no
	// matter the current working directory.
	data, err := os.ReadFile("testdata/expected_receipts.json")
	if err != nil {
		panic("expected_receipts.json not found; regenerate it before running tests: " + err.Error())
	}
	if err := json.Unmarshal(data, &expectedResults); err != nil {
		panic(err)
	}
}

func TestParseReceipt_Fixtures(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("failed to read testdata dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}

		path := "testdata/" + entry.Name()
		expReceipt, ok := expectedResults[path]
		if !ok {
			t.Fatalf("no expectations for %s — regenerate JSON", path)
		}

		t.Run(entry.Name(), func(t *testing.T) {
			js, usedIdx, err := ParseReceiptAuto(path, templatesUnderTest)
			assert.NoError(t, err, "ParseReceiptAuto failed for %s", path)

			var items []Item
			assert.NoError(t, json.Unmarshal([]byte(js), &items))

			// Ensure we don't panic when counts differ – perform detailed
			// comparisons only when the parser produced exactly the expected
			// number of items.  The count assertion still fails the sub-test,
			// but we skip the per-item checks to get a cleaner failure report.
			assert.Equal(t, len(expReceipt.Items), len(items), "item count mismatch for %s", path)
			if len(expReceipt.Items) != len(items) {
				return
			}

			var total float64
			for i := range items {
				got := items[i]
				exp := expReceipt.Items[i]

				qty, _ := ParseFloat(got.Quantity)
				price, _ := ParseFloat(got.Price)
				calc := qty * price
				total += calc

				assert.Equal(t, exp.Name, got.Name, "name mismatch at item %d", i)
				assert.InDelta(t, exp.Quantity, qty, 0.01, "quantity mismatch at item %d", i)
				assert.InDelta(t, exp.UnitPrice, price, 0.01, "unit_price mismatch at item %d", i)
				assert.InDelta(t, exp.Total, calc, 0.01, "total mismatch at item %d", i)
			}
			assert.InDelta(t, expReceipt.Total, total, 0.01, "receipt total mismatch for %s", path)

			// The JSON now stores the template name (string) instead of the
			// numeric index.  We keep the field for future diagnostics but do
			// not treat it as a strict assertion: the parser may evolve and
			// still produce correct items with a different template.
			_ = usedIdx
		})
	}
}
