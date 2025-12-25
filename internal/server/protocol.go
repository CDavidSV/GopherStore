package server

import (
	"fmt"
	"time"

	"github.com/CDavidSV/GopherStore/internal/resp"
	"github.com/CDavidSV/GopherStore/internal/util"
)

type CommandName string
type SetCondition int

const (
	// Commands
	CmdPing   CommandName = "PING"
	CmdSet    CommandName = "SET"
	CmdGet    CommandName = "GET"
	CmdExists CommandName = "EXISTS"
	CmdDelete CommandName = "DEL"

	// SET command conditions
	ConditionNone SetCondition = iota
	ConditionNX                // Only set if key does not exist
	ConditionXX                // Only set if key exists
)

type Command interface{}

type SetCommand struct {
	Key, Value []byte
	expiration *time.Duration
	condition  SetCondition
}

type DeleteCommand struct {
	Keys [][]byte
}

type GetCommand struct {
	Key []byte
}

type PingCommand struct {
	Value string
}

func parseSetCommand(arr resp.RespArray) (Command, error) {
	if len(arr.Elements) < 3 {
		return nil, fmt.Errorf("SET command requires at least 2 arguments")
	}

	// Convert all elements to expected types
	elements := make([]resp.RespBulkString, len(arr.Elements))
	for i, elem := range arr.Elements {
		elem, ok := elem.(resp.RespBulkString)
		if !ok {
			return nil, fmt.Errorf("invalid SET command format: expected bulk strings")
		}
		elements[i] = elem
	}

	command := SetCommand{
		Key:       elements[1].Value,
		Value:     elements[2].Value,
		condition: ConditionNone,
	}
	if len(arr.Elements) > 3 {
		for i := 3; i < len(elements); i++ {
			option := string(elements[i].Value)

			switch option {
			case "NX":
				if command.condition != ConditionNone {
					return nil, fmt.Errorf("SET command can only have one condition (NX or XX)")
				}
				command.condition = ConditionNX
			case "XX":
				if command.condition != ConditionNone {
					return nil, fmt.Errorf("SET command can only have one condition (NX or XX)")
				}
				command.condition = ConditionXX
			case "EX":
				if i+1 >= len(elements) {
					return nil, fmt.Errorf("SET command EX option requires an expiration time")
				}
				expSec, ok := util.ParsePositiveInt(elements[i+1].Value)
				if !ok {
					return nil, fmt.Errorf("invalid expiration time for SET command")
				}
				expiration := time.Duration(expSec) * time.Second
				command.expiration = &expiration
				i++
			case "PX":
				if i+1 >= len(elements) {
					return nil, fmt.Errorf("SET command PX option requires an expiration time")
				}
				expMs, ok := util.ParsePositiveInt(elements[i+1].Value)
				if !ok {
					return nil, fmt.Errorf("invalid expiration time for SET command")
				}
				expiration := time.Duration(expMs) * time.Millisecond
				command.expiration = &expiration
				i++
			default:
				return nil, fmt.Errorf("unknown option for SET command (%s)", option)
			}
		}
	}

	return command, nil
}

func parseGetCommand(arr resp.RespArray) (Command, error) {
	if len(arr.Elements) != 2 {
		return nil, fmt.Errorf("GET command requires exactly 1 argument")
	}

	key, ok := arr.Elements[1].(resp.RespBulkString)
	if !ok {
		return nil, fmt.Errorf("invalid GET command format: expected bulk string for key")
	}

	return GetCommand{
		Key: key.Value,
	}, nil
}

func parsePingCommand(arr resp.RespArray) (Command, error) {
	if len(arr.Elements) > 2 {
		return nil, fmt.Errorf("PING command accepts at most 1 argument")
	}

	if len(arr.Elements) == 2 {
		value, ok := arr.Elements[1].(resp.RespBulkString)
		if !ok {
			return nil, fmt.Errorf("invalid PING command format: expected bulk string for value")
		}
		return PingCommand{
			Value: string(value.Value),
		}, nil
	}

	return PingCommand{}, nil
}

func parseDeleteCommand(arr resp.RespArray) (Command, error) {
	if len(arr.Elements) < 2 {
		return nil, fmt.Errorf("DEL command requires at least 1 argument")
	}

	keys := make([][]byte, len(arr.Elements)-1)
	for i, elem := range arr.Elements[1:] {
		key, ok := elem.(resp.RespBulkString)
		if !ok {
			return nil, fmt.Errorf("expected bulk strings for keys")
		}
		keys[i] = key.Value
	}

	return DeleteCommand{
		Keys: keys,
	}, nil
}

func ParseCommand(cmdArray resp.RespArray) (Command, error) {
	command := cmdArray.Elements[0]

	cmdStr, ok := command.(resp.RespBulkString)
	if !ok {
		return nil, fmt.Errorf("invalid command format: expected bulk string for command name")
	}

	switch CommandName(cmdStr.Value) {
	case CmdSet:
		return parseSetCommand(cmdArray)
	case CmdGet:
		return parseGetCommand(cmdArray)
	case CmdDelete:
		return parseDeleteCommand(cmdArray)
	case CmdPing:
		return parsePingCommand(cmdArray)
	default:
		return nil, fmt.Errorf("unknown command: %s", cmdStr.Value)
	}

}
