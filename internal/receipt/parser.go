package receipt

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Item represents a single purchase line in a receipt.
// Price and Quantity are kept as strings because different shops may use
// different units (шт., кг, л и т.д.) or decimal delimiters (comma/dot).
// A downstream consumer can normalise them if necessary.
//
// The struct tags allow the list of items to be marshalled directly to JSON.
// Example JSON output:
// [
//
//	{
//	  "name": "Хлеб", "quantity": "2 шт.", "price": "49.99"
//	}
//
// ]
//
// If you need numeric values, convert price/quantity manually after parsing.
// We avoid assumptions here to stay template-agnostic.
type Item struct {
	Name     string `json:"name"`
	Quantity string `json:"quantity"`
	Price    string `json:"price"`
}

// Template describes CSS selectors that allow the parser to locate the
// repeating blocks and the desired fields inside each block.
//
//   - ItemSelector   – selects the root node for every item (executed first).
//   - NameSelector   – looked-up inside an item node, must yield exactly one element that contains the item name.
//   - PriceSelector  – looked-up inside an item node, extracts the price (unit or total – depends on template).
//   - QtySelector    – looked-up inside an item node, extracts the quantity text.
//
// Selectors follow standard CSS syntax as used by goquery (jQuery-like).
// The defaults for the current Beeline/OFD "Перекрёсток" cheque are provided
// via DefaultBeelineTemplate and may be used as a starting point for other
// layouts.
//
// If a selector yields multiple matches, only the first one is used.
// Empty matches are skipped.
//
// You can optionally specify a PriceCleanupRegexp (e.g. "[^0-9.,]") and
// QtyCleanupRegexp to post-process raw strings before they are written to
// Item.Price and Item.Quantity.
type Template struct {
	ItemSelector  string
	NameSelector  string
	PriceSelector string
	QtySelector   string

	PriceCleanupRegexp *regexp.Regexp
	QtyCleanupRegexp   *regexp.Regexp
}

// Шаблоны по умолчанию вынесены в templates.go

var companyRegexp = regexp.MustCompile(`(?i)(АО|ООО|ОАО|ЗАО|ИП)`) // Russian company forms

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.TrimSpace(s)
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return s
}

// ParseReceiptFile reads the given HTML file and extracts items using the
// supplied template.  The result is a JSON-encoded []Item.
func ParseReceiptFile(path string, tpl Template) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	return parseReceipt(f, tpl)
}

// parseReceipt is the shared implementation for ParseReceiptFile and tests.
func parseReceipt(r io.Reader, tpl Template) (string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return "", fmt.Errorf("html parse: %w", err)
	}

	var items []Item

	doc.Find(tpl.ItemSelector).Each(func(_ int, itemSel *goquery.Selection) {
		// Name
		name := strings.TrimSpace(itemSel.Find(tpl.NameSelector).Last().Text())
		if name == "" {
			return // cannot identify – skip
		}

		// Price
		priceRaw := strings.TrimSpace(itemSel.Find(tpl.PriceSelector).First().Text())
		if tpl.PriceCleanupRegexp != nil {
			priceRaw = tpl.PriceCleanupRegexp.ReplaceAllString(priceRaw, "")
		}

		// Quantity
		qtyRaw := strings.TrimSpace(itemSel.Find(tpl.QtySelector).First().Text())
		if tpl.QtyCleanupRegexp != nil {
			qtyRaw = tpl.QtyCleanupRegexp.ReplaceAllString(qtyRaw, "")
		}

		items = append(items, Item{
			Name:     name,
			Quantity: qtyRaw,
			Price:    priceRaw,
		})
	})

	// Marshal
	js, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return string(js), nil
}

// ParseReceiptAuto tries each template from the slice and returns the
// first successful parse that yields at least one item.
// It returns the JSON string and the index of the template used.
func ParseReceiptAuto(path string, templates []Template) (jsonOut string, used int, err error) {
	for i, tpl := range templates {
		js, e := ParseReceiptFile(path, tpl)
		if e != nil {
			continue
		}
		var tmp []Item
		if err2 := json.Unmarshal([]byte(js), &tmp); err2 != nil {
			continue
		}
		if len(tmp) > 0 {
			return js, i, nil
		}
	}
	return "", -1, fmt.Errorf("no template matched %s", path)
}

// --- Helpers that might be useful in future extensions ---

// ParseFloat tries to convert Russian/European decimal strings (comma as
// separator) into float64.  If conversion fails, returns 0 and false.
func ParseFloat(raw string) (float64, bool) {
	raw = strings.ReplaceAll(raw, " ", "") // remove thin spaces etc.
	raw = strings.ReplaceAll(raw, ",", ".")
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// ExtractShop tries to fetch seller name.
func ExtractShop(doc *goquery.Document) string {
	// Taxcom layout.
	if name := cleanText(doc.Find("div.receipt-company-name span, div.receipt-company-name").First().Text()); companyRegexp.MatchString(name) {
		return name
	}
	// Beeline OFD — найти абзац, содержащий форму юрлица.
	var found string
	doc.Find("p").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		txt := cleanText(sel.Text())
		if companyRegexp.MatchString(txt) {
			found = txt
			return false // stop
		}
		return true
	})
	if found != "" {
		return found
	}
	return ""
}
