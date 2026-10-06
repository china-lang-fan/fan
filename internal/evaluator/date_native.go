package evaluator

import (
	"fmt"
	"time"

	"fan/internal/object"
)

func nativeDateUnix(args []object.Object) ([]object.Object, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("dateUnix 需要 3 个参数")
	}
	year, ok := args[0].(*object.Integer)
	if !ok {
		return nil, fmt.Errorf("dateUnix 第 1 个参数必须是整数")
	}
	month, ok := args[1].(*object.Integer)
	if !ok {
		return nil, fmt.Errorf("dateUnix 第 2 个参数必须是整数")
	}
	day, ok := args[2].(*object.Integer)
	if !ok {
		return nil, fmt.Errorf("dateUnix 第 3 个参数必须是整数")
	}
	if month.Value < 1 || month.Value > 12 || day.Value < 1 || day.Value > 31 {
		return []object.Object{&object.Integer{Value: 0}, object.NewError("日期不合法")}, nil
	}
	t := time.Date(int(year.Value), time.Month(month.Value), int(day.Value), 0, 0, 0, 0, time.Local)
	if t.Year() != int(year.Value) || int(t.Month()) != int(month.Value) || t.Day() != int(day.Value) {
		return []object.Object{&object.Integer{Value: 0}, object.NewError("日期不存在")}, nil
	}
	return []object.Object{&object.Integer{Value: t.Unix()}, object.Null}, nil
}

func init() {
	RegisterNativeFunction("dateUnix", nativeDateUnix)
}
