package redis

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"

	"github.com/dolab/objconv"
	"github.com/dolab/objconv/objutil"
)

// A Command represent a Redis command used withing a Request.
type Command struct {
	// Cmd is the Redis command that's being sent with this request.
	Cmd string

	// Args is the list of arguments for the request's command. This field
	// may be nil for client requests if there are no arguments to send with
	// the request.
	//
	// For server request, Args is never nil, even if there are no values in
	// the argument list.
	Args Args

	// for retry
	args [][]byte
	argn int
}

// ParseArgs parses the list of arguments from the command into the destination
// pointers, returning an error if something went wrong.
func (cmd *Command) ParseArgs(dsts ...interface{}) error {
	return ParseArgs(cmd.Args, dsts...)
}

// newCommand returns a command for request reuse.
//
// NOTE: It CANNOT be exported cause it should ensure the command is idempotent, see Response.Retry() for details!
func (cmd *Command) newCommand() Command {
	return Command{
		Cmd:  cmd.Cmd,
		Args: &byteArgs{args: cmd.args},
	}
}

func (cmd *Command) getKeys(keys []string) []string {
	lastIndex := len(keys)
	keys = append(keys, "")

	if cmd.Args != nil {
		// TODO: for now we assume commands have only one key
		if cmd.Args.Next(&keys[lastIndex]) {
			cmd.Args = MultiArgs(List(keys[lastIndex]), cmd.Args)
		} else {
			keys = keys[:lastIndex]
		}
	}

	return keys
}

func (cmd *Command) loadByteArgs() {
	if cmd.Args == nil {
		return
	}

	var (
		list [][]byte
		arg  []byte
	)

	for cmd.Args.Next(&arg) {
		list = append(list, arg)
		arg = nil
	}

	if err := cmd.Args.Close(); err != nil {
		cmd.Args = newArgsError(err)
	} else {
		cmd.Args = &byteArgs{args: list}
	}
}

func (cmd *Command) appendArg(arg []byte) {
	if len(cmd.args) >= cmd.argn {
		cmd.args = append(cmd.args, arg)
	} else {
		cmd.args[cmd.argn] = make([]byte, len(arg))
		copy(cmd.args[cmd.argn], arg)
	}

	cmd.argn++
}

// CommandReader is a type produced by the Conn.ReadCommands method to read a
// single command or a sequence of commands belonging to the same transaction.
type CommandReader struct {
	mutex sync.Mutex
	conn  *Conn
	dec   objconv.StreamDecoder
	multi bool
	done  bool
	retry bool
	err   error
}

// Close closes the command reader, it must be called when all commands have been
// read from the reader in order to release the parent connection's read lock.
func (r *CommandReader) Close() error {
	r.mutex.Lock()
	err := r.err

	if r.conn != nil {
		if !r.done || err != nil {
			r.conn.Close()
		}
		r.conn.rmutex.Unlock()
		r.conn = nil
	}

	r.done = true
	r.mutex.Unlock()
	return err
}

// Read reads the next command from the command reader, filling cmd with the
// name and list of arguments. The command's arguments Close method must be
// called in order to release the reader's lock before any other methods of
// the reader are called.
//
// The method returns true if a command could be read, or false if there were
// no more commands to read from the reader.
func (r *CommandReader) Read(cmd *Command) bool {
	r.mutex.Lock()
	r.resetDecoder()

	if r.done {
		r.mutex.Unlock()
		return false
	}

	*cmd = Command{}

	if err := r.dec.Decode(&cmd.Cmd); err != nil {
		r.err = r.dec.Err()
		r.done = true
		r.mutex.Unlock()
		return false
	}

	if r.multi {
		r.done = r.multi && (cmd.Cmd == "EXEC" || cmd.Cmd == "DISCARD")
	} else {
		r.multi = cmd.Cmd == "MULTI"
		r.done = !r.multi
	}

	cmd.Args = newCmdArgsReader(r, cmd)
	return true
}

// Pipe reads the next command from the command reader, it spec for pipeline feature of
// redis RESP.
func (r *CommandReader) Pipe(cmd *Command) bool {
	if r.multi || !r.done {
		return false
	}

	r.resetReader()

	return r.Read(cmd)
}

func (r *CommandReader) resetReader() {
	r.done = false
}

func (r *CommandReader) resetDecoder() {
	r.dec = objconv.StreamDecoder{Parser: r.dec.Parser}
}

func newCmdArgsReader(r *CommandReader, cmd *Command) *cmdArgsReader {
	if r.retry {
		cmd.args = make([][]byte, 0, r.dec.Len())
	}

	args := &cmdArgsReader{r: r, cmd: cmd}
	args.b = args.a[:0]
	return args
}

type cmdArgsReader struct {
	once sync.Once
	cmd  *Command
	err  error
	r    *CommandReader
	b    []byte
	a    [128]byte
}

func (args *cmdArgsReader) Close() error {
	args.once.Do(func() {
		var err error

		for args.r.dec.Decode(nil) == nil {
			// discard all remaining values
		}

		err = args.r.dec.Err()
		if args.err == nil {
			args.err = err
		}

		// Unlocking the parent command reader allows it to make progress and
		// read the next command.
		if args.r != nil {
			args.r.err = err
			args.r.mutex.Unlock()
		}
	})

	return args.err
}

func (args *cmdArgsReader) Len() int {
	if args.err != nil {
		return 0
	}

	return args.r.dec.Len()
}

func (args *cmdArgsReader) Next(val interface{}) bool {
	if args.err != nil {
		return false
	}

	if args.r.dec.Len() != 0 {
		t, _ := args.r.dec.Parser.ParseType()
		if t == objconv.Error {
			args.r.dec.Decode(&args.err)
			return false
		}
	}

	args.b = args.b[:0]

	if err := args.r.dec.Decode(&args.b); err != nil {
		args.err = args.r.dec.Err()
		return false
	}

	if v := reflect.ValueOf(val); v.IsValid() {
		if err := args.parse(v.Elem()); err != nil {
			args.err = err
			return false
		}

		if args.r.retry {
			args.cmd.appendArg(args.b[:])
		}
	}

	return true
}

func (args *cmdArgsReader) parse(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Bool:
		return args.parseBool(v)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return args.parseInt(v)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return args.parseUint(v)

	case reflect.Float32, reflect.Float64:
		return args.parseFloat(v)

	case reflect.String:
		return args.parseString(v)

	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return args.parseBytes(v)
		}

	case reflect.Interface:
		return args.parseValue(v)
	}

	return fmt.Errorf("unsupported output type for value in argument of a redis command: %s", v.Type())
}

func (args *cmdArgsReader) parseBool(v reflect.Value) error {
	i, err := objutil.ParseInt(args.b)
	if err != nil {
		return err
	}

	v.SetBool(i != 0)
	return nil
}

func (args *cmdArgsReader) parseInt(v reflect.Value) error {
	i, err := objutil.ParseInt(args.b)
	if err != nil {
		return err
	}

	v.SetInt(i)
	return nil
}

func (args *cmdArgsReader) parseUint(v reflect.Value) error {
	u, err := strconv.ParseUint(string(args.b), 10, 64) // this could be optimized
	if err != nil {
		return err
	}

	v.SetUint(u)
	return nil
}

func (args *cmdArgsReader) parseFloat(v reflect.Value) error {
	f, err := strconv.ParseFloat(string(args.b), 64)
	if err != nil {
		return err
	}

	v.SetFloat(f)
	return nil
}

func (args *cmdArgsReader) parseString(v reflect.Value) error {
	v.SetString(string(args.b))
	return nil
}

func (args *cmdArgsReader) parseBytes(v reflect.Value) error {
	v.SetBytes(append(v.Bytes()[:0], args.b...))
	return nil
}

func (args *cmdArgsReader) parseValue(v reflect.Value) error {
	v.Set(reflect.ValueOf(append(make([]byte, 0, len(args.b)), args.b...)))
	return nil
}
