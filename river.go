package river

import (
	"net/http"
	"path"

	"github.com/julienschmidt/httprouter"
)

// River is a REST server handler and toolkit.
type River struct {
	r *httprouter.Router
	handlerChain
	renderer Renderer
	verbose
}

// New creates a new River.
// Optional params middlewares are the middlewares to initiate with.
// Middlewares can also be added with river.Use* methods.
// Renderer defaults to JSONRenderer.
func New(middlewares ...Handler) *River {
	r := httprouter.New()
	r.HandleMethodNotAllowed = true
	r.HandleOPTIONS = true
	r.RedirectTrailingSlash = true

	return (&River{r: r, handlerChain: middlewares}).
		NotFound(notFound).
		NotAllowed(notAllowed).
		Renderer(JSONRenderer)
}

func (rv *River) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rv.r.ServeHTTP(w, r)
}

// Handle handles endpoint at path p.
// This should only be called after Endpoint requests have been handled.
func (rv *River) Handle(p string, e *Endpoint) *River {
	rv.handle(p, e)
	return rv
}

func (rv *River) routerHandle(handler Handler, e *Endpoint) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		c := &Context{
			rw:          w,
			Request:     r,
			params:      p,
			renderer:    notNilRenderer(e.renderer, rv.renderer),
			middlewares: append(rv.handlerChain, append(e.handlerChain, handler)...),
		}
		c.Next()
	}
}

func (rv *River) routerHandleNoEndpoint(handler Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := &Context{
			rw:          w,
			Request:     r,
			renderer:    notNilRenderer(rv.renderer),
			middlewares: append(rv.handlerChain, handler),
		}
		c.Next()
	}
}

func (rv *River) handle(p string, e *Endpoint) {
	for subpath := range e.handlers {
		fullPath := path.Join(p, subpath)
		for method, handler := range e.handlers[subpath] {
			rv.r.Handle(method, fullPath, rv.routerHandle(handler, e))
			rv.handledPaths.add(method, fullPath, nameOf(handler))
		}
	}
}

// Run starts River as an http server.
func (rv *River) Run(addr string) error {
	log.printf("Server started on %s", addr)
	rv.Dump()
	return http.ListenAndServe(addr, rv)
}

// Renderer sets output renderer.
// An endpoint renderer overrules this.
func (rv *River) Renderer(r Renderer) *River {
	rv.renderer = r
	return rv
}

// NotAllowed replaces the default handler for methods not handled by
// any endpoint with h.
func (rv *River) NotAllowed(h Handler) *River {
	rv.r.MethodNotAllowed = rv.routerHandleNoEndpoint(h)
	return rv
}

// NotFound replaces the default handler for request paths without
// any endpoint.
func (rv *River) NotFound(h Handler) *River {
	rv.r.NotFound = rv.routerHandleNoEndpoint(h)
	return rv
}

func notFound(c *Context) {
	c.RenderEmpty(http.StatusNotFound)
}

func notAllowed(c *Context) {
	c.RenderEmpty(http.StatusMethodNotAllowed)
}

func notNilRenderer(r ...Renderer) Renderer {
	for i := range r {
		if r[i] != nil {
			return r[i]
		}
	}
	return PlainRenderer
}
