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

// Шаблон Яндекс.Маркет (чек с таблицей .receipt-table)
var YandexMarketTemplate = Template{
	ItemSelector:       "table.receipt-table tr",
	NameSelector:       "td:nth-child(2)",
	PriceSelector:      "td:nth-child(3)",
	QtySelector:        "td:nth-child(4)",
	PriceCleanupRegexp: regexp.MustCompile(`[^0-9.,]`),
	QtyCleanupRegexp:   regexp.MustCompile(`[^0-9.,]`),
}

// Шаблон OFD.ru (noreply@ofd.ru)
var DefaultOFDruTemplate = Template{
	// Каждая позиция: <tr> с <td align="left"><b>...</b></td> и <td align="right"><span>1 X 1100.00</span></td>
	ItemSelector:       "tr",
	NameSelector:       "td[align='left'] b",
	QtySelector:        "td[align='right'] span",
	PriceSelector:      "td[align='right'] span",
	PriceCleanupRegexp: regexp.MustCompile(`([0-9]+[.,][0-9]+)$`), // цена после X
	QtyCleanupRegexp:   regexp.MustCompile(`^(\d+)\s*[xXХ]`),      // количество до X
}

// Шаблон OFD.ru (noreply@ofd.ru) — вложенная таблица с <b> и <span>1 X 1100.00</span>
var OFDruNestedTableTemplate = Template{
	ItemSelector:       "tr",
	NameSelector:       "td:first-child b",
	QtySelector:        "td:last-child span",
	PriceSelector:      "td:last-child span",
	PriceCleanupRegexp: regexp.MustCompile(`X\s*([0-9]+[.,]?[0-9]*)`), // после X
	QtyCleanupRegexp:   regexp.MustCompile(`([0-9]+)\s*X`),            // до X
}

// Альтернативная раскладка писем ofdreceipt@beeline.ru («OTHER_OFD_100+»):
// блок товара – это любая таблица, внутри которой есть строка "Цена*Кол".
// HTML по-прежнему использует жирный шрифт для названия, поэтому селекторы
// совпадают с DefaultBeelineTemplate, но ItemSelector упрощён, чтобы не
// зависеть от inline-стилей.
var BeelinePriceTableTemplate = Template{
	ItemSelector:       "table[style*='font-size: 15px'][style*='line-height: 19px']",
	NameSelector:       "td span[style*='font-weight: bold']",
	PriceSelector:      "td:contains('Цена*Кол') + td",
	QtySelector:        "td:contains('Цена*Кол') + td + td",
	PriceCleanupRegexp: regexp.MustCompile(`[^0-9.,]`),
	QtyCleanupRegexp:   regexp.MustCompile(`[^0-9.,]`),
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
	ItemSelector: ".check-section",
	NameSelector: ".check-product-name",
	// В той же секции строка с "1 х 5520.00" — берём правую колонку
	QtySelector:        ".check-col-right",
	PriceSelector:      ".check-col-right",
	PriceCleanupRegexp: nil, // обработка в parser.go
	QtyCleanupRegexp:   nil, // обработка в parser.go
}

// Шаблон Яндекс.ОФД (простая таблица: №, Наименование, Сумма)
var YandexOFDPlainTableTemplate = Template{
	ItemSelector:       "table tr",
	NameSelector:       "td.text_left",
	PriceSelector:      "td:last-child",
	QtySelector:        "td:last-child",
	PriceCleanupRegexp: nil, // обработка в parser.go
	QtyCleanupRegexp:   nil, // обработка в parser.go
}

// Шаблон MTS (receipt@mts.ru) — платеж, не товарный чек
var MTSPaymentTemplate = Template{
	ItemSelector:       "table[style*='width: 450px'] tr",
	NameSelector:       "td span:contains('Сумма (итого)')",
	PriceSelector:      "td span:contains('Сумма (итого)') ~ td span",
	QtySelector:        "td span:contains('Сумма (итого)') ~ td span", // quantity всегда 1
	PriceCleanupRegexp: regexp.MustCompile(`([0-9]+[.,]?[0-9]*)`),
	QtyCleanupRegexp:   regexp.MustCompile(`.*`), // quantity всегда 1, обработка в коде
}

// Шаблон Uniteller (check@uniteller.ru)
var UnitellerTemplate = Template{
	ItemSelector:       "table.goods tr.newline",
	NameSelector:       "td.item-name",
	PriceSelector:      "td:nth-child(4)",
	QtySelector:        "td:nth-child(5)",
	PriceCleanupRegexp: nil,
	QtyCleanupRegexp:   nil,
}

// Шаблон ofd_ya_kassa (no-reply@ofd-ya-kassa.ru)
var OfdYaKassaTemplate = Template{
	ItemSelector:       "table.check_item",
	NameSelector:       "tr:nth-child(1) td",
	QtySelector:        "tr:nth-child(2) td:first-child",
	PriceSelector:      "tr:nth-child(2) td:last-child",
	PriceCleanupRegexp: regexp.MustCompile(`x\s*([0-9]+[.,]?[0-9]*)`), // после x
	QtyCleanupRegexp:   regexp.MustCompile(`([0-9]+)\s*x`),            // до x
}

// Шаблон для OTHER_OFD_100 (ofdreceipt@beeline.ru) - специальный случай
var BelineOFD100Template = Template{
	ItemSelector:       "table[style*='color: #4a4a4a'][style*='font-size: 15px'][style*='line-height: 19px']",
	NameSelector:       "td:not([width='44']) span[style*='font-weight: bold']",
	PriceSelector:      "td:contains('Цена*Кол') + td",
	QtySelector:        "td:contains('Цена*Кол') + td + td",
	PriceCleanupRegexp: regexp.MustCompile(`[^0-9.,]`),
	QtyCleanupRegexp:   regexp.MustCompile(`[^0-9.,]`),
}

// Шаблон для случаев без товаров (другое)
var EmptyReceiptTemplate = Template{
	ItemSelector:       "body", // всегда найдется, но не содержит товаров
	NameSelector:       "nonexistent",
	PriceSelector:      "nonexistent",
	QtySelector:        "nonexistent",
	PriceCleanupRegexp: nil,
	QtyCleanupRegexp:   nil,
}

// Шаблон для OFD.ru авансов (noreply@ofd.ru)
var OfdRuAdvanceTemplate = Template{
	ItemSelector:       "tr",
	NameSelector:       "td[align='left'][width='50%'] span b",
	PriceSelector:      "td[align='right'] span",
	QtySelector:        "td[align='right'] span",
	PriceCleanupRegexp: regexp.MustCompile(`([0-9]+[.,]?[0-9]*)\s*[Xx×]\s*([0-9]+[.,]?[0-9]*)`), // извлекаем вторую часть (цена за единицу)
	QtyCleanupRegexp:   regexp.MustCompile(`([0-9]+[.,]?[0-9]*)\s*[Xx×]\s*([0-9]+[.,]?[0-9]*)`), // извлекаем первую часть (количество)
}

// Шаблон для Mail.ru авансов (fiscal@corp.mail.ru)
var MailruAdvanceTemplate = Template{
	ItemSelector:       "tr",
	NameSelector:       "td[colspan='2'] b",
	PriceSelector:      "td div span",
	QtySelector:        "td div span",
	PriceCleanupRegexp: regexp.MustCompile(`([0-9]+[.,]?[0-9]*)$`), // последнее число (цена)
	QtyCleanupRegexp:   regexp.MustCompile(`^([0-9]+)`),            // первое число (количество)
}

// Шаблон для ofd.ru со сложной структурой
var OFDRuComplexLayout = Template{
	ItemSelector:  "table[width='460'] table[width='100%'] > tbody > tr",
	NameSelector:  "td:nth-child(1)",
	PriceSelector: "td:nth-child(2)",
	QtySelector:   "td:nth-child(2)",
}
