// Package packs computes pack-shipment combinations under three rules:
//
//  1. Only whole packs are used.
//  2. Among all combinations satisfying (1), the total items shipped is
//     the minimum that meets-or-exceeds the order.
//  3. Among all combinations satisfying (1) and (2), the number of packs is
//     minimized. Rule 2 takes precedence over rule 3.
//
// The implementation uses a two-phase dynamic program. A naive greedy
// approach is insufficient because it can fail on pack sets such as
// [23, 31, 53] for order=263 (the "changing a dollar" case).
//
// Phase A — minimum reachable total.
//
//	Build a reachability table reachable[0..order+maxSize-1] via unbounded
//	knapsack: reachable[i] is true iff i can be expressed as a non-negative
//	integer combination of sizes. Scan upward from order; the first
//	reachable index is the minimum total T that satisfies rule 2. The range
//	is always sufficient because maxSize itself is reachable, so at least
//	one value in [order, order+maxSize-1] is reachable.
//
// Phase B — minimum packs for exactly T items.
//
//	Unbounded knapsack: dp[i] is the minimum pack count to reach exactly i,
//	with a parent table recording which size was used to land on i.
//	Backtrack from T to reconstruct the multiset of packs.
//
// Complexity: O((order + maxSize) * len(sizes)) time and O(order + maxSize)
// space.
package packs

import (
	"errors"
	"fmt"
	"sort"
)

// Defensive upper bounds. The dynamic program allocates two arrays of size
// order+maxSize, so we cap the order and pack sizes to keep memory bounded
// under hostile input. These limits are generous for the case-study workload.
const (
	MaxOrder      = 10_000_000
	MaxPackSize   = 10_000_000
	MaxDistinctSz = 100
)

// Sentinel errors. Callers should compare with errors.Is.
var (
	ErrInvalidOrder   = errors.New("order must be >= 0")
	ErrEmptySizes     = errors.New("pack sizes must not be empty")
	ErrInvalidSize    = errors.New("pack sizes must be > 0")
	ErrDuplicateSize  = errors.New("pack sizes must be unique")
	ErrLimitExceeded  = errors.New("input exceeds configured limits")
	ErrUnreconcilable = errors.New("no pack combination reaches the order") // unreachable in current invariants; defensive
)

// PackCount is a single bucket of one pack size used in a result.
type PackCount struct {
	Size  int `json:"size"`
	Count int `json:"count"`
}

// Result is the answer to a single Calculate call.
//
// Packs is sorted by Size in descending order so that consumers can render
// "2 x 5000 + 1 x 2000 + ..." without re-sorting.
type Result struct {
	Packs      []PackCount `json:"packs"`
	TotalItems int         `json:"total_items"`
	TotalPacks int         `json:"total_packs"`
}

// Calculate returns the pack combination that satisfies rules 1, 2, and 3
// (in that priority order) for the given order quantity and the given set
// of pack sizes.
//
// sizes may be passed in any order and is not mutated. The Calculate
// function does not depend on any package-level state, so it is safe for
// concurrent use.
func Calculate(order int, sizes []int) (Result, error) {
	if err := validate(order, sizes); err != nil {
		return Result{}, err
	}
	if order == 0 {
		return Result{}, nil
	}

	// Work on a defensive copy sorted ascending; sorting does not change the
	// answer but keeps the inner loops predictable.
	work := make([]int, len(sizes))
	copy(work, sizes)
	sort.Ints(work)
	maxSize := work[len(work)-1]

	// Upper bound for the search: any T in [order, order+maxSize-1] suffices
	// because maxSize itself is reachable by a single pack.
	upper := order + maxSize - 1

	// Phase A: reachability up to `upper`.
	reachable := make([]bool, upper+1)
	reachable[0] = true
	for i := 1; i <= upper; i++ {
		for _, s := range work {
			if i-s < 0 {
				break // work is ascending; further sizes are larger
			}
			if reachable[i-s] {
				reachable[i] = true
				break
			}
		}
	}

	target := -1
	for i := order; i <= upper; i++ {
		if reachable[i] {
			target = i
			break
		}
	}
	if target == -1 {
		// Invariant violation: maxSize is always reachable and lies in range
		// when order >= 1. Kept for defense-in-depth.
		return Result{}, fmt.Errorf("%w: order=%d sizes=%v", ErrUnreconcilable, order, sizes)
	}

	// Phase B: minimum packs summing to exactly `target`.
	const inf = int(^uint(0) >> 1)
	dp := make([]int, target+1)
	parent := make([]int, target+1) // size used to reach i; -1 if unreachable
	for i := range dp {
		dp[i] = inf
		parent[i] = -1
	}
	dp[0] = 0
	for i := 1; i <= target; i++ {
		for _, s := range work {
			if i-s < 0 {
				break
			}
			if dp[i-s] != inf && dp[i-s]+1 < dp[i] {
				dp[i] = dp[i-s] + 1
				parent[i] = s
			}
		}
	}

	// Reconstruct counts by walking back along parent[] from target to 0.
	counts := make(map[int]int, len(work))
	for i := target; i > 0; {
		s := parent[i]
		if s <= 0 {
			// Shouldn't happen given reachability proved above; defensive.
			return Result{}, fmt.Errorf("%w: backtrack failed at i=%d", ErrUnreconcilable, i)
		}
		counts[s]++
		i -= s
	}

	out := make([]PackCount, 0, len(counts))
	for s, c := range counts {
		out = append(out, PackCount{Size: s, Count: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Size > out[j].Size })

	return Result{
		Packs:      out,
		TotalItems: target,
		TotalPacks: dp[target],
	}, nil
}

// ValidateSizes enforces the contract for a pack-size set: non-empty, no
// duplicates, all positive, and within the configured upper bounds. It is
// shared between the calculator and the store so both layers agree on what
// a "valid" pack set looks like. Errors are wrapped via fmt.Errorf so call
// sites can use errors.Is to match the sentinel while still receiving
// context-rich messages.
func ValidateSizes(sizes []int) error {
	if len(sizes) == 0 {
		return ErrEmptySizes
	}
	if len(sizes) > MaxDistinctSz {
		return fmt.Errorf("%w: %d distinct sizes exceeds max=%d", ErrLimitExceeded, len(sizes), MaxDistinctSz)
	}
	seen := make(map[int]struct{}, len(sizes))
	for _, s := range sizes {
		if s <= 0 {
			return fmt.Errorf("%w: got %d", ErrInvalidSize, s)
		}
		if s > MaxPackSize {
			return fmt.Errorf("%w: size=%d exceeds max=%d", ErrLimitExceeded, s, MaxPackSize)
		}
		if _, dup := seen[s]; dup {
			return fmt.Errorf("%w: %d appears more than once", ErrDuplicateSize, s)
		}
		seen[s] = struct{}{}
	}
	return nil
}

// validate enforces the full Calculate contract: a valid order plus a
// valid sizes set.
func validate(order int, sizes []int) error {
	if order < 0 {
		return fmt.Errorf("%w: got %d", ErrInvalidOrder, order)
	}
	if order > MaxOrder {
		return fmt.Errorf("%w: order=%d exceeds max=%d", ErrLimitExceeded, order, MaxOrder)
	}
	return ValidateSizes(sizes)
}
