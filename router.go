package mobilicy

type Router interface {
	Command(path string, handlers ...Handler) Router
	Add(method Method, path string, handlers ...Handler) Router
}

type Route struct {
	Method   Method
	Path     string
	Handlers []Handler
}

func (r *Route) match(path string) bool {
	return r.Path == path
}
