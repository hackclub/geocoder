package api

import (
	"html/template"
	"net/http"
	"path/filepath"
)

func (h *Handlers) HandleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	tmplPath := filepath.Join("web", "templates", "dashboard.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Template execution failed", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) HandleDocs(w http.ResponseWriter, r *http.Request) {
	tmplPath := filepath.Join("web", "templates", "docs.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Documentation template not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Template execution failed", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) HandleTestMap(w http.ResponseWriter, r *http.Request) {
	tmplPath := filepath.Join("web", "templates", "test_map.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Test template not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Template execution failed", http.StatusInternalServerError)
		return
	}
}
