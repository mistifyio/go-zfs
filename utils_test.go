package zfs

import (
	"errors"
	"testing"
)

func TestUnescapeFilePath(t *testing.T) {
	var tests = []struct {
		escaped   string
		unescaped string
		err       error
	}{
		{"i heart unicode", "i heart unicode", nil},
		{"i \\0040\\0342\\0235\\0244 unicode", "", errors.New("invalid octal code: too short")},
		{"i \\0040\\0999\\0235\\0244\\0040 unicode", "", errors.New(`invalid octal code: strconv.ParseUint: parsing "0999": invalid syntax`)},
		{"i \\0040\\0342\\0235\\0244\\0040 unicode", "i ❤ unicode", nil},
		{"i \\0040\\0342\\0235\\0244\\0040\\0040\\0342\\0235\\0244\\0040 unicode", "i ❤❤ unicode", nil},
		{"i \\0040\\0342\\0235\\0244\\0040 \\0040\\0342\\0235\\0244\\0040 unicode", "i ❤ ❤ unicode", nil},
		{"i \\040\\342\\235\\244\\040 unicode", "i ❤ unicode", nil},
		{"i \\040\\342\\235\\244\\040\\040\\342\\235\\244\\040 unicode", "i ❤❤ unicode", nil},
		{"i \\040\\342\\235\\244\\040 \\040\\342\\235\\244\\040 unicode", "i ❤ ❤ unicode", nil},
	}

	for _, test := range tests {
		t.Log(test.unescaped)
		unescaped, err := unescapeFilepath(test.escaped)
		if err != nil && test.err != nil {
			if err.Error() != test.err.Error() {
				t.Fatalf("1mismatched errors: want:%v, got:%v\n", test.err, err)
			}
		} else if err != test.err {
			t.Fatalf("mismatched errors: want:%v, got:%v\n", test.err, err)
		}
		if unescaped != test.unescaped {
			t.Fatalf("mismatched unescaped:\nwant:|%v|\n got:|%v|\n",
				[]byte(test.unescaped), []byte(unescaped))
		}
	}
}
