package bono

import (
	"errors"
	"log"
	"net/http"
	"os"
)

type Response struct {
	Status int
	Body   []byte
	Writer http.ResponseWriter
}

type Context struct {
	Request    *http.Request
	Response   *Response
	Attributes map[string]interface{}
}

func (c *Context) Set(key string, value interface{}) {
	if c.Attributes == nil {
		c.Attributes = make(map[string]interface{})
	}
	c.Attributes[key] = value
}

func (c *Context) Get(key string) interface{} {
	return c.Attributes[key]
}

func (c *Context) Redirect(url string, status ...int) error {
	if len(status) == 0 {
		status = append(status, 302)
	}
	c.Response.Status = status[0]
	c.Response.Writer.Header().Set("Location", url)
	return errors.New("Stop")
}

type Next func() error

type Middleware func(c *Context, next Next) error

type App struct {
	middlewares []Middleware
}

func (a *App) dispatchMiddleware(i int, context *Context) error {
	if len(a.middlewares) > i {
		middleware := a.middlewares[i]
		return middleware(context, func() error {
			return a.dispatchMiddleware(i+1, context)
		})
	}
	return nil
}

func (a *App) Callback() func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		response := Response{
			Status: 404,
			Writer: rw,
		}

		context := &Context{
			Response: &response,
			Request:  req,
		}
		err := a.dispatchMiddleware(0, context)

		if err != nil {
			if err.Error() == "Delegated" {
				return
			} else if err.Error() != "Stop" {
				log.Printf("Caught error: %s", err.Error())
				response.Status = 500
				response.Body = []byte(err.Error() + "\n")
			}
		}

		if response.Status == 404 && response.Body != nil {
			response.Status = 200
		}

		rw.WriteHeader(context.Response.Status)
		rw.Write(context.Response.Body)
	}
}

func (a *App) Use(m Middleware) *App {
	a.middlewares = append(a.middlewares, m)
	return a
}

func (a *App) Listen(address string) {
	http.HandleFunc("/", a.Callback())

	log.Printf("Listening to %s", address)
	http.ListenAndServe(address, nil)
}

func New() *App {
	return &App{}
}

func StaticMiddleware(base string) Middleware {
	return func(context *Context, next Next) error {
		if stat, err := os.Stat(base + context.Request.URL.Path); os.IsNotExist(err) || stat.IsDir() {
			return next()
		}

		http.ServeFile(context.Response.Writer, context.Request, base+context.Request.URL.Path)
		return errors.New("Delegated")
	}
}
