package receipt

import "regexp"

// Шаблон Beeline (Перекрёсток, письма ofdreceipt@beeline.ru)
var DefaultBeelineTemplate = Template{
	ItemSelector:       "table[style*='color: #4a4a4a'][style*='line-height: 19px']",
	NameSelector:       "td:not([width='44']) span[style*='font-weight: bold']",
	PriceSelector:      "td:contains('Цена*Кол') + td",
	QtySelector:        "td:contains('Цена*Кол') + td + td",
	PriceCleanupRegexp: regexp.MustCompile(`[^0-9.,]`),
	QtyCleanupRegexp:   regexp.MustCompile(`[^0-9.,]`),
}

// Шаблон Taxcom (noreply@taxcom.ru)
var DefaultTaxcomTemplate = Template{
	ItemSelector:       "div.item",
	NameSelector:       "span.receipt-value-1030",
	QtySelector:        "span.receipt-value-1023",
	PriceSelector:      "span.receipt-value-1079",
	PriceCleanupRegexp: regexp.MustCompile(`[^0-9.,]`),
	QtyCleanupRegexp:   regexp.MustCompile(`[^0-9.,a-zA-Zа-яА-Я ]`),
}

// Шаблон Yandex Music (music@support.yandex.ru). Селекторы основаны
// на типичном HTML чека: таблица с классом "item".
var DefaultMusicTemplate = Template{
	ItemSelector:       "table.item",
	NameSelector:       "td.name, span.name",
	QtySelector:        "td.qty",
	PriceSelector:      "td.price",
	PriceCleanupRegexp: regexp.MustCompile(`[^0-9.,]`),
	QtyCleanupRegexp:   regexp.MustCompile(`[^0-9.,]`),
}

// Шаблон Yandex OFD (no-reply@ofd.yandex.ru)
var DefaultYandexOFDTemplate = Template{
	// Каждая позиция находится внутри вложенной таблицы в блоке <tr class="content-row">.
	// Берём строку таблицы как item.
	ItemSelector:       "tr.content-row table tr",
	NameSelector:       "td:first-child",
	QtySelector:        "td span",
	PriceSelector:      "td:last-child",
	PriceCleanupRegexp: regexp.MustCompile(`[^0-9.,]`),
	QtyCleanupRegexp:   regexp.MustCompile(`[^0-9.,]`),
}

// Шаблон Yandex Market (noreply@market.yandex.ru)
var DefaultYandexMarketTemplate = Template{
	// Каждая позиция – строка таблицы, содержащая ссылку на товар (domain market.yandex.ru/product)
	ItemSelector:       "tr",
	NameSelector:       "a[href*='product']",
	QtySelector:        "td:last-child span", // кол-во обычно в той же ячейке
	PriceSelector:      "td:last-child",
	PriceCleanupRegexp: regexp.MustCompile(`[^0-9.,]`),
	QtyCleanupRegexp:   regexp.MustCompile(`[^0-9.,]`),
}

// Шаблон OFD.ru (noreply@ofd.ru)
var DefaultOFDruTemplate = Template{
	// Каждая позиция описана в строке <tr> с именем в теге <b>
	ItemSelector:       "tr",
	NameSelector:       "b",
	QtySelector:        "span:contains(' X ')",
	PriceSelector:      "span:contains('=')",
	PriceCleanupRegexp: regexp.MustCompile(`[^0-9.,]`),
	// Кол-во берём без очистки, чтобы сохранить формулу «1 X 500.00»
}

// Шаблон Первый ОФД (noreply@1-ofd.ru)
var DefaultFirstOFDTemplate = Template{
	// Таблица Courier New содержит строки позиций; выбираем все <tr> внутри неё.
	ItemSelector:       "table[style*='Courier New'] tr",
	NameSelector:       "td:nth-child(2)",
	PriceSelector:      "td:nth-child(3)",
	QtySelector:        "td:nth-child(4)",
	PriceCleanupRegexp: regexp.MustCompile(`[^0-9.,]`),
	QtyCleanupRegexp:   regexp.MustCompile(`[^0-9]`),
}

// Шаблон Платформа ОФД (noreply@chek.pofd.ru)
var DefaultPlatformaOFDTemplate = Template{
	// Секция товара — div.check-section, внутри есть div.check-product-name
	ItemSelector: "div.check-section",
	NameSelector: "div.check-product-name",
	// В той же секции строка с "1 х 5520.00" — берём правую колонку
	QtySelector:        "div.check-col-right:contains(' х ')",
	PriceSelector:      "div.check-col-right:contains(' х ')",
	PriceCleanupRegexp: regexp.MustCompile(`[^0-9.,]`),
	QtyCleanupRegexp:   regexp.MustCompile(`[^0-9]`),
}
