// Package zfs provides wrappers around the ZFS command line tools.
package zfs

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ZFS dataset types, which can indicate if a dataset is a filesystem,
// snapshot, or volume.
const (
	DatasetFilesystem = "filesystem"
	DatasetSnapshot   = "snapshot"
	DatasetVolume     = "volume"
)

// Dataset is a ZFS dataset.  A dataset could be a clone, filesystem, snapshot,
// or volume.  The Type struct member can be used to determine a dataset's type.
//
// The field definitions can be found in the ZFS manual:
// http://www.freebsd.org/cgi/man.cgi?zfs(8).
type Dataset struct {
	Name          string
	Origin        string
	Used          uint64
	Avail         uint64
	Mountpoint    string
	Compression   string
	Type          string
	Written       uint64
	Volsize       uint64
	Logicalused   uint64
	Usedbydataset uint64
	Quota         uint64
	Referenced    uint64
}

// InodeType is the type of inode as reported by Diff
type InodeType int

// Types of Inodes
const (
	_                     = iota // 0 == unknown type
	BlockDevice InodeType = iota
	CharacterDevice
	Directory
	Door
	NamedPipe
	SymbolicLink
	EventPort
	Socket
	File
)

// ChangeType is the type of inode change as reported by Diff
type ChangeType int

// Types of Changes
const (
	_                  = iota // 0 == unknown type
	Removed ChangeType = iota
	Created
	Modified
	Renamed
)

// DestroyFlag is the options flag passed to Destroy
type DestroyFlag int64

// Valid destroy options
const (
	DestroyDefault         DestroyFlag = 1 << iota
	DestroyRecursive                   = 1 << iota
	DestroyRecursiveClones             = 1 << iota
	DestroyDeferDeletion               = 1 << iota
	DestroyForceUmount                 = 1 << iota
)

// SendFlag is the options flags passed to SendSnapshot
type SendFlag int64

// Valid send options
const (
	SendDefault	   SendFlag = 1 << iota
	IncrementalStream	    = 1 << iota
	IncrementalPackage	    = 1 << iota
	ReplicationStream	    = 1 << iota
)

// InodeChange represents a change as reported by Diff
type InodeChange struct {
	Change               ChangeType
	Type                 InodeType
	Path                 string
	NewPath              string
	ReferenceCountChange int
}

// Logger can be used to log commands/actions
type Logger interface {
	Log(cmd []string)
}

type defaultLogger struct{}

func (*defaultLogger) Log(cmd []string) {
	return
}

var logger Logger = &defaultLogger{}

// SetLogger set a log handler to log all commands including arguments before
// they are executed
func SetLogger(l Logger) {
	if l != nil {
		logger = l
	}
}

// zfs is a helper function to wrap typical calls to zfs.
func zfs(arg ...string) ([][]string, error) {
	c := command{Command: "zfs"}
	return c.Run(arg...)
}

// Datasets returns a slice of ZFS datasets, regardless of type.
// A filter argument may be passed to select a dataset with the matching name,
// or empty string ("") may be used to select all datasets.
func Datasets(filter string) ([]*Dataset, error) {
	return listByType("all", filter)
}

// Snapshots returns a slice of ZFS snapshots.
// A filter argument may be passed to select a snapshot with the matching name,
// or empty string ("") may be used to select all snapshots.
func Snapshots(filter string) ([]*Dataset, error) {
	return listByType(DatasetSnapshot, filter)
}

// Filesystems returns a slice of ZFS filesystems.
// A filter argument may be passed to select a filesystem with the matching name,
// or empty string ("") may be used to select all filesystems.
func Filesystems(filter string) ([]*Dataset, error) {
	return listByType(DatasetFilesystem, filter)
}

// Volumes returns a slice of ZFS volumes.
// A filter argument may be passed to select a volume with the matching name,
// or empty string ("") may be used to select all volumes.
func Volumes(filter string) ([]*Dataset, error) {
	return listByType(DatasetVolume, filter)
}

// GetDataset retrieves a single ZFS dataset by name.  This dataset could be
// any valid ZFS dataset type, such as a clone, filesystem, snapshot, or volume.
func GetDataset(name string) (*Dataset, error) {
	out, err := zfs("list", "-Hp", "-o", dsPropListOptions, name)
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

// Clone clones a ZFS snapshot and returns a clone dataset.
// An error will be returned if the input dataset is not of snapshot type.
func (d *Dataset) Clone(dest string, properties map[string]string) (*Dataset, error) {
	if d.Type != DatasetSnapshot {
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

// Unmount unmounts currently mounted ZFS file systems.
func (d *Dataset) Unmount(force bool) (*Dataset, error) {
	if d.Type == DatasetSnapshot {
		return nil, errors.New("cannot unmount snapshots")
	}
	args := make([]string, 1, 3)
	args[0] = "umount"
	if force {
		args = append(args, "-f")
	}
	args = append(args, d.Name)
	_, err := zfs(args...)
	if err != nil {
		return nil, err
	}
	return GetDataset(d.Name)
}

// Mount mounts ZFS file systems.
func (d *Dataset) Mount(overlay bool, options []string) (*Dataset, error) {
	if d.Type == DatasetSnapshot {
		return nil, errors.New("cannot mount snapshots")
	}
	args := make([]string, 1, 5)
	args[0] = "mount"
	if overlay {
		args = append(args, "-O")
	}
	if options != nil {
		args = append(args, "-o")
		args = append(args, strings.Join(options, ","))
	}
	args = append(args, d.Name)
	_, err := zfs(args...)
	if err != nil {
		return nil, err
	}
	return GetDataset(d.Name)
}

// ReceiveSnapshot receives a ZFS stream from the input io.Reader, creates a
// new snapshot with the specified name, and streams the input data into the
// newly-created snapshot.
func ReceiveSnapshot(input io.Reader, name string) (*Dataset, error) {
	c := command{Command: "zfs", Stdin: input}
	_, err := c.Run("receive", name)
	if err != nil {
		return nil, err
	}
	return GetDataset(name)
}

// ReceiveSnapshotRollback forces a rollback of the file system to the most recent
// snapshot before the receive is initiated. After, receives a ZFS stream from the
// input io.Reader like ReceiveSnapshot does.
func ReceiveSnapshotRollback(input io.Reader, name string, overwrite bool) (*Dataset, error) {
	c := command{Command: "zfs", Stdin: input}
	_, err := c.Run("receive", "-F", name)
	if err != nil {
		return nil, err
	}
	return GetDataset(name)
}

// SendSnapshot sends a ZFS stream of a snapshot to the input io.Writer.
// An error will be returned if the input dataset is not of snapshot type.
func (d *Dataset) SendSnapshot(output io.Writer, flags SendFlag) error {
	if d.Type != DatasetSnapshot {
		return errors.New("can only send snapshots")
	}
	c := command{Command: "zfs", Stdout: output}

	// Flags for SendSnapshot
	if flags&ReplicationStream !=0 {
		_, err := c.Run("send", "-R", d.Name)
		return err
	} else {
		_, err := c.Run("send", d.Name)
		return err
	}
}

// SendSnapshotIncremental sends a ZFS incremental stream to the input io.Writer.
// Includes options -i and -I to send an incremental stream or a stream package respectively.
func SendSnapshotIncremental(output io.Writer, d1 *Dataset, d2 *Dataset, replication bool, flags SendFlag) error {
	if d1.Type != DatasetSnapshot || d2.Type != DatasetSnapshot {
		return errors.New("can only send snapshots")
	}

	// Flags for SendSnapshot
	option := ""
	if flags&IncrementalStream !=0 {
		option = "-i"
	}
	if flags&IncrementalPackage !=0 {
		option = "-I"
	}
	c := command{Command: "zfs", Stdout: output}
	if replication == true {
		stream := "-R"
		_, err := c.Run("send", stream, option, d1.Name, d2.Name)
		return err
	} else {
		_, err := c.Run("send", option, d1.Name, d2.Name)
		return err
	}
}

// CreateVolume creates a new ZFS volume with the specified name, size, and
// properties.
// A full list of available ZFS properties may be found here:
// https://www.freebsd.org/cgi/man.cgi?zfs(8).
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

// Destroy destroys a ZFS dataset. If the destroy bit flag is set, any
// descendents of the dataset will be recursively destroyed, including snapshots.
// If the deferred bit flag is set, the snapshot is marked for deferred
// deletion.
func (d *Dataset) Destroy(flags DestroyFlag) error {
	args := make([]string, 1, 3)
	args[0] = "destroy"
	if flags&DestroyRecursive != 0 {
		args = append(args, "-r")
	}

	if flags&DestroyRecursiveClones != 0 {
		args = append(args, "-R")
	}

	if flags&DestroyDeferDeletion != 0 {
		args = append(args, "-d")
	}

	if flags&DestroyForceUmount != 0 {
		args = append(args, "-f")
	}

	args = append(args, d.Name)
	_, err := zfs(args...)
	return err
}

// SetProperty sets a ZFS property on the receiving dataset.
// A full list of available ZFS properties may be found here:
// https://www.freebsd.org/cgi/man.cgi?zfs(8).
func (d *Dataset) SetProperty(key, val string) error {
	prop := strings.Join([]string{key, val}, "=")
	_, err := zfs("set", prop, d.Name)
	return err
}

// GetProperty returns the current value of a ZFS property from the
// receiving dataset.
// A full list of available ZFS properties may be found here:
// https://www.freebsd.org/cgi/man.cgi?zfs(8).
func (d *Dataset) GetProperty(key string) (string, error) {
	out, err := zfs("get", "-H", key, d.Name)
	if err != nil {
		return "", err
	}

	return out[0][2], nil
}

// Rename renames a dataset.
func (d *Dataset) Rename(name string, createParent bool, recursiveRenameSnapshots bool) (*Dataset, error) {
	args := make([]string, 3, 5)
	args[0] = "rename"
	args[1] = d.Name
	args[2] = name
	if createParent {
		args = append(args, "-p")
	}
	if recursiveRenameSnapshots {
		args = append(args, "-r")
	}
	_, err := zfs(args...)
	if err != nil {
		return d, err
	}

	return GetDataset(name)
}

// Snapshots returns a slice of all ZFS snapshots of a given dataset.
func (d *Dataset) Snapshots() ([]*Dataset, error) {
	return Snapshots(d.Name)
}

// CreateFilesystem creates a new ZFS filesystem with the specified name and
// properties.
// A full list of available ZFS properties may be found here:
// https://www.freebsd.org/cgi/man.cgi?zfs(8).
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

// Snapshot creates a new ZFS snapshot of the receiving dataset, using the
// specified name.  Optionally, the snapshot can be taken recursively, creating
// snapshots of all descendent filesystems in a single, atomic operation.
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

// Rollback rolls back the receiving ZFS dataset to a previous snapshot.
// Optionally, intermediate snapshots can be destroyed.  A ZFS snapshot
// rollback cannot be completed without this option, if more recent
// snapshots exist.
// An error will be returned if the input dataset is not of snapshot type.
func (d *Dataset) Rollback(destroyMoreRecent bool) error {
	if d.Type != DatasetSnapshot {
		return errors.New("can only rollback snapshots")
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

// Children returns a slice of children of the receiving ZFS dataset.
// A recursion depth may be specified, or a depth of 0 allows unlimited
// recursion.
func (d *Dataset) Children(depth uint64) ([]*Dataset, error) {
	args := []string{"list"}
	if depth > 0 {
		args = append(args, "-d")
		args = append(args, strconv.FormatUint(depth, 10))
	} else {
		args = append(args, "-r")
	}
	args = append(args, "-t", "all", "-Hp", "-o", dsPropListOptions)
	args = append(args, d.Name)

	out, err := zfs(args...)
	if err != nil {
		return nil, err
	}

	var datasets []*Dataset
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
	return datasets[1:], nil
}

// Diff returns changes between a snapshot and the given ZFS dataset.
// The snapshot name must include the filesystem part as it is possible to
// compare clones with their origin snapshots.
func (d *Dataset) Diff(snapshot string) ([]*InodeChange, error) {
	args := []string{"diff", "-FH", snapshot, d.Name}[:]
	out, err := zfs(args...)
	if err != nil {
		return nil, err
	}
	inodeChanges, err := parseInodeChanges(out)
	if err != nil {
		return nil, err
	}
	return inodeChanges, nil
}
root@poczfs-node6:~/go/src/github.com/mistifyio/go-zfs # cat zfs_test.go 
package zfs_test

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	zfs "github.com/mistifyio/go-zfs"
)

func sleep(delay int) {
	time.Sleep(time.Duration(delay) * time.Second)
}

func pow2(x int) int64 {
	return int64(math.Pow(2, float64(x)))
}

//https://github.com/benbjohnson/testing
// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// nok fails the test if an err is nil.
func nok(tb testing.TB, err error) {
	if err == nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: expected error: %s\033[39m\n\n", filepath.Base(file), line)
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

func zpoolTest(t *testing.T, fn func()) {
	tempfiles := make([]string, 3)
	for i := range tempfiles {
		f, _ := ioutil.TempFile("/tmp/", "zfs-")
		defer f.Close()
		err := f.Truncate(pow2(30))
		ok(t, err)
		tempfiles[i] = f.Name()
		defer os.Remove(f.Name())
	}

	pool, err := zfs.CreateZpool("test", nil, tempfiles...)
	ok(t, err)
	defer pool.Destroy()
	ok(t, err)
	fn()

}

func TestDatasets(t *testing.T) {
	zpoolTest(t, func() {
		_, err := zfs.Datasets("")
		ok(t, err)

		ds, err := zfs.GetDataset("test")
		ok(t, err)
		equals(t, zfs.DatasetFilesystem, ds.Type)
		equals(t, "", ds.Origin)
		if runtime.GOOS != "solaris" {
			assert(t, ds.Logicalused != 0, "Logicalused is not greater than 0")
		}
	})
}

func TestDatasetGetProperty(t *testing.T) {
	zpoolTest(t, func() {
		ds, err := zfs.GetDataset("test")
		ok(t, err)

		prop, err := ds.GetProperty("foobarbaz")
		nok(t, err)
		equals(t, "", prop)

		prop, err = ds.GetProperty("compression")
		ok(t, err)
		equals(t, "off", prop)
	})
}

func TestSnapshots(t *testing.T) {

	zpoolTest(t, func() {
		snapshots, err := zfs.Snapshots("")
		ok(t, err)

		for _, snapshot := range snapshots {
			equals(t, zfs.DatasetSnapshot, snapshot.Type)
		}
	})
}

func TestFilesystems(t *testing.T) {
	zpoolTest(t, func() {
		f, err := zfs.CreateFilesystem("test/filesystem-test", nil)
		ok(t, err)

		filesystems, err := zfs.Filesystems("")
		ok(t, err)

		for _, filesystem := range filesystems {
			equals(t, zfs.DatasetFilesystem, filesystem.Type)
		}

		ok(t, f.Destroy(zfs.DestroyDefault))
	})
}

func TestCreateFilesystemWithProperties(t *testing.T) {
	zpoolTest(t, func() {
		props := map[string]string{
			"compression": "lz4",
		}

		f, err := zfs.CreateFilesystem("test/filesystem-test", props)
		ok(t, err)

		equals(t, "lz4", f.Compression)

		filesystems, err := zfs.Filesystems("")
		ok(t, err)

		for _, filesystem := range filesystems {
			equals(t, zfs.DatasetFilesystem, filesystem.Type)
		}

		ok(t, f.Destroy(zfs.DestroyDefault))
	})
}

func TestVolumes(t *testing.T) {
	zpoolTest(t, func() {
		v, err := zfs.CreateVolume("test/volume-test", uint64(pow2(23)), nil)
		ok(t, err)

		// volumes are sometimes "busy" if you try to manipulate them right away
		sleep(1)

		equals(t, zfs.DatasetVolume, v.Type)
		volumes, err := zfs.Volumes("")
		ok(t, err)

		for _, volume := range volumes {
			equals(t, zfs.DatasetVolume, volume.Type)
		}

		ok(t, v.Destroy(zfs.DestroyDefault))
	})
}

func TestSnapshot(t *testing.T) {
	zpoolTest(t, func() {
		f, err := zfs.CreateFilesystem("test/snapshot-test", nil)
		ok(t, err)

		filesystems, err := zfs.Filesystems("")
		ok(t, err)

		for _, filesystem := range filesystems {
			equals(t, zfs.DatasetFilesystem, filesystem.Type)
		}

		s, err := f.Snapshot("test", false)
		ok(t, err)

		equals(t, zfs.DatasetSnapshot, s.Type)

		equals(t, "test/snapshot-test@test", s.Name)

		ok(t, s.Destroy(zfs.DestroyDefault))

		ok(t, f.Destroy(zfs.DestroyDefault))
	})
}

func TestClone(t *testing.T) {
	zpoolTest(t, func() {
		f, err := zfs.CreateFilesystem("test/snapshot-test", nil)
		ok(t, err)

		filesystems, err := zfs.Filesystems("")
		ok(t, err)

		for _, filesystem := range filesystems {
			equals(t, zfs.DatasetFilesystem, filesystem.Type)
		}

		s, err := f.Snapshot("test", false)
		ok(t, err)

		equals(t, zfs.DatasetSnapshot, s.Type)
		equals(t, "test/snapshot-test@test", s.Name)

		c, err := s.Clone("test/clone-test", nil)
		ok(t, err)

		equals(t, zfs.DatasetFilesystem, c.Type)

		ok(t, c.Destroy(zfs.DestroyDefault))

		ok(t, s.Destroy(zfs.DestroyDefault))

		ok(t, f.Destroy(zfs.DestroyDefault))
	})
}

func TestSendSnapshot(t *testing.T) {
	zpoolTest(t, func() {
		f, err := zfs.CreateFilesystem("test/snapshot-test", nil)
		ok(t, err)

		filesystems, err := zfs.Filesystems("")
		ok(t, err)

		for _, filesystem := range filesystems {
			equals(t, zfs.DatasetFilesystem, filesystem.Type)
		}

		s, err := f.Snapshot("test", false)
		ok(t, err)

		file, _ := ioutil.TempFile("/tmp/", "zfs-")
		defer file.Close()
		err = file.Truncate(pow2(30))
		ok(t, err)
		defer os.Remove(file.Name())

		err = s.SendSnapshot(file, zfs.SendDefault)
		ok(t, err)

		ok(t, s.Destroy(zfs.DestroyDefault))

		ok(t, f.Destroy(zfs.DestroyDefault))
	})
}

func TestSendSnapshotIncremental(t *testing.T) {
	zpoolTest(t, func() {
		f, err := zfs.CreateFilesystem("test/snapshot-test", nil)
		ok(t, err)

		filesystems, err := zfs.Filesystems("")
		ok(t, err)

		for _, filesystem := range filesystems {
			equals(t, zfs.DatasetFilesystem, filesystem.Type)
		}

		s1, err := f.Snapshot("snap1", false)
		ok(t, err)
		s2, err := f.Snapshot("snap2", false)
		ok(t, err)

		file, _ := ioutil.TempFile("/tmp/", "zfs-")
		defer file.Close()
		err = file.Truncate(pow2(30))
		ok(t, err)
		defer os.Remove(file.Name())

		err = zfs.SendSnapshotIncremental(file, s1, s2, true, zfs.IncrementalStream)
		ok(t, err)

		ok(t, s2.Destroy(zfs.DestroyDefault))
		ok(t, s1.Destroy(zfs.DestroyDefault))

		ok(t, f.Destroy(zfs.DestroyDefault))
	})
}

func TestChildren(t *testing.T) {
	zpoolTest(t, func() {
		f, err := zfs.CreateFilesystem("test/snapshot-test", nil)
		ok(t, err)

		s, err := f.Snapshot("test", false)
		ok(t, err)

		equals(t, zfs.DatasetSnapshot, s.Type)
		equals(t, "test/snapshot-test@test", s.Name)

		children, err := f.Children(0)
		ok(t, err)

		equals(t, 1, len(children))
		equals(t, "test/snapshot-test@test", children[0].Name)

		ok(t, s.Destroy(zfs.DestroyDefault))
		ok(t, f.Destroy(zfs.DestroyDefault))
	})
}

func TestListZpool(t *testing.T) {
	zpoolTest(t, func() {
		pools, err := zfs.ListZpools()
		ok(t, err)
		for _, pool := range pools {
			if pool.Name == "test" {
				equals(t, "test", pool.Name)
				return
			}
		}
		t.Fatal("Failed to find test pool")
	})
}

func TestRollback(t *testing.T) {
	zpoolTest(t, func() {
		f, err := zfs.CreateFilesystem("test/snapshot-test", nil)
		ok(t, err)

		filesystems, err := zfs.Filesystems("")
		ok(t, err)

		for _, filesystem := range filesystems {
			equals(t, zfs.DatasetFilesystem, filesystem.Type)
		}

		s1, err := f.Snapshot("test", false)
		ok(t, err)

		_, err = f.Snapshot("test2", false)
		ok(t, err)

		s3, err := f.Snapshot("test3", false)
		ok(t, err)

		err = s3.Rollback(false)
		ok(t, err)

		err = s1.Rollback(false)
		assert(t, err != nil, "should error when rolling back beyond most recent without destroyMoreRecent = true")

		err = s1.Rollback(true)
		ok(t, err)

		ok(t, s1.Destroy(zfs.DestroyDefault))

		ok(t, f.Destroy(zfs.DestroyDefault))
	})
}

func TestDiff(t *testing.T) {
	zpoolTest(t, func() {
		fs, err := zfs.CreateFilesystem("test/origin", nil)
		ok(t, err)

		linkedFile, err := os.Create(filepath.Join(fs.Mountpoint, "linked"))
		ok(t, err)

		movedFile, err := os.Create(filepath.Join(fs.Mountpoint, "file"))
		ok(t, err)

		snapshot, err := fs.Snapshot("snapshot", false)
		ok(t, err)

		unicodeFile, err := os.Create(filepath.Join(fs.Mountpoint, "i ❤ unicode"))
		ok(t, err)

		err = os.Rename(movedFile.Name(), movedFile.Name()+"-new")
		ok(t, err)

		err = os.Link(linkedFile.Name(), linkedFile.Name()+"_hard")
		ok(t, err)

		inodeChanges, err := fs.Diff(snapshot.Name)
		ok(t, err)
		equals(t, 4, len(inodeChanges))

		equals(t, "/test/origin/", inodeChanges[0].Path)
		equals(t, zfs.Directory, inodeChanges[0].Type)
		equals(t, zfs.Modified, inodeChanges[0].Change)

		equals(t, "/test/origin/linked", inodeChanges[1].Path)
		equals(t, zfs.File, inodeChanges[1].Type)
		equals(t, zfs.Modified, inodeChanges[1].Change)
		equals(t, 1, inodeChanges[1].ReferenceCountChange)

		equals(t, "/test/origin/file", inodeChanges[2].Path)
		equals(t, "/test/origin/file-new", inodeChanges[2].NewPath)
		equals(t, zfs.File, inodeChanges[2].Type)
		equals(t, zfs.Renamed, inodeChanges[2].Change)

		equals(t, "/test/origin/i ❤ unicode", inodeChanges[3].Path)
		equals(t, zfs.File, inodeChanges[3].Type)
		equals(t, zfs.Created, inodeChanges[3].Change)

		ok(t, movedFile.Close())
		ok(t, unicodeFile.Close())
		ok(t, linkedFile.Close())
		ok(t, snapshot.Destroy(zfs.DestroyForceUmount))
		ok(t, fs.Destroy(zfs.DestroyForceUmount))
	})
}
