package server

import (
	"fmt"

	"github.com/CDavidSV/GopherStore/internal/resp"
)

type CommandName string

const (
	CmdPing CommandName = "PING"
	CmdSet  CommandName = "SET"
	CmdGet  CommandName = "GET"
)

type Command interface{}

type SetCommand struct {
	Key, Value string
}

type GetCommand struct {
	Key string
}

func validateSetCommand(arr resp.RespArray) (Command, error) {
	if len(arr.Elements) != 3 {
		return nil, fmt.Errorf("SET command requires exactly 2 arguments")
	}

	key, ok := arr.Elements[1].(resp.RespBulkString)
	if !ok {
		return nil, fmt.Errorf("invalid SET command format: expected bulk string for key")
	}

	value, ok := arr.Elements[2].(resp.RespBulkString)
	if !ok {
		return nil, fmt.Errorf("invalid SET command format: expected bulk string for value")
	}

	return SetCommand{
		Key:   key.Value,
		Value: value.Value,
	}, nil
}

func validateGetCommand(arr resp.RespArray) (Command, error) {
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

func ParseCommand(cmdArray resp.RespArray) (Command, error) {
	command := cmdArray.Elements[0]

	cmdStr, ok := command.(resp.RespBulkString)
	if !ok {
		return nil, fmt.Errorf("invalid command format: expected bulk string for command name")
	}

	switch CommandName(cmdStr.Value) {
	case CmdSet:
		return validateSetCommand(cmdArray)
	case CmdGet:
		return validateGetCommand(cmdArray)
	default:
		return nil, fmt.Errorf("unknown command: %s", cmdStr.Value)
	}

}
