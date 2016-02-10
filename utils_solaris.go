// +build solaris

package zfs

import (
	"strings"
)

// List of ZFS properties to retrieve from zfs list command on a Solaris platform
var dsPropList = []string{"name", "origin", "used", "available", "mountpoint", "compression", "type", "volsize", "quota"}

var dsPropListOptions = strings.Join(dsPropList, ",")
