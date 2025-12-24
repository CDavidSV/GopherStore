package resp

func EncodeSimpleString(value string) []byte {
	return []byte("+" + value + "\r\n")
}

func EncodeError(value string) []byte {
	return []byte("-" + value + "\r\n")
}
