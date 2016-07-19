package river

import (
	"fmt"
	"net/http"
	"time"

	"github.com/fatih/color"
)

// handlerChain is middleware chain.
type handlerChain []Handler

// Use adds middlewares to the middleware chain.
func (c *handlerChain) Use(middlewares ...Handler) {
	*c = append(*c, middlewares...)
}

// UseHandler adds any http.Handler as middleware to the middleware chain.
func (c *handlerChain) UseHandler(middlewares ...http.Handler) {
	for i := range middlewares {
		c.Use(toHandler(middlewares[i]))
	}
}

func toHandler(h http.Handler) Handler {
	return func(c *Context) {
		h.ServeHTTP(c, c.Request)
		c.Next()
	}
}

// Logger is a middleware that logs requests in a colourful way.
// Useful for development.
func Logger() Handler {
	return func(c *Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)

		bg := color.BgBlack
		switch {
		case c.Status() >= 200 && c.Status() < 300:
			bg = color.BgGreen
		case c.Status() >= 300 && c.Status() < 400:
			bg = color.BgBlue
		case c.Status() >= 400 && c.Status() < 500:
			bg = color.BgYellow
		case c.Status() >= 500 && c.Status() < 600:
			bg = color.BgRed
		}

		paint := color.New(bg, color.FgWhite, color.Bold).SprintFunc()
		status := paint(fmt.Sprintf("  %d  ", c.Status()))

		fmt.Printf("%s %v %s %15v %-4s %s\n",
			log.prefix(),
			time.Now().Format("2006-01-02 15:04:05"),
			status, duration, c.Method, c.URL.Path,
		)

	}
}

// Recovery creates a panic recovery middleware. If handler is not nil,
// it calls handler after recovery
func Recovery(handler func(*Context, interface{})) Handler {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				if handler != nil {
					handler(c, err)
				} else {
					c.Render(http.StatusInternalServerError, err)
				}
			}
		}()
		c.Next()
	}
}
