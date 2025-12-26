package resp

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
	Value []byte
}

type RespSimpleString struct {
	Value string
}

type RespErrorValue struct {
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
	return e.Msg
}

func (e *RESPError) Unwrap() error {
	return e.Err
}
