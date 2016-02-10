// +build !solaris

package zfs

import (
	"strings"
)

// List of ZFS properties to retrieve from zfs list command on a non-Solaris platform
var dsPropList = []string{"name", "origin", "used", "available", "mountpoint", "compression", "type", "volsize", "quota", "written", "logicalused", "usedbydataset"}

var dsPropListOptions = strings.Join(dsPropList, ",")

func (z *Zpool) parseLine(line []string) error {
	prop := line[1]
	val := line[2]

	var err error

	switch prop {
	case "name":
		setString(&z.Name, val)
	case "health":
		setString(&z.Health, val)
	case "allocated":
		err = setUint(&z.Allocated, val)
	case "size":
		err = setUint(&z.Size, val)
	case "free":
		err = setUint(&z.Free, val)
	}
	return err
}
