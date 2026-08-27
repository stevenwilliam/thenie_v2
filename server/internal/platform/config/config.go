// Package config loads configuration from the environment.
//
// Secrets only ever arrive via env. Nothing secret is in git: .env is
// git-ignored and .env.example is the documented surface.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the whole configuration surface of the service.
type Config struct {
	AppEnv      string
	AppPort     int
	AppTimezone string
	LogLevel    string
	LogJSON     bool

	DatabaseURL     string
	TestDatabaseURL string
	DBMaxOpenConns  int
	DBMaxIdleConns  int
	DBSlowQuery     time.Duration

	// AdminToken guards every write endpoint. It is a single shared bearer
	// token rather than a user system: this service has exactly one class of
	// caller (whoever edits the menu) and inventing accounts for that would be
	// scaffolding nobody asked for. If more than one person ever needs
	// distinct access, that is the moment to add real identity, not now.
	AdminToken string

	// CORSOrigins are the origins allowed to read the public config document.
	// The published site is served from a different origin to this API, so the
	// browser will preflight; an empty list means same-origin only.
	CORSOrigins []string

	// SitePath is the built page the `publish` command reads and rewrites.
	SitePath    string
	PublishPath string
}

// Load reads the environment, optionally seeded from an env file, and validates
// it. It returns every problem at once rather than one at a time: a deployment
// with three missing variables should say so in one run.
func Load(envFile string) (*Config, error) {
	if envFile != "" {
		if err := loadEnvFile(envFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	c := &Config{
		AppEnv:          get("APP_ENV", "development"),
		AppPort:         getInt("APP_PORT", 8082),
		AppTimezone:     get("APP_TIMEZONE", "Asia/Jakarta"),
		LogLevel:        get("LOG_LEVEL", "info"),
		LogJSON:         getBool("LOG_JSON", false),
		DatabaseURL:     get("DATABASE_URL", ""),
		TestDatabaseURL: get("TEST_DATABASE_URL", ""),
		DBMaxOpenConns:  getInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:  getInt("DB_MAX_IDLE_CONNS", 5),
		DBSlowQuery:     time.Duration(getInt("DB_SLOW_QUERY_MS", 500)) * time.Millisecond,
		AdminToken:      get("ADMIN_TOKEN", ""),
		SitePath:        get("SITE_PATH", "../dist/index.html"),
		PublishPath:     get("PUBLISH_PATH", ""),
	}
	if raw := get("CORS_ORIGINS", ""); raw != "" {
		for _, o := range strings.Split(raw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				c.CORSOrigins = append(c.CORSOrigins, o)
			}
		}
	}
	if c.PublishPath == "" {
		c.PublishPath = c.SitePath
	}

	var problems []string
	if c.DatabaseURL == "" {
		problems = append(problems, "DATABASE_URL is required")
	}
	if c.AppPort < 1 || c.AppPort > 65535 {
		problems = append(problems, fmt.Sprintf("APP_PORT %d is out of range", c.AppPort))
	}
	if _, err := time.LoadLocation(c.AppTimezone); err != nil {
		problems = append(problems, fmt.Sprintf("APP_TIMEZONE %q is not a known timezone", c.AppTimezone))
	}
	// A production deployment with no admin token would expose every write
	// endpoint to the internet. Refuse to start rather than serve that.
	if c.AppEnv == "production" && c.AdminToken == "" {
		problems = append(problems, "ADMIN_TOKEN is required when APP_ENV=production")
	}
	if c.AdminToken != "" && len(c.AdminToken) < 24 {
		problems = append(problems, "ADMIN_TOKEN must be at least 24 characters")
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("config: %s", strings.Join(problems, "; "))
	}
	return c, nil
}

// Location returns the operating timezone. Every business-date comparison goes
// through this, never through the server's local time.
func (c *Config) Location() *time.Location {
	loc, err := time.LoadLocation(c.AppTimezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// IsProduction reports whether this is a production deployment.
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

// loadEnvFile reads KEY=VALUE lines. Values already present in the real
// environment win, so a container's env always overrides a stray file.
func loadEnvFile(path string) error {
	f, err := os.Open(path) // #nosec G304 -- operator-supplied path, by design
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	return sc.Err()
}

func get(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
