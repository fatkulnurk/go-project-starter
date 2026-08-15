package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	appcache "github.com/fatkulnurk/go-project-starter/internal/application/cache"
)

type fakeCache struct {
	data map[string][]byte
}

func newFakeCache() *fakeCache { return &fakeCache{data: map[string][]byte{}} }

func (f *fakeCache) Get(ctx context.Context, key string) ([]byte, error) {
	if v, ok := f.data[key]; ok {
		return v, nil
	}
	return nil, appcache.ErrNotFound
}

func (f *fakeCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.data[key] = value
	return nil
}

func (f *fakeCache) Delete(ctx context.Context, key string) error {
	delete(f.data, key)
	return nil
}

func (f *fakeCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	var v int64
	if b, ok := f.data[key]; ok {
		_ = json.Unmarshal(b, &v)
	}
	v += delta
	b, _ := json.Marshal(v)
	f.data[key] = b
	return v, nil
}

func (f *fakeCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return nil
}

func (f *fakeCache) Ping(ctx context.Context) error { return nil }

func (f *fakeCache) Close() error { return nil }

func TestBumpIncrementsVersion(t *testing.T) {
	pc := NewPermissionCache(newFakeCache(), time.Minute)
	ctx := context.Background()

	if err := pc.Bump(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, err := pc.CurrentVersion(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 1 {
		t.Fatalf("version = %d, want 1", v)
	}
}

func TestCurrentVersion_DefaultZero(t *testing.T) {
	pc := NewPermissionCache(newFakeCache(), time.Minute)
	v, err := pc.CurrentVersion(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 0 {
		t.Fatalf("version = %d, want 0", v)
	}
}

func TestSetGetUser_Roundtrip(t *testing.T) {
	pc := NewPermissionCache(newFakeCache(), time.Minute)
	ctx := context.Background()

	if err := pc.SetUser(ctx, "u1", 3, []string{"editor"}, []string{"posts.edit"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	roles, perms, ok, err := pc.GetUser(ctx, "u1", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(roles) != 1 || roles[0] != "editor" {
		t.Fatalf("roles = %v", roles)
	}
	if len(perms) != 1 || perms[0] != "posts.edit" {
		t.Fatalf("permissions = %v", perms)
	}
}

func TestGetUser_StaleVersion(t *testing.T) {
	pc := NewPermissionCache(newFakeCache(), time.Minute)
	ctx := context.Background()

	if err := pc.SetUser(ctx, "u1", 3, []string{"editor"}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, _, ok, err := pc.GetUser(ctx, "u1", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected stale entry to be rejected")
	}
}

func TestGetUser_Missing(t *testing.T) {
	pc := NewPermissionCache(newFakeCache(), time.Minute)
	_, _, ok, err := pc.GetUser(context.Background(), "ghost", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected cache miss")
	}
}
