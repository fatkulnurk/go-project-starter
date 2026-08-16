# `internal/platform/cache/`

**Technical infrastructure** — `cache.Cache` driver implementations and the
factory that picks one from `CACHE_DRIVER`.

## Drivers

| Driver    | Type      | Constructor      | Notes                                          |
|-----------|-----------|------------------|------------------------------------------------|
| `redis`   | `Redis`   | `NewRedis(cfg)`  | go-redis backed; pool owned until `Close`; atomic `GETDEL`/`INCRBY` |
| `memory`  | `Memory`  | `NewMemory()`    | in-process; dev/tests only, do not span replicas |
| `db`      | `Database`| `NewDatabase(db, driver)` | Laravel-style `cache` table (TEXT values); shares the SQL pool, `Close` is a no-op |

## Factory

`New(cfg config.CacheConfig, db *sql.DB, dbDriver string) (cache.Cache, error)`
— returns the driver for `cfg.Driver`; `db`/`dbDriver` are only used when the
driver is `db` (nil otherwise). Unknown drivers return an error.

## Usage

```go
c, err := cache.New(cfg.Cache, db, cfg.Database.Driver)
if err != nil { return err }

if err := c.Set(ctx, "otp:"+userID, []byte(code), cfg.Auth.OTPTTL); err != nil { ... }
v, err := c.GetDelete(ctx, "otp:"+userID) // single-use
defer c.Close()
```

`Cache` supports a shared Redis instance with the queue: cache keys live in
`REDIS_DB` (default `0`), queue keys in `QUEUE_REDIS_DB` (default `1`).

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
