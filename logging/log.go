package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
)

var Output io.Writer = ioutil.Discard

func New(prefix string, out io.Writer) *log.Logger {
	if out == nil {
		out = Output
	}
	return log.New(out, fmt.Sprintf("[%s] ", prefix), log.Ldate|log.Lmicroseconds)
}

var DiscardingLogger = New("", ioutil.Discard)

// LogTo makes loggers output to the given writer by replacing Output. It returns a function that
// sets Output to whatever value it had before the call to LogTo.
//
// As an example
//     defer LogTo(os.Stderr)()
// would start logging to stderr immediately and defer restoring the loggers.
func LogTo(w io.Writer) func() {
	oldOutput := Output
	Output = w
	return func() {
		Output = oldOutput
	}
}
