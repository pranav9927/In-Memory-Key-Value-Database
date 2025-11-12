package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Command struct {
	cmd  string
	args []string
}
type ClientConn struct {
	conn             net.Conn
	transactionQueue []Command
	isTransaction    bool
}

func parseCmd(r RespData) (Command, error) {
	if r.Type == SimpleString && r.Str == "PING" {
		return Command{cmd: "PING", args: nil}, nil
	}
	if r.Type != Array {
		return Command{}, fmt.Errorf("expected array, got %v", r.Type)
	}

	if len(r.Array) < 1 {
		return Command{}, fmt.Errorf("expected at least one element in array")
	}

	// Validate command name is a bulk string
	if r.Array[0].Type != BulkString {
		return Command{}, fmt.Errorf("expected bulk string for command, got %v", r.Array[0].Type)
	}
	cmd := r.Array[0].Str
	args := make([]string, len(r.Array)-1)
	for i, item := range r.Array[1:] {
		if item.Type != BulkString {
			return Command{}, fmt.Errorf("expected bulk string, got %v", item.Type)
		}
		args[i] = item.Str
	}

	return Command{cmd: cmd, args: args}, nil
}

// executeCommand handles the command logic and returns RespData
func executeCommand(cmd Command, clientConn *ClientConn, context bool) RespData {
	switch strings.ToLower(cmd.cmd) {
	case "multi":
		return handleMultiCommand(cmd, clientConn)
	case "exec":
		return handleExecCommand(clientConn)
	case "discard":
		return handleDiscardCommand(clientConn)

	}
	if clientConn.isTransaction && !context {
		clientConn.transactionQueue = append(clientConn.transactionQueue, cmd)
		return RespData{Type: SimpleString, Str: "QUEUED"}
	}

	switch strings.ToLower(cmd.cmd) {
	case "ping":
		return RespData{Type: SimpleString, Str: "PONG"}

	case "echo":
		return RespData{Type: BulkString, Str: cmd.args[0]}

	case "set":
		return handleSetCommand(cmd)
	case "delete":
		return handleDeleteCommand(cmd)

	case "get":
		return handleGetCommand(cmd)

	case "save":
		if err := db.SaveRDB(); err != nil {
			return RespData{Type: Error, Str: fmt.Sprintf("ERR %v", err)}
		}
		return RespData{Type: SimpleString, Str: "OK"}

	case "config":
		return handleConfigCommand(cmd)

	case "keys":
		return handleKeysCommand(cmd)

	case "info":
		// Minimal INFO without replication details
		return RespData{Type: BulkString, Str: "role:master"}

	case "incr":
		return handleIncrCommand(cmd)

	case "lpush":
		return handleLPushCommand(cmd)
	case "rpush":
		return handleRPushCommand(cmd)
	case "lpop":
		return handleLPopCommand(cmd)
	case "rpop":
		return handleRPopCommand(cmd)
	case "llen":
		return handleLLenCommand(cmd)
	case "lrange":
		return handleLRangeCommand(cmd)
	case "type":
		return handleTypeCommand(cmd)
	case "xadd":
		return handleXAddCommand(cmd)
	case "xlen":
		return handleXLenCommand(cmd)
	case "xrange":
		return handleXRangeCommand(cmd)
	case "xread":
		return handleXReadCommand(cmd)

	default:
		return RespData{Type: Error, Str: "ERR unknown command '" + cmd.cmd + "'"}
	}
}

// handleCommand executes the command and writes the result
func handleCommand(cmd Command, r *RESPreader, clientConn *ClientConn) {
	result := executeCommand(cmd, clientConn, false)

	// Handle special cases that need additional operations after getting result
	switch strings.ToLower(cmd.cmd) {
	case "config":
		if strings.ToLower(cmd.args[0]) == "get" {
			handleConfigGetResponse(cmd, r)
			return
		}
	}

	// Standard response writing
	if result.Type == Array && len(result.Array) > 0 {
		r.WriteArray(result.Array)
	} else {
		r.Write(result)
	}

	// Replication removed: no command propagation
}

func handleTypeCommand(cmd Command) RespData {
	if len(cmd.args) != 1 {
		return RespData{Type: Error, Str: "ERR wrong number of arguments for 'type' command"}
	}

	val := db.GetType(cmd.args[0])
	if val == nil {
		return RespData{Type: SimpleString, Str: "none"}
	}

	return RespData{Type: SimpleString, Str: *val}
}

// Helper functions for individual command logic
func handleSetCommand(cmd Command) RespData {
	if len(cmd.args) == 2 {
		db.Add(cmd.args[0], cmd.args[1])
		return RespData{Type: SimpleString, Str: "OK"}
	} else if len(cmd.args) == 4 && strings.ToLower(cmd.args[2]) == "px" {
		num, err := strconv.Atoi(cmd.args[3])
		if err != nil {
			return RespData{Type: Error, Str: "wrong numeric value for expiry"}
		}
		db.Addex(cmd.args[0], cmd.args[1], int64(num))
		return RespData{Type: SimpleString, Str: "OK"}
	}
	return RespData{Type: Error, Str: "ERR wrong number of arguments for 'set' command"}
}

func handleGetCommand(cmd Command) RespData {
	if len(cmd.args) != 1 {
		return RespData{Type: Error, Str: "ERR wrong number of arguments for 'get' command"}
	}

	val := db.Get(cmd.args[0])
	if val == nil {
		return RespData{Type: BulkString, IsNull: true}
	}
	return RespData{Type: BulkString, Str: *val}
}

func handleConfigCommand(cmd Command) RespData {
	if len(cmd.args) < 2 {
		return RespData{Type: Error, Str: "ERR wrong number of arguments for config command"}
	}

	switch strings.ToLower(cmd.args[0]) {
	case "get":
		return handleConfigGet(cmd.args[1])
	case "set":
		return handleConfigSet(cmd.args[1], cmd.args[2])
	default:
		return RespData{Type: Error, Str: "ERR unknown config subcommand"}
	}
}

func handleConfigGet(param string) RespData {
	switch param {
	case "dir":
		return RespData{
			Type: Array,
			Array: []RespData{
				{Type: BulkString, Str: "dir"},
				{Type: BulkString, Str: db.dir},
			},
		}
	case "dbfilename":
		return RespData{
			Type: Array,
			Array: []RespData{
				{Type: BulkString, Str: "dbfilename"},
				{Type: BulkString, Str: db.dbfilename},
			},
		}
	case "port":
		return RespData{
			Type: Array,
			Array: []RespData{
				{Type: BulkString, Str: "port"},
				{Type: BulkString, Str: db.port},
			},
		}
	default:
		return RespData{Type: Array, IsNull: true}
	}
}

func handleConfigSet(param, value string) RespData {
	switch param {
	case "dir":
		db.dir = value
		return RespData{Type: SimpleString, Str: "OK"}
	case "dbfilename":
		db.dbfilename = value
		return RespData{Type: SimpleString, Str: "OK"}
	default:
		return RespData{Type: Error, Str: "ERR unsupported config parameter"}
	}
}

func handleKeysCommand(cmd Command) RespData {
	if len(cmd.args) != 1 {
		return RespData{Type: Error, Str: "ERR wrong number of arguments for 'keys' command"}
	}

	keys := make([]RespData, 0, len(db.M))
	for k := range db.M {
		fmt.Println(k)
		keys = append(keys, RespData{Type: BulkString, Str: k})
	}

	if len(keys) == 0 {
		return RespData{Type: Array, IsNull: true}
	}
	return RespData{Type: Array, Array: keys}
}

func handleIncrCommand(cmd Command) RespData {
	if len(cmd.args) != 1 {
		return RespData{Type: Error, Str: "ERR wrong number of arguments for 'incr' command"}
	}

	err := db.Incr(cmd.args[0])
	if err != nil {
		return RespData{Type: Error, Str: err.Error()}
	}
	return RespData{Type: SimpleString, Str: "OK"}
}

func handleConfigGetResponse(cmd Command, r *RESPreader) {
	result := handleConfigGet(cmd.args[1])
	if result.Type == Array {
		r.WriteArray(result.Array)
	} else {
		r.Write(result)
	}
}

// replication-specific slave handlers removed

func handleDeleteCommand(cmd Command) RespData {
	if len(cmd.args) != 1 {
		return RespData{Type: Error, Str: "ERR wrong number of arguments for 'delete' command"}
	}

	db.Delete(cmd.args[0])

	return RespData{Type: SimpleString, Str: "OK"}
}
