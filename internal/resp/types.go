package resp

import "fmt"

type RespType int

const (
	// RESP data types.
	SimpleString RespType = iota
	Error
	Integer
	BulkString
	Array
)

// RESP value interface.
type RespValue interface{}

// RESP data types.
type RespArray struct {
	Elements []RespValue
}

type RespBulkString struct {
	Value string
}

type RespSimpleString struct {
	Value string
}

type RespError struct {
	Message string
}

type RespInteger struct {
	Value int64
}

// RESPError wraps parsing errors with context.
type RESPError struct {
	Msg string
	Err error
}

func (e *RESPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("invalid RESP format: %s: %v", e.Msg, e.Err)
	}
	return fmt.Sprintf("invalid RESP format: %s", e.Msg)
}

func (e *RESPError) Unwrap() error {
	return e.Err
}
