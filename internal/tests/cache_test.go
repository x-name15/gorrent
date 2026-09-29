package tests

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/x-name15/gorrent/pkg/search"
)

type dummySource struct {
	id    string
	calls int32
}

func (d *dummySource) ID() string   { return d.id }
func (d *dummySource) Name() string { return d.id }
func (d *dummySource) Search(ctx context.Context, query string) ([]search.TorrentResult, error) {
	atomic.AddInt32(&d.calls, 1)
	return []search.TorrentResult{
		{
			InfoHash: "0123456789abcdef0123456789abcdef01234567",
			Name:     "Result for " + query,
			Seeders:  10,
		},
	}, nil
}

func TestCachingSource_CeilingAndLRU(t *testing.T) {
	dummy := &dummySource{id: "test-source"}
	max := 3
	cs := search.NewCachingSourceWithLimit(dummy, 1*time.Minute, max)

	ctx := context.Background()

	// 1. Fill cache with 3 distinct items: q0, q1, q2
	for i := 0; i < max; i++ {
		res, err := cs.Search(ctx, fmt.Sprintf("q%d", i))
		if err != nil || len(res) == 0 {
			t.Fatalf("unexpected search error: %v", err)
		}
		if res[0].Name != fmt.Sprintf("Result for q%d", i) {
			t.Fatalf("unexpected result name: %s", res[0].Name)
		}
	}
	if atomic.LoadInt32(&dummy.calls) != int32(max) {
		t.Fatalf("expected %d source calls, got %d", max, dummy.calls)
	}

	// 2. Access q0 again -> should be a cache hit (calls must not increment)
	hitRes, _ := cs.Search(ctx, "q0")
	if hitRes[0].Name != "Result for q0" {
		t.Fatalf("cache returned wrong result for q0: %s", hitRes[0].Name)
	}
	if atomic.LoadInt32(&dummy.calls) != int32(max) {
		t.Fatalf("cache hit failed: calls incremented to %d", dummy.calls)
	}

	// 3. Insert q3 -> total distinct searches = 4 > max(3).
	// Because q0 was recently accessed, the least recently used entry was q1!
	// Therefore, q1 must be evicted.
	_, _ = cs.Search(ctx, "q3")
	if atomic.LoadInt32(&dummy.calls) != int32(max+1) {
		t.Fatalf("expected calls to be %d, got %d", max+1, dummy.calls)
	}

	// 4. Verify q0 is still cached (calls should stay max+1)
	_, _ = cs.Search(ctx, "q0")
	if atomic.LoadInt32(&dummy.calls) != int32(max+1) {
		t.Fatalf("expected q0 to remain cached due to LRU promotion, but call count increased to %d", dummy.calls)
	}

	// 5. Verify q1 was evicted (calls must increment by 1)
	_, _ = cs.Search(ctx, "q1")
	if atomic.LoadInt32(&dummy.calls) != int32(max+2) {
		t.Fatalf("expected q1 to be evicted and hit backend, calls = %d", dummy.calls)
	}
}
