package gogi

/*
#cgo pkg-config: glib-2.0
#include <glib.h>
#include <girepository.h>

GList *empty_glist = NULL;

char *from_gchar(gchar *str) { return (char*)str; }
gchar *to_gchar(char *str) { return (gchar*)str; }

// bitflag support
gboolean and(gint flags, gint position) {
	return flags & position;
}
*/
import "C"
import (
	"container/list"
	"fmt"
	"reflect"
)

var goTypes = map[int]string {
	(int)(C.GI_TYPE_TAG_VOID):     "",
	(int)(C.GI_TYPE_TAG_BOOLEAN):  "bool",
	(int)(C.GI_TYPE_TAG_INT8):     "int8",
	(int)(C.GI_TYPE_TAG_INT16):    "int16",
	(int)(C.GI_TYPE_TAG_INT32):    "int32",
	(int)(C.GI_TYPE_TAG_INT64):    "int64",
	(int)(C.GI_TYPE_TAG_UINT8):    "uint8",
	(int)(C.GI_TYPE_TAG_UINT16):   "uint16",
	(int)(C.GI_TYPE_TAG_UINT32):   "uint32",
	(int)(C.GI_TYPE_TAG_UINT64):   "uint64",
	(int)(C.GI_TYPE_TAG_FLOAT):    "float32",
	(int)(C.GI_TYPE_TAG_DOUBLE):   "float64",
	(int)(C.GI_TYPE_TAG_UTF8):     "string",
	(int)(C.GI_TYPE_TAG_FILENAME): "string",
	// skip a couple
	(int)(C.GI_TYPE_TAG_GLIST):    "list.List",
	(int)(C.GI_TYPE_TAG_GSLIST):   "list.List",
	// skip a couple
	(int)(C.GI_TYPE_TAG_UNICHAR):  "rune",
}

var cTypes = map[int]string {
	(int)(C.GI_TYPE_TAG_VOID):     "void",
	(int)(C.GI_TYPE_TAG_BOOLEAN):  "gboolean",
	(int)(C.GI_TYPE_TAG_INT8):     "gint8",
	(int)(C.GI_TYPE_TAG_INT16):    "gint16",
	(int)(C.GI_TYPE_TAG_INT32):    "gint32",
	(int)(C.GI_TYPE_TAG_INT64):    "gint64",
	(int)(C.GI_TYPE_TAG_UINT8):    "guint8",
	(int)(C.GI_TYPE_TAG_UINT16):   "guint16",
	(int)(C.GI_TYPE_TAG_UINT32):   "guint32",
	(int)(C.GI_TYPE_TAG_UINT64):   "guint64",
	(int)(C.GI_TYPE_TAG_FLOAT):    "gfloat",
	(int)(C.GI_TYPE_TAG_DOUBLE):   "gdouble",
	(int)(C.GI_TYPE_TAG_UTF8):     "gchar",
	(int)(C.GI_TYPE_TAG_FILENAME): "gchar",
	// skip a couple
	(int)(C.GI_TYPE_TAG_GLIST):    "GList*",
	(int)(C.GI_TYPE_TAG_GSLIST):   "GSList*",
	// skip a couple
	(int)(C.GI_TYPE_TAG_UNICHAR):  "gunichar",
}

// returns the C type and the necessary marshaling code
func GoToC(typeInfo *GiInfo, arg Argument, cvar string) (ctype string, marshal string) {
	govar := arg.name

	dir := arg.info.GetDirection()
	ref := refOut(dir)

	tag := typeInfo.GetTag()
	if tag == ArrayTag {
		// do array stuff
		switch typeInfo.GetArrayType() {
		case C.GI_ARRAY_TYPE_C:
			arg.name = govar + "_ar"
			ar_ctype, _ := GoToC(typeInfo.GetParamType(0), arg, cvar + "_ar")
			ctype = "*" + ar_ctype
			cvar_len := cvar + "_len"
			cvar_val := cvar + "_val"
			marshal = cvar_len + " := len(" + ref + govar + ")\n\t" +
			          cvar_val + " := make([]" + ar_ctype + ", " + cvar_len + ")\n\t" +
			          "for i := 0; i < " + cvar_len + "; i++ {\n\t" +
					  "\t" + cvar_val + "[i] = (*C.gchar)(C.CString((" + ref + govar + ")[i]))\n\t" +
					  "}\n\t" +
					  cvar + " = (" + ref + ar_ctype + ")(unsafe.Pointer(&" + cvar_val + "))"
		}
	} else {
		switch tag {
			case C.GI_TYPE_TAG_INT8:
				ctype = "C.gint8"
				marshal = fmt.Sprintf("%s = (%s)(%s)", cvar, ctype, ref + govar)
			case C.GI_TYPE_TAG_INT16:
				ctype = "C.gint16"
				marshal = fmt.Sprintf("%s = (%s)(%s)", cvar, ctype, ref + govar)
			case C.GI_TYPE_TAG_INT32:
				ctype = "C.gint32"
				marshal = fmt.Sprintf("%s = (%s)(%s)", cvar, ctype, ref + govar)
			case C.GI_TYPE_TAG_INT64:
				ctype = "C.gint64"
				marshal = fmt.Sprintf("%s = (%s)(%s)", cvar, ctype, ref + govar)
			case C.GI_TYPE_TAG_UINT8:
				ctype = "C.guint8"
				marshal = fmt.Sprintf("%s = (%s)(%s)", cvar, ctype, ref + govar)
			case C.GI_TYPE_TAG_UINT16:
				ctype = "C.guint16"
				marshal = fmt.Sprintf("%s = (%s)(%s)", cvar, ctype, ref + govar)
			case C.GI_TYPE_TAG_UINT32:
				ctype = "C.guint32"
				marshal = fmt.Sprintf("%s = (%s)(%s)", cvar, ctype, ref + govar)
			case C.GI_TYPE_TAG_UINT64:
				ctype = "C.guint64"
				marshal = fmt.Sprintf("%s = (%s)(%s)", cvar, ctype, ref + govar)
			case C.GI_TYPE_TAG_UTF8, C.GI_TYPE_TAG_FILENAME:
				ctype = "*C.gchar"
				marshal = "// TODO: marshal strings"
			case C.GI_TYPE_TAG_INTERFACE:
				interfaceInfo := typeInfo.GetTypeInterface()
				switch interfaceInfo.Type {
					case Enum:
						ctype = "gint"
						marshal = fmt.Sprintf("%s = (%s)(%s)", cvar, ctype, ref + govar)
				}
			case C.GI_TYPE_TAG_GLIST:
				ctype = "C.GList*"
				marshal = "// TODO: marshal glist"
			case C.GI_TYPE_TAG_GSLIST:
				ctype = "C.GSList*"
				marshal = "// TODO: marshal gslist"
			default:
				ctype = "<MISSING CTYPE: " + TypeTagToString(tag) + ">"
		}
	}
	return
}

func CToGo(typeInfo *GiInfo, govar string, cvar string) (gotype string, marshal string) {
	tag := typeInfo.GetTag()
	if tag == ArrayTag {
		// TODO: implement
	} else {
		switch tag {
			case C.GI_TYPE_TAG_BOOLEAN:
				gotype = "bool"
				marshal = fmt.Sprintf("var %s %s\n", govar, gotype)
				marshal += fmt.Sprintf("\tif %s == 0 {", cvar) + "\n\t" +
						   fmt.Sprintf("\t%s = false", govar) + "\n\t" +
						   "} else {\n\t" +
						   fmt.Sprintf("\t%s = true", govar) + "\n\t" +
				           "}"
			case C.GI_TYPE_TAG_INTERFACE:
				interfaceInfo := typeInfo.GetTypeInterface()
				switch interfaceInfo.Type {
					case Object:
						gotype = "*" + interfaceInfo.GetName()
						marshal = "// marshal?"
				}
			default:
				gotype = "<MISSING GOTYPE: " + TypeTagToString(tag) + ">"
		}
	}
	return
}

func GoType(typeInfo *GiInfo, dir Direction) string {
	tag := typeInfo.GetTag()
	if tag == ArrayTag {
		return (refOut(dir) + "[]" + GoType(typeInfo.GetParamType(0), In))
	} else {
		ptr := refPointer(typeInfo, dir)
		val, ok := goTypes[(int)(tag)]
		if ok {
			return (ptr + val)
		}

		// check non-primitive tags
		switch tag {
			case C.GI_TYPE_TAG_INTERFACE:
				interfaceType := typeInfo.GetTypeInterface()
				switch interfaceType.Type {
					case Object:
						return (ptr + interfaceType.GetName())
					default:
						return interfaceType.GetName()
				}
		}
	}

	return "<MISSING GOTYPE: " + TypeTagToString(tag) + ">"
}

func CType(typeInfo *GiInfo, dir Direction) string {
	ptr := refPointer(typeInfo, dir)
	tag := typeInfo.GetTag()
	if tag == ArrayTag {
		return CType(typeInfo.GetParamType(0), In) + ptr
	} else {
		val, ok := cTypes[(int)(tag)]
		if ok {
			return val + ptr
		}

		switch tag {
			case C.GI_TYPE_TAG_INTERFACE:
				interfaceType := typeInfo.GetTypeInterface()
				switch interfaceType.Type {
					case Object:
						return interfaceType.GetObjectTypeName() + ptr
					default:
						return interfaceType.GetName()
				}
		}
	}

	return "<MISSING CTYPE: " + TypeTagToString(tag) + ">"
}


func GoBool(b C.gboolean) bool {
	if b == C.gboolean(0) {
		return false
	}
	return true
}

func GlibBool(b bool) C.gboolean {
	if b {
		return C.gboolean(1)
	}
	return C.gboolean(0)
}

func GoChar(c C.gchar) int8 {
	return int8(c)
}

func GlibChar(i int8) C.gchar {
	return C.gchar(i)
}

func GoUChar(c C.guchar) uint {
	return uint(c)
}

func GlibUChar(i uint) C.guchar {
	return C.guchar(i)
}

func GoInt(i C.gint) int {
	return int(i)
}

func GlibInt(i int) C.gint {
	return C.gint(i)
}

func GoUInt(i C.guint) uint {
	return uint(i)
}

func GlibUInt(i uint) C.guint {
	return C.guint(i)
}

func GoInt8(i C.gint8) int8 {
	return int8(i)
}

func GlibInt8(i int8) C.gint8 {
	return C.gint8(i)
}

func GoUInt8(i C.guint8) uint8 {
	return uint8(i)
}

func GlibUInt8(i uint8) C.guint8 {
	return C.guint8(i)
}

func GoInt16(i C.gint16) int16 {
	return int16(i)
}

func GlibInt16(i int16) C.gint16 {
	return C.gint16(i)
}

func GoUInt16(i C.guint16) uint16 {
	return uint16(i)
}

func GlibUInt16(i uint16) C.guint16 {
	return C.guint16(i)
}

func GoInt32(i C.gint32) int32 {
	return int32(i)
}

func GlibInt32(i int32) C.gint32 {
	return C.gint32(i)
}

func GoUInt32(i C.guint32) uint32 {
	return uint32(i)
}

func GlibUInt32(i uint32) C.guint32 {
	return C.guint32(i)
}

func GoInt64(i C.gint64) int64 {
	return int64(i)
}

func GlibInt64(i int64) C.gint64 {
	return C.gint64(i)
}

func GoUInt64(i C.guint64) uint64 {
	return uint64(i)
}

func GlibUInt64(i uint64) C.guint64 {
	return C.guint64(i)
}

func GoShort(s C.gshort) int16 {
	return int16(s)
}

func GlibShort(s int16) C.gshort {
	return C.gshort(s)
}

func GoUShort(s C.gushort) uint16 {
	return uint16(s)
}

func GlibUShort(s uint16) C.gushort {
	return C.gushort(s)
}

func GoLong(l C.glong) int64 {
	return int64(l)
}

func GlibLong(l int64) C.glong {
	return C.glong(l)
}

func GoULong(l C.gulong) uint64 {
	return uint64(l)
}

func GlibULong(l uint64) C.gulong {
	return C.gulong(l)
}

// TODO: gint8, gint16, etc.

func GoFloat(f C.gfloat) float32 {
	return float32(f)
}

func GlibFloat(f float32) C.gfloat {
	return C.gfloat(f)
}

func GoDouble(d C.gdouble) float64 {
	return float64(d)
}

func GlibDouble(d float64) C.gdouble {
	return C.gdouble(d)
}

func GoString(str *C.gchar) string {
	return C.GoString(C.from_gchar(str))
}

func GlibString(str string) *C.gchar {
	return C.to_gchar(C.CString(str))
}

func GListToGo(glist *C.GList) *list.List {
	result := list.New()
	for glist != C.empty_glist {
		result.PushBack(glist.data)
		glist = glist.next
	}
	return result
}

func PopulateFlags(data interface{}, bits C.gint, flags []C.gint) {
	value := reflect.ValueOf(data).Elem()
	for i := range flags {
		value.Field(i).SetBool(GoBool(C.and(bits, flags[i])))
	}
}

func refOut(dir Direction) string {
	if dir == Out || dir == InOut {
		return "*"
	}
	return ""
}

func refPointer(typeInfo *GiInfo, dir Direction) string {
	ptr := refOut(dir)
	if typeInfo.IsPointer() {
		ptr += "*"
	}
	return ptr
}
