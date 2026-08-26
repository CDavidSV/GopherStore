package server

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CDavidSV/GopherStore/internal/resp"
)

func bulkCommand(values ...string) resp.RespArray {
	elements := make([]resp.RespValue, len(values))
	for i, value := range values {
		elements[i] = resp.RespBulkString{Value: []byte(value)}
	}
	return resp.RespArray{Elements: elements}
}

func TestParseCommand(t *testing.T) {
	second := 2 * time.Second
	millisecond := 2 * time.Millisecond

	tests := []struct {
		name    string
		input   resp.RespArray
		want    Command
		errText string
	}{
		{name: "PING", input: bulkCommand("PING"), want: PingCommand{}},
		{name: "PING with message", input: bulkCommand("PING", "hello"), want: PingCommand{Value: "hello"}},
		{name: "SET", input: bulkCommand("SET", "key", "value"), want: SetCommand{Key: []byte("key"), Value: []byte("value")}},
		{name: "SET NX", input: bulkCommand("SET", "key", "value", "NX"), want: SetCommand{Key: []byte("key"), Value: []byte("value"), condition: ConditionNX}},
		{name: "SET XX", input: bulkCommand("SET", "key", "value", "XX"), want: SetCommand{Key: []byte("key"), Value: []byte("value"), condition: ConditionXX}},
		{name: "SET EX", input: bulkCommand("SET", "key", "value", "EX", "2"), want: SetCommand{Key: []byte("key"), Value: []byte("value"), expiration: &second}},
		{name: "SET PX", input: bulkCommand("SET", "key", "value", "PX", "2"), want: SetCommand{Key: []byte("key"), Value: []byte("value"), expiration: &millisecond}},
		{name: "GET", input: bulkCommand("GET", "key"), want: GetCommand{Key: []byte("key")}},
		{name: "DEL multiple", input: bulkCommand("DEL", "one", "two"), want: DeleteCommand{Keys: [][]byte{[]byte("one"), []byte("two")}}},
		{name: "EXISTS multiple", input: bulkCommand("EXISTS", "one", "two"), want: ExistsCommand{Keys: [][]byte{[]byte("one"), []byte("two")}}},
		{name: "EXPIRE", input: bulkCommand("EXPIRE", "key", "2"), want: ExpireCommand{Key: []byte("key"), TTL: 2 * time.Second}},
		{name: "PEXPIRE", input: bulkCommand("PEXPIRE", "key", "2"), want: ExpireCommand{Key: []byte("key"), TTL: 2 * time.Millisecond}},
		{name: "LPUSH", input: bulkCommand("LPUSH", "list", "one", "two"), want: PushCommand{Key: []byte("list"), Vals: [][]byte{[]byte("one"), []byte("two")}, pushAtFront: true}},
		{name: "RPUSH", input: bulkCommand("RPUSH", "list", "one"), want: PushCommand{Key: []byte("list"), Vals: [][]byte{[]byte("one")}}},
		{name: "LPOP", input: bulkCommand("LPOP", "list"), want: PopCommand{Key: []byte("list"), popAtFront: true}},
		{name: "RPOP", input: bulkCommand("RPOP", "list"), want: PopCommand{Key: []byte("list")}},
		{name: "LLEN", input: bulkCommand("LLEN", "list"), want: LLenCommand{Key: []byte("list")}},
		{name: "LRANGE", input: bulkCommand("LRANGE", "list", "-2", "-1"), want: LRangeCommand{Key: []byte("list"), Start: -2, End: -1}},
		{name: "empty command", input: resp.RespArray{}, errText: "empty command array"},
		{name: "unknown command", input: bulkCommand("NOPE"), errText: "unknown command"},
		{name: "SET missing value", input: bulkCommand("SET", "key"), errText: "at least 2 arguments"},
		{name: "SET duplicate conditions", input: bulkCommand("SET", "key", "value", "NX", "XX"), errText: "only have one condition"},
		{name: "SET missing EX value", input: bulkCommand("SET", "key", "value", "EX"), errText: "requires an expiration time"},
		{name: "SET invalid expiration", input: bulkCommand("SET", "key", "value", "PX", "bad"), errText: "invalid expiration time"},
		{name: "GET missing key", input: bulkCommand("GET"), errText: "exactly 1 argument"},
		{name: "EXPIRE invalid TTL", input: bulkCommand("EXPIRE", "key", "bad"), errText: "invalid TTL value"},
		{name: "LPUSH missing values", input: bulkCommand("LPUSH", "list"), errText: "at least 2 arguments"},
		{name: "LRANGE invalid start", input: bulkCommand("LRANGE", "list", "bad", "1"), errText: "invalid start index"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCommand(tt.input)
			if tt.errText != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errText) {
					t.Fatalf("ParseCommand() error = %v, want error containing %q", err, tt.errText)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCommand() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseCommand() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
