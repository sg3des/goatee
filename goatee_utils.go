package main

import "bytes"

//not printable special symbols
var specialBytes = []byte(`\+*?()|[]{}^$`)

//if byte in special symbols
func special(b byte) bool {
	return bytes.IndexByte(specialBytes, b) >= 0
}

// ClearMeta clear meta symbols.
func ClearMeta(str string) string {
	s := []byte(str)
	var cs []byte
	for n, b := range s {
		if special(b) {
			if n != 0 && s[n-1] != '\\' {
				cs = append(cs, b)
			}
		} else {
			cs = append(cs, b)
		}
	}
	// fmt.Println(cs)

	return string(cs)
}
