package mobilicy

type Router interface {
	Command(path string, handlers ...Handler) Router
	Add(method Method, path string, handlers ...Handler) Router
}

type Route struct {
	Method   Method
	Path     string
	Handlers []func(*Ctx) error
}

func (r *Route) match(path string) bool {
	return r.Path == path
}
