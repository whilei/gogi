package gogi

/*
#cgo pkg-config: glib-2.0 gobject-introspection-1.0
#include <glib.h>
#include <girepository.h>
#include <errno.h>

gboolean load_namespace(const gchar *namespace) {
	GError *error = NULL;
	GITypelib *lib = g_irepository_require(NULL, namespace, NULL, 0, &error);
	return (error == NULL && lib != NULL);
}

GList *get_namespaces() {
	GList *results = NULL;
	gchar **namespaces = g_irepository_get_loaded_namespaces(NULL);
	gint i = 0;
	while (namespaces[i] != NULL) {
		results = g_list_prepend(results, namespaces[i]);
		i++;
	}
	return g_list_reverse(results);
}

GList *get_infos(const gchar *namespace) {
	GList *results = NULL;
	gint n = g_irepository_get_n_infos(NULL, namespace);
	gint i;
	for (i = 0; i<n; i++) {
		results = g_list_prepend(results, g_irepository_get_info(NULL, namespace, i));
	}

	return g_list_reverse(results);
}
*/
import "C"
import (
	"container/list"
	"fmt"
	//"reflect"
)

func GetNamespaces() *list.List {
	raw_list := GListToGo(C.get_namespaces())
	namespaces := list.New()
	for e := raw_list.Front(); e != nil; e = e.Next() {
		namespaces.PushBack(GoString(e.Value.(*C.gchar)))
	}
	return namespaces
}

func GetInfos(namespace string) ([]*GiInfo, error) {
	_namespace := GlibString(namespace) ; defer C.g_free((C.gpointer)(_namespace))
	loaded := GoBool(C.load_namespace(_namespace))
	if !loaded {
		return nil, fmt.Errorf("gogi: namespace '%s' not found", namespace)
	}
	raw_list := GListToGo(C.get_infos(_namespace))
	results := make([]*GiInfo, raw_list.Len())
	for i, e := 0, raw_list.Front(); e != nil; i, e = i + 1, e.Next() {
		ptr := (*C.GIBaseInfo)(e.Value.(C.gpointer))
		results[i] = NewGiInfo(ptr)
	}
	return results, nil
}
