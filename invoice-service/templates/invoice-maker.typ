// ── Helpers ──────────────────────────────────────────────────────────

#let add-zeros(num) = {
  let parts = str(num).split(".")
  let (whole, decimal) = if parts.len() == 2 { parts } else { (num, "00") }
  str(whole) + "." + (str(decimal) + "00").slice(0, 2)
}

#let parse-date(date-str) = {
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

// UK-style date display: "1 May 2026"
#let fmt-date(date-str) = {
  let d = parse-date(date-str)
  d.display("[day padding:none] [month repr:long] [year]")
}

// Month display: "May 2026"
#let fmt-month(date-str) = {
  let d = parse-date(date-str)
  d.display("[month repr:long] [year]")
}

// Currency formatter with thousands separator
#let fmt-price(num, currency: "£") = {
  let dec = add-zeros(num)
  let parts = dec.split(".")
  let integer = parts.at(0)
  let decimal = parts.at(1)

  let formatted = ""
  let len = integer.len()
  for ii in range(len) {
    if ii > 0 and calc.rem(ii, 3) == 0 {
      formatted = "," + formatted
    }
    formatted = integer.at(-ii - 1) + formatted
  }

  currency + formatted + "." + decimal
}

#let TODO = box(
  inset: (x: 0.5em),
  outset: (y: 0.2em),
  radius: 0.2em,
  fill: rgb(255, 180, 170),
)[#text(size: 0.8em, weight: 600, fill: rgb(100, 68, 64))[TODO]]

// ── Labels (UK English) ─────────────────────────────────────────────

#let labels = (
  recipient: "To",
  biller: "From",
  invoice: "INVOICE",
  invoice-id: "Invoice Number",
  issuing-date: "Date",
  delivery-date: "Period",
  due: "Due Date",
  items: "Services",
  number: "#",
  description: "Description",
  service-dates: "Dates",
  duration: "Hours",
  price: "Rate",
  total: "Amount",
  total-time: "Total Hours",
  no-vat: "Not VAT Registered",
  subtotal-label: "Subtotal",
  total-label: "Total",
  due-text: val => [Payment due by *#val*],
  bank: "Bank",
  account-name: "Account Name",
  sort-code: "Sort Code",
  utr: "UTR",
  account-number: "Account No.",
  closing: "Thank you for your business!",
)

// ── Address formatting ──────────────────────────────────────────────

#let join-address-lines(entity) = {
  let lines = ()
  if entity.name != "" { lines.push(entity.name) }
  if "title" in entity and entity.title != "" { lines.push(entity.title) }
  if entity.address.street != "" { lines.push(entity.address.street) }
  if entity.address.city != "" or entity.address.postal-code != "" {
    lines.push(entity.address.city + " " + entity.address.postal-code)
  }
  if "country" in entity.address and entity.address.country != "" {
    lines.push(entity.address.country)
  }
  lines.map(line => [#line]).join([#linebreak()])
}

#let join-address-inline(entity) = {
  let parts = ()
  if entity.name != "" { parts.push(entity.name) }
  if "title" in entity and entity.title != "" { parts.push(entity.title) }
  if entity.address.street != "" { parts.push(entity.address.street) }
  if entity.address.city != "" or entity.address.postal-code != "" {
    parts.push(entity.address.city + " " + entity.address.postal-code)
  }
  if "country" in entity.address and entity.address.country != "" {
    parts.push(entity.address.country)
  }
  parts.join(", ")
}

// ── Main invoice function ───────────────────────────────────────────

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
  // ── Styling defaults ─────────────────────────────────────────────

  styling.font = styling.at("font", default: "Noto Sans")
  styling.font-size = styling.at("font-size", default: 10pt)
  styling.margin = styling.at("margin", default: (
    top: 14mm,
    right: 18mm,
    bottom: 14mm,
    left: 18mm,
  ))

  let accent = rgb("1a1a2e")
  let rule-color = rgb("cccccc")
  let muted = rgb("666666")

  // ── Date handling ────────────────────────────────────────────────

  let invoice-id-value = if invoice-id != none { invoice-id } else { TODO }
  let issue-date-value = if issuing-date != none {
    issuing-date
  } else {
    datetime.today().display("[year]-[month]-[day]")
  }
  let period-value = if delivery-date != none { delivery-date } else { TODO }
  let show-payment-due = biller.at("show-payment-due", default: true)
  let due-date-value = if due-date != none {
    due-date
  } else {
    (parse-date(issue-date-value) + duration(days: 14)).display("[year]-[month]-[day]")
  }

  // ── Document setup ───────────────────────────────────────────────

  set document(title: title, keywords: keywords, date: parse-date(issue-date-value))
  set page(margin: styling.margin)
  set par(justify: false)
  set text(
    font: if styling.font == none { "Noto Sans" } else { styling.font },
    size: styling.font-size,
  )
  set table(stroke: none)

  // ── Compute line items ───────────────────────────────────────────

  let rows = items
    .enumerate()
    .map(((index, row)) => {
      let hours = row.at("dur-min", default: 0) / 60
      let hourly-rate = row.at("hourly-rate", default: 0)
      let line-total = if row.at("dur-min", default: 0) == 0 {
        row.at("price", default: 0) * row.at("quantity", default: 1)
      } else {
        calc.round(hourly-rate * hours, digits: 2)
      }

      (
        number: row.at("number", default: index + 1),
        description: row.description,
        service-dates: row.at("service-dates", default: ""),
        hours-fmt: if row.at("dur-min", default: 0) == 0 { "" } else { add-zeros(hours) },
        rate-fmt: fmt-price(hourly-rate),
        total-fmt: fmt-price(line-total),
        raw-total: line-total,
        raw-minutes: row.at("dur-min", default: 0),
      )
    })

  let subtotal = rows.map(row => row.raw-total).sum()
  let total-hours = rows.map(row => row.raw-minutes).sum() / 60
  let tax = subtotal * vat
  let total = subtotal + tax

  // ══════════════════════════════════════════════════════════════════
  //  HEADER
  // ══════════════════════════════════════════════════════════════════

  // Header: INVOICE  |  INV-YYYY-MM-SLUG on one line
  grid(
    columns: (1fr, 1fr),
    align: (left, right),
    text(weight: "bold", size: 1.8em, fill: accent)[INVOICE], text(size: 1em, fill: muted)[#invoice-id-value],
  )

  v(0.8em)

  // Dates on one line: Date: ...   Period: ...   Due Date: ...
  let date-cells = (
    text(fill: muted, size: 0.9em)[#labels.issuing-date:],
    text(weight: "semibold", size: 0.9em)[#fmt-date(issue-date-value)],
    h(1.2em),
    text(fill: muted, size: 0.9em)[#labels.delivery-date:],
    text(weight: "semibold", size: 0.9em)[#fmt-month(period-value)],
  )
  if show-payment-due {
    date-cells.push(h(1.2em))
    date-cells.push(text(fill: muted, size: 0.9em)[#labels.due:])
    date-cells.push(text(weight: "semibold", size: 0.9em)[#fmt-date(due-date-value)])
  }

  grid(
    columns: if show-payment-due {
      (auto, auto, auto, auto, auto, auto, auto, auto)
    } else {
      (auto, auto, auto, auto, auto)
    },
    column-gutter: 0.4em,
    ..date-cells,
  )

  v(1em)
  line(length: 100%, stroke: 1.5pt + accent)
  v(0.9em)

  // ══════════════════════════════════════════════════════════════════
  //  PARTIES (From / To)
  // ══════════════════════════════════════════════════════════════════

  grid(
    columns: (1fr, 1fr),
    column-gutter: 0.4em,
    align: (left, left),
    text(fill: muted, size: 0.9em)[#labels.biller:] + text(weight: "semibold", size: 0.9em)[ #join-address-inline(biller)],
    text(fill: muted, size: 0.9em)[#labels.recipient:] + text(weight: "semibold", size: 0.9em)[ #join-address-inline(recipient)],
  )

  v(0.7em)
  line(length: 100%, stroke: 0.5pt + rule-color)
  v(0.8em)

  // ══════════════════════════════════════════════════════════════════
  //  SERVICES TABLE
  // ══════════════════════════════════════════════════════════════════

  text(weight: "bold", size: 1.1em, fill: accent)[#labels.items:]
  v(0.5em)

  table(
    columns: (auto, 1fr, 1fr, auto, auto, auto),
    align: (col, row) => if row == 0 {
      (right, left, left, center, right, right).at(col)
    } else {
      (right, left, left, right, right, right).at(col)
    },
    inset: (x: 7pt, y: 5pt),
    table.header(
      table.hline(stroke: 0.8pt + accent),
      [*#labels.number*],
      [*#labels.description*],
      [*#labels.service-dates*],
      [*#labels.duration*],
      [*#labels.price*],
      [*#labels.total* #text(size: 0.85em, fill: muted)[(#currency)]],
      table.hline(stroke: 0.4pt + rule-color),
    ),
    ..rows
      .map(row => (
        str(row.number),
        row.description,
        row.service-dates,
        row.hours-fmt,
        row.rate-fmt,
        row.total-fmt,
      ))
      .flatten(),
    table.hline(stroke: 0.8pt + accent),
  )

  v(0.7em)

  // ══════════════════════════════════════════════════════════════════
  //  TOTALS
  // ══════════════════════════════════════════════════════════════════

  let summary-rows = (
    (text(fill: muted, size: 0.95em)[#labels.subtotal-label], text(weight: "semibold")[#fmt-price(subtotal)]),
    if vat != 0 {
      (
        text(fill: muted, size: 0.95em)[VAT (#calc.round(vat * 100, digits: 0)%)],
        text(weight: "semibold")[#fmt-price(tax)],
      )
    },
    (
      text(fill: muted, size: 0.95em)[#labels.total-time],
      text(weight: "semibold")[#add-zeros(calc.round(total-hours, digits: 2)) h],
    ),
    if vat == 0 {
      (text(fill: muted, size: 0.9em)[#labels.no-vat], [])
    },
  ).filter(row => row != none)

  align(right)[
    #table(
      columns: (auto, auto),
      column-gutter: 2em,
      align: (right, right),
      inset: (x: 0pt, y: 3pt),
      ..summary-rows.flatten(),
    )
    #v(0.5em)
    #table(
      columns: (auto, auto),
      column-gutter: 2em,
      align: (right, right),
      fill: accent,
      inset: (x: 10pt, y: 6pt),
      text(weight: "bold", fill: white, size: 1.05em)[#labels.total-label],
      text(weight: "bold", fill: white, size: 1.05em)[#fmt-price(total)],
    )
  ]

  v(1.4em)

  // ══════════════════════════════════════════════════════════════════
  //  PAYMENT DETAILS
  // ══════════════════════════════════════════════════════════════════

  line(length: 100%, stroke: 0.5pt + rule-color)
  v(0.8em)

  text(weight: "bold", size: 1em, fill: accent)[Payment Details:]
  v(0.4em)

  grid(
    columns: (1fr, 1fr),
    row-gutter: 1em,
    column-gutter: 2em,
    text(
      fill: muted,
      size: 0.9em,
    )[#labels.bank: #text(fill: black, weight: "semibold")[#biller.at("bank", default: "—")]],
    text(
      fill: muted,
      size: 0.9em,
    )[#labels.sort-code: #text(fill: black, weight: "semibold")[#biller.at("sort-code", default: "—")]],
    text(
      fill: muted,
      size: 0.9em,
    )[#labels.utr: #text(fill: black, weight: "semibold")[#biller.at("utr", default: "—")]],

    text(
      fill: muted,
      size: 0.9em,
    )[#labels.account-name: #text(fill: black, weight: "semibold")[#biller.at("account-name", default: biller.name)]],
    text(
      fill: muted,
      size: 0.9em,
    )[#labels.account-number: #text(fill: black, weight: "semibold")[#biller.at("account-number", default: "—")]],
  )

  v(0.7em)
  line(length: 100%, stroke: 0.5pt + rule-color)
  v(0.8em)

  // ══════════════════════════════════════════════════════════════════
  //  FOOTER
  // ══════════════════════════════════════════════════════════════════

  if show-payment-due {
    (labels.due-text)(fmt-date(due-date-value))

    if "payment-terms" in biller and biller.payment-terms != "" [
      #v(0.3em)
      #text(fill: muted, size: 0.9em)[#biller.payment-terms]
    ]
  }

  v(0.8em)
  align(center)[
    #text(fill: muted, style: "italic")[#labels.closing]
  ]
  doc
}
