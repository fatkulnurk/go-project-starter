// Package dbdriver defines the supported database driver names. It lives in
// its own package so config, platform/database, and module repositories can
// reference the same constants without import cycles.
package dbdriver

// Supported database drivers.
const (
	MySQL    = "mysql"
	Postgres = "postgres"
)
