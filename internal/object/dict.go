package object

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const KindDict Kind = "字典"

type Dict struct {
	Keys    []Object
	Entries map[string]Object
}

func NewDict() *Dict {
	return &Dict{Entries: map[string]Object{}}
}

func (d *Dict) Kind() Kind { return KindDict }

func (d *Dict) Inspect() string {
	parts := make([]string, 0, len(d.Keys))
	for _, k := range d.Keys {
		parts = append(parts, Format(k)+": "+Format(d.Entries[dictKey(k)]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func dictKey(k Object) string {
	switch v := k.(type) {
	case *Integer:
		return "i:" + strconv.FormatInt(v.Value, 10)
	case *Float:
		return "f:" + strconv.FormatFloat(v.Value, 'g', -1, 64)
	case *String:
		return "s:" + v.Value
	case *Bool:
		if v.Value {
			return "b:1"
		}
		return "b:0"
	case *Byte:
		return "y:" + strconv.Itoa(int(v.Value))
	case *Char:
		return "c:" + string(v.Value)
	}
	return ""
}

func ValidDictKey(k Object) bool {
	switch k.(type) {
	case *Integer, *Float, *String, *Bool, *Byte, *Char:
		return true
	}
	return false
}

func (d *Dict) Get(k Object) (Object, bool) {
	v, ok := d.Entries[dictKey(k)]
	return v, ok
}

func (d *Dict) Set(k, v Object) {
	key := dictKey(k)
	if _, exists := d.Entries[key]; !exists {
		d.Keys = append(d.Keys, k)
	}
	d.Entries[key] = v
}

func (d *Dict) Len() int {
	return len(d.Keys)
}

func (d *Dict) SortedKeys() []Object {
	keys := append([]Object(nil), d.Keys...)
	sort.SliceStable(keys, func(i, j int) bool {
		return fmt.Sprintf("%v", keys[i]) < fmt.Sprintf("%v", keys[j])
	})
	return keys
}
