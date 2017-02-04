package zfs_test

import (
	"testing"

	zfs "github.com/mistifyio/go-zfs"
)

func TestGetZpool(t *testing.T) {
	zpoolTest(t, func() {
		pool, err := zfs.GetZpool("test")
		ok(t, err)

		equals(t, "test", pool.Name)
		equals(t, "ONLINE", pool.Health)
		equals(t, false, pool.ReadOnly)
		size := 3 * (1<<30 - 1<<24)
		equals(t, uint64(size), pool.Size)
		equals(t, uint64(size), pool.Free+pool.Allocated)
		equals(t, uint64(0), pool.Fragmentation)
		equals(t, uint64(0), pool.Freeing)
		equals(t, uint64(0), pool.Leaked)
		equals(t, 1.0, pool.DedupRatio)
	})
}
