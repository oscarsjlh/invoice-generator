package handler

import "net/http"

func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	year := q.Get("year")
	month := q.Get("month")

	years, err := a.store.ListAvailableYears()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	months, err := a.store.ListAvailableMonths(year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	summary, totalHours, totalAmount, err := a.store.FilteredSummary(year, month)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	invoices, err := a.store.ListInvoices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(invoices) > 8 {
		invoices = invoices[:8]
	}

	a.renderPage(w, http.StatusOK, "dashboard.html", DashboardPageData{
		Summary:        summary,
		RecentInvoices: invoices,
		Years:          years,
		Months:         months,
		SelectedYear:   year,
		SelectedMonth:  month,
		TotalHours:     totalHours,
		TotalAmount:    totalAmount,
		Notice:         noticeFromRequest(r),
	})
}
