package api

import (
	"html/template"
	"net/http"
	"path/filepath"
)

type WebHandler struct {
	templates map[string]*template.Template
}

func NewWebHandler() *WebHandler {
	h := &WebHandler{
		templates: make(map[string]*template.Template),
	}
	
	layout := filepath.Join("web", "templates", "layout.html")
	pages := []string{"index", "login", "register", "dashboard", "keys", "account", "script-editor", "privacy", "gdpr", "setup", "docs", "admin", "terms", "community", "api-reference", "secrets"}
	
	for _, page := range pages {
		pagePath := filepath.Join("web", "templates", page+".html")
		h.templates[page] = template.Must(template.ParseFiles(layout, pagePath))
	}
	
	return h
}

func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	h.render(w, "index", map[string]interface{}{
		"Title": "Home",
	})
}

func (h *WebHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.render(w, "login", map[string]interface{}{
		"Title": "Login",
	})
}

func (h *WebHandler) Register(w http.ResponseWriter, r *http.Request) {
	h.render(w, "register", map[string]interface{}{
		"Title": "Register",
	})
}

func (h *WebHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	h.render(w, "dashboard", map[string]interface{}{
		"Title": "Dashboard",
	})
}

func (h *WebHandler) Keys(w http.ResponseWriter, r *http.Request) {
	h.render(w, "keys", map[string]interface{}{
		"Title": "Key Management",
	})
}

func (h *WebHandler) Account(w http.ResponseWriter, r *http.Request) {
	h.render(w, "account", map[string]interface{}{
		"Title": "Account Settings",
	})
}

func (h *WebHandler) ScriptEditor(w http.ResponseWriter, r *http.Request) {
	h.render(w, "script-editor", map[string]interface{}{
		"Title": "Script Editor",
	})
}

func (h *WebHandler) Privacy(w http.ResponseWriter, r *http.Request) {
	h.render(w, "privacy", map[string]interface{}{
		"Title": "Privacy Policy",
	})
}

func (h *WebHandler) GDPR(w http.ResponseWriter, r *http.Request) {
	h.render(w, "gdpr", map[string]interface{}{
		"Title": "GDPR Information",
	})
}

func (h *WebHandler) Setup(w http.ResponseWriter, r *http.Request) {
	h.render(w, "setup", map[string]interface{}{
		"Title": "Setup",
	})
}

func (h *WebHandler) Docs(w http.ResponseWriter, r *http.Request) {
	h.render(w, "docs", map[string]interface{}{
		"Title": "Documentation",
	})
}

func (h *WebHandler) Admin(w http.ResponseWriter, r *http.Request) {
	h.render(w, "admin", map[string]interface{}{
		"Title": "Admin Panel",
	})
}

func (h *WebHandler) Terms(w http.ResponseWriter, r *http.Request) {
	h.render(w, "terms", map[string]interface{}{
		"Title": "Terms of Service",
	})
}

func (h *WebHandler) Community(w http.ResponseWriter, r *http.Request) {
	h.render(w, "community", map[string]interface{}{
		"Title": "Community Scripts",
	})
}

func (h *WebHandler) APIReference(w http.ResponseWriter, r *http.Request) {
	h.render(w, "api-reference", map[string]interface{}{
		"Title": "API Reference",
	})
}

func (h *WebHandler) Secrets(w http.ResponseWriter, r *http.Request) {
	h.render(w, "secrets", map[string]interface{}{
		"Title": "Secrets",
	})
}

func (h *WebHandler) render(w http.ResponseWriter, tmpl string, data interface{}) {
	if err := h.templates[tmpl].ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
