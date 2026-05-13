package store

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/oguzhantasimaz/packcalc/api/internal/packs"
)

func TestMemory_SeedsDefaults(t *testing.T) {
	s := NewMemory()
	snap, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := normalize(DefaultSizes)
	if !reflect.DeepEqual(snap.Sizes, want) {
		t.Errorf("seed: got %v, want %v", snap.Sizes, want)
	}
	if snap.Version == "" {
		t.Errorf("seed version is empty")
	}
}

func TestMemory_GetReturnsCopy(t *testing.T) {
	s := NewMemory()
	snap, _ := s.Get(context.Background())
	if len(snap.Sizes) == 0 {
		t.Fatal("expected non-empty seed")
	}
	// Mutating the returned slice must not affect the store.
	snap.Sizes[0] = 9_999_999
	again, _ := s.Get(context.Background())
	if again.Sizes[0] == 9_999_999 {
		t.Errorf("store leaks internal slice: %v", again.Sizes)
	}
}

func TestMemory_SetNormalizesAndBumpsVersion(t *testing.T) {
	s := NewMemory()
	before, _ := s.Get(context.Background())

	got, err := s.Set(context.Background(), []int{100, 50, 200}, "")
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !reflect.DeepEqual(got.Sizes, []int{200, 100, 50}) {
		t.Errorf("normalize: got %v, want [200 100 50]", got.Sizes)
	}
	if got.Version == before.Version {
		t.Errorf("version not bumped: %s", got.Version)
	}
}

func TestMemory_SetWithoutIfMatchAlwaysSucceeds(t *testing.T) {
	s := NewMemory()
	_, err := s.Set(context.Background(), []int{1, 2, 3}, "")
	if err != nil {
		t.Fatalf("first set: %v", err)
	}
	_, err = s.Set(context.Background(), []int{4, 5, 6}, "")
	if err != nil {
		t.Fatalf("second set with empty ifMatch must succeed: %v", err)
	}
}

func TestMemory_SetWithStaleIfMatchFails(t *testing.T) {
	s := NewMemory()
	first, _ := s.Get(context.Background())

	// Advance the version.
	if _, err := s.Set(context.Background(), []int{10}, ""); err != nil {
		t.Fatalf("advance: %v", err)
	}

	// Now try to write with the stale version.
	_, err := s.Set(context.Background(), []int{20}, first.Version)
	if !errors.Is(err, ErrVersionMismatch) {
		t.Errorf("expected ErrVersionMismatch, got %v", err)
	}

	// The store must be unchanged.
	cur, _ := s.Get(context.Background())
	if !reflect.DeepEqual(cur.Sizes, []int{10}) {
		t.Errorf("store mutated despite failed CAS: %v", cur.Sizes)
	}
}

func TestMemory_SetWithCurrentIfMatchSucceeds(t *testing.T) {
	s := NewMemory()
	cur, _ := s.Get(context.Background())
	got, err := s.Set(context.Background(), []int{42}, cur.Version)
	if err != nil {
		t.Fatalf("Set with current ifMatch: %v", err)
	}
	if !reflect.DeepEqual(got.Sizes, []int{42}) {
		t.Errorf("sizes: %v", got.Sizes)
	}
}

func TestMemory_SetValidatesInput(t *testing.T) {
	cases := []struct {
		name      string
		sizes     []int
		wantErrIs error
	}{
		{"empty", []int{}, packs.ErrEmptySizes},
		{"zero", []int{0}, packs.ErrInvalidSize},
		{"negative", []int{-1}, packs.ErrInvalidSize},
		{"duplicate", []int{100, 100}, packs.ErrDuplicateSize},
		{"over size limit", []int{packs.MaxPackSize + 1}, packs.ErrLimitExceeded},
		{"too many", overflowSizes(packs.MaxDistinctSz + 1), packs.ErrLimitExceeded},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			s := NewMemory()
			_, err := s.Set(context.Background(), c.sizes, "")
			if !errors.Is(err, c.wantErrIs) {
				t.Errorf("err = %v, want errors.Is(%v)", err, c.wantErrIs)
			}
		})
	}
}

func overflowSizes(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i + 1
	}
	return out
}

// TestMemory_ConcurrentSetsAreRaceSafe runs many goroutines hammering Set.
// With -race the test fails if there's any unsynchronized access; with
// or without -race, the final state must be a valid Snapshot (sizes
// normalized, version non-empty) consistent with one of the inputs.
func TestMemory_ConcurrentSetsAreRaceSafe(t *testing.T) {
	s := NewMemory()
	const goroutines = 50
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		g := g
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_, err := s.Set(context.Background(), []int{g*1000 + i + 1}, "")
				if err != nil {
					t.Errorf("Set: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	final, _ := s.Get(context.Background())
	if len(final.Sizes) != 1 {
		t.Errorf("final sizes len = %d, want 1", len(final.Sizes))
	}
	if final.Version == "" {
		t.Errorf("final version empty")
	}
}

// TestMemory_CASInterleavedWritesOneWinner verifies that under contention
// for the same starting version, exactly one writer succeeds with that
// ifMatch and all others receive ErrVersionMismatch.
func TestMemory_CASInterleavedWritesOneWinner(t *testing.T) {
	s := NewMemory()
	start, _ := s.Get(context.Background())

	const writers = 20
	var wins int64
	var fails int64
	var wg sync.WaitGroup
	wg.Add(writers)
	for i := 0; i < writers; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, err := s.Set(context.Background(), []int{i + 1}, start.Version)
			switch {
			case err == nil:
				atomic.AddInt64(&wins, 1)
			case errors.Is(err, ErrVersionMismatch):
				atomic.AddInt64(&fails, 1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if wins != 1 {
		t.Errorf("wins=%d, want 1", wins)
	}
	if fails != int64(writers-1) {
		t.Errorf("fails=%d, want %d", fails, writers-1)
	}
}

// Ensure Memory satisfies PackStore at compile time.
var _ PackStore = (*Memory)(nil)
