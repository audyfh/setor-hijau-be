package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pinger defines an interface for components capable of checking database connectivity.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Options configures the database connection pool.
type Options struct {
	MaxConns    int32
	MinConns    int32
	PingTimeout time.Duration
}

// DB wraps pgxpool.Pool and implements Pinger and common database utilities.
type DB struct {
	Pool *pgxpool.Pool
}

// Connect establishes a connection pool to the database using default options.
func Connect(ctx context.Context, databaseURL string) (*DB, error) {
	return ConnectWithOptions(ctx, databaseURL, Options{})
}

// ConnectWithOptions establishes a connection pool to the database with explicit pool options.
// It configures pgx to be compatible with serverless Postgres (e.g. NeonDB) and transaction poolers.
func ConnectWithOptions(ctx context.Context, databaseURL string, opts Options) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	maxConns := opts.MaxConns
	if maxConns <= 0 {
		maxConns = 10
	}

	pingTimeout := opts.PingTimeout
	if pingTimeout <= 0 {
		pingTimeout = 10 * time.Second
	}

	config.MaxConns = maxConns
	config.MinConns = opts.MinConns // 0 by default so serverless DBs (NeonDB) can auto-suspend
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	// Avoid server-side named prepared statement caching unless explicitly requested in DATABASE_URL,
	// ensuring seamless portability behind transaction poolers (NeonDB PgBouncer, Supabase Supavisor).
	if !strings.Contains(strings.ToLower(databaseURL), "default_query_exec_mode=") {
		config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Ping checks if the database is reachable.
func (d *DB) Ping(ctx context.Context) error {
	if d.Pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}
	return d.Pool.Ping(ctx)
}

// Close closes all connections in the pool.
func (d *DB) Close() {
	if d.Pool != nil {
		d.Pool.Close()
	}
}
