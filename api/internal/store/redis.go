package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/oguzhantasimaz/packcalc/api/internal/packs"
)

// RedisKey is the single key under which the pack-size configuration is
// stored. Exported so operators can inspect it via redis-cli.
const RedisKey = "packcalc:packs"

// Redis is a PackStore backed by a single JSON-encoded key in Redis. It
// uses the WATCH/MULTI/EXEC primitive to provide read-modify-write
// atomicity under concurrent writes, and an ifMatch precondition (mapped
// to a ULID version stored alongside the sizes) to surface conflicting
// edits to the caller.
type Redis struct {
	rdb *redis.Client
}

// NewRedis returns a Redis-backed PackStore. If the configured key is
// absent on construction, DefaultSizes are seeded atomically via SETNX so
// the API works out of the box on a fresh Redis instance.
func NewRedis(ctx context.Context, rdb *redis.Client) (*Redis, error) {
	r := &Redis{rdb: rdb}
	seed := Snapshot{Sizes: normalize(DefaultSizes), Version: newVersion()}
	b, err := json.Marshal(seed)
	if err != nil {
		return nil, fmt.Errorf("marshal seed: %w", err)
	}
	if _, err := rdb.SetNX(ctx, RedisKey, b, 0).Result(); err != nil {
		return nil, fmt.Errorf("seed: %w", err)
	}
	return r, nil
}

// ErrMissing is returned by Get if the key is absent. Construction via
// NewRedis seeds the key, so this is only observable if an operator
// manually deletes it.
var ErrMissing = errors.New("packcalc: pack-size key is missing")

func (r *Redis) Get(ctx context.Context) (Snapshot, error) {
	raw, err := r.rdb.Get(ctx, RedisKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return Snapshot{}, ErrMissing
		}
		return Snapshot{}, fmt.Errorf("redis get: %w", err)
	}
	var snap Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return Snapshot{}, fmt.Errorf("decode snapshot: %w", err)
	}
	return snap, nil
}

func (r *Redis) Set(ctx context.Context, sizes []int, ifMatch string) (Snapshot, error) {
	if err := packs.ValidateSizes(sizes); err != nil {
		return Snapshot{}, err
	}
	next := Snapshot{Sizes: normalize(sizes), Version: newVersion()}
	nextBytes, err := json.Marshal(next)
	if err != nil {
		return Snapshot{}, fmt.Errorf("marshal: %w", err)
	}

	txf := func(tx *redis.Tx) error {
		raw, err := tx.Get(ctx, RedisKey).Bytes()
		if err != nil && !errors.Is(err, redis.Nil) {
			return err
		}
		if ifMatch != "" {
			if errors.Is(err, redis.Nil) {
				return ErrVersionMismatch
			}
			var cur Snapshot
			if uerr := json.Unmarshal(raw, &cur); uerr != nil {
				return fmt.Errorf("decode current: %w", uerr)
			}
			if cur.Version != ifMatch {
				return ErrVersionMismatch
			}
		}
		_, pipeErr := tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
			p.Set(ctx, RedisKey, nextBytes, 0)
			return nil
		})
		return pipeErr
	}

	err = r.rdb.Watch(ctx, txf, RedisKey)
	switch {
	case err == nil:
		return next, nil
	case errors.Is(err, redis.TxFailedErr):
		// Another writer modified the key between our WATCH and EXEC; map
		// to the canonical CAS error so callers see one consistent type.
		return Snapshot{}, ErrVersionMismatch
	case errors.Is(err, ErrVersionMismatch):
		return Snapshot{}, ErrVersionMismatch
	default:
		return Snapshot{}, fmt.Errorf("redis set: %w", err)
	}
}

// Ping returns nil if Redis is reachable; intended for /readyz.
func (r *Redis) Ping(ctx context.Context) error {
	return r.rdb.Ping(ctx).Err()
}

// Compile-time interface guard.
var _ PackStore = (*Redis)(nil)
