package util

import "strconv"

func ParsePositiveInt(s []byte) (int, bool) {
	n, err := strconv.Atoi(string(s))
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func ParseInt(s []byte) (int, bool) {
	n, err := strconv.Atoi(string(s))
	if err != nil {
		return 0, false
	}
	return n, true
}

func ReverseSlice[T any](s [][]T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func SliceList[T any](list []T, start, end int) []T {
	length := len(list)

	startIndex := start
	endIndex := end

	if startIndex < 0 {
		startIndex = length + startIndex
	}

	if endIndex < 0 {
		endIndex = length + endIndex
	}

	startIndex = max(0, startIndex)
	endIndex = min(length-1, endIndex)

	if startIndex > endIndex {
		return []T{}
	} else {
		return list[startIndex : endIndex+1]
	}
}
