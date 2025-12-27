package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	// Constants for RESP parsing.
	terminator byte = '\n'
)

// Checks if bytes at the given offset end with \r\n.
func hasValidTerminator(bytes []byte, offset int) bool {
	return len(bytes) > offset+1 && bytes[offset] == '\r' && bytes[offset+1] == '\n'
}

func readAndParseLength(r *bufio.Reader) (int, error) {
	// Read until the line terminator to get the length of elements in the array.
	bytes, err := r.ReadBytes(terminator)
	if err != nil {
		return 0, err
	}

	// Trim the actual separator and convert to integer.
	countStr := strings.TrimSuffix(string(bytes), "\r\n")
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return 0, &RESPError{Msg: "invalid length", Err: err}
	}
	return count, nil
}

// Reads an array from the RESP protocol.
func ReadArray(r *bufio.Reader) (RespArray, error) {
	count, err := readAndParseLength(r)
	if err != nil {
		return RespArray{}, err
	}

	// Handle null array case.
	if count == -1 {
		return RespArray{Elements: nil}, nil
	}

	// Once we have the actual cound, we read each element and recursively call ReadRESP to append to the array.
	elements := make([]RespValue, 0, count)
	for range count {
		// Parse each individual element in the array and handle any errors.
		elem, err := ReadRESP(r)
		if err != nil {
			return RespArray{}, err
		}

		// Append the parsed element to the elements slice.
		elements = append(elements, elem)
	}

	return RespArray{Elements: elements}, nil
}

// Reads a bulk string from the RESP protocol.
func ReadBulkString(r *bufio.Reader) (RespBulkString, error) {
	count, err := readAndParseLength(r)
	if err != nil {
		return RespBulkString{}, err
	}

	if count == -1 {
		return RespBulkString{Value: nil}, nil
	}

	bytes := make([]byte, count+2) // +2 for \r\n
	_, err = io.ReadFull(r, bytes)
	if err != nil {
		return RespBulkString{}, err
	}

	// Ensure that it ends with \r\n
	if !hasValidTerminator(bytes, count) {
		return RespBulkString{}, &RESPError{Msg: "bulk string not terminated properly"}
	}

	value := bytes[:count]
	return RespBulkString{Value: value}, nil
}

func ReadSimpleString(r *bufio.Reader) (RespSimpleString, error) {
	line, err := r.ReadString(terminator)
	if err != nil {
		return RespSimpleString{}, err
	}

	if !hasValidTerminator([]byte(line), len(line)-2) {
		return RespSimpleString{}, &RESPError{Msg: "simple string not terminated properly"}
	}

	// Trim the trailing \r\n
	value := strings.TrimSuffix(line, "\r\n")
	return RespSimpleString{Value: value}, nil
}

func ReadError(r *bufio.Reader) (RespErrorValue, error) {
	line, err := r.ReadString(terminator)
	if err != nil {
		return RespErrorValue{}, err
	}

	if !hasValidTerminator([]byte(line), len(line)-2) {
		return RespErrorValue{}, &RESPError{Msg: "error not terminated properly"}
	}

	// Trim the trailing \r\n
	value := strings.TrimSuffix(line, "\r\n")
	return RespErrorValue{Message: value}, nil
}

func ReadInteger(r *bufio.Reader) (RespInteger, error) {
	line, err := r.ReadString(terminator)
	if err != nil {
		return RespInteger{}, err
	}

	if !hasValidTerminator([]byte(line), len(line)-2) {
		return RespInteger{}, &RESPError{Msg: "integer not terminated properly"}
	}

	// Trim the trailing \r\n
	valueStr := strings.TrimSuffix(line, "\r\n")
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return RespInteger{}, &RESPError{Msg: "invalid integer", Err: err}
	}

	return RespInteger{Value: value}, nil
}

// Reads a RESP value from the reader.
func ReadRESP(r *bufio.Reader) (RespValue, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch prefix {
	case '*':
		return ReadArray(r)
	case '$':
		return ReadBulkString(r)
	case '+':
		return ReadSimpleString(r)
	case '-':
		return ReadError(r)
	case ':':
		return ReadInteger(r)
	default:
		return nil, &RESPError{Msg: fmt.Sprintf("unknown RESP type prefix: %c", prefix)}
	}
}
