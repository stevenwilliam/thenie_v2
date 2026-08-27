// Command thenied is the Thenie site-configuration service.
//
//	thenied serve            run the HTTP API
//	thenied migrate up       apply pending migrations
//	thenied migrate down [n] roll back n migrations (default 1, 0 = all)
//	thenied migrate status   show applied and pending
//	thenied seed             load the captured page's content into an empty database
//	thenied validate         re-check the stored content against the domain rules
//
// The page it configures is a frozen, byte-exact capture that is never edited
// (docs/07-fidelity-and-verification.md). This service does not rewrite it --
// it serves the content a hydration overlay reads at runtime.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	adapterhttp "github.com/stevenwilliam/thenie_v2/server/internal/adapter/http"
	"github.com/stevenwilliam/thenie_v2/server/internal/adapter/http/adminui"
	"github.com/stevenwilliam/thenie_v2/server/internal/adapter/postgres"
	"github.com/stevenwilliam/thenie_v2/server/internal/app/admin"
	"github.com/stevenwilliam/thenie_v2/server/internal/app/siteconfig"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/config"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/database"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/logging"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/migrate"
)

// version is stamped at build time: -ldflags "-X main.version=$(git rev-parse --short HEAD)"
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "thenied: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("no command given")
	}

	cfg, err := config.Load(envFile())
	if err != nil {
		return err
	}
	log := logging.New(cfg.LogLevel, cfg.LogJSON)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch args[0] {
	case "serve":
		return serve(ctx, cfg, log)
	case "migrate":
		return runMigrate(ctx, cfg, log, args[1:])
	case "seed":
		return runSeed(ctx, cfg, log, args[1:])
	case "validate":
		return runValidate(ctx, cfg, log)
	case "user":
		return runUser(ctx, cfg, log, args[1:])
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `thenied — Thenie site-configuration service

  thenied serve              run the HTTP API
  thenied migrate up         apply pending migrations
  thenied migrate down [n]   roll back n migrations (default 1, 0 = all)
  thenied migrate status     show applied and pending
  thenied seed [--force]     load the captured page's content into an empty database
  thenied validate           re-check stored content against the domain rules
  thenied user ...           manage admin accounts (list, create, password, roles)

Configuration comes from the environment; see .env.example.
`)
}

// envFile lets an operator point at a specific file; otherwise the conventional
// one next to the binary's working directory is used if it exists.
func envFile() string {
	if v := os.Getenv("ENV_FILE"); v != "" {
		return v
	}
	for _, candidate := range []string{".env", "../.env"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func serve(ctx context.Context, cfg *config.Config, log *slogLogger) error {
	db, err := database.Open(ctx, database.Options{
		URL:          cfg.DatabaseURL,
		MaxOpenConns: cfg.DBMaxOpenConns,
		MaxIdleConns: cfg.DBMaxIdleConns,
		SlowQuery:    cfg.DBSlowQuery,
		Debug:        !cfg.IsProduction(),
	}, log)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(db) }()

	// Refuse to start on a database that has not been migrated. A service
	// answering 500 to every request because a table is missing is a much worse
	// failure than one that will not start and says why.
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	status, err := migrate.Current(ctx, sqlDB)
	if err != nil {
		return fmt.Errorf("check migrations: %w", err)
	}
	if len(status.Pending) > 0 {
		return fmt.Errorf("%d migration(s) pending (%v) — run: thenied migrate up",
			len(status.Pending), status.Pending)
	}

	cfgSvc := siteconfig.NewService(postgres.NewConfigRepo(db))
	adminSvc := admin.NewService(postgres.NewAdminRepo(db), cfg.AdminToken)

	// Expired sessions accumulate forever otherwise. One sweep at start-up plus
	// an hourly tick is enough for a table this small.
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			if n, err := adminSvc.PurgeExpiredSessions(ctx); err == nil && n > 0 {
				log.Info("purged expired sessions", "count", n)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	router := adapterhttp.New(adapterhttp.Deps{
		Config:  cfgSvc,
		Menu:    postgres.NewMenuRepo(db),
		Params:  postgres.NewParamRepo(db),
		Rates:   postgres.NewRateRepo(db),
		Rules:   postgres.NewRulesRepo(db),
		Admin:   adminSvc,
		AdminUI: adminui.Handler(),
		// Secure cookies require HTTPS. In development the admin UI is served
		// over plain HTTP on a LAN address, and a Secure cookie would simply
		// never be sent — the login would appear to succeed and then not stick.
		SecureCookie: cfg.IsProduction(),
		Log:          log,
		AdminToken:   cfg.AdminToken,
		CORSOrigins:  cfg.CORSOrigins,
		Version:      version,
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.AppPort),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening",
			"addr", srv.Addr, "env", cfg.AppEnv, "version", version,
			"service_token", cfg.AdminToken != "", "admin_ui", "/admin/")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func runMigrate(ctx context.Context, cfg *config.Config, log *slogLogger, args []string) error {
	db, err := database.Open(ctx, database.Options{URL: cfg.DatabaseURL}, log)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(db) }()
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	action := "up"
	if len(args) > 0 {
		action = args[0]
	}
	switch action {
	case "up":
		n, err := migrate.Up(ctx, sqlDB, log)
		if err != nil {
			return err
		}
		fmt.Printf("applied %d migration(s)\n", n)
	case "down":
		steps := 1
		if len(args) > 1 {
			if steps, err = strconv.Atoi(args[1]); err != nil {
				return fmt.Errorf("down: %q is not a number", args[1])
			}
		}
		n, err := migrate.Down(ctx, sqlDB, steps, log)
		if err != nil {
			return err
		}
		fmt.Printf("rolled back %d migration(s)\n", n)
	case "status":
		st, err := migrate.Current(ctx, sqlDB)
		if err != nil {
			return err
		}
		fmt.Printf("applied: %v\npending: %v\n", st.Applied, st.Pending)
	default:
		return fmt.Errorf("migrate: unknown action %q", action)
	}
	return nil
}

func runValidate(ctx context.Context, cfg *config.Config, log *slogLogger) error {
	db, err := database.Open(ctx, database.Options{URL: cfg.DatabaseURL}, log)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(db) }()

	doc, err := siteconfig.NewService(postgres.NewConfigRepo(db)).Get(ctx)
	if err != nil {
		return err
	}
	problems := doc.Validate()
	if len(problems) == 0 {
		fmt.Printf("revision %d: OK — %d plans, %d tier products, %d menu cycles\n",
			doc.Revision, len(doc.Plans), len(doc.TierProducts), len(doc.Menu.Cycles))
		return nil
	}
	fmt.Printf("revision %d: %d problem(s)\n", doc.Revision, len(problems))
	for _, p := range problems {
		fmt.Printf("  - %v\n", p)
	}
	return errors.New("validation failed")
}
