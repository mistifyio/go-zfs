// +build solaris

package zfs

import (
	"errors"
	"strings"
)

// List of ZFS properties to retrieve from zfs list command on a Solaris platform
var dsPropList = []string{"name", "origin", "used", "available", "mountpoint", "compression", "type", "volsize", "quota"}

var dsPropListOptions = strings.Join(dsPropList, ",")

func (z *Zpool) parseLine(line []string) error {
	if len(line) != len(zpoolPropList) {
		return errors.New("Output not what is expected on this platform")
	}
	setString(&z.Name, line[0])
	setString(&z.Health, line[1])
	setString(&z.Allocated, line[2])
	setString(&z.Size, line[3])
	setString(&z.Free, line[4])

	return nil
}
