#let add-zeros = (num) => {
  let parts = str(num).split(".")
  let (whole, decimal) = if parts.len() == 2 { parts } else { (num, "00") }
  str(whole) + "." + (str(decimal) + "00").slice(0, 2)
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
  fill: rgb(255, 180, 170),
)[#text(size: 0.8em, weight: 600, fill: rgb(100, 68, 64))[TODO]]

#let labels = (
  recipient: "To",
  biller: "From",
  invoice: "Invoice",
  invoice-id: "Invoice Number",
  issuing-date: "Invoice Date",
  delivery-date: "Period",
  items: "Services",
  number: "#",
  description: "Description",
  duration: "Hours",
  price: "Rate (£)",
  total-time: "Total Hours",
  no-vat: "Not VAT Registered",
  total: "Total",
  due-text: val => [Payment due by *#val*],
  bank: "Bank",
  account-name: "Account Name",
  sort-code: "Sort Code",
  account-number: "Account No.",
  closing: "Thank you for your business!",
)

#let join-address-lines = entity => {
  let lines = ()

  if entity.name != "" { lines.push(entity.name) }
  if "title" in entity { lines.push(entity.title) }
  if entity.address.street != "" { lines.push(entity.address.street) }
  if entity.address.city != "" or entity.address.postal-code != "" {
    lines.push(entity.address.city + " " + entity.address.postal-code)
  }
  if "country" in entity.address and entity.address.country != "" {
    lines.push(entity.address.country)
  }

  lines.map(line => [#line]).join([#linebreak()])
}

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
  styling.font = styling.at("font", default: "Noto Sans")
  styling.font-size = styling.at("font-size", default: 11pt)
  styling.margin = styling.at("margin", default: (
    top: 20mm,
    right: 25mm,
    bottom: 20mm,
    left: 25mm,
  ))

  let invoice-id-value = if invoice-id != none { invoice-id } else { TODO }
  let issue-date-value = if issuing-date != none {
    issuing-date
  } else {
    datetime.today().display("[year]-[month]-[day]")
  }
  let period-value = if delivery-date != none { delivery-date } else { TODO }

  // Basic document setup.
  set document(title: title, keywords: keywords, date: parse-date(issue-date-value))
  set page(margin: styling.margin)
  set par(justify: false)
  set text(
    font: if styling.font == none { "Noto Sans" } else { styling.font },
    size: styling.font-size,
  )
  set table(stroke: none)

  let rule-color = rgb("8a8a8a")
  let total-fill = luma(242)

  // Convert each line item into the values we actually render.
  let rows = items.enumerate().map(((index, row)) => {
    let hours = row.at("dur-min", default: 0) / 60
    let unit-price = row.at(
      "price",
      default: calc.round(row.at("hourly-rate", default: 0) * hours, digits: 2),
    )
    let line-total = if row.at("dur-min", default: 0) == 0 {
      row.price * row.at("quantity", default: 1)
    } else {
      calc.round(row.at("hourly-rate", default: 0) * hours, digits: 2)
    }

    (
      number: row.at("number", default: index + 1),
      description: row.description,
      hours: if row.at("dur-min", default: 0) == 0 { "" } else { add-zeros(hours) },
      rate: add-zeros(unit-price),
      total: add-zeros(line-total),
      raw-total: line-total,
      raw-minutes: row.at("dur-min", default: 0),
    )
  })

  let subtotal = rows.map(row => row.raw-total).sum()
  let total-hours = rows.map(row => row.raw-minutes).sum() / 60
  let tax = subtotal * vat
  let total = subtotal + tax

  let summary-rows = (
    ([#labels.total-time:], [#add-zeros(calc.round(total-hours, digits: 2))#sym.space h]),
    if vat == 0 { ([#labels.no-vat], []) },
  ).filter(entry => entry != none)

  // Title and centered invoice metadata.
  align(center)[
    #block(inset: 1.2em)[
      #text(weight: "bold", size: 1.9em)[#(if title != none { title } else { labels.invoice })]
    ]

    #table(
      columns: 2,
      align: (right, left),
      column-gutter: 0.7em,
      inset: 2pt,
      [#labels.invoice-id:], [*#invoice-id-value*],
      [#labels.issuing-date:], [*#issue-date-value*],
      [#labels.delivery-date:], [*#period-value*],
    )
  ]

  v(2.4em)

  // Top three-column block: recipient, biller, and bank details.
  table(
    columns: (1fr, 1fr, 1.1fr),
    column-gutter: 2.2em,
    align: (left, left, left),
    [#text(weight: "bold", size: 1.15em)[#labels.recipient]],
    [#text(weight: "bold", size: 1.15em)[#labels.biller]],
    [#text(weight: "bold", size: 1.15em)[Bank Details]],
    [#v(0.35em) #join-address-lines(recipient)],
    [#v(0.35em) #join-address-lines(biller)],
    [
      #v(0.35em)
      #table(
        columns: (auto, 1fr),
        column-gutter: 0.9em,
        inset: (x: 0pt, y: 2pt),
        align: (left, left),
        [#labels.bank:], [#biller.at("bank", default: "")],
        [#labels.account-name:], [#biller.at("account-name", default: biller.name)],
        [#labels.sort-code:], [#biller.at("sort-code", default: "")],
        [#labels.account-number:], [#biller.at("account-number", default: "")],
      )
    ],
  )

  v(1.9em)

  // Main services table.
  text(weight: "bold", size: 1.35em)[#labels.items]
  v(0.9em)

  table(
    columns: (auto, 1fr, auto, auto, auto),
    align: (col, row) => if row == 0 {
      (right, left, center, right, right).at(col)
    } else {
      (right, left, right, right, right).at(col)
    },
    inset: (x: 6pt, y: 5pt),
    table.header(
      table.hline(stroke: 0.7pt + rule-color),
      [*#labels.number*],
      [*#labels.description*],
      [*#labels.duration*],
      [*#labels.price*],
      [*#labels.total* #text(size: 0.8em)[(#currency)]],
      table.hline(stroke: 0.5pt + rule-color),
    ),
    ..rows
      .map(row => (
        str(row.number),
        row.description,
        row.hours,
        row.rate,
        row.total,
      ))
      .flatten(),
    table.hline(stroke: 0.7pt + rule-color),
  )

  align(right)[
    // Small totals block aligned to the right under the table.
    #table(
      columns: 2,
      align: (right, right),
      column-gutter: 1em,
      inset: (x: 0pt, y: 3pt),
      ..summary-rows.flatten(),
    )
    #table(
      columns: 2,
      align: (right, right),
      fill: total-fill,
      inset: (x: 9pt, y: 5pt),
      stroke: (x: 0pt, y: 0.6pt + rule-color),
      [#labels.total:],
      [#add-zeros(total)#sym.space#currency],
    )
  ]

  v(1.4em)

  // Payment note and footer text.
  let due-date-value = if due-date != none {
    due-date
  } else {
    (parse-date(issue-date-value) + duration(days: 14)).display("[year]-[month]-[day]")
  }

  (labels.due-text)(due-date-value)
  v(1em)

  if "payment-terms" in biller [
    #biller.at("payment-terms")
    #v(1em)
  ]

  labels.closing
  doc
}
