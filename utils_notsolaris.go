// +build !solaris

package zfs

import (
	"errors"
	"strings"
)

// List of ZFS properties to retrieve from zfs list command on a non-Solaris platform
var dsPropList = []string{"name", "origin", "used", "available", "mountpoint", "compression", "type", "volsize", "quota", "written", "logicalused", "usedbydataset"}

var dsPropListOptions = strings.Join(dsPropList, ",")

func (z *Zpool) parseLine(line []string) error {
	if len(line) != len(zpoolPropList) {
		return errors.New("Output not what is expected on this platform")
	}
	setString(&z.Name, line[0])
	setString(&z.Health, line[1])
	if err := setUint(&z.Allocated, line[2]); err != nil {
		return err
	}
	if err := setUint(&z.Size, line[3]); err != nil {
		return err
	}
	if err := setUint(&z.Free, line[4]); err != nil {
		return err
	}

	return nil
}
