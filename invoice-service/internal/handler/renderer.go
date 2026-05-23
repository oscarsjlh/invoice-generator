package handler

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"reflect"

	"invoice-app/internal/db"
	"invoice-app/templates"
)

type Renderer struct {
	baseTmpl *template.Template
	logger   *slog.Logger
	authOn   func() bool
	ocrOn    func() bool
}

func NewRenderer(logger *slog.Logger, authOn func() bool, ocrOn func() bool) *Renderer {
	r := &Renderer{
		logger: logger,
		authOn: authOn,
		ocrOn:  ocrOn,
	}
	r.baseTmpl = r.compileTemplates()
	return r
}

func (r *Renderer) compileTemplates() *template.Template {
	funcMap := template.FuncMap{
		"money":       money,
		"numfmt":      numfmt,
		"dateLabel":   dateLabel,
		"selected":    selected,
		"monthName":   monthName,
		"mul":         func(a float64, b float64) float64 { return a * b },
		"divf":        func(a float64, b float64) float64 { return a / b },
		"div":         func(a, b int64) int64 { return a / b },
		"authEnabled": r.authOn,
		"ocrEnabled":  r.ocrOn,
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templates.FS, "layout.html")
	if err != nil {
		r.logger.Error("compile base template", "error", err)
		panic(fmt.Sprintf("failed to compile base template: %v", err))
	}
	return tmpl
}

func (r *Renderer) Page(w http.ResponseWriter, req *http.Request, status int, page string, data any, extra ...string) {
	files := append([]string{"layout.html"}, extra...)
	files = append(files, page)

	tmpl, err := r.baseTmpl.Clone()
	if err != nil {
		http.Error(w, fmt.Sprintf("clone template: %v", err), http.StatusInternalServerError)
		return
	}
	if _, err := tmpl.ParseFS(templates.FS, files...); err != nil {
		http.Error(w, fmt.Sprintf("parse template: %v", err), http.StatusInternalServerError)
		return
	}

	data = injectUser(data, UserFromContext(req.Context()))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, fmt.Sprintf("execute template: %v", err), http.StatusInternalServerError)
	}
}

func (r *Renderer) Partial(w http.ResponseWriter, status int, name string, data any, files ...string) {
	tmpl, err := r.baseTmpl.Clone()
	if err != nil {
		http.Error(w, fmt.Sprintf("clone template: %v", err), http.StatusInternalServerError)
		return
	}
	if _, err := tmpl.ParseFS(templates.FS, files...); err != nil {
		http.Error(w, fmt.Sprintf("parse partial: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, fmt.Sprintf("execute partial: %v", err), http.StatusInternalServerError)
	}
}

func injectUser(data any, user *db.User) any {
	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return data
	}

	elem := reflect.Indirect(v)
	if elem.IsValid() && elem.Kind() == reflect.Struct && v != elem {
		f := elem.FieldByName("User")
		if f.IsValid() && f.CanSet() && f.Type() == reflect.TypeOf(user) {
			f.Set(reflect.ValueOf(user))
		}
		return data
	}

	if v.Kind() == reflect.Struct {
		ptr := reflect.New(v.Type())
		ptr.Elem().Set(v)
		f := ptr.Elem().FieldByName("User")
		if f.IsValid() && f.CanSet() && f.Type() == reflect.TypeOf(user) {
			f.Set(reflect.ValueOf(user))
		}
		return ptr.Interface()
	}
	return data
}
