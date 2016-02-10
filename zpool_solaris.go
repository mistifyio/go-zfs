// +build solaris
package zfs

// Zpool is a ZFS zpool.  A pool is a top-level structure in ZFS, and can
// contain many descendent datasets.
type Zpool struct {
	Name      string
	Health    string
	Allocated string
	Size      string
	Free      string
}

//Zpool on Solaris does not support the -p option
const zpoolListArgs = "-o"
