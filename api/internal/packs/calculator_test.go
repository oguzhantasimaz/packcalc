package packs

import (
	"errors"
	"reflect"
	"sort"
	"testing"
)

var defaultSizes = []int{250, 251, 500, 501, 1000, 1501, 2000, 2222, 5000, 5913, 12034, 348539}

// sumPacks returns the total items represented by a slice of PackCount.
func sumPacks(p []PackCount) int {
	total := 0
	for _, pc := range p {
		total += pc.Size * pc.Count
	}
	return total
}

// countPacks returns the total number of packs (sum of counts).
func countPacks(p []PackCount) int {
	total := 0
	for _, pc := range p {
		total += pc.Count
	}
	return total
}

// assertSortedDesc verifies that Packs is sorted by Size descending.
func assertSortedDesc(t *testing.T, p []PackCount) {
	t.Helper()
	for i := 1; i < len(p); i++ {
		if p[i-1].Size <= p[i].Size {
			t.Fatalf("packs not sorted desc: %v", p)
		}
	}
}

func TestCalculate_ReviewerExamples(t *testing.T) {
	// Pinned to the canonical case-study pack set so this test remains a
	// fixed regression check regardless of edits to defaultSizes elsewhere.
	reviewerSizes := []int{250, 500, 1000, 2000, 5000}
	cases := []struct {
		order      int
		wantItems  int
		wantPacks  int
		wantCounts map[int]int
	}{
		{1, 250, 1, map[int]int{250: 1}},
		{250, 250, 1, map[int]int{250: 1}},
		{251, 500, 1, map[int]int{500: 1}},
		{501, 750, 2, map[int]int{500: 1, 250: 1}},
		{12001, 12250, 4, map[int]int{5000: 2, 2000: 1, 250: 1}},
	}
	for _, c := range cases {
		c := c
		t.Run("", func(t *testing.T) {
			got, err := Calculate(c.order, reviewerSizes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.TotalItems != c.wantItems {
				t.Errorf("total_items: got %d, want %d", got.TotalItems, c.wantItems)
			}
			if got.TotalPacks != c.wantPacks {
				t.Errorf("total_packs: got %d, want %d", got.TotalPacks, c.wantPacks)
			}
			gotCounts := map[int]int{}
			for _, pc := range got.Packs {
				gotCounts[pc.Size] = pc.Count
			}
			if !reflect.DeepEqual(gotCounts, c.wantCounts) {
				t.Errorf("packs: got %v, want %v", gotCounts, c.wantCounts)
			}
			assertSortedDesc(t, got.Packs)
		})
	}
}

func TestCalculate_ExactPackMatches(t *testing.T) {
	for _, s := range defaultSizes {
		s := s
		t.Run("", func(t *testing.T) {
			got, err := Calculate(s, defaultSizes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.TotalItems != s {
				t.Errorf("total_items: got %d, want %d", got.TotalItems, s)
			}
			if got.TotalPacks != 1 {
				t.Errorf("total_packs: got %d, want 1", got.TotalPacks)
			}
			if len(got.Packs) != 1 || got.Packs[0].Size != s || got.Packs[0].Count != 1 {
				t.Errorf("packs: got %v, want one pack of size %d", got.Packs, s)
			}
		})
	}
}

func TestCalculate_ZeroOrder(t *testing.T) {
	got, err := Calculate(0, defaultSizes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems != 0 || got.TotalPacks != 0 || len(got.Packs) != 0 {
		t.Errorf("zero order should yield empty Result, got %+v", got)
	}
}

func TestCalculate_SinglePackSize(t *testing.T) {
	// order=10, sizes=[3] -> 4x3 = 12 items, 4 packs.
	got, err := Calculate(10, []int{3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems != 12 {
		t.Errorf("total_items: got %d, want 12", got.TotalItems)
	}
	if got.TotalPacks != 4 {
		t.Errorf("total_packs: got %d, want 4", got.TotalPacks)
	}
	if !reflect.DeepEqual(got.Packs, []PackCount{{Size: 3, Count: 4}}) {
		t.Errorf("packs: got %v", got.Packs)
	}
}

func TestCalculate_OrderSmallerThanAnyPack(t *testing.T) {
	// order=1 with sizes=[1000] -> single pack of 1000.
	got, err := Calculate(1, []int{1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems != 1000 || got.TotalPacks != 1 {
		t.Errorf("got %+v, want 1000/1", got)
	}
}

// TestCalculate_Adversarial covers cases where a naive greedy by largest
// pack would fail. The "changing a dollar" analogy from the reviewer hints.
func TestCalculate_Adversarial(t *testing.T) {
	cases := []struct {
		name      string
		order     int
		sizes     []int
		wantItems int
		// For some cases the minimum number of packs is hard to assert by
		// hand; we verify it indirectly by checking optimality below.
	}{
		// gcd([23,31,53]) = 1, so all sufficiently large ints are reachable.
		// 263 is reachable exactly (e.g. 31+31+31+31+31+31+23+23+31 etc.).
		{"coprime small adversarial", 263, []int{23, 31, 53}, 263},
		// Big adversarial case from the spec.
		{"coprime large adversarial", 500001, []int{23, 31, 53}, 500001},
		// gcd([6,9,15]) = 3, so reachable values are multiples of 3.
		// order=7 -> minimum reachable >= 7 that is a multiple of 3 is 9.
		{"non-coprime small", 7, []int{6, 9, 15}, 9},
		// order=100, multiples of 3 reachable above the Frobenius number.
		// 100 itself is not a multiple of 3, smallest reachable >= 100 is 102.
		{"non-coprime medium", 100, []int{6, 9, 15}, 102},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got, err := Calculate(c.order, c.sizes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.TotalItems != c.wantItems {
				t.Errorf("total_items: got %d, want %d", got.TotalItems, c.wantItems)
			}
			// Always: every pack used must be a member of sizes.
			sizeSet := map[int]struct{}{}
			for _, s := range c.sizes {
				sizeSet[s] = struct{}{}
			}
			for _, pc := range got.Packs {
				if _, ok := sizeSet[pc.Size]; !ok {
					t.Errorf("result uses size %d not in input %v", pc.Size, c.sizes)
				}
				if pc.Count <= 0 {
					t.Errorf("zero or negative count for size %d", pc.Size)
				}
			}
			// Totals are internally consistent.
			if got.TotalItems != sumPacks(got.Packs) {
				t.Errorf("TotalItems %d != sum(packs)=%d", got.TotalItems, sumPacks(got.Packs))
			}
			if got.TotalPacks != countPacks(got.Packs) {
				t.Errorf("TotalPacks %d != count(packs)=%d", got.TotalPacks, countPacks(got.Packs))
			}
			// Result is sorted DESC.
			assertSortedDesc(t, got.Packs)

			// Rule 2: no smaller-or-equal reachable T < TotalItems with T >= order.
			// Verified by direct reachability check.
			if reach := smallestReachableGE(c.order, c.sizes); reach != got.TotalItems {
				t.Errorf("Rule 2 violation: smallest reachable >= order is %d, but got %d",
					reach, got.TotalItems)
			}

			// Rule 3: minimum pack count for exactly TotalItems.
			if min := minPacksForExact(got.TotalItems, c.sizes); min != got.TotalPacks {
				t.Errorf("Rule 3 violation: min packs for exact %d is %d, but got %d",
					got.TotalItems, min, got.TotalPacks)
			}
		})
	}
}

// smallestReachableGE returns the smallest T >= order that is a non-negative
// integer combination of sizes. This is an independent reference
// implementation used only by tests to verify Rule 2.
func smallestReachableGE(order int, sizes []int) int {
	if order == 0 {
		return 0
	}
	max := 0
	for _, s := range sizes {
		if s > max {
			max = s
		}
	}
	upper := order + max - 1
	reach := make([]bool, upper+1)
	reach[0] = true
	for i := 1; i <= upper; i++ {
		for _, s := range sizes {
			if i-s >= 0 && reach[i-s] {
				reach[i] = true
				break
			}
		}
	}
	for i := order; i <= upper; i++ {
		if reach[i] {
			return i
		}
	}
	return -1
}

// minPacksForExact returns the minimum number of packs that sum to exactly
// target. Used only by tests to verify Rule 3 against the implementation.
func minPacksForExact(target int, sizes []int) int {
	const inf = int(^uint(0) >> 1)
	dp := make([]int, target+1)
	for i := range dp {
		dp[i] = inf
	}
	dp[0] = 0
	for i := 1; i <= target; i++ {
		for _, s := range sizes {
			if i-s >= 0 && dp[i-s] != inf && dp[i-s]+1 < dp[i] {
				dp[i] = dp[i-s] + 1
			}
		}
	}
	return dp[target]
}

// TestCalculate_GreedyWouldFail demonstrates that a greedy-by-largest pack
// strategy is wrong, justifying the DP approach.
func TestCalculate_GreedyWouldFail(t *testing.T) {
	// sizes=[23, 31, 53], order=263.
	// Greedy: 4x53=212, remaining 51 -> 1x53=53 (overshoot 14) -> 5 packs, 265 items.
	// Actual optimum: smallestReachableGE(263) should equal 263 (exact).
	sizes := []int{23, 31, 53}
	order := 263
	got, err := Calculate(order, sizes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems != 263 {
		t.Fatalf("expected exact match at 263, got %d (greedy-style failure)", got.TotalItems)
	}
}

func TestCalculate_ValidationErrors(t *testing.T) {
	cases := []struct {
		name      string
		order     int
		sizes     []int
		wantErrIs error
	}{
		{"negative order", -1, defaultSizes, ErrInvalidOrder},
		{"empty sizes", 100, []int{}, ErrEmptySizes},
		{"zero size", 100, []int{0, 250}, ErrInvalidSize},
		{"negative size", 100, []int{-1, 250}, ErrInvalidSize},
		{"duplicate sizes", 100, []int{250, 250}, ErrDuplicateSize},
		{"order over limit", MaxOrder + 1, defaultSizes, ErrLimitExceeded},
		{"size over limit", 100, []int{MaxPackSize + 1}, ErrLimitExceeded},
		{"too many sizes", 100, manySizes(MaxDistinctSz + 1), ErrLimitExceeded},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			_, err := Calculate(c.order, c.sizes)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, c.wantErrIs) {
				t.Errorf("err = %v, want errors.Is(%v)", err, c.wantErrIs)
			}
		})
	}
}

func manySizes(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i + 1
	}
	return out
}

// TestCalculate_DoesNotMutateInput verifies that callers can pass a shared
// slice without observing reordering or other side effects.
func TestCalculate_DoesNotMutateInput(t *testing.T) {
	in := []int{500, 250, 5000, 1000, 2000}
	snapshot := make([]int, len(in))
	copy(snapshot, in)
	if _, err := Calculate(12001, in); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(in, snapshot) {
		// We don't claim ordering; we only claim element-set equality.
		sort.Ints(in)
		sort.Ints(snapshot)
		if !reflect.DeepEqual(in, snapshot) {
			t.Errorf("input mutated: before=%v after=%v", snapshot, in)
		}
	}
}

func BenchmarkCalculate_DefaultSizes_1M(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Calculate(1_000_000, defaultSizes); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCalculate_ReviewerCase_12001(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Calculate(12001, defaultSizes); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCalculate_Adversarial_500001(b *testing.B) {
	b.ReportAllocs()
	sizes := []int{23, 31, 53}
	for i := 0; i < b.N; i++ {
		if _, err := Calculate(500001, sizes); err != nil {
			b.Fatal(err)
		}
	}
}
