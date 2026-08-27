// Package database opens and configures the PostgreSQL connection.
//
// gorm is the ORM. Money is BIGINT whole rupiah and every arithmetic path uses
// integers -- floating point is prohibited anywhere near a price.
package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Options struct {
	URL          string
	MaxOpenConns int
	MaxIdleConns int
	SlowQuery    time.Duration
	Debug        bool
}

// Open connects to PostgreSQL and verifies the connection.
func Open(ctx context.Context, opts Options, log *slog.Logger) (*gorm.DB, error) {
	if opts.URL == "" {
		return nil, errors.New("database: URL is empty")
	}
	if opts.SlowQuery == 0 {
		opts.SlowQuery = 500 * time.Millisecond
	}

	level := gormlogger.Warn
	if opts.Debug {
		level = gormlogger.Info
	}

	db, err := gorm.Open(postgres.Open(opts.URL), &gorm.Config{
		Logger: gormlogger.New(slogWriter{log}, gormlogger.Config{
			SlowThreshold:             opts.SlowQuery,
			LogLevel:                  level,
			IgnoreRecordNotFoundError: true,
			// Never log parameter values.
			ParameterizedQueries: true,
			Colorful:             false,
		}),
		SkipDefaultTransaction: true,
		TranslateError:         true,
	})
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(orDefault(opts.MaxOpenConns, 25))
	sqlDB.SetMaxIdleConns(orDefault(opts.MaxIdleConns, 5))
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	// Every session works in UTC. Business-date logic (the same-day cut-offs in
	// BR-7.4 above all) converts to Asia/Jakarta explicitly, never through the
	// server's local time.
	if err := db.Exec("SET TIME ZONE 'UTC'").Error; err != nil {
		return nil, fmt.Errorf("database: set timezone: %w", err)
	}
	return db, nil
}

// Close releases the pool.
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// InTx runs fn inside a transaction. Content publishing uses it so a menu
// cycle and all of its days land together or not at all.
func InTx(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}

type slogWriter struct{ log *slog.Logger }

// Printf receives gorm's SQL trace. It logs at debug so a development run does
// not drown in statements; slow queries are surfaced by gorm's own threshold.
func (w slogWriter) Printf(format string, args ...any) {
	if w.log == nil {
		return
	}
	w.log.Debug(fmt.Sprintf(format, args...))
}

func orDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}
