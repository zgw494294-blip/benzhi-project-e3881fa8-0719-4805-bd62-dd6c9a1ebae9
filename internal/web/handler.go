package web

import (
	"embed"
	"io/fs"
	"net/http"
	"time"

	"caption-release-gate/internal/workflow"
)

//go:embed static/*
var assets embed.FS

type Handler struct {
	workflow *workflow.Service
	static   http.Handler
}

func NewHandler(service *workflow.Service) *Handler {
	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	return &Handler{workflow: service, static: http.FileServer(http.FS(staticFS))}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.HandleRoot)
	mux.HandleFunc("GET /workbench", h.HandleWorkbench)
	mux.HandleFunc("GET /assets/{name}", h.HandleAsset)
	mux.HandleFunc("GET /healthz", h.HandleHealth)
	mux.HandleFunc("GET /api/v1/packages", h.HandleListPackages)
	mux.HandleFunc("POST /api/v1/packages", h.HandleCreatePackage)
	mux.HandleFunc("GET /api/v1/packages/{packageId}", h.HandleGetPackage)
	mux.HandleFunc("POST /api/v1/packages/{packageId}/revisions", h.HandleImportRevision)
	mux.HandleFunc("POST /api/v1/packages/{packageId}/checks", h.HandleRunChecks)
	mux.HandleFunc("POST /api/v1/packages/{packageId}/exceptions", h.HandleRequestException)
	mux.HandleFunc("POST /api/v1/packages/{packageId}/replacements", h.HandleReplacement)
	mux.HandleFunc("POST /api/v1/packages/{packageId}/reviews", h.HandleReview)
	mux.HandleFunc("POST /api/v1/packages/{packageId}/freeze", h.HandleFreeze)
	mux.HandleFunc("POST /api/v1/packages/{packageId}/manifest", h.HandleIssueManifest)
	mux.HandleFunc("GET /api/v1/packages/{packageId}/history", h.HandleHistory)
	mux.HandleFunc("POST /api/v1/manifests/verify", h.HandleVerifyManifest)
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; connect-src 'self'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/workbench", http.StatusTemporaryRedirect)
}

func (h *Handler) HandleWorkbench(w http.ResponseWriter, r *http.Request) {
	data, err := assets.ReadFile("static/workbench.html")
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

func (h *Handler) HandleAsset(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = "/" + r.PathValue("name")
	h.static.ServeHTTP(w, r)
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "caption-release-gate", "time": time.Now().UTC()})
}
