package zfs

// Zpool represents a ZFS Pool
type Zpool struct {
	Name      string
	Health    string
	Allocated uint64
	Size      uint64
	Free      uint64
}

// helper function to wrap typical calls to zpool
func zpool(arg ...string) ([][]string, error) {
	c := command{Command: "zpool"}
	return c.Run(arg...)
}

func prepend(s []string, v ...string) []string {
	l := len(v)
	t := make([]string, len(s)+l)
	copy(t[l:], s)
	for i := range v {
		t[i] = v[i]
	}
	return t
}

// GetZpool retrieves a Zpool
func GetZpool(name string) (*Zpool, error) {
	out, err := zpool("get", "all", "-p", name)
	if err != nil {
		return nil, err
	}

	// there is no -H
	out = out[1:len(out)]

	z := &Zpool{Name: name}
	for _, line := range out {
		z.parseLine(line)
	}

	return z, nil
}

// Datasets returns a slice of all datasets in a zpool
func (z *Zpool) Datasets() ([]*Dataset, error) {
	return Datasets(z.Name)
}

// Snapshots returns a slice of all snapshots in a zpool
func (z *Zpool) Snapshots() ([]*Dataset, error) {
	return Snapshots(z.Name)
}

// CreateZpool creates a new zpool
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

// Destroy destroys a zpool
func (z *Zpool) Destroy() error {
	_, err := zpool("destroy", z.Name)
	return err
}

// ListZpools list all zpools
func ListZpools() ([]*Zpool, error) {
	args := []string{"list", "-Ho", "name"}
	out, err := zpool(args...)
	if err != nil {
		return nil, err
	}

	// there is no -H
	out = out[1:len(out)]

	pools := make([]*Zpool, 0)
	for _, line := range out {
		z, err := GetZpool(line[0])
		if err != nil {
			return nil, err
		}
		pools = append(pools, z)
	}
	return pools, nil
}
