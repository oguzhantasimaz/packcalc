// Package store persists the configurable pack-size set behind a single
// interface (PackStore). Implementations are interchangeable; the HTTP
// layer constructs one at boot based on environment configuration and
// holds it as an interface value, so handler code is independent of the
// chosen backend.
package store

import (
	"context"
	"errors"
	"sort"

	"github.com/oklog/ulid/v2"
)

// ErrVersionMismatch is returned by Set when a non-empty ifMatch
// precondition does not match the store's current version.
var ErrVersionMismatch = errors.New("version mismatch")

// DefaultSizes is the canonical pack set seeded into a fresh store when
// no prior value exists. Defined here (not in packs) because seeding is a
// storage concern, not an algorithm concern.
var DefaultSizes = []int{250, 500, 1000, 2000, 5000}

// Snapshot is the value held in the store: a normalized set of pack
// sizes (sorted DESC) and an opaque version that callers may echo back
// to Set as an If-Match precondition.
type Snapshot struct {
	Sizes   []int  `json:"sizes"`
	Version string `json:"version"`
}

// PackStore is the persistence seam for the pack-size configuration. All
// implementations must be safe for concurrent use by multiple goroutines.
//
// Set semantics:
//   - An empty ifMatch means "no precondition" — the write proceeds
//     regardless of the current version.
//   - A non-empty ifMatch must equal the current version, or the call
//     returns ErrVersionMismatch and the store is unchanged.
//   - The returned Snapshot carries the new version and the normalized
//     (sorted DESC) sizes, so callers do not need to re-Get.
type PackStore interface {
	Get(ctx context.Context) (Snapshot, error)
	Set(ctx context.Context, sizes []int, ifMatch string) (Snapshot, error)
}

// normalize returns a new slice containing the input sizes sorted in
// descending order. The input is not mutated.
func normalize(sizes []int) []int {
	out := make([]int, len(sizes))
	copy(out, sizes)
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

// newVersion mints a fresh opaque version identifier. ULIDs are used so
// versions are both unique and lexicographically time-ordered, which makes
// the stream of writes inspectable in logs.
func newVersion() string {
	return ulid.Make().String()
}
