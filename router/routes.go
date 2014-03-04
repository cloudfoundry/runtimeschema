package router

import (
	"fmt"
	"github.com/bmizerany/pat"
	"net/http"
	"strings"
)

type Handlers map[string]http.Handler

type Route struct {
	Handler string
	Method  string
	Path    string
}

type Routes []Route

func (r Routes) Router(actions Handlers) (http.Handler, error) {
	p := pat.New()
	for _, route := range r {
		handler, ok := actions[route.Handler]
		if !ok {
			return nil, fmt.Errorf("missing handler %s", route.Handler)
		}
		switch strings.ToUpper(route.Method) {
		case "GET":
			p.Get(route.Path, handler)
		case "POST":
			p.Post(route.Path, handler)
		case "PUT":
			p.Put(route.Path, handler)
		case "DELETE":
			p.Del(route.Path, handler)
		default:
			return nil, fmt.Errorf("invalid verb: %s", route.Method)
		}
	}
	return p, nil
}
