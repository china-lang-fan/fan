package object

func DeepEqual(a, b Object) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Kind() != b.Kind() {
		return false
	}
	switch left := a.(type) {
	case *Integer:
		return left.Value == b.(*Integer).Value
	case *Float:
		return left.Value == b.(*Float).Value
	case *String:
		return left.Value == b.(*String).Value
	case *Bool:
		return left.Value == b.(*Bool).Value
	case *Byte:
		return left.Value == b.(*Byte).Value
	case *Char:
		return left.Value == b.(*Char).Value
	case *Nil:
		return true
	case *Array:
		right := b.(*Array)
		if len(left.Elements) != len(right.Elements) {
			return false
		}
		for i := range left.Elements {
			if !DeepEqual(left.Elements[i], right.Elements[i]) {
				return false
			}
		}
		return true
	case *Dict:
		right := b.(*Dict)
		if left.Len() != right.Len() {
			return false
		}
		for _, k := range left.Keys {
			lv, lok := left.Get(k)
			rv, rok := right.Get(k)
			if !lok || !rok || !DeepEqual(lv, rv) {
				return false
			}
		}
		return true
	}
	return false
}
