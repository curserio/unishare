package httpapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/curserio/unishare/internal/config"
	"github.com/curserio/unishare/internal/store"
)

func TestLoginCreateAndListItems(t *testing.T) {
	itemStore, err := store.NewFileStore(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	app := New(config.Config{
		Users:          []config.User{{ID: "main", Token: "secret"}},
		PublicBaseURL:  "https://share.example.com",
		StaticDir:      "static",
		CookieSecure:   "auto",
		MaxUploadBytes: 1 << 20,
	}, itemStore)
	handler := app.Handler()

	loginBody := bytes.NewBufferString(`{"token":"secret"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", loginBody)
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("unexpected login status: %d", loginRec.Code)
	}
	cookies := (&http.Response{Header: loginRec.Result().Header}).Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not set a cookie")
	}

	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	if err := writer.WriteField("text", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("url", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	createReq := httptest.NewRequest(http.MethodPost, "/api/items", &form)
	createReq.Header.Set("Content-Type", writer.FormDataContentType())
	createReq.AddCookie(cookies[0])
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("unexpected create status: %d", createRec.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	listReq.URL = &url.URL{Scheme: "https", Host: "share.example.com", Path: "/api/items"}
	listReq.AddCookie(cookies[0])
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d", listRec.Code)
	}

	var items []PublicItem
	if err := json.NewDecoder(listRec.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ShareText != "hello\nhttps://example.com" {
		t.Fatalf("unexpected share text: %q", items[0].ShareText)
	}
}

func TestHealthz(t *testing.T) {
	itemStore, err := store.NewFileStore(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	app := New(config.Config{Users: []config.User{{ID: "main", Token: "secret"}}, StaticDir: "static", MaxUploadBytes: 1 << 20}, itemStore)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestCookieSecureOverride(t *testing.T) {
	itemStore, err := store.NewFileStore(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	app := New(config.Config{Users: []config.User{{ID: "main", Token: "secret"}}, StaticDir: "static", CookieSecure: "true", MaxUploadBytes: 1 << 20}, itemStore)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(`{"token":"secret"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	app.Handler().ServeHTTP(loginRec, loginReq)

	cookies := (&http.Response{Header: loginRec.Result().Header}).Cookies()
	if len(cookies) == 0 || !cookies[0].Secure {
		t.Fatal("expected secure auth cookie")
	}
}

func TestUsersAreIsolated(t *testing.T) {
	itemStore, err := store.NewFileStore(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	app := New(config.Config{
		Users:          []config.User{{ID: "main", Token: "main-token"}, {ID: "second", Token: "second-token"}},
		StaticDir:      "static",
		MaxUploadBytes: 1 << 20,
	}, itemStore)
	handler := app.Handler()

	mainCookie := loginCookie(t, handler, "main-token")
	secondCookie := loginCookie(t, handler, "second-token")

	createItem(t, handler, mainCookie, "main item")
	createItem(t, handler, secondCookie, "second item")

	mainItems := listItems(t, handler, mainCookie)
	secondItems := listItems(t, handler, secondCookie)
	if len(mainItems) != 1 || mainItems[0].Text != "main item" {
		t.Fatalf("unexpected main items: %+v", mainItems)
	}
	if len(secondItems) != 1 || secondItems[0].Text != "second item" {
		t.Fatalf("unexpected second items: %+v", secondItems)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/items/"+secondItems[0].ID, nil)
	deleteReq.AddCookie(mainCookie)
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNotFound {
		t.Fatalf("expected cross-user delete to fail, got %d", deleteRec.Code)
	}
}

func TestTamperedSessionUserIDIsRejected(t *testing.T) {
	itemStore, err := store.NewFileStore(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	app := New(config.Config{
		Users:          []config.User{{ID: "main", Token: "main-token"}, {ID: "second", Token: "second-token"}},
		StaticDir:      "static",
		MaxUploadBytes: 1 << 20,
	}, itemStore)
	handler := app.Handler()

	cookie := loginCookie(t, handler, "main-token")
	_, signature, ok := strings.Cut(cookie.Value, ".")
	if !ok {
		t.Fatalf("unexpected session cookie value: %q", cookie.Value)
	}
	tampered := *cookie
	tampered.Value = "second." + signature

	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	req.AddCookie(&tampered)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected tampered cookie to be rejected, got %d", rec.Code)
	}
}

func TestMalformedSessionCookieIsRejected(t *testing.T) {
	itemStore, err := store.NewFileStore(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	app := New(config.Config{
		Users:          []config.User{{ID: "main", Token: "main-token"}},
		StaticDir:      "static",
		MaxUploadBytes: 1 << 20,
	}, itemStore)
	handler := app.Handler()

	for _, value := range []string{"", "main", ".signature", "main.", "main.signature.extra"} {
		t.Run(value, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
			req.AddCookie(&http.Cookie{Name: "unishare_session", Value: value})
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected malformed cookie %q to be rejected, got %d", value, rec.Code)
			}
		})
	}
}

func TestBasePathRoutesAndManifest(t *testing.T) {
	itemStore, err := store.NewFileStore(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	app := New(config.Config{
		Users:          []config.User{{ID: "main", Token: "secret"}},
		BasePath:       "/unishare",
		StaticDir:      "static",
		MaxUploadBytes: 1 << 20,
	}, itemStore)
	handler := app.Handler()

	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRec := httptest.NewRecorder()
	handler.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("unexpected root health status: %d", healthRec.Code)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/unishare/api/login", bytes.NewBufferString(`{"token":"secret"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("unexpected prefixed login status: %d", loginRec.Code)
	}

	manifestReq := httptest.NewRequest(http.MethodGet, "/unishare/manifest.webmanifest", nil)
	manifestRec := httptest.NewRecorder()
	handler.ServeHTTP(manifestRec, manifestReq)
	if manifestRec.Code != http.StatusOK {
		t.Fatalf("unexpected manifest status: %d", manifestRec.Code)
	}
	var manifest map[string]any
	if err := json.NewDecoder(manifestRec.Body).Decode(&manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["start_url"] != "/unishare/" || manifest["scope"] != "/unishare/" {
		t.Fatalf("unexpected manifest paths: %+v", manifest)
	}
}

func loginCookie(t *testing.T, handler http.Handler, token string) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(fmt.Sprintf(`{"token":%q}`, token)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected login status: %d", rec.Code)
	}
	cookies := (&http.Response{Header: rec.Result().Header}).Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not set cookie")
	}
	return cookies[0]
}

func createItem(t *testing.T, handler http.Handler, cookie *http.Cookie, text string) {
	t.Helper()
	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	if err := writer.WriteField("text", text); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/items", &form)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected create status: %d", rec.Code)
	}
}

func listItems(t *testing.T, handler http.Handler, cookie *http.Cookie) []PublicItem {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d", rec.Code)
	}
	var items []PublicItem
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	return items
}
