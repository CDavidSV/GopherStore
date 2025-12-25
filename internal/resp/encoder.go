package resp

import "strconv"

func EncodeSimpleString(value string) []byte {
	return []byte("+" + value + "\r\n")
}

func EncodeError(value string) []byte {
	return []byte("-" + value + "\r\n")
}

func EncodeBulkString(value []byte) []byte {
	// Handle nil values
	if value == nil {
		return []byte("$-1\r\n")
	}

	return []byte("$" + strconv.Itoa(len(value)) + "\r\n" + string(value) + "\r\n")
}

func EncodeInteger(value int64) []byte {
	return []byte(":" + strconv.FormatInt(value, 10) + "\r\n")
}
