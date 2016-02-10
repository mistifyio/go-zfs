// +build !solaris

package zfs

// List of ZFS properties to retrieve from zfs list command on a non-Solaris platform
var DsPropList = []string{"name", "origin", "used", "available", "mountpoint", "compression", "type", "volsize", "quota", "written", "logicalused"}

// List of Zpool properties to retrieve from zpool list command on a non-Solaris platform
var ZpoolPropList = []string{"name", "health", "allocated", "size", "free"}
