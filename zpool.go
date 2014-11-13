package zfs

import (
	"strings"
)

// Zpool is a ZFS zpool.  A pool is a top-level structure in ZFS, and can
// contain many descendent datasets.
type Zpool struct {
	Name      string
	Health    string
	Allocated uint64
	Size      uint64
	Free      uint64
}

// zpool is a helper function to wrap typical calls to zpool.
func zpool(arg ...string) ([][]string, error) {
	c := command{Command: "zpool"}
	return c.Run(arg...)
}

// GetZpool retrieves a single ZFS zpool by name.
func GetZpool(name string) (*Zpool, error) {
	out, err := zpool("list", "-Ho", strings.Join(zpoolPropertyFields, ","), name)
	if err != nil {
		return nil, err
	}

	return parseZpoolLine(out[0])
}

// Datasets returns a slice of all ZFS datasets in a zpool.
func (z *Zpool) Datasets() ([]*Dataset, error) {
	return Datasets(z.Name)
}

// Snapshots returns a slice of all ZFS snapshots in a zpool.
func (z *Zpool) Snapshots() ([]*Dataset, error) {
	return Snapshots(z.Name)
}

// CreateZpool creates a new ZFS zpool with the specified name, properties,
// and optional arguments.
// A full list of available ZFS properties and command-line arguments may be
// found here: https://www.freebsd.org/cgi/man.cgi?zfs(8).
func CreateZpool(name string, properties map[string]string, args ...string) (*Zpool, error) {
	cli := make([]string, 1, 4)
	cli[0] = "create"
	if properties != nil {
		cli = append(cli, propsSlice(properties)...)
	}
	cli = append(cli, name)
	cli = append(cli, args...)
	_, err := zpool(cli...)
	if err != nil {
		return nil, err
	}

	return &Zpool{Name: name}, nil
}

// Destroy destroys a ZFS zpool by name.
func (z *Zpool) Destroy() error {
	_, err := zpool("destroy", z.Name)
	return err
}

// ListZpools list all ZFS zpools accessible on the current system.
func ListZpools() ([]*Zpool, error) {
	out, err := zpool("list", "-Ho", strings.Join(zpoolPropertyFields, ","))
	if err != nil {
		return nil, err
	}
	return parseZpoolLines(out)
}
