#let nbh = "‑"

// Truncate a number to 2 decimal places
// and add trailing zeros if necessary
#let add-zeros = (num) => {
    let frags = str(num).split(".")
    let (intp, decp) = if frags.len() == 2 { frags } else { (num, "00") }
    str(intp) + "." + (str(decp) + "00").slice(0, 2)
  }

// IBAN length and BBAN structure per country.
#let iban-formats = (
    AD: (24, "8n12c"),    AE: (23, "19n"),       AL: (28, "8n16c"),
    AT: (20, "16n"),      AZ: (28, "4a20c"),     BA: (20, "16n"),
    BE: (16, "12n"),      BG: (22, "4a6n8c"),    BH: (22, "4a14c"),
    BI: (27, "23n"),      BR: (29, "23n1a1c"),   BY: (28, "4c4n16c"),
    CH: (21, "5n12c"),    CR: (22, "18n"),       CY: (28, "8n16c"),
    CZ: (24, "20n"),      DE: (22, "18n"),       DJ: (27, "23n"),
    DK: (18, "14n"),      DO: (28, "4c20n"),     EE: (20, "16n"),
    EG: (29, "25n"),      ES: (24, "20n"),       FI: (18, "14n"),
    FK: (18, "2a12n"),    FO: (18, "14n"),       FR: (27, "10n11c2n"),
    GB: (22, "4a14n"),    GE: (22, "2a16n"),     GI: (23, "4a15c"),
    GL: (18, "14n"),      GR: (27, "7n16c"),     GT: (28, "24c"),
    HN: (28, "4a20n"),    HR: (21, "17n"),       HU: (28, "24n"),
    IE: (22, "4a14n"),    IL: (23, "19n"),       IQ: (23, "4a15n"),
    IS: (26, "22n"),      IT: (27, "1a10n12c"),  JO: (30, "4a4n18c"),
    KW: (30, "4a22c"),    KZ: (20, "3n13c"),     LB: (28, "4n20c"),
    LC: (32, "4a24c"),    LI: (21, "5n12c"),     LT: (20, "16n"),
    LU: (20, "3n13c"),    LV: (21, "4a13c"),     LY: (25, "21n"),
    MC: (27, "10n11c2n"), MD: (24, "20c"),       ME: (22, "18n"),
    MK: (19, "3n10c2n"),  MN: (20, "16n"),       MR: (27, "23n"),
    MT: (31, "4a5n18c"),  MU: (30, "4a19n3a"),   NI: (28, "4a20n"),
    NL: (18, "4a10n"),    NO: (15, "11n"),       OM: (23, "3n16c"),
    PK: (24, "4a16c"),    PL: (28, "24n"),       PS: (29, "4a21c"),
    PT: (25, "21n"),      QA: (29, "4a21c"),     RO: (24, "4a16c"),
    RS: (22, "18n"),      RU: (33, "14n15c"),    SA: (24, "2n18c"),
    SC: (31, "4a20n3a"),  SD: (18, "14n"),       SE: (24, "20n"),
    SI: (19, "15n"),      SK: (24, "20n"),       SM: (27, "1a10n12c"),
    SO: (23, "19n"),      ST: (25, "21n"),       SV: (28, "4a20n"),
    TL: (23, "19n"),      TN: (24, "20n"),       TR: (26, "5n17c"),
    UA: (29, "6n19c"),    VA: (22, "18n"),       VG: (24, "4a16n"),
    XK: (20, "16n"),      YE: (30, "4a4n18c"),
  )

#let _bban-fmt-to-regex = (fmt) => {
    let parts = ()
    let count = ""
    for c in fmt {
      let code = c.to-unicode()
      if code >= 48 and code <= 57 {
        count += c
      } else {
        let cls = if c == "n" { "[0-9]" }
                  else if c == "a" { "[A-Z]" }
                  else if c == "c" { "[A-Z0-9]" }
                  else { panic("Unknown BBAN format spec: " + c) }
        parts.push(cls + "{" + count + "}")
        count = ""
      }
    }
    "^" + parts.join("") + "$"
  }

#let mod-97 = (digits) => {
    let r = 0
    for c in digits {
      r = calc.rem(r * 10 + int(c), 97)
    }
    r
  }

#let verify-iban = (iban) => {
    if iban == "" { return true }
    let normalized = upper(iban).replace(" ", "").replace("-", "")
    if normalized.find(regex("^[A-Z]{2}[0-9]{2}[A-Z0-9]+$")) == none {
      return false
    }
    let country = normalized.slice(0, 2)
    let spec = iban-formats.at(country, default: none)
    if spec == none { return false }
    let (expected-length, bban-fmt) = spec
    if normalized.len() != expected-length { return false }
    let bban = normalized.slice(4)
    if bban.find(regex(_bban-fmt-to-regex(bban-fmt))) == none {
      return false
    }
    let rearranged = bban + normalized.slice(0, 4)
    let numeric = ""
    for c in rearranged {
      let code = c.to-unicode()
      numeric += if code >= 48 and code <= 57 { c }
                 else { str(code - 55) }
    }
    mod-97(numeric) == 1
}

#let parse-date = (date-str) => {
  let parts = date-str.split("-")
  if parts.len() != 3 {
    panic("Invalid date string: " + date-str)
  }
  datetime(
    year: int(parts.at(0)),
    month: int(parts.at(1)),
    day: int(parts.at(2)),
  )
}

#let TODO = box(
  inset: (x: 0.5em),
  outset: (y: 0.2em),
  radius: 0.2em,
  fill: rgb(255,180,170),
)[#text(size: 0.8em, weight: 600, fill: rgb(100,68,64))[TODO]]

#let horizontalrule = [
  #v(8mm)
  #line(start: (20%,0%), end: (80%,0%), stroke: 0.8pt + gray)
  #v(8mm)
]

#let languages = (
    en: (
      id: "en", country: "GB",
      recipient: "To", biller: "From",
      invoice: "Invoice",
      cancellation-invoice: "Cancellation Invoice",
      cancellation-notice: (id, issuing-date) => [
        As agreed, you will receive a credit note
        for the invoice *#id* dated *#issuing-date*.
      ],
      invoice-id: "Invoice Number",
      issuing-date: "Invoice Date",
      delivery-date: "Period",
      items: "Services",
      closing: "Thank you for your business!",
      payment-terms: "Payment Terms",
      number: "#",
      date: "Date",
      description: "Description",
      duration: "Hours",
      quantity: "Qty",
      price: "Rate (£)",
      total-time: "Total Hours",
      subtotal: "Subtotal",
      discount-of: "Discount",
      vat: "VAT",
      no-vat: "Not VAT Registered",
      reverse-charge: "Reverse Charge",
      total: "Total",
      due-text: val => [Payment due by *#val* to:],
      bank: "Bank",
      account-name: "Account Name",
      sort-code: "Sort Code",
      account-number: "Account No.",
      iban: "IBAN",
    ),
    fr: (
      id: "fr", country: "FR",
      recipient: "Destinataire", biller: "Émetteur",
      invoice: "Facture",
      cancellation-invoice: "Annulation de facture",
      cancellation-notice: (id, issuing-date) => [
        Comme convenu, vous recevrez un crédit
        pour la facture *#id* du *#issuing-date*.
      ],
      invoice-id: "Facture N°", issuing-date: "Date d'émission",
      delivery-date: "Date de livraison", items: "Produits",
      closing: "Merci !", number: "N°", date: "Date",
      description: "Description", duration: "Durée",
      quantity: "Quantité", price: "Prix",
      total-time: "Temps total travaillé", subtotal: "Sous-total",
      discount-of: "Remise de", vat: "TVA",
      no-vat: "Non sujet à la TVA", reverse-charge: "Facturation inversée",
      total: "Total",
      due-text: val => [Merci de régler d'ici le *#val* par virement au compte bancaire suivant:],
      bank: "Banque", account-name: "Titulaire",
      sort-code: "Code Banque", account-number: "N° Compte",
      iban: "IBAN",
    ),
    de: (
      id: "de", country: "DE",
      recipient: "Empfänger", biller: "Aussteller",
      invoice: "Rechnung",
      cancellation-invoice: "Stornorechnung",
      cancellation-notice: (id, issuing-date) => [
        Vereinbarungsgemäß erhalten Sie hiermit eine Gutschrift
        zur Rechnung *#id* vom *#issuing-date*.
      ],
      invoice-id: "Rechnungsnummer", issuing-date: "Ausstellungsdatum",
      delivery-date: "Lieferdatum", items: "Leistungen",
      closing: "Vielen Dank für die gute Zusammenarbeit!", number: "Nr",
      date: "Datum", description: "Beschreibung", duration: "Dauer",
      quantity: "Menge", price: "Preis",
      total-time: "Gesamtarbeitszeit", subtotal: "Zwischensumme",
      discount-of: "Rabatt von", vat: "Umsatzsteuer von",
      no-vat: "Nicht Umsatzsteuerpflichtig",
      reverse-charge: "Steuerschuldnerschaft des\nLeistungsempfängers",
      total: "Gesamt",
      due-text: val => [Bitte überweise den Betrag bis *#val* auf folgendes Konto:],
      bank: "Bank", account-name: "Inhaber",
      sort-code: "Bankleitzahl", account-number: "Kontonummer",
      iban: "IBAN",
    ),
  )

#let invoice(
  language: "en",
  currency: "£",
  title: none,
  invoice-id: none,
  cancellation-id: none,
  issuing-date: none,
  delivery-date: none,
  due-date: none,
  biller: (:),
  recipient: (:),
  keywords: (),
  styling: (:),
  items: (),
  discount: none,
  vat: 0,
  data: none,
  override-translation: none,
  target: "pdf",
  doc,
) = {
  let is-html = target == "html"
  let center-block = if is-html { it => it } else { it => align(center, it) }
  let v-space = if is-html { _ => [] } else { v }
  let pad-with = if is-html { (..args, body) => body } else { (..args, body) => pad(..args, body) }
  let box-with = if is-html { (..args, body) => body } else { (..args, body) => box(..args, body) }
  let block-with = if is-html { (..args, body) => body } else { (..args, body) => block(..args, body) }
  let columns-with = if is-html { (count, ..args, body) => body } else { (count, ..args, body) => columns(count, ..args, body) }

  styling.font = styling.at("font", default: "Liberation Sans")
  styling.font-size = styling.at("font-size", default: 11pt)
  styling.margin = styling.at("margin", default: (
    top: 20mm, right: 25mm, bottom: 20mm, left: 25mm,
  ))

  language = if data != none { data.at("language", default: language) } else { language }

  let t = if type(language) == str { languages.at(language) }
          else { language }

  if override-translation != none {
    for k in t.keys() {
      if override-translation.at(k, default: none) != none {
        t.insert(k, override-translation.at(k))
      }
    }
  }

  if data != none {
    language = data.at("language", default: language)
    currency = data.at("currency", default: currency)
    title = data.at("title", default: title)
    invoice-id = data.at("invoice-id", default: invoice-id)
    cancellation-id = data.at("cancellation-id", default: cancellation-id)
    issuing-date = data.at("issuing-date", default: issuing-date)
    delivery-date = data.at("delivery-date", default: delivery-date)
    due-date = data.at("due-date", default: due-date)
    biller = data.at("biller", default: biller)
    recipient = data.at("recipient", default: recipient)
    keywords = data.at("keywords", default: keywords)
    styling = data.at("styling", default: styling)
    items = data.at("items", default: items)
    discount = data.at("discount", default: discount)
    vat = data.at("vat", default: vat)
  }

  let signature = ""
  let issuing-date = if issuing-date != none { issuing-date }
        else { datetime.today().display("[year]-[month]-[day]") }

  set document(title: title, keywords: keywords, date: parse-date(issuing-date))

  let body = {
  set par(justify: true)
  set text(
    lang: t.id,
    font: if styling.font == none { "Libertinus Serif" } else { styling.font },
    size: styling.font-size,
  )
  set table(stroke: none)

  center-block(block-with(inset: 2em)[
    #text(weight: "bold", size: 2em)[
      #(if title != none { title } else {
        if cancellation-id != none { t.cancellation-invoice }
        else { t.invoice }
      })
    ]
  ])

  let invoice-id-norm = if invoice-id != none {
          if cancellation-id != none { cancellation-id }
          else { invoice-id }
        }
        else { TODO }

  let delivery-date = if delivery-date != none { delivery-date }
        else { TODO }

  center-block(
    table(
      columns: 2,
      align: (right, left),
      inset: 4pt,
      [#t.invoice-id:], [*#invoice-id-norm*],
      [#t.issuing-date:], [*#issuing-date*],
      [#t.delivery-date:], [*#delivery-date*],
    )
  )

  v-space(2em)

  box-with(height: 12em)[
    #columns-with(2, gutter: 4em)[
      === #t.recipient
      #v-space(0.5em)
      #recipient.name \
      #{if "title" in recipient { [#recipient.title \ ] }}
      #{if "country" in recipient.address { [#recipient.address.country \ ] }}
      #recipient.address.city #recipient.address.postal-code \
      #recipient.address.street

      === #t.biller
      #v-space(0.5em)
      #biller.name \
      #{if "title" in biller { [#biller.title \ ] }}
      #{if "country" in biller.address { [#biller.address.country \ ] }}
      #biller.address.city #biller.address.postal-code \
      #biller.address.street
    ]
  ]

  [== #t.items]

  v-space(1em)

  let getRowTotal = row => {
    if row.at("dur-min", default: 0) == 0 {
      row.price * row.at("quantity", default: 1)
    }
    else {
      calc.round(row.at("hourly-rate", default: 0) * (row.dur-min / 60), digits: 2)
    }
  }

  let cancel-neg = if cancellation-id != none { -1 } else { 1 }

  table(
    columns: (auto, 1fr, auto, auto, auto),
    align: (col, row) =>
        if row == 0 {
          (right,left,center,right,right,).at(col)
        }
        else {
          (right,left,right,right,right,).at(col)
        },
    inset: 6pt,
    table.header(
      table.hline(stroke: 0.5pt),
      [*#t.number*],
      [*#t.description*],
      [*#t.duration*],
      [*#t.price*],
      [*#t.total*\ #text(size: 0.8em)[( #currency )]],
      table.hline(stroke: 0.5pt),
    ),
    ..items
      .enumerate()
      .map(((index, row)) => {
        let dur-min = row.at("dur-min", default: 0)
        (
          row.at("number", default: index + 1),
          row.description,
          str(if dur-min == 0 { "" } else { dur-min }),
          str(add-zeros(cancel-neg *
           row.at("price", default: calc.round(
             row.at("hourly-rate", default: 0) * (dur-min / 60),
             digits: 2
           ))
          )),
          str(add-zeros(cancel-neg * getRowTotal(row))),
        )
      })
      .flatten()
      .map(str),
    table.hline(stroke: 0.5pt),
  )

  let sub-total = items.map(getRowTotal).sum()

  let total-duration = items
        .map(row => int(row.at("dur-min", default: 0)))
        .sum()

  let discount-value = 0
  let tax = sub-total * vat
  let total = sub-total - discount-value + tax

  let table-entries = (
    if total-duration != 0 {
      ([#t.total-time:], [*#str(total-duration) min*])
    },
    if vat != 0 {
      ([#t.subtotal:], [#{add-zeros(cancel-neg * sub-total)} #currency])
    },
    if vat != 0 {
      ([#t.vat #{vat * 100} %:], [#{add-zeros(cancel-neg * tax)} #currency])
    },
    if vat == 0 {([#t.no-vat], [ ])},
    (
      [*#t.total*:],
      [*#add-zeros(cancel-neg * total) #currency*]
    ),
  )
  .filter(entry => entry != none)

  let grayish = luma(245)

  align(right,
    table(
      columns: 2,
      fill: (col, row) =>
        if row == table-entries.len() - 1 { grayish }
        else { none },
      stroke: (col, row) =>
        if row == table-entries.len() - 1 { (y: 0.5pt, x: 0pt) }
        else { none },
      ..table-entries.flatten(),
    )
  )

  v-space(1em)

  if cancellation-id == none {
    let due-date = if due-date != none { due-date }
          else {
            (parse-date(issuing-date) + duration(days: 14))
              .display("[year]-[month]-[day]")
          }

    (t.due-text)(due-date)

    v-space(1em)
    center-block[
      #table(
        fill: grayish,
        columns: (10em, auto),
        inset: (col, row) =>
          if col == 0 {
            if row == 0 { (top: 1.2em, right: 0.6em, bottom: 0.6em) }
            else { (top: 0.6em, right: 0.6em, bottom: 1.2em) }
          }
          else {
            if row == 0 { (top: 1.2em, right: 2em, bottom: 0.6em, left: 0.6em) }
            else { (top: 0.6em, right: 2em, bottom: 1.2em, left: 0.6em) }
          },
        align: (col, row) => (right,left,).at(col),
        table.hline(stroke: 0.5pt),
        [#t.bank:], [*#biller.at("bank", default: "")*],
        [#t.account-name:], [*#biller.at("account-name", default: biller.name)*],
        [#t.sort-code:], [*#biller.at("sort-code", default: "")*],
        [#t.account-number:], [*#biller.at("account-number", default: "")*],
        table.hline(stroke: 0.5pt),
      )
    ]

    v-space(1em)

    if "payment-terms" in biller {
      [#biller.at("payment-terms")]
      v-space(1em)
    }

    t.closing
  }
  else {
    v-space(1em)
    center-block(strong(t.closing))
  }

  doc
  }

  if is-html {
    body
  } else {
    set page(
      margin: styling.margin,
      numbering: (current, total) => if total > 1 [#current / #total],
    )
    body
  }
}
