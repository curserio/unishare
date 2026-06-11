package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var userIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type Config struct {
	Addr           string
	DataDir        string
	StaticDir      string
	PublicBaseURL  string
	BasePath       string
	Users          []User
	CookieSecure   string
	MaxUploadMB    int64
	MaxUploadBytes int64
}

type User struct {
	ID    string
	Token string
}

func Load() (Config, error) {
	maxUploadMB := int64Env("UNISHARE_MAX_UPLOAD_MB", 50)
	users, err := ParseUsers(os.Getenv("UNISHARE_USERS"), os.Getenv("UNISHARE_TOKEN"))
	if err != nil {
		return Config{}, err
	}
	basePath, err := NormalizeBasePath(os.Getenv("UNISHARE_BASE_PATH"))
	if err != nil {
		return Config{}, err
	}
	return Config{
		Addr:           env("UNISHARE_ADDR", ":8080"),
		DataDir:        env("UNISHARE_DATA_DIR", "/data"),
		StaticDir:      env("UNISHARE_STATIC_DIR", "static"),
		PublicBaseURL:  strings.TrimRight(os.Getenv("UNISHARE_PUBLIC_BASE_URL"), "/"),
		BasePath:       basePath,
		Users:          users,
		CookieSecure:   cookieSecureEnv("UNISHARE_COOKIE_SECURE", "auto"),
		MaxUploadMB:    maxUploadMB,
		MaxUploadBytes: maxUploadMB << 20,
	}, nil
}

func ParseUsers(rawUsers, fallbackToken string) ([]User, error) {
	rawUsers = strings.TrimSpace(rawUsers)
	if rawUsers == "" {
		fallbackToken = strings.TrimSpace(fallbackToken)
		if fallbackToken == "" {
			return nil, nil
		}
		return []User{{ID: "default", Token: fallbackToken}}, nil
	}

	seenIDs := map[string]bool{}
	var users []User
	for _, part := range strings.Split(rawUsers, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, token, ok := strings.Cut(part, ":")
		id = strings.TrimSpace(id)
		token = strings.TrimSpace(token)
		if !ok || id == "" || token == "" {
			return nil, fmt.Errorf("invalid user entry %q", part)
		}
		if !userIDPattern.MatchString(id) {
			return nil, fmt.Errorf("invalid user id %q", id)
		}
		if seenIDs[id] {
			return nil, fmt.Errorf("duplicate user id %q", id)
		}
		seenIDs[id] = true
		users = append(users, User{ID: id, Token: token})
	}
	if len(users) == 0 {
		return nil, errors.New("UNISHARE_USERS does not contain users")
	}
	return users, nil
}

func NormalizeBasePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "/" {
		return "", nil
	}
	if !strings.HasPrefix(value, "/") {
		return "", errors.New("base path must start with /")
	}
	value = strings.TrimRight(value, "/")
	if strings.Contains(value, "//") {
		return "", errors.New("base path must not contain //")
	}
	return value, nil
}

func cookieSecureEnv(key, fallback string) string {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch value {
	case "auto", "true", "false":
		return value
	case "":
		return fallback
	default:
		return fallback
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func int64Env(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	var value int64
	if _, err := fmt.Sscanf(raw, "%d", &value); err == nil && value > 0 {
		return value
	}
	return fallback
}
