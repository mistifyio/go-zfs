package zfs

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

type command struct {
	Command string
	Stdin   io.Reader
	Stdout  io.Writer
}

func (c *command) Run(arg ...string) ([][]string, error) {

	cmd := exec.Command(c.Command, arg...)

	var stdout, stderr bytes.Buffer

	if c.Stdout == nil {
		cmd.Stdout = &stdout
	} else {
		cmd.Stdout = c.Stdout
	}

	if c.Stdin != nil {
		cmd.Stdin = c.Stdin

	}
	cmd.Stderr = &stderr

	debug := strings.Join([]string{cmd.Path, strings.Join(cmd.Args, " ")}, " ")
	err := cmd.Run()

	if err != nil {
		return nil, fmt.Errorf("%s: '%s' => %s", err, debug, stderr.String())
	}

	// assume if you passed in something for stdout, that you know what to do with it
	if c.Stdout != nil {
		return nil, nil
	}

	lines := strings.Split(stdout.String(), "\n")

	//last line is always blank
	lines = lines[0 : len(lines)-1]
	output := make([][]string, len(lines))

	for i, l := range lines {
		output[i] = strings.Split(l, "\t")
	}

	return output, nil
}

var propertyFields = make([]string, 0, 66)
var propertyMap = map[string]string{}

var zpoolPropertyFields = make([]string, 0, 66)
var zpoolPropertyMap = map[string]string{}

func init() {
	st := reflect.TypeOf(Dataset{})
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		// only look at exported values
		if f.PkgPath == "" {
			key := strings.ToLower(f.Name)
			propertyMap[key] = f.Name
			propertyFields = append(propertyFields, key)
		}
	}

	st = reflect.TypeOf(Zpool{})
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		// only look at exported values
		if f.PkgPath == "" {
			key := strings.ToLower(f.Name)
			zpoolPropertyMap[key] = f.Name
			zpoolPropertyFields = append(zpoolPropertyFields, key)
		}
	}
}

func parseDatasetLine(line []string) (*Dataset, error) {
	dataset := Dataset{}

	st := reflect.ValueOf(&dataset).Elem()

	for j, field := range propertyFields {

		fieldName := propertyMap[field]

		if fieldName == "" {
			continue
		}

		f := st.FieldByName(fieldName)
		value := line[j]

		switch f.Kind() {

		case reflect.Uint64:
			var v uint64
			if value != "-" {
				v, _ = strconv.ParseUint(value, 10, 64)
			}
			f.SetUint(v)

		case reflect.String:
			v := ""
			if value != "-" {
				v = value
			}
			f.SetString(v)

		}
	}
	return &dataset, nil
}

func parseDatasetLines(lines [][]string) ([]*Dataset, error) {
	datasets := make([]*Dataset, len(lines))

	for i, line := range lines {
		d, _ := parseDatasetLine(line)
		datasets[i] = d
	}

	return datasets, nil
}

func listByType(t, filter string) ([]*Dataset, error) {
	args := []string{"list", "-t", t, "-rHpo", strings.Join(propertyFields, ",")}[:]
	if filter != "" {
		args = append(args, filter)
	}
	out, err := zfs(args...)
	if err != nil {
		return nil, err
	}
	return parseDatasetLines(out)
}

func propsSlice(properties map[string]string) []string {
	args := make([]string, 0, len(properties)*3)
	for k, v := range properties {
		args = append(args, "-o")
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}
	return args
}

// based on https://github.com/dustin/go-humanize/blob/master/bytes.go
const (
	Byte  = 1
	KByte = Byte * 1024
	MByte = KByte * 1024
	GByte = MByte * 1024
	TByte = GByte * 1024
	PByte = TByte * 1024
	EByte = PByte * 1024
)

var bytesSizeTable = map[string]uint64{
	"b": Byte,
	"k": KByte,
	"m": MByte,
	"g": GByte,
	"t": TByte,
	"p": PByte,
	"e": EByte,
}

func parseBytes(s string) (uint64, error) {
	lastDigit := 0
	for _, r := range s {
		if !(unicode.IsDigit(r) || r == '.') {
			break
		}
		lastDigit++
	}

	f, err := strconv.ParseFloat(s[:lastDigit], 64)
	if err != nil {
		return 0, err
	}

	extra := strings.ToLower(strings.TrimSpace(s[lastDigit:]))
	if m, ok := bytesSizeTable[extra]; ok {
		f *= float64(m)
		if f >= math.MaxUint64 {
			return 0, fmt.Errorf("too large: %v", s)
		}
		return uint64(f), nil
	}

	return 0, fmt.Errorf("unhandled size name: %v", extra)
}

func parseZpoolLine(line []string) (*Zpool, error) {
	pool := Zpool{}

	st := reflect.ValueOf(&pool).Elem()

	for j, field := range zpoolPropertyFields {

		fieldName := zpoolPropertyMap[field]

		if fieldName == "" {
			continue
		}

		f := st.FieldByName(fieldName)
		value := line[j]

		switch f.Kind() {

		//sizes in zpool are apparently only availible in "human readable" form
		case reflect.Uint64:
			var v uint64
			if value != "-" {
				v, _ = parseBytes(value)
			}
			f.SetUint(v)

		case reflect.String:
			v := ""
			if value != "-" {
				v = value
			}
			f.SetString(v)

		}
	}
	return &pool, nil
}

func parseZpoolLines(lines [][]string) ([]*Zpool, error) {
	pools := make([]*Zpool, len(lines))

	for i, line := range lines {
		p, _ := parseZpoolLine(line)
		pools[i] = p
	}

	return pools, nil
}
