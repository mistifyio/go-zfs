// +build !solaris

package zfs

// Zpool is a ZFS zpool.  A pool is a top-level structure in ZFS, and can
// contain many descendent datasets.
type Zpool struct {
	Name      string
	Health    string
	Allocated uint64
	Size      uint64
	Free      uint64
}

var zpoolArgs = []string{"get", zpoolPropListOptions, "-p"}
