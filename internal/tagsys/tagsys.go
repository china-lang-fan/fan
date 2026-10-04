package tagsys

type Object any

type Callable interface {
	Call(args []Object) ([]Object, error)
}

type TargetKind string

const (
	TargetClass    TargetKind = "模型"
	TargetFunction TargetKind = "函数"
	TargetMethod   TargetKind = "方法"
)

type Target interface {
	TargetName() string
	TargetKind() TargetKind
	Tags() []Object
}

type Context interface {
	Tag() Object
	Target() Target
	Callable() Callable
	ReplaceCallable(Callable)
	ReplaceFunction(Callable)
	ReplaceMethod(Callable)
}

type Handler interface {
	Handle(Context) error
}

type HandlerFunc func(Context) error

func (fn HandlerFunc) Handle(ctx Context) error {
	return fn(ctx)
}

type Resolver interface {
	Resolve(name string) (Object, bool)
}
