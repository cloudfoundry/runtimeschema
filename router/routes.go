package router

import (
	"fmt"
	"github.com/gorilla/pat"
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
			p.Get(route.Path, handler.ServeHTTP)
		case "POST":
			p.Post(route.Path, handler.ServeHTTP)
		case "PUT":
			p.Put(route.Path, handler.ServeHTTP)
		case "DELETE":
			p.Delete(route.Path, handler.ServeHTTP)
		default:
			return nil, fmt.Errorf("invalid verb: %s", route.Method)
		}
	}
	return p, nil
}
