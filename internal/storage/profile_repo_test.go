package storage_test

import (
	"context"
	"errors"
	"testing"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/storage"
	"hop.top/kit/go/runtime/domain"
)

// newRepo returns a fresh ProfileRepo pointed at an isolated APS_DATA_PATH.
func newRepo(t *testing.T) *storage.ProfileRepo {
	t.Helper()
	t.Setenv("APS_DATA_PATH", t.TempDir())
	return storage.NewProfileRepo()
}

func TestProfileRepo_CreateAndGet(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	p := &core.Profile{ID: "noor", DisplayName: "Noor", Email: "noor@example.com"}
	if err := r.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := r.Get(ctx, "noor")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "noor" || got.DisplayName != "Noor" || got.Email != "noor@example.com" {
		t.Errorf("got = %+v", got)
	}
}

func TestProfileRepo_Create_DuplicateReturnsConflict(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	p := &core.Profile{ID: "dup", DisplayName: "Dup"}
	if err := r.Create(ctx, p); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	err := r.Create(ctx, p)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("err = %v, want ErrConflict", err)
	}
}

func TestProfileRepo_Get_MissingReturnsNotFound(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	_, err := r.Get(ctx, "nope")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestProfileRepo_Update_ReplacesProfile(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	p := &core.Profile{ID: "kai", DisplayName: "Kai"}
	if err := r.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	p.DisplayName = "Kai Renamed"
	if err := r.Update(ctx, p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := r.Get(ctx, "kai")
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.DisplayName != "Kai Renamed" {
		t.Errorf("display = %q, want Kai Renamed", got.DisplayName)
	}
}

func TestProfileRepo_Update_MissingReturnsNotFound(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	err := r.Update(ctx, &core.Profile{ID: "ghost", DisplayName: "Ghost"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestProfileRepo_Delete_RemovesProfile(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	if err := r.Create(ctx, &core.Profile{ID: "rami", DisplayName: "Rami"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := r.Delete(ctx, "rami"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := r.Get(ctx, "rami")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("after delete err = %v, want ErrNotFound", err)
	}
}

func TestProfileRepo_Delete_MissingReturnsNotFound(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	err := r.Delete(ctx, "nope")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestProfileRepo_List(t *testing.T) {
	r := newRepo(t)
	ctx := context.Background()

	for _, id := range []string{"a", "b", "c"} {
		if err := r.Create(ctx, &core.Profile{ID: id, DisplayName: id}); err != nil {
			t.Fatalf("Create %s: %v", id, err)
		}
	}

	got, err := r.List(ctx, domain.Query{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d profiles, want 3", len(got))
	}

	// With Limit
	limited, err := r.List(ctx, domain.Query{Limit: 2})
	if err != nil {
		t.Fatalf("List limit: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("limited got %d, want 2", len(limited))
	}

	// Search by DisplayName
	matched, err := r.List(ctx, domain.Query{Search: "a"})
	if err != nil {
		t.Fatalf("List search: %v", err)
	}
	if len(matched) != 1 || matched[0].ID != "a" {
		t.Errorf("search got %+v, want one profile a", matched)
	}
}

func TestProfile_GetID(t *testing.T) {
	p := core.Profile{ID: "x"}
	if p.GetID() != "x" {
		t.Errorf("GetID() = %q, want x", p.GetID())
	}
	// Pointer receiver too — confirms Entity interface satisfied either way.
	pp := &core.Profile{ID: "y"}
	if pp.GetID() != "y" {
		t.Errorf("(*Profile).GetID() = %q, want y", pp.GetID())
	}
}

// TestProfileRepo_SatisfiesDomainRepository verifies at compile-time that
// ProfileRepo implements domain.Repository[core.Profile]. If it ever stops
// satisfying the interface, this fails to compile.
func TestProfileRepo_SatisfiesDomainRepository(t *testing.T) {
	var _ domain.Repository[core.Profile] = storage.NewProfileRepo()
}
