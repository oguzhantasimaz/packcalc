package store

import (
	"context"
	"sync"

	"github.com/oguzhantasimaz/packcalc/api/internal/packs"
)

// Memory is an in-process PackStore backed by a sync.RWMutex-guarded
// snapshot. It is seeded with DefaultSizes when constructed via
// NewMemory and is intended for single-replica deployments and tests.
type Memory struct {
	mu       sync.RWMutex
	snapshot Snapshot
}

// NewMemory returns a Memory store pre-populated with DefaultSizes so
// the API works out of the box on first boot.
func NewMemory() *Memory {
	return &Memory{snapshot: Snapshot{Sizes: normalize(DefaultSizes), Version: newVersion()}}
}

func (m *Memory) Get(_ context.Context) (Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Snapshot{Sizes: cloneInts(m.snapshot.Sizes), Version: m.snapshot.Version}, nil
}

func (m *Memory) Set(_ context.Context, sizes []int, ifMatch string) (Snapshot, error) {
	if err := packs.ValidateSizes(sizes); err != nil {
		return Snapshot{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if ifMatch != "" && ifMatch != m.snapshot.Version {
		return Snapshot{}, ErrVersionMismatch
	}
	next := Snapshot{Sizes: normalize(sizes), Version: newVersion()}
	m.snapshot = next
	return Snapshot{Sizes: cloneInts(next.Sizes), Version: next.Version}, nil
}

func cloneInts(s []int) []int {
	out := make([]int, len(s))
	copy(out, s)
	return out
}
