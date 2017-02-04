// +build !solaris

package zfs

import (
	"strings"
)

// List of ZFS properties to retrieve from zfs list command on a non-Solaris platform
var dsPropList = []string{"name", "origin", "used", "available", "mountpoint", "compression", "type", "volsize", "quota", "written", "logicalused", "usedbydataset"}

// List of Zpool properties to retrieve from zpool list command on a non-Solaris platform
var zpoolPropList = []string{"name", "health", "allocated", "size", "free", "readonly", "dedupratio", "fragmentation", "freeing", "leaked"}
var zpoolPropListOptions = strings.Join(zpoolPropList, ",")
var zpoolArgs = []string{"get", "-p", zpoolPropListOptions}
