package gogi

import (
	"fmt"
	"strings"
)

type Argument struct {
	info *GiInfo
	typ *GiInfo
	dir Direction
	name string
	cname string
	marshal string
}

// return a marshaled Go function and any necessary C wrapper
func WriteFunction(info *GiInfo, owner *GiInfo) (g string, c string) {
	symbol := info.GetSymbol()
	if blacklist[symbol] || cExports[symbol] {
		return
	}
	cExports[symbol] = true
	prefix := GetPrefix(info)

	flags := info.GetFunctionFlags()
	argc := info.GetNArgs()
	retc := 0

	for i := 0; i < info.GetNArgs(); i++ {
		dir := info.GetArg(i).GetDirection()
		switch dir {
			case In: // default, do nothing
			case Out: argc-- ; retc++
			case InOut: return "", "" // quit early
		}
	}

	var ownerName string
	if owner != nil {
		ownerName = owner.GetName()
		castFunc(prefix, ownerName, &c)
	}

	g += "func "

	returnType := info.GetReturnType() ; defer returnType.Free()
	{
		ctype, cp := CType(returnType)
		if ctype == "" {
			g = ""; c = ""
			return
		} else if (ctype == "gchar" && cp != "") {
			ctype = "const " + ctype
		}

		c += ctype + " " + cp
	}

	if owner != nil {
		g += ownerName
	}
	g += CamelCase(info.GetName()) + "("
	c += "gogi_" + symbol + "("

	cArgLine := make([]string, 0)
	gArgLine := make([]string, 0)

	if owner != nil && flags.IsMethod {
		cArgLine = append(cArgLine, prefix + ownerName + " *self")
		gArg := "self "
		if owner.Type == Struct {
			gArg += "*"
		}
		gArg += ownerName
		gArgLine = append(gArgLine, gArg)
	}

	args := make([]Argument, 0)
	rets := make([]Argument, 0)
	argsAndRets := make([]Argument, 0)
	for i := 0; i < argc + retc; i++ {
		arg := info.GetArg(i)
		dir := arg.GetDirection()
		gotype, gp := GoType(arg.GetType())
		ctype, cp := CType(arg.GetType())
		if gotype == "" || ctype == "" || blacklist[gotype] {
			// argument failed to marshal
			return "", ""
		}

		name := arg.GetName()
		newArg := Argument{arg,arg.GetType(),dir,name,"c_"+name,""}
		argsAndRets = append(argsAndRets, newArg)
		if dir == In {
			args = append(args, newArg)
			gArgLine = append(gArgLine, fmt.Sprintf("%s %s", noKeywords(name), gp + gotype))
			cArgLine = append(cArgLine, fmt.Sprintf("%s %s", ctype, cp + name))
		} else if dir == Out {
			rets = append(rets, newArg)
			cArgLine = append(cArgLine, fmt.Sprintf("%s *%s", ctype, cp + name))
		}
	}
	if flags.Throws {
		cArgLine = append(cArgLine, "GError **error")
	}
	g += strings.Join(gArgLine, ", ") + ") "
	c += strings.Join(cArgLine, ", ") + ") "

	var returns bool
	if returnType.GetTag() != VoidTag || returnType.IsPointer() {
		retc++
		rets = append(rets, Argument{nil,returnType,In,"retval","c_retval",""})
		returns = true
	}

	retLine := make([]string, 0)
	for i, ret := range rets {
		retType, retMarshal := MarshalToGo(ret.typ, ret.name, ret.cname)
		if retType == "" {
			return "", ""
		}
		if blacklist[strings.Trim(retType, "*")] {
			return "", ""
		}
		retLine = append(retLine, retType)
		rets[i].marshal = retMarshal
	}
	if flags.Throws {
		retLine = append(retLine, "error")
	}
	if len(retLine) > 0 {
		g += "(" + strings.Join(retLine, ", ") + ") "
	}

	g += "{\n"
	c += "{\n"

	// marshal
	for _, arg := range args {
		ctype, marshal := MarshalToC(arg.typ, arg, arg.cname)
		// TODO: remove the check for "C.", it shouldn't be needed
		if ctype == "" || ctype == "C." {
			g = ""; c = ""
			return
		}
		g += fmt.Sprintf("\tvar %s %s\n", arg.cname, ctype)
		g += fmt.Sprintf("\t%s\n", marshal)
	}

	for i, ret := range rets {
		if i == len(rets)-1 && returns {
			break
		}
		ctype, cp := CType(ret.typ)
		g += fmt.Sprintf("\tvar %s %sC.%s\n", ret.cname, cp, ctype)
	}
	if flags.Throws {
		g += "\tvar c_error *C.GError\n"
	}
	g += "\t"
	if returns {
		g += "c_retval := "
	}

	cgoLine := make([]string, 0)
	if owner != nil && flags.IsMethod {
		switch owner.Type {
			case Object:
				cgoLine = append(cgoLine, fmt.Sprintf("self.As%s()", ownerName))
			case Struct:
				cgoLine = append(cgoLine, "self.ptr")
		}
	}
	for _, arg := range argsAndRets {
		name := arg.cname
		if arg.dir == Out {
			name = "&" + name
		}
		cgoLine = append(cgoLine, name)
	}
	if flags.Throws {
		cgoLine = append(cgoLine, "&c_error")
	}
	g += fmt.Sprintf("C.gogi_%s(%s)\n", symbol, strings.Join(cgoLine, ", "))

	for _, ret := range rets {
		g += "\t" + ret.marshal + "\n"
	}
	if retc > 0 {
		g += "\treturn "
		for i, ret := range rets {
			if i > 0 {
				g += ", "
			}
			g += ret.name
		}
		// TODO: marshal the GError
		if flags.Throws {
			if retc > 0 {
				g += ", "
			}
			g += "nil"
		}
		g += "\n"
	}

	// TODO: marshal return values back

	// TODO: catch errno
	c += "\t"
	if returns {
		c += "return "
	}
	c += info.GetSymbol() + "("
	if owner != nil && flags.IsMethod {
		c += "self"
	}

	//c += strings.Join(c_argnames, ", ")
	for i, arg := range argsAndRets {
		if i > 0 || (owner != nil && flags.IsMethod) {
			c += ", "
		}
		c += arg.name
	}

	if flags.Throws {
		if argc > 0 || flags.IsMethod {
			c += ", "
		}
		c += "error"
	}
	c += ");\n"

	g += "}\n"
	c += "}\n"

	return
}

func WriteStruct(info *GiInfo) (g string, c string) {
	// for now, skip gtype and foreign structs
	if info.IsGTypeStruct() || info.IsForeign() {
		return
	}

	name := info.GetName()

	if blacklist[name] {
		return
	}

	prefix := GetPrefix(info)

	g += fmt.Sprintf("type %s struct {\n", name)
	g += fmt.Sprintf("\tptr *C.%s\n", prefix + name)
	g += "}\n"

	// do its methods
	method_count := info.GetNStructMethods()
	for i := 0; i < method_count; i++ {
		method := info.GetStructMethod(i)
		if method.IsDeprecated() {
			continue
		}
		g_, c_ := WriteFunction(method, info)
		g += g_ + "\n"
		c += c_ + "\n"
	}

	g += "\n"
	if c != "" {
		c += "\n"
	}

	return
}

func WriteObject(info *GiInfo) (g string, c string) {
	iter := info
	name := iter.GetName()

	if blacklist[name] {
		return
	}

	prefix := GetPrefix(info)

	// interface
	g += fmt.Sprintf("type %s interface {\n", name)
	g += fmt.Sprintf("\tAs%s() *C.%s\n", name, prefix + name)
	g += "}\n"

	// implementation
	// ???: does it matter if it's abstract?
	implName := GetImplName(name)
	g += fmt.Sprintf("type %s struct {\n", implName)
	g += fmt.Sprintf("\tptr *C.%s\n", prefix + name)
	g += "}\n"

	// ???: do this for abstract types?
	for {
		if !blacklist[prefix + name] {
			cast := castFunc(prefix, name, &c)
			g += fmt.Sprintf("func (ob %s) As%s() *C.%s {\n", implName, name, prefix + name)
			g += fmt.Sprintf("\treturn C.%s((C.gpointer)(ob.ptr))\n", cast)
			g += "}\n"
		}
		// ???: better way to tell when to stop?
		if name == "Object" || name == "ParamSpec" {
			break
		}
		// workaround for this sometimes being written out twice
		oldName := name
		for name == oldName {
			iter = iter.GetParent() ; defer iter.Free()
			name = iter.GetName()
		}
	}

	// do its methods
	method_count := info.GetNObjectMethods()
	for i := 0; i < method_count; i++ {
		method := info.GetObjectMethod(i)
		if method.IsDeprecated() {
			continue
		}
		g_, c_ := WriteFunction(method, info)
		g += g_ + "\n"
		c += c_ + "\n"
	}

	g += "\n"
	if c != "" {
		c += "\n"
	}

	return
}

func WriteEnum(info *GiInfo) (g string, c string) {
	name := info.GetName()
	prefix := GetPrefix(info)
	symbol := prefix + info.GetName()
	g += fmt.Sprintf("type %s C.%s\n", name, symbol)
	g += "const (\n"

	value_count := info.GetNEnumValues()
	for i := 0; i < value_count; i++ {
		value := info.GetEnumValue(i) ; defer value.Free()
		// ???: how to avoid name clashes?
		g += fmt.Sprintf("\t%s = %d\n", enumValueName(name, CamelCase(value.GetName())), value.GetValue())
	}
	g += ")\n"

	return
}

// Some argument names overlap with Go keywords; use this method to rename them
func noKeywords(name string) string {
	switch name {
		case "type": return "typ"
		case "func": return "fun"
		case "len": return "length"
		case "string": return "str"
	}
	return name
}

// Gets the name for an enum value. Used to avoid naming conflicts
func enumValueName(enum, value string) string {
	return enum + value
}

// Gets the C function for casting to a specific type and writes it if it hasn't been yet
func castFunc(prefix, n string, c *string) string {
	name := "as_" + strings.ToLower(n)
	if !cExports[name] {
		cExports[name] = true
		(*c) += fmt.Sprintf("%s *%s(gpointer ob) {\n", prefix + n, name)
		(*c) += fmt.Sprintf("\treturn (%s*)ob;\n", prefix + n)
		(*c) += "}\n"
	}
	return name
}
