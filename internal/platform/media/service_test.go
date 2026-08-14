package media

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	appid "github.com/fatkulnurk/go-project-starter/internal/application/id"
	"github.com/fatkulnurk/go-project-starter/internal/application/media"
	appstorage "github.com/fatkulnurk/go-project-starter/internal/application/storage"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

func TestMain(m *testing.M) {
	appid.SetDefault(testIDGen{})
	os.Exit(m.Run())
}

type testIDGen struct{}

func (testIDGen) New() string { return "id-1" }

type fakeRepo struct {
	byID map[string]*media.Media
	list []*media.Media
	// overrides
	saveErr error
}

func (f *fakeRepo) Save(_ context.Context, m *media.Media) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if f.byID == nil {
		f.byID = map[string]*media.Media{}
	}
	f.byID[m.ID] = m
	return nil
}

func (f *fakeRepo) FindByID(_ context.Context, id string) (*media.Media, error) {
	if m, ok := f.byID[id]; ok {
		return m, nil
	}
	return nil, nil
}

func (f *fakeRepo) ListByModel(_ context.Context, modelType, modelID, collection string) ([]*media.Media, error) {
	var out []*media.Media
	for _, m := range f.list {
		if m.ModelType == modelType && m.ModelID == modelID {
			if collection == "" || m.CollectionName == collection {
				out = append(out, m)
			}
		}
	}
	return out, nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

type fakeStore struct {
	files map[string]string
}

func newFakeStore() *fakeStore { return &fakeStore{files: map[string]string{}} }

func (s *fakeStore) Put(_ context.Context, key string, r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.files[key] = string(b)
	return nil
}

func (s *fakeStore) Get(context.Context, string) (io.ReadCloser, error) { return nil, nil }
func (s *fakeStore) Delete(_ context.Context, key string) error         { delete(s.files, key); return nil }
func (s *fakeStore) Attributes(context.Context, string) (appstorage.ObjectAttrs, error) {
	return appstorage.ObjectAttrs{}, nil
}

type fakeURLGen struct{ url string }

func (f *fakeURLGen) URL(context.Context, string) (string, error) {
	if f.url == "" {
		return "", appstorage.ErrNoURL
	}
	return f.url, nil
}

func newService(repo *fakeRepo, store *fakeStore, urlGen appstorage.URLGenerator) *Service {
	return New(Deps{
		Repo:         repo,
		Storage:      store,
		URLGenerator: urlGen,
		Disk:         "local",
		Clock:        clock.Fixed{T: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	})
}

func TestAddMediaPersistsAndStores(t *testing.T) {
	repo := &fakeRepo{}
	store := newFakeStore()
	svc := newService(repo, store, nil)

	m, err := svc.AddMedia(context.Background(), media.AddMediaInput{
		ModelType:  "user",
		ModelID:    "u1",
		Collection: "avatar",
		Name:       "profile.jpg",
		MimeType:   "image/jpeg",
		Size:       5,
		Reader:     strings.NewReader("hello"),
	})
	if err != nil {
		t.Fatalf("AddMedia error: %v", err)
	}
	if m.ID == "" || m.ModelType != "user" || m.CollectionName != "avatar" {
		t.Fatalf("unexpected media record: %+v", m)
	}
	if len(store.files) != 1 {
		t.Fatalf("storage files = %d, want 1", len(store.files))
	}
	if repo.byID[m.ID] == nil {
		t.Fatal("media not persisted in repo")
	}
}

func TestAddMediaRollsBackStorageOnSaveFailure(t *testing.T) {
	repo := &fakeRepo{saveErr: errors.New("boom")}
	store := newFakeStore()
	svc := newService(repo, store, nil)

	if _, err := svc.AddMedia(context.Background(), media.AddMediaInput{
		ModelType: "user", ModelID: "u1", Name: "a.jpg", Reader: strings.NewReader("x"),
	}); err == nil {
		t.Fatal("AddMedia succeeded, want error")
	}
	if len(store.files) != 0 {
		t.Fatalf("storage files = %d after rollback, want 0", len(store.files))
	}
}

func TestGetMediaNotFound(t *testing.T) {
	svc := newService(&fakeRepo{}, newFakeStore(), nil)
	if _, err := svc.GetMedia(context.Background(), "nope"); !errors.Is(err, media.ErrNotFound) {
		t.Fatalf("GetMedia error = %v, want media.ErrNotFound", err)
	}
}

func TestURLUsesGenerator(t *testing.T) {
	repo := &fakeRepo{}
	svc := newService(repo, newFakeStore(), &fakeURLGen{url: "https://cdn.example.com/a.jpg"})

	m := &media.Media{ID: "m1", FileName: "media/user/u1/a.jpg"}
	repo.byID = map[string]*media.Media{m.ID: m}

	u, err := svc.URL(context.Background(), "m1")
	if err != nil {
		t.Fatalf("URL error: %v", err)
	}
	if u != "https://cdn.example.com/a.jpg" {
		t.Fatalf("URL = %q", u)
	}
}

func TestURLNoGenerator(t *testing.T) {
	repo := &fakeRepo{}
	svc := newService(repo, newFakeStore(), nil)

	m := &media.Media{ID: "m1", FileName: "media/user/u1/a.jpg"}
	repo.byID = map[string]*media.Media{m.ID: m}

	if _, err := svc.URL(context.Background(), "m1"); !errors.Is(err, media.ErrNoURL) {
		t.Fatalf("URL error = %v, want media.ErrNoURL", err)
	}
}

func TestRemoveMediaDeletesFromStorageAndRepo(t *testing.T) {
	repo := &fakeRepo{}
	store := newFakeStore()
	svc := newService(repo, store, nil)

	m := &media.Media{ID: "m1", FileName: "media/user/u1/a.jpg"}
	repo.byID = map[string]*media.Media{m.ID: m}
	store.files[m.FileName] = "x"

	if err := svc.RemoveMedia(context.Background(), "m1"); err != nil {
		t.Fatalf("RemoveMedia error: %v", err)
	}
	if len(store.files) != 0 {
		t.Fatalf("storage files = %d after remove, want 0", len(store.files))
	}
	if repo.byID["m1"] != nil {
		t.Fatal("media still in repo after remove")
	}
}
