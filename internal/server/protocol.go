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
	CmdPing    CommandName = "PING"
	CmdSet     CommandName = "SET"
	CmdGet     CommandName = "GET"
	CmdLPush   CommandName = "LPUSH"
	CmdRPush   CommandName = "RPUSH"
	CmdLPop    CommandName = "LPOP"
	CmdRPop    CommandName = "RPOP"
	CmdExists  CommandName = "EXISTS"
	CmdDelete  CommandName = "DEL"
	CmdExpire  CommandName = "EXPIRE"
	CmdPExpire CommandName = "PEXPIRE"

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

type ExistsCommand struct {
	Keys [][]byte
}

type GetCommand struct {
	Key []byte
}

type PingCommand struct {
	Value string
}

type ExpireCommand struct {
	Key []byte
	TTL time.Duration
}

type PushCommand struct {
	Key         []byte
	Vals        [][]byte
	pushAtFront bool
}

type PopCommand struct {
	Key        []byte
	popAtFront bool
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

func parseExistsCommand(arr resp.RespArray) (Command, error) {
	if len(arr.Elements) < 2 {
		return nil, fmt.Errorf("EXISTS command requires at least 1 argument")
	}

	keys := make([][]byte, len(arr.Elements)-1)
	for i, elem := range arr.Elements[1:] {
		key, ok := elem.(resp.RespBulkString)
		if !ok {
			return nil, fmt.Errorf("expected bulk strings for keys")
		}
		keys[i] = key.Value
	}

	return ExistsCommand{
		Keys: keys,
	}, nil
}

func parseExpireCommand(arr resp.RespArray) (Command, error) {
	if len(arr.Elements) != 3 {
		return nil, fmt.Errorf("EXPIRE/PEXPIRE command requires exactly 2 arguments")
	}

	key, ok := arr.Elements[1].(resp.RespBulkString)
	if !ok {
		return nil, fmt.Errorf("invalid EXPIRE/PEXPIRE command format: expected bulk string for key")
	}

	ttl, ok := arr.Elements[2].(resp.RespBulkString)
	if !ok {
		return nil, fmt.Errorf("invalid EXPIRE/PEXPIRE command format: expected bulk string for TTL")
	}

	ttlInt, err := util.ParsePositiveInt(ttl.Value)
	if !err {
		return nil, fmt.Errorf("invalid TTL value")
	}

	var duration time.Duration
	if string(arr.Elements[0].(resp.RespBulkString).Value) == "EXPIRE" {
		duration = time.Duration(ttlInt) * time.Second
	} else {
		duration = time.Duration(ttlInt) * time.Millisecond
	}

	return ExpireCommand{
		Key: key.Value,
		TTL: duration,
	}, nil
}

func parsePushCommand(arr resp.RespArray) (Command, error) {
	if len(arr.Elements) < 3 {
		return nil, fmt.Errorf("LPUSH/RPUSH command requires at least 2 arguments")
	}

	key, ok := arr.Elements[1].(resp.RespBulkString)
	if !ok {
		return nil, fmt.Errorf("invalid LPUSH/RPUSH command format: expected bulk string for key")
	}

	values := make([][]byte, len(arr.Elements)-2)
	for i, elem := range arr.Elements[2:] {
		val, ok := elem.(resp.RespBulkString)
		if !ok {
			return nil, fmt.Errorf("invalid LPUSH/RPUSH command format: expected bulk strings for values")
		}
		values[i] = val.Value
	}

	cmd := PushCommand{
		Key:  key.Value,
		Vals: values,
	}

	if string(arr.Elements[0].(resp.RespBulkString).Value) == "LPUSH" {
		cmd.pushAtFront = true
	} else {
		cmd.pushAtFront = false
	}
	return cmd, nil
}

func parsePopCommand(arr resp.RespArray) (Command, error) {
	if len(arr.Elements) != 2 {
		return nil, fmt.Errorf("LPOP/RPOP command requires exactly 1 argument")
	}

	key, ok := arr.Elements[1].(resp.RespBulkString)
	if !ok {
		return nil, fmt.Errorf("invalid LPOP/RPOP command format: expected bulk string for key")
	}

	cmd := PopCommand{
		Key: key.Value,
	}

	if string(arr.Elements[0].(resp.RespBulkString).Value) == "LPOP" {
		cmd.popAtFront = true
	} else {
		cmd.popAtFront = false
	}

	return cmd, nil
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
	case CmdExists:
		return parseExistsCommand(cmdArray)
	case CmdPing:
		return parsePingCommand(cmdArray)
	case CmdExpire, CmdPExpire:
		return parseExpireCommand(cmdArray)
	case CmdLPush, CmdRPush:
		return parsePushCommand(cmdArray)
	case CmdLPop, CmdRPop:
		return parsePopCommand(cmdArray)
	default:
		return nil, fmt.Errorf("unknown command: %s", cmdStr.Value)
	}
}
