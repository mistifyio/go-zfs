// Package zfs provides wrappers around the ZFS command line tools.
package zfs

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Dataset is a zfs dataset.  This could be a volume, filesystem, snapshot. Check the type field
// The field definitions can be found in the zfs manual: http://www.freebsd.org/cgi/man.cgi?zfs(8)
type Dataset struct {
	Name          string
	Used          uint64
	Avail         uint64
	Mountpoint    string
	Compression   string
	Type          string
	Written       uint64
	Volsize       uint64
	Usedbydataset uint64
	Quota         uint64
}

// helper function to wrap typical calls to zfs
func zfs(arg ...string) ([][]string, error) {
	c := command{Command: "zfs"}
	return c.Run(arg...)
}

// Datasets returns a slice of all datasets
func Datasets(filter string) ([]*Dataset, error) {
	return listByType("all", filter)
}

// Snapshots returns a slice of all snapshots
func Snapshots(filter string) ([]*Dataset, error) {
	return listByType("snapshot", filter)
}

// Filesystems returns a slice of all filesystems
func Filesystems(filter string) ([]*Dataset, error) {
	return listByType("filesystem", filter)
}

// Volumes returns a slice of all volumes
func Volumes(filter string) ([]*Dataset, error) {
	return listByType("volume", filter)
}

// GetDataset retrieves a single dataset
func GetDataset(name string) (*Dataset, error) {
	out, err := zfs("get", "all", "-Hp", name)
	if err != nil {
		return nil, err
	}

	ds := &Dataset{Name: name}
	for _, line := range out {
		if err := ds.parseLine(line); err != nil {
			return nil, err
		}
	}

	return ds, nil
}

// Clone clones a snapshot. An error will be returned if a non-snapshot is used
func (d *Dataset) Clone(dest string, properties map[string]string) (*Dataset, error) {
	if d.Type != "snapshot" {
		return nil, errors.New("can only clone snapshots")
	}
	args := make([]string, 2, 4)
	args[0] = "clone"
	args[1] = "-p"
	if properties != nil {
		args = append(args, propsSlice(properties)...)
	}
	args = append(args, []string{d.Name, dest}...)
	_, err := zfs(args...)
	if err != nil {
		return nil, err
	}
	return GetDataset(dest)
}

// ReceiveSnapshot receives a zfs stream into a new snapshot
func ReceiveSnapshot(input io.Reader, name string) (*Dataset, error) {
	c := command{Command: "zfs", Stdin: input}
	_, err := c.Run("receive", name)
	if err != nil {
		return nil, err
	}
	return GetDataset(name)
}

// SendSnapshot sends a snapshot as a zfs stream
func (d *Dataset) SendSnapshot(output io.Writer) error {
	if d.Type != "snapshot" {
		return errors.New("can only send snapshots")
	}

	c := command{Command: "zfs", Stdout: output}
	_, err := c.Run("send", d.Name)
	return err
}

// CreateVolume creates a new volume
func CreateVolume(name string, size uint64, properties map[string]string) (*Dataset, error) {
	args := make([]string, 4, 5)
	args[0] = "create"
	args[1] = "-p"
	args[2] = "-V"
	args[3] = strconv.FormatUint(size, 10)
	if properties != nil {
		args = append(args, propsSlice(properties)...)
	}
	args = append(args, name)
	_, err := zfs(args...)
	if err != nil {
		return nil, err
	}
	return GetDataset(name)
}

// Destroy destroys a dataset
func (d *Dataset) Destroy(recursive bool) error {
	args := make([]string, 1, 3)
	args[0] = "destroy"
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, d.Name)
	_, err := zfs(args...)
	return err
}

// SetProperty sets a property
func (d *Dataset) SetProperty(key, val string) error {
	prop := strings.Join([]string{key, val}, "=")
	_, err := zfs("set", prop, d.Name)
	return err
}

// GetProperty Gets a property
func (d *Dataset) GetProperty(key string) (string, error) {
	out, err := zfs("get", key, d.Name)
	if err != nil {
		return "", err
	}

	return out[0][2], nil
}

// Snapshots returns a slice of all snapshots of a given dataset
func (d *Dataset) Snapshots() ([]*Dataset, error) {
	return listByType("snapshot", d.Name)
}

// CreateFilesystem creates a new filesystem
func CreateFilesystem(name string, properties map[string]string) (*Dataset, error) {
	args := make([]string, 1, 4)
	args[0] = "create"

	if properties != nil {
		args = append(args, propsSlice(properties)...)
	}

	args = append(args, name)
	_, err := zfs(args...)
	if err != nil {
		return nil, err
	}
	return GetDataset(name)
}

// Snapshot creates a snapshot
func (d *Dataset) Snapshot(name string, recursive bool) (*Dataset, error) {
	args := make([]string, 1, 4)
	args[0] = "snapshot"
	if recursive {
		args = append(args, "-r")
	}
	snapName := fmt.Sprintf("%s@%s", d.Name, name)
	args = append(args, snapName)
	_, err := zfs(args...)
	if err != nil {
		return nil, err
	}
	return GetDataset(snapName)
}

// Rollback rolls back a given dataset to a previous snapshot
func (d *Dataset) Rollback(destroyMoreRecent bool) error {
	if d.Type != "snapshot" {
		errors.New("can only rollback snapshots")
	}

	args := make([]string, 1, 3)
	args[0] = "rollback"
	if destroyMoreRecent {
		args = append(args, "-r")
	}
	args = append(args, d.Name)

	_, err := zfs(args...)
	return err
}

// Children returns the children of the dataset. Depth of 0 does not limit recursion.
func (d *Dataset) Children(depth uint64) ([]*Dataset, error) {
	args := []string{"get", "all", "-t", "all", "-Hp"}
	if depth > 0 {
		args = append(args, "-d")
		args = append(args, strconv.FormatUint(depth, 10))
	} else {
		args = append(args, "-r")
	}
	args = append(args, d.Name)

	out, err := zfs(args...)
	if err != nil {
		return nil, err
	}

	datasets := make([]*Dataset, 0)
	name := ""
	var ds *Dataset
	for _, line := range out {
		if name != line[0] {
			name = line[0]
			ds = &Dataset{Name: name}
			datasets = append(datasets, ds)
		}
		if err := ds.parseLine(line); err != nil {
			return nil, err
		}
	}

	return datasets[1:len(datasets)], nil
}
