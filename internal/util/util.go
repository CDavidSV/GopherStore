package util

import "strconv"

func ParsePositiveInt(s []byte) (int, bool) {
	n, err := strconv.Atoi(string(s))
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}
