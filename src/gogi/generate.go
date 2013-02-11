package gogi

import (
	"fmt"
	"strings"
)

type Argument struct {
	info *GiInfo
	name string
	cname string
	typ *GiInfo
}

// return a marshaled Go function and any necessary C wrapper
func WriteFunction(info *GiInfo) (string, string) {
	var text string = "func "
	var wrapper string = "void "
	c_func := info.GetFullName()

	// TODO: check if this is a method on an object

	text += strings.Title(info.GetName()) + "("
	wrapper += "gogi_" + c_func + "("

	argc := info.GetNArgs()
	args := make([]Argument, argc)
	for i := 0; i < argc; i++ {
		arg := info.GetArg(i)
		args[i] = Argument{arg,arg.GetName(),"",arg.GetType()}
		text += fmt.Sprintf("%s %s", args[i].name, GoType(args[i].typ, arg.GetDirection()))
		wrapper += fmt.Sprintf("%s %s", CType(args[i].typ, args[i].info.GetDirection()), args[i].name)
		if i < argc-1 {
			text += ", "
			wrapper += ", "
		}
	}
	text += ") "
	wrapper += ") "

	// TODO: check for a return value

	text += "{\n" // Go function open
	wrapper += "{\n"
	// marshal
	for i := 0; i < argc; i++ {
		args[i].cname = "c_" + args[i].name
		ctype, marshal := GoToC(args[i].typ, args[i], args[i].cname)
		text += fmt.Sprintf("\tvar %s %s\n", args[i].cname, ctype)
		text += fmt.Sprintf("\t%s\n", marshal)
		text += fmt.Sprint("\n")
	}
	go_argnames := make([]string, len(args))
	c_argnames := make([]string, len(args))
	for i, arg := range args {
		dir := arg.info.GetDirection()
		if dir == Out || dir == InOut {
			go_argnames[i] += "&"
			//c_argnames[i] += "&"
		}
		go_argnames[i] += arg.cname
		c_argnames[i] += arg.name
	}
	text += "\tC.gogi_" + c_func + "(" + strings.Join(go_argnames, ", ") + ")\n"
	wrapper += "\t" + c_func + "(" + strings.Join(c_argnames, ", ") + ");\n"

	text += "}"
	wrapper += "}"

	return text, wrapper
}
