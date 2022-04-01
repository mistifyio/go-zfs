package zfs

import (
	"reflect"
	"testing"
)

func TestParseLine(t *testing.T) {
	for name, test := range map[string]struct {
		prop  string
		value string
		want  Zpool
	}{
		"some fragmentation": {
			prop:  "fragmentation",
			value: "15%",
			want:  Zpool{Fragmentation: 15},
		},
		"no fragmentation": {
			prop:  "fragmentation",
			value: "0",
			want:  Zpool{Fragmentation: 0},
		},
		"untracked fragmentation": {
			prop:  "fragmentation",
			value: "-",
			want:  Zpool{Fragmentation: 0},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := Zpool{}
			got.parseLine([]string{"", test.prop, test.value})
			if !reflect.DeepEqual(test.want, got) {
				t.Fatalf("parse failure: wanted: %v, got: %v", test.want, got)
			}
		})
	}
}
