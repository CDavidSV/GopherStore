package resp

import (
	"bufio"
	"bytes"
	"testing"
)

func TestEncodeSimpleString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []byte
	}{
		{
			name:  "simple OK",
			input: "OK",
			want:  []byte("+OK\r\n"),
		},
		{
			name:  "simple string with text",
			input: "hello world",
			want:  []byte("+hello world\r\n"),
		},
		{
			name:  "empty string",
			input: "",
			want:  []byte("+\r\n"),
		},
		{
			name:  "string with numbers",
			input: "12345",
			want:  []byte("+12345\r\n"),
		},
		{
			name:  "string with special characters",
			input: "test!@#$%",
			want:  []byte("+test!@#$%\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeSimpleString(tt.input)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("EncodeSimpleString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEncodeError(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []byte
	}{
		{
			name:  "simple error",
			input: "ERR unknown command",
			want:  []byte("-ERR unknown command\r\n"),
		},
		{
			name:  "error with type prefix",
			input: "WRONGTYPE Operation against a key holding the wrong kind of value",
			want:  []byte("-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"),
		},
		{
			name:  "empty error",
			input: "",
			want:  []byte("-\r\n"),
		},
		{
			name:  "error with numbers",
			input: "ERR404",
			want:  []byte("-ERR404\r\n"),
		},
		{
			name:  "syntax error",
			input: "ERR syntax error",
			want:  []byte("-ERR syntax error\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeError(tt.input)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("EncodeError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEncodeBulkString(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  []byte
	}{
		{
			name:  "simple bulk string",
			input: []byte("hello"),
			want:  []byte("$5\r\nhello\r\n"),
		},
		{
			name:  "empty bulk string",
			input: []byte(""),
			want:  []byte("$0\r\n\r\n"),
		},
		{
			name:  "null bulk string",
			input: nil,
			want:  []byte("$-1\r\n"),
		},
		{
			name:  "bulk string with newlines",
			input: []byte("hello\nworld"),
			want:  []byte("$11\r\nhello\nworld\r\n"),
		},
		{
			name:  "bulk string with special characters",
			input: []byte("hello\r\nworld"),
			want:  []byte("$12\r\nhello\r\nworld\r\n"),
		},
		{
			name:  "bulk string with binary data",
			input: []byte{0x00, 0x01, 0x02, 0xFF},
			want:  []byte("$4\r\n\x00\x01\x02\xFF\r\n"),
		},
		{
			name:  "long bulk string",
			input: []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit"),
			want:  []byte("$55\r\nLorem ipsum dolor sit amet, consectetur adipiscing elit\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeBulkString(tt.input)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("EncodeBulkString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEncodeInteger(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  []byte
	}{
		{
			name:  "positive integer",
			input: 42,
			want:  []byte(":42\r\n"),
		},
		{
			name:  "negative integer",
			input: -100,
			want:  []byte(":-100\r\n"),
		},
		{
			name:  "zero",
			input: 0,
			want:  []byte(":0\r\n"),
		},
		{
			name:  "large positive integer",
			input: 9223372036854775807,
			want:  []byte(":9223372036854775807\r\n"),
		},
		{
			name:  "large negative integer",
			input: -9223372036854775808,
			want:  []byte(":-9223372036854775808\r\n"),
		},
		{
			name:  "small positive integer",
			input: 1,
			want:  []byte(":1\r\n"),
		},
		{
			name:  "small negative integer",
			input: -1,
			want:  []byte(":-1\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeInteger(tt.input)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("EncodeInteger() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEncodeBulkStringArray(t *testing.T) {
	tests := []struct {
		name  string
		input [][]byte
		want  []byte
	}{
		{
			name:  "empty array",
			input: [][]byte{},
			want:  []byte("*0\r\n"),
		},
		{
			name:  "null array",
			input: nil,
			want:  []byte("*-1\r\n"),
		},
		{
			name: "array with one element",
			input: [][]byte{
				[]byte("hello"),
			},
			want: []byte("*1\r\n$5\r\nhello\r\n"),
		},
		{
			name: "array with multiple elements",
			input: [][]byte{
				[]byte("foo"),
				[]byte("bar"),
				[]byte("baz"),
			},
			want: []byte("*3\r\n$3\r\nfoo\r\n$3\r\nbar\r\n$3\r\nbaz\r\n"),
		},
		{
			name: "array with empty string",
			input: [][]byte{
				[]byte("hello"),
				[]byte(""),
				[]byte("world"),
			},
			want: []byte("*3\r\n$5\r\nhello\r\n$0\r\n\r\n$5\r\nworld\r\n"),
		},
		{
			name: "array with nil element",
			input: [][]byte{
				[]byte("hello"),
				nil,
				[]byte("world"),
			},
			want: []byte("*3\r\n$5\r\nhello\r\n$-1\r\n$5\r\nworld\r\n"),
		},
		{
			name: "array with binary data",
			input: [][]byte{
				{0x00, 0x01},
				{0xFF, 0xFE},
			},
			want: []byte("*2\r\n$2\r\n\x00\x01\r\n$2\r\n\xFF\xFE\r\n"),
		},
		{
			name: "array representing PING command",
			input: [][]byte{
				[]byte("PING"),
			},
			want: []byte("*1\r\n$4\r\nPING\r\n"),
		},
		{
			name: "array representing SET command",
			input: [][]byte{
				[]byte("SET"),
				[]byte("key"),
				[]byte("value"),
			},
			want: []byte("*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"),
		},
		{
			name: "array representing GET command",
			input: [][]byte{
				[]byte("GET"),
				[]byte("mykey"),
			},
			want: []byte("*2\r\n$3\r\nGET\r\n$5\r\nmykey\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeBulkStringArray(tt.input)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("EncodeBulkStringArray() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRoundTrip tests encoding and then decoding to ensure data integrity
func TestRoundTrip(t *testing.T) {
	t.Run("bulk string round trip", func(t *testing.T) {
		original := []byte("hello world")
		encoded := EncodeBulkString(original)

		// Skip the prefix byte and decode
		reader := bufio.NewReader(bytes.NewReader(encoded[1:]))
		decoded, err := ReadBulkString(reader)
		if err != nil {
			t.Fatalf("ReadBulkString() error = %v", err)
		}

		if !bytes.Equal(decoded.Value, original) {
			t.Errorf("Round trip failed: got %q, want %q", decoded.Value, original)
		}
	})

	t.Run("null bulk string round trip", func(t *testing.T) {
		encoded := EncodeBulkString(nil)

		reader := bufio.NewReader(bytes.NewReader(encoded[1:]))
		decoded, err := ReadBulkString(reader)
		if err != nil {
			t.Fatalf("ReadBulkString() error = %v", err)
		}

		if decoded.Value != nil {
			t.Errorf("Round trip failed: got %v, want nil", decoded.Value)
		}
	})

	t.Run("integer round trip", func(t *testing.T) {
		original := int64(12345)
		encoded := EncodeInteger(original)

		reader := bufio.NewReader(bytes.NewReader(encoded[1:]))
		decoded, err := ReadInteger(reader)
		if err != nil {
			t.Fatalf("ReadInteger() error = %v", err)
		}

		if decoded.Value != original {
			t.Errorf("Round trip failed: got %d, want %d", decoded.Value, original)
		}
	})

	t.Run("simple string round trip", func(t *testing.T) {
		original := "OK"
		encoded := EncodeSimpleString(original)

		reader := bufio.NewReader(bytes.NewReader(encoded[1:]))
		decoded, err := ReadSimpleString(reader)
		if err != nil {
			t.Fatalf("ReadSimpleString() error = %v", err)
		}

		if decoded.Value != original {
			t.Errorf("Round trip failed: got %q, want %q", decoded.Value, original)
		}
	})

	t.Run("error round trip", func(t *testing.T) {
		original := "ERR unknown command"
		encoded := EncodeError(original)

		reader := bufio.NewReader(bytes.NewReader(encoded[1:]))
		decoded, err := ReadError(reader)
		if err != nil {
			t.Fatalf("ReadError() error = %v", err)
		}

		if decoded.Message != original {
			t.Errorf("Round trip failed: got %q, want %q", decoded.Message, original)
		}
	})
}
