package util

import "strconv"

func ParsePositiveInt(s []byte) (int, bool) {
	n, err := strconv.Atoi(string(s))
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func ReverseSlice[T any](s [][]T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
