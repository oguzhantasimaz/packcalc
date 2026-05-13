package store

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/oguzhantasimaz/packcalc/api/internal/packs"
)

// newTestRedis spins up an in-process miniredis and returns a Redis
// store pointed at it. Cleanup is handled via t.Cleanup.
func newTestRedis(t *testing.T) (*Redis, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	s, err := NewRedis(context.Background(), rdb)
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	return s, mr
}

func TestRedis_SeedsDefaultsIfEmpty(t *testing.T) {
	s, _ := newTestRedis(t)
	snap, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := normalize(DefaultSizes)
	if !reflect.DeepEqual(snap.Sizes, want) {
		t.Errorf("seed: got %v, want %v", snap.Sizes, want)
	}
	if snap.Version == "" {
		t.Errorf("version empty after seed")
	}
}

func TestRedis_NewRedisDoesNotOverwriteExisting(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	// First boot seeds defaults.
	s1, err := NewRedis(context.Background(), rdb)
	if err != nil {
		t.Fatalf("first NewRedis: %v", err)
	}
	if _, err := s1.Set(context.Background(), []int{7, 11}, ""); err != nil {
		t.Fatalf("Set: %v", err)
	}
	cur, _ := s1.Get(context.Background())

	// Second boot must not clobber operator edits.
	s2, err := NewRedis(context.Background(), rdb)
	if err != nil {
		t.Fatalf("second NewRedis: %v", err)
	}
	after, _ := s2.Get(context.Background())
	if !reflect.DeepEqual(after.Sizes, cur.Sizes) {
		t.Errorf("second NewRedis clobbered sizes: %v -> %v", cur.Sizes, after.Sizes)
	}
	if after.Version != cur.Version {
		t.Errorf("second NewRedis clobbered version: %s -> %s", cur.Version, after.Version)
	}
}

func TestRedis_SetNormalizesAndBumpsVersion(t *testing.T) {
	s, _ := newTestRedis(t)
	before, _ := s.Get(context.Background())

	got, err := s.Set(context.Background(), []int{100, 50, 200}, "")
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !reflect.DeepEqual(got.Sizes, []int{200, 100, 50}) {
		t.Errorf("normalize: got %v, want [200 100 50]", got.Sizes)
	}
	if got.Version == before.Version {
		t.Errorf("version not bumped")
	}

	// Read-back must reflect the write.
	read, _ := s.Get(context.Background())
	if !reflect.DeepEqual(read.Sizes, got.Sizes) || read.Version != got.Version {
		t.Errorf("readback mismatch: %+v vs %+v", read, got)
	}
}

func TestRedis_SetWithStaleIfMatchFails(t *testing.T) {
	s, _ := newTestRedis(t)
	first, _ := s.Get(context.Background())

	if _, err := s.Set(context.Background(), []int{10}, ""); err != nil {
		t.Fatalf("advance: %v", err)
	}

	_, err := s.Set(context.Background(), []int{20}, first.Version)
	if !errors.Is(err, ErrVersionMismatch) {
		t.Errorf("expected ErrVersionMismatch, got %v", err)
	}

	cur, _ := s.Get(context.Background())
	if !reflect.DeepEqual(cur.Sizes, []int{10}) {
		t.Errorf("store mutated despite failed CAS: %v", cur.Sizes)
	}
}

func TestRedis_SetWithCurrentIfMatchSucceeds(t *testing.T) {
	s, _ := newTestRedis(t)
	cur, _ := s.Get(context.Background())
	got, err := s.Set(context.Background(), []int{42}, cur.Version)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !reflect.DeepEqual(got.Sizes, []int{42}) {
		t.Errorf("sizes: %v", got.Sizes)
	}
}

func TestRedis_GetMissingKeyReturnsErrMissing(t *testing.T) {
	s, mr := newTestRedis(t)
	mr.Del(RedisKey)
	_, err := s.Get(context.Background())
	if !errors.Is(err, ErrMissing) {
		t.Errorf("expected ErrMissing, got %v", err)
	}
}

func TestRedis_SetValidatesInput(t *testing.T) {
	cases := []struct {
		name      string
		sizes     []int
		wantErrIs error
	}{
		{"empty", []int{}, packs.ErrEmptySizes},
		{"zero", []int{0}, packs.ErrInvalidSize},
		{"duplicate", []int{100, 100}, packs.ErrDuplicateSize},
		{"over size limit", []int{packs.MaxPackSize + 1}, packs.ErrLimitExceeded},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			s, _ := newTestRedis(t)
			_, err := s.Set(context.Background(), c.sizes, "")
			if !errors.Is(err, c.wantErrIs) {
				t.Errorf("err = %v, want errors.Is(%v)", err, c.wantErrIs)
			}
		})
	}
}

func TestRedis_PingReportsHealth(t *testing.T) {
	// Silence go-redis's internal retry log spam during this test — we
	// deliberately break the connection and the retry messages would
	// otherwise pollute the test output.
	redis.SetLogger(quietRedisLogger{})

	s, mr := newTestRedis(t)
	if err := s.Ping(context.Background()); err != nil {
		t.Fatalf("ping before close: %v", err)
	}
	mr.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := s.Ping(ctx); err == nil {
		t.Errorf("ping after close should error, got nil")
	}
}

// quietRedisLogger implements go-redis's logging interface and discards
// everything. Used to keep test output clean when intentionally
// triggering connection failures.
type quietRedisLogger struct{}

func (quietRedisLogger) Printf(_ context.Context, _ string, _ ...interface{}) {}
