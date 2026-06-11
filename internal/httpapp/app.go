package httpapp

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/curserio/unishare/internal/config"
	"github.com/curserio/unishare/internal/model"
)

const maxMemory = 32 << 20

type Store interface {
	List(userID string) ([]model.Item, error)
	Add(userID, title, text, rawURL string, uploads []*multipart.FileHeader) (model.Item, error)
	Delete(userID, id string) error
	File(userID, itemID, fileID string) (model.StoredFile, string, bool)
}

type App struct {
	cfg   config.Config
	store Store
}

type PublicItem struct {
	model.Item
	ShareText string `json:"shareText"`
}

func New(cfg config.Config, store Store) *App {
	return &App{cfg: cfg, store: store}
}

func (a *App) Handler() http.Handler {
	app := a.appHandler()
	if a.cfg.BasePath == "" {
		return securityHeaders(app)
	}

	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", a.healthz)
	root.Handle(a.cfg.BasePath+"/", http.StripPrefix(a.cfg.BasePath, app))
	root.HandleFunc(a.cfg.BasePath, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, a.cfg.BasePath+"/", http.StatusMovedPermanently)
	})
	return securityHeaders(root)
}

func (a *App) appHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.healthz)
	mux.HandleFunc("GET /config.js", a.configJS)
	mux.HandleFunc("GET /manifest.webmanifest", a.manifest)
	mux.HandleFunc("POST /api/login", a.login)
	mux.HandleFunc("POST /api/logout", a.logout)
	mux.HandleFunc("GET /api/session", a.session)
	mux.HandleFunc("GET /api/items", a.requireAuth(a.listItems))
	mux.HandleFunc("POST /api/items", a.requireAuth(a.createItem))
	mux.HandleFunc("DELETE /api/items/{id}", a.requireAuth(a.deleteItem))
	mux.HandleFunc("GET /files/{itemID}/{fileID}", a.requireAuth(a.file))
	mux.HandleFunc("POST /share-target", a.requireAuth(a.shareTarget))
	mux.Handle("/", http.FileServer(http.Dir(a.cfg.StaticDir)))
	return mux
}

func (a *App) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func (a *App) configJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write([]byte("window.__UNISHARE_CONFIG__ = "))
	_ = json.NewEncoder(w).Encode(map[string]any{
		"basePath": a.cfg.BasePath,
	})
	_, _ = w.Write([]byte(";"))
}

func (a *App) manifest(w http.ResponseWriter, r *http.Request) {
	base := a.cfg.BasePath
	if base == "" {
		base = ""
	}
	w.Header().Set("Content-Type", "application/manifest+json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"name":             "Unishare",
		"short_name":       "Unishare",
		"description":      "Personal dropbox for links, text, and files",
		"start_url":        base + "/",
		"scope":            base + "/",
		"display":          "standalone",
		"background_color": "#f5f7f6",
		"theme_color":      "#0f766e",
		"icons": []map[string]string{
			{"src": base + "/icons/icon-192.png", "sizes": "192x192", "type": "image/png", "purpose": "any"},
			{"src": base + "/icons/icon-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any"},
			{"src": base + "/icons/icon-maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable"},
		},
		"share_target": map[string]any{
			"action":  base + "/share-target",
			"method":  "POST",
			"enctype": "multipart/form-data",
			"params": map[string]any{
				"title": "title",
				"text":  "text",
				"url":   "url",
				"files": []map[string]any{
					{"name": "files", "accept": []string{"image/*", "video/*", "audio/*", "application/pdf", "text/*", "*/*"}},
				},
			},
		},
	}); err != nil {
		slog.Error("failed to write manifest", "err", err)
	}
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	userID, ok := a.userIDForToken(body.Token)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, a.authCookie(r, a.sessionValue(userID)))
	writeJSON(w, map[string]bool{"ok": true})
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "unishare_session",
		Value:    "",
		Path:     a.cookiePath(),
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		Secure:   a.secureCookie(r),
	})
	writeJSON(w, map[string]bool{"ok": true})
}

func (a *App) session(w http.ResponseWriter, r *http.Request) {
	_, ok := a.authenticated(r)
	writeJSON(w, map[string]bool{"authenticated": ok})
}

func (a *App) listItems(w http.ResponseWriter, r *http.Request, userID string) {
	items, err := a.store.List(userID)
	if err != nil {
		http.Error(w, "failed to list items", http.StatusInternalServerError)
		return
	}
	public := make([]PublicItem, 0, len(items))
	for _, item := range items {
		public = append(public, PublicItem{Item: item, ShareText: a.shareText(r, item)})
	}
	writeJSON(w, public)
}

func (a *App) createItem(w http.ResponseWriter, r *http.Request, userID string) {
	r.Body = http.MaxBytesReader(w, r.Body, a.maxRequestBytes())
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	item, err := a.store.Add(
		userID,
		r.FormValue("title"),
		r.FormValue("text"),
		r.FormValue("url"),
		r.MultipartForm.File["files"],
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, PublicItem{Item: item, ShareText: a.shareText(r, item)})
}

func (a *App) deleteItem(w http.ResponseWriter, r *http.Request, userID string) {
	if err := a.store.Delete(userID, r.PathValue("id")); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) file(w http.ResponseWriter, r *http.Request, userID string) {
	file, path, ok := a.store.File(userID, r.PathValue("itemID"), r.PathValue("fileID"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	if file.ContentType != "" {
		w.Header().Set("Content-Type", file.ContentType)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.Name))
	http.ServeFile(w, r, path)
}

func (a *App) shareTarget(w http.ResponseWriter, r *http.Request, userID string) {
	r.Body = http.MaxBytesReader(w, r.Body, a.maxRequestBytes())
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	var files []*multipart.FileHeader
	if r.MultipartForm != nil {
		files = append(files, r.MultipartForm.File["files"]...)
		files = append(files, r.MultipartForm.File["file"]...)
	}
	_, err := a.store.Add(
		userID,
		r.FormValue("title"),
		firstNonEmpty(r.FormValue("text"), r.FormValue("description")),
		r.FormValue("url"),
		files,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "./", http.StatusSeeOther)
}

func (a *App) requireAuth(next func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := a.authenticated(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r, userID)
	}
}

func (a *App) authenticated(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("unishare_session")
	if err != nil {
		return "", false
	}
	userID, signature, ok := strings.Cut(cookie.Value, ".")
	if !ok || userID == "" || signature == "" {
		return "", false
	}
	if sameSecret(cookie.Value, a.sessionValue(userID)) {
		return userID, true
	}
	return "", false
}

func (a *App) sessionValue(userID string) string {
	token, ok := a.tokenForUserID(userID)
	if !ok {
		return ""
	}
	sum := sha256.Sum256([]byte("unishare:" + userID + ":" + token))
	return userID + "." + hex.EncodeToString(sum[:])
}

func (a *App) userIDForToken(token string) (string, bool) {
	tokenSum := sha256.Sum256([]byte(token))
	matchID := ""
	for _, user := range a.cfg.Users {
		userSum := sha256.Sum256([]byte(user.Token))
		if subtle.ConstantTimeCompare(tokenSum[:], userSum[:]) == 1 {
			matchID = user.ID
		}
	}
	return matchID, matchID != ""
}

func (a *App) tokenForUserID(userID string) (string, bool) {
	for _, user := range a.cfg.Users {
		if user.ID == userID {
			return user.Token, true
		}
	}
	return "", false
}

func (a *App) shareText(r *http.Request, item model.Item) string {
	parts := []string{}
	for _, value := range []string{item.Title, item.Text, item.URL} {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, strings.TrimSpace(value))
		}
	}
	for _, file := range item.Files {
		parts = append(parts, a.absoluteURL(r, "/files/"+item.ID+"/"+file.ID))
	}
	return strings.Join(parts, "\n")
}

func (a *App) absoluteURL(r *http.Request, path string) string {
	if a.cfg.PublicBaseURL != "" {
		return a.cfg.PublicBaseURL + a.cfg.BasePath + path
	}
	scheme := "http"
	if isHTTPS(r) {
		scheme = "https"
	}
	return scheme + "://" + r.Host + a.cfg.BasePath + path
}

func (a *App) authCookie(r *http.Request, value string) *http.Cookie {
	return &http.Cookie{
		Name:     "unishare_session",
		Value:    value,
		Path:     a.cookiePath(),
		MaxAge:   int((180 * 24 * time.Hour).Seconds()),
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		Secure:   a.secureCookie(r),
	}
}

func (a *App) cookiePath() string {
	if a.cfg.BasePath != "" {
		return a.cfg.BasePath
	}
	return "/"
}

func (a *App) secureCookie(r *http.Request) bool {
	switch a.cfg.CookieSecure {
	case "true":
		return true
	case "false":
		return false
	default:
		return isHTTPS(r)
	}
}

func (a *App) maxRequestBytes() int64 {
	// Multipart framing and text fields need a small amount of headroom over the
	// configured per-file limit.
	return a.cfg.MaxUploadBytes + maxMemory
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func sameSecret(left, right string) bool {
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("failed to write json", "err", err)
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; media-src 'self' blob:; style-src 'self' 'unsafe-inline'; script-src 'self';")
		next.ServeHTTP(w, r)
	})
}
