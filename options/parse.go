// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package options

import (
	"strings"
)

// Parse processes the command line options
// returning the remaining arguments
func Parse(args []string) {
	i := 0
loop:
	for ; i < len(args); i++ {
		arg := args[i]
		if arg == "" {
			continue
		}
		if arg[0] != '-' {
			break
		}
		switch arg {
		case "-c", "-client":
			Client = "127.0.0.1"
			if i+1 < len(args) && args[i+1][0] != '-' {
				i++
				Client = args[i]
			}
		case "-r", "-repl":
			Repl = true
		case "-p", "-port":
			if i+1 < len(args) {
				i++
				Port = args[i]
			}
		case "-u", "-unattended":
			Unattended = true
		case "-v", "-version":
			Version = true
		case "--":
			i++
			break loop
		default:
			Help = true
		}
	}
	CmdLine = remainder(args[i:])
	if Client != "" {
		Errlog = "error" + Port + ".log"
		Outlog = "output" + Port + ".log"
	}
}

func remainder(args []string) string {
	var sb strings.Builder
	sep := ""
	for _, arg := range args {
		sb.WriteString(sep)
		sep = " "
		sb.WriteString(EscapeArg(arg))
	}
	return sb.String()
}