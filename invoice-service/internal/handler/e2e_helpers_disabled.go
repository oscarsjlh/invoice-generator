//go:build !e2e

package handler

import "net/http"

func (a *App) registerE2EHelperRoutes(mux *http.ServeMux) {}

func isE2EPublicPath(path string) bool {
	return false
}
