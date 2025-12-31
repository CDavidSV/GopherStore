package resp

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestReadAndParseLength(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        int
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid single digit length",
			input:   "5\r\n",
			want:    5,
			wantErr: false,
		},
		{
			name:    "valid multi-digit length",
			input:   "123\r\n",
			want:    123,
			wantErr: false,
		},
		{
			name:    "zero length",
			input:   "0\r\n",
			want:    0,
			wantErr: false,
		},
		{
			name:    "negative length",
			input:   "-1\r\n",
			want:    -1,
			wantErr: false,
		},
		{
			name:        "invalid non-numeric input",
			input:       "abc\r\n",
			want:        0,
			wantErr:     true,
			errContains: "invalid length",
		},
		{
			name:        "empty input",
			input:       "\r\n",
			want:        0,
			wantErr:     true,
			errContains: "invalid length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := readAndParseLength(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("readAndParseLength() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("readAndParseLength() error = %v, should contain %q", err, tt.errContains)
				}
			}
			if got != tt.want {
				t.Errorf("readAndParseLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadSimpleString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        RespSimpleString
		wantErr     bool
		errContains string
	}{
		{
			name:    "simple OK",
			input:   "OK\r\n",
			want:    RespSimpleString{Value: "OK"},
			wantErr: false,
		},
		{
			name:    "simple string with spaces",
			input:   "hello world\r\n",
			want:    RespSimpleString{Value: "hello world"},
			wantErr: false,
		},
		{
			name:    "empty simple string",
			input:   "\r\n",
			want:    RespSimpleString{Value: ""},
			wantErr: false,
		},
		{
			name:    "simple string with numbers",
			input:   "12345\r\n",
			want:    RespSimpleString{Value: "12345"},
			wantErr: false,
		},
		{
			name:        "simple string with incorrect terminator",
			input:       "hello\n\n",
			want:        RespSimpleString{},
			wantErr:     true,
			errContains: "not terminated properly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := ReadSimpleString(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadSimpleString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ReadSimpleString() error = %v, should contain %q", err, tt.errContains)
				}
			}
			if !tt.wantErr && got.Value != tt.want.Value {
				t.Errorf("ReadSimpleString() = %v, want %v", got.Value, tt.want.Value)
			}
		})
	}
}

func TestReadError(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        RespErrorValue
		wantErr     bool
		errContains string
	}{
		{
			name:    "simple error message",
			input:   "ERR unknown command\r\n",
			want:    RespErrorValue{Message: "ERR unknown command"},
			wantErr: false,
		},
		{
			name:    "error with special characters",
			input:   "WRONGTYPE Operation against a key holding the wrong kind of value\r\n",
			want:    RespErrorValue{Message: "WRONGTYPE Operation against a key holding the wrong kind of value"},
			wantErr: false,
		},
		{
			name:    "empty error message",
			input:   "\r\n",
			want:    RespErrorValue{Message: ""},
			wantErr: false,
		},
		{
			name:    "error with numbers",
			input:   "ERR404\r\n",
			want:    RespErrorValue{Message: "ERR404"},
			wantErr: false,
		},
		{
			name:        "error with incorrect terminator",
			input:       "ERR test\n\n",
			want:        RespErrorValue{},
			wantErr:     true,
			errContains: "not terminated properly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := ReadError(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ReadError() error = %v, should contain %q", err, tt.errContains)
				}
			}
			if !tt.wantErr && got.Message != tt.want.Message {
				t.Errorf("ReadError() = %v, want %v", got.Message, tt.want.Message)
			}
		})
	}
}

func TestReadInteger(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        RespInteger
		wantErr     bool
		errContains string
	}{
		{
			name:    "positive integer",
			input:   "42\r\n",
			want:    RespInteger{Value: 42},
			wantErr: false,
		},
		{
			name:    "negative integer",
			input:   "-100\r\n",
			want:    RespInteger{Value: -100},
			wantErr: false,
		},
		{
			name:    "zero",
			input:   "0\r\n",
			want:    RespInteger{Value: 0},
			wantErr: false,
		},
		{
			name:    "large positive integer",
			input:   "9223372036854775807\r\n",
			want:    RespInteger{Value: 9223372036854775807},
			wantErr: false,
		},
		{
			name:    "large negative integer",
			input:   "-9223372036854775808\r\n",
			want:    RespInteger{Value: -9223372036854775808},
			wantErr: false,
		},
		{
			name:        "invalid integer - non-numeric",
			input:       "abc\r\n",
			want:        RespInteger{},
			wantErr:     true,
			errContains: "invalid integer",
		},
		{
			name:        "invalid integer - with spaces",
			input:       "12 34\r\n",
			want:        RespInteger{},
			wantErr:     true,
			errContains: "invalid integer",
		},
		{
			name:        "integer with incorrect terminator",
			input:       "42\n\n",
			want:        RespInteger{},
			wantErr:     true,
			errContains: "not terminated properly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := ReadInteger(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadInteger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ReadInteger() error = %v, should contain %q", err, tt.errContains)
				}
			}
			if !tt.wantErr && got.Value != tt.want.Value {
				t.Errorf("ReadInteger() = %v, want %v", got.Value, tt.want.Value)
			}
		})
	}
}

func TestReadBulkString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        RespBulkString
		wantErr     bool
		errContains string
	}{
		{
			name:    "simple bulk string",
			input:   "5\r\nhello\r\n",
			want:    RespBulkString{Value: []byte("hello")},
			wantErr: false,
		},
		{
			name:    "empty bulk string",
			input:   "0\r\n\r\n",
			want:    RespBulkString{Value: []byte{}},
			wantErr: false,
		},
		{
			name:    "bulk string with special characters",
			input:   "13\r\nhello\nworld\r\n\r\n",
			want:    RespBulkString{Value: []byte("hello\nworld\r\n")},
			wantErr: false,
		},
		{
			name:    "null bulk string",
			input:   "-1\r\n",
			want:    RespBulkString{Value: nil},
			wantErr: false,
		},
		{
			name:    "bulk string with binary data",
			input:   "3\r\n\x00\x01\x02\r\n",
			want:    RespBulkString{Value: []byte{0x00, 0x01, 0x02}},
			wantErr: false,
		},
		{
			name:        "bulk string with incorrect terminator",
			input:       "5\r\nhello\n\n",
			want:        RespBulkString{},
			wantErr:     true,
			errContains: "not terminated properly",
		},
		{
			name:        "bulk string too short",
			input:       "10\r\nhello\r\n",
			want:        RespBulkString{},
			wantErr:     true,
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := ReadBulkString(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadBulkString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ReadBulkString() error = %v, should contain %q", err, tt.errContains)
				}
			}
			if !tt.wantErr {
				if tt.want.Value == nil {
					if got.Value != nil {
						t.Errorf("ReadBulkString() = %v, want nil", got.Value)
					}
				} else if !bytes.Equal(got.Value, tt.want.Value) {
					t.Errorf("ReadBulkString() = %v, want %v", got.Value, tt.want.Value)
				}
			}
		})
	}
}

func TestReadArray(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantLen     int
		validate    func(t *testing.T, arr RespArray)
		wantErr     bool
		errContains string
	}{
		{
			name:    "empty array",
			input:   "0\r\n",
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "array with one bulk string",
			input:   "1\r\n$5\r\nhello\r\n",
			wantLen: 1,
			validate: func(t *testing.T, arr RespArray) {
				bs, ok := arr.Elements[0].(RespBulkString)
				if !ok {
					t.Errorf("Expected RespBulkString, got %T", arr.Elements[0])
					return
				}
				if !bytes.Equal(bs.Value, []byte("hello")) {
					t.Errorf("Expected 'hello', got %s", bs.Value)
				}
			},
			wantErr: false,
		},
		{
			name:    "array with multiple bulk strings",
			input:   "3\r\n$3\r\nfoo\r\n$3\r\nbar\r\n$3\r\nbaz\r\n",
			wantLen: 3,
			validate: func(t *testing.T, arr RespArray) {
				expected := []string{"foo", "bar", "baz"}
				for i, exp := range expected {
					bs, ok := arr.Elements[i].(RespBulkString)
					if !ok {
						t.Errorf("Element %d: Expected RespBulkString, got %T", i, arr.Elements[i])
						continue
					}
					if !bytes.Equal(bs.Value, []byte(exp)) {
						t.Errorf("Element %d: Expected %q, got %s", i, exp, bs.Value)
					}
				}
			},
			wantErr: false,
		},
		{
			name:    "nested array",
			input:   "2\r\n*1\r\n$5\r\nhello\r\n*1\r\n$5\r\nworld\r\n",
			wantLen: 2,
			validate: func(t *testing.T, arr RespArray) {
				// First element should be an array
				nestedArr1, ok := arr.Elements[0].(RespArray)
				if !ok {
					t.Errorf("First element: Expected RespArray, got %T", arr.Elements[0])
					return
				}
				if len(nestedArr1.Elements) != 1 {
					t.Errorf("First nested array: Expected 1 element, got %d", len(nestedArr1.Elements))
					return
				}
				bs1, ok := nestedArr1.Elements[0].(RespBulkString)
				if !ok {
					t.Errorf("First nested array element: Expected RespBulkString, got %T", nestedArr1.Elements[0])
					return
				}
				if !bytes.Equal(bs1.Value, []byte("hello")) {
					t.Errorf("First nested array: Expected 'hello', got %s", bs1.Value)
				}

				// Second element should be an array
				nestedArr2, ok := arr.Elements[1].(RespArray)
				if !ok {
					t.Errorf("Second element: Expected RespArray, got %T", arr.Elements[1])
					return
				}
				if len(nestedArr2.Elements) != 1 {
					t.Errorf("Second nested array: Expected 1 element, got %d", len(nestedArr2.Elements))
					return
				}
				bs2, ok := nestedArr2.Elements[0].(RespBulkString)
				if !ok {
					t.Errorf("Second nested array element: Expected RespBulkString, got %T", nestedArr2.Elements[0])
					return
				}
				if !bytes.Equal(bs2.Value, []byte("world")) {
					t.Errorf("Second nested array: Expected 'world', got %s", bs2.Value)
				}
			},
			wantErr: false,
		},
		{
			name:    "array with null bulk string",
			input:   "2\r\n$5\r\nhello\r\n$-1\r\n",
			wantLen: 2,
			validate: func(t *testing.T, arr RespArray) {
				bs1, ok := arr.Elements[0].(RespBulkString)
				if !ok {
					t.Errorf("First element: Expected RespBulkString, got %T", arr.Elements[0])
					return
				}
				if !bytes.Equal(bs1.Value, []byte("hello")) {
					t.Errorf("First element: Expected 'hello', got %s", bs1.Value)
				}

				bs2, ok := arr.Elements[1].(RespBulkString)
				if !ok {
					t.Errorf("Second element: Expected RespBulkString, got %T", arr.Elements[1])
					return
				}
				if bs2.Value != nil {
					t.Errorf("Second element: Expected nil, got %v", bs2.Value)
				}
			},
			wantErr: false,
		},
		{
			name:        "invalid array count",
			input:       "abc\r\n",
			wantErr:     true,
			errContains: "invalid length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := ReadArray(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadArray() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ReadArray() error = %v, should contain %q", err, tt.errContains)
				}
				return
			}
			if !tt.wantErr {
				if len(got.Elements) != tt.wantLen {
					t.Errorf("ReadArray() length = %d, want %d", len(got.Elements), tt.wantLen)
					return
				}
				if tt.validate != nil {
					tt.validate(t, got)
				}
			}
		})
	}
}

func TestReadRESP(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantType    string
		validate    func(t *testing.T, val RespValue)
		wantErr     bool
		errContains string
	}{
		{
			name:     "bulk string",
			input:    "$5\r\nhello\r\n",
			wantType: "RespBulkString",
			validate: func(t *testing.T, val RespValue) {
				bs, ok := val.(RespBulkString)
				if !ok {
					t.Errorf("Expected RespBulkString, got %T", val)
					return
				}
				if !bytes.Equal(bs.Value, []byte("hello")) {
					t.Errorf("Expected 'hello', got %s", bs.Value)
				}
			},
			wantErr: false,
		},
		{
			name:     "simple string",
			input:    "+OK\r\n",
			wantType: "RespSimpleString",
			validate: func(t *testing.T, val RespValue) {
				ss, ok := val.(RespSimpleString)
				if !ok {
					t.Errorf("Expected RespSimpleString, got %T", val)
					return
				}
				if ss.Value != "OK" {
					t.Errorf("Expected 'OK', got %s", ss.Value)
				}
			},
			wantErr: false,
		},
		{
			name:     "error",
			input:    "-ERR unknown command\r\n",
			wantType: "RespErrorValue",
			validate: func(t *testing.T, val RespValue) {
				err, ok := val.(RespErrorValue)
				if !ok {
					t.Errorf("Expected RespErrorValue, got %T", val)
					return
				}
				if err.Message != "ERR unknown command" {
					t.Errorf("Expected 'ERR unknown command', got %s", err.Message)
				}
			},
			wantErr: false,
		},
		{
			name:     "integer positive",
			input:    ":1000\r\n",
			wantType: "RespInteger",
			validate: func(t *testing.T, val RespValue) {
				intVal, ok := val.(RespInteger)
				if !ok {
					t.Errorf("Expected RespInteger, got %T", val)
					return
				}
				if intVal.Value != 1000 {
					t.Errorf("Expected 1000, got %d", intVal.Value)
				}
			},
			wantErr: false,
		},
		{
			name:     "integer negative",
			input:    ":-42\r\n",
			wantType: "RespInteger",
			validate: func(t *testing.T, val RespValue) {
				intVal, ok := val.(RespInteger)
				if !ok {
					t.Errorf("Expected RespInteger, got %T", val)
					return
				}
				if intVal.Value != -42 {
					t.Errorf("Expected -42, got %d", intVal.Value)
				}
			},
			wantErr: false,
		},
		{
			name:     "array",
			input:    "*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n",
			wantType: "RespArray",
			validate: func(t *testing.T, val RespValue) {
				arr, ok := val.(RespArray)
				if !ok {
					t.Errorf("Expected RespArray, got %T", val)
					return
				}
				if len(arr.Elements) != 2 {
					t.Errorf("Expected 2 elements, got %d", len(arr.Elements))
					return
				}
				bs1, ok := arr.Elements[0].(RespBulkString)
				if !ok || !bytes.Equal(bs1.Value, []byte("foo")) {
					t.Errorf("First element: Expected 'foo', got %v", arr.Elements[0])
				}
				bs2, ok := arr.Elements[1].(RespBulkString)
				if !ok || !bytes.Equal(bs2.Value, []byte("bar")) {
					t.Errorf("Second element: Expected 'bar', got %v", arr.Elements[1])
				}
			},
			wantErr: false,
		},
		{
			name:     "empty array",
			input:    "*0\r\n",
			wantType: "RespArray",
			validate: func(t *testing.T, val RespValue) {
				arr, ok := val.(RespArray)
				if !ok {
					t.Errorf("Expected RespArray, got %T", val)
					return
				}
				if len(arr.Elements) != 0 {
					t.Errorf("Expected 0 elements, got %d", len(arr.Elements))
				}
			},
			wantErr: false,
		},
		{
			name:     "null bulk string",
			input:    "$-1\r\n",
			wantType: "RespBulkString",
			validate: func(t *testing.T, val RespValue) {
				bs, ok := val.(RespBulkString)
				if !ok {
					t.Errorf("Expected RespBulkString, got %T", val)
					return
				}
				if bs.Value != nil {
					t.Errorf("Expected nil value, got %v", bs.Value)
				}
			},
			wantErr: false,
		},
		{
			name:        "unknown prefix",
			input:       "@invalid\r\n",
			wantErr:     true,
			errContains: "unknown RESP type prefix",
		},
		{
			name:        "invalid prefix character",
			input:       "#invalid\r\n",
			wantErr:     true,
			errContains: "unknown RESP type prefix",
		},
		{
			name:     "complex nested structure",
			input:    "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n*2\r\n$5\r\nvalue\r\n$4\r\ntest\r\n",
			wantType: "RespArray",
			validate: func(t *testing.T, val RespValue) {
				arr, ok := val.(RespArray)
				if !ok {
					t.Errorf("Expected RespArray, got %T", val)
					return
				}
				if len(arr.Elements) != 3 {
					t.Errorf("Expected 3 elements, got %d", len(arr.Elements))
					return
				}
				// Validate first element (SET)
				bs1, ok := arr.Elements[0].(RespBulkString)
				if !ok || !bytes.Equal(bs1.Value, []byte("SET")) {
					t.Errorf("First element: Expected 'SET', got %v", arr.Elements[0])
				}
				// Validate second element (key)
				bs2, ok := arr.Elements[1].(RespBulkString)
				if !ok || !bytes.Equal(bs2.Value, []byte("key")) {
					t.Errorf("Second element: Expected 'key', got %v", arr.Elements[1])
				}
				// Validate third element (nested array)
				nestedArr, ok := arr.Elements[2].(RespArray)
				if !ok {
					t.Errorf("Third element: Expected RespArray, got %T", arr.Elements[2])
					return
				}
				if len(nestedArr.Elements) != 2 {
					t.Errorf("Nested array: Expected 2 elements, got %d", len(nestedArr.Elements))
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := ReadRESP(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadRESP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ReadRESP() error = %v, should contain %q", err, tt.errContains)
				}
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}
