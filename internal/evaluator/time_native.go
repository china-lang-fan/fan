package evaluator

import (
	"fmt"
	"time"

	"fan/internal/object"
)

func nativeNowUnix(_ []object.Object) ([]object.Object, error) {
	return []object.Object{&object.Integer{Value: time.Now().Unix()}}, nil
}

func nativeNowUnixMilli(_ []object.Object) ([]object.Object, error) {
	return []object.Object{&object.Integer{Value: time.Now().UnixMilli()}}, nil
}

func nativeFormatTime(args []object.Object) ([]object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("formatTime 需要 2 个参数")
	}
	timestamp, ok := args[0].(*object.Integer)
	if !ok {
		return nil, fmt.Errorf("formatTime 第 1 个参数必须是整数时间戳")
	}
	layout, ok := args[1].(*object.String)
	if !ok {
		return nil, fmt.Errorf("formatTime 第 2 个参数必须是字符串格式")
	}
	t := time.Unix(timestamp.Value, 0)
	return []object.Object{&object.String{Value: t.Format(layout.Value)}}, nil
}

func nativeParseTime(args []object.Object) ([]object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("parseTime 需要 2 个参数")
	}
	layout, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("parseTime 第 1 个参数必须是字符串格式")
	}
	value, ok := args[1].(*object.String)
	if !ok {
		return nil, fmt.Errorf("parseTime 第 2 个参数必须是字符串时间")
	}
	t, err := time.ParseInLocation(layout.Value, value.Value, time.Local)
	if err != nil {
		return []object.Object{&object.Integer{Value: 0}, object.NewError("无法解析时间：" + err.Error())}, nil
	}
	return []object.Object{&object.Integer{Value: t.Unix()}, object.Null}, nil
}

func nativeSleep(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sleep 需要 1 个参数")
	}
	seconds, ok := args[0].(*object.Integer)
	if !ok {
		return nil, fmt.Errorf("sleep 参数必须是整数秒")
	}
	time.Sleep(time.Duration(seconds.Value) * time.Second)
	return []object.Object{object.Null}, nil
}

func nativeTimeParts(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("timeParts 需要 1 个参数")
	}
	timestamp, ok := args[0].(*object.Integer)
	if !ok {
		return nil, fmt.Errorf("timeParts 参数必须是整数时间戳")
	}
	t := time.Unix(timestamp.Value, 0)
	dict := object.NewDict()
	put := func(key string, value object.Object) {
		dict.Set(&object.String{Value: key}, value)
	}
	put("year", &object.Integer{Value: int64(t.Year())})
	put("month", &object.Integer{Value: int64(t.Month())})
	put("day", &object.Integer{Value: int64(t.Day())})
	put("hour", &object.Integer{Value: int64(t.Hour())})
	put("minute", &object.Integer{Value: int64(t.Minute())})
	put("second", &object.Integer{Value: int64(t.Second())})
	put("weekday", &object.Integer{Value: int64(t.Weekday())})
	return []object.Object{dict}, nil
}

func init() {
	RegisterNativeFunction("nowUnix", nativeNowUnix)
	RegisterNativeFunction("nowUnixMilli", nativeNowUnixMilli)
	RegisterNativeFunction("formatTime", nativeFormatTime)
	RegisterNativeFunction("parseTime", nativeParseTime)
	RegisterNativeFunction("sleep", nativeSleep)
	RegisterNativeFunction("timeParts", nativeTimeParts)
}
