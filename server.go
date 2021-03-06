package main

import (
	"bono"
	"bytes"
	"html/template"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/russross/blackfriday"
)

type Server struct {
	*bono.App
	debug   bool
	dataDir string
}

func (s *Server) UseStaticMiddleware(path string) error {
	if !s.debug {
		s.Use(func(c *bono.Context, next bono.Next) error {
			filename := "www" + c.Request.URL.Path
			_, err := AssetInfo(filename)
			if err != nil {
				return next()
			}

			contentType := mime.TypeByExtension(filepath.Ext(filename))
			if contentType != "" {
				c.Response.Writer.Header().Set("Content-Type", contentType)
			}

			content, err := Asset(filename)
			if err != nil {
				return err
			}
			c.Response.Body = content
			return nil
		})
	} else {
		s.Use(bono.StaticMiddleware(path))
	}

	return nil
}

func (s *Server) UseLogMiddleware() error {
	s.Use(func(c *bono.Context, next bono.Next) error {
		if c.Request.URL.Path == "/favicon.ico" {
			return nil
		}
		log.Printf("%s %s", c.Request.Method, c.Request.RequestURI)
		return next()
	})
	return nil
}

func (s *Server) UseMainHandler() error {
	s.Use(func(c *bono.Context, next bono.Next) error {
		if c.Request.URL.Path == "/index" {
			return s.showIndex(c, next)
		}
		return next()
	})

	s.Use(func(c *bono.Context, next bono.Next) error {
		action := c.Request.URL.Query().Get("do")
		switch action {
		case "update":
			if c.Request.Method == "POST" {
				return s.updateContent(c, next)
			} else {
				return s.updateContentForm(c, next)
			}
		case "delete":
			return s.deleteContent(c, next)
		default:
			return s.showContent(c, next)
		}
	})

	return nil
}

func (s *Server) deleteContent(c *bono.Context, next bono.Next) error {
	os.Remove(s.getMarkdownFile(c.Request.URL.Path))
	return c.Redirect("/index")
}

func NewServer(dataDir string) (*Server, error) {
	debug := true
	if os.Getenv("DEBUG") == "" {
		debug = false
	}
	server := &Server{
		App:     bono.New(),
		dataDir: dataDir,
		debug:   debug,
	}

	if err := server.UseStaticMiddleware("www"); err != nil {
		return nil, err
	}

	if err := server.UseLogMiddleware(); err != nil {
		return nil, err
	}

	if err := server.UseMainHandler(); err != nil {
		return nil, err
	}

	return server, nil
}

func (s *Server) getMarkdownFile(uri string) string {
	file := uri
	if file == "/" {
		file = "/index"
	}
	return s.dataDir + file + ".md"
}

func (s *Server) showIndex(c *bono.Context, next bono.Next) error {
	files := []string{}

	dataDirStrLen := len(s.dataDir)
	filepath.Walk(s.dataDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, "/index.md") {
			file := path
			if s.dataDir != "." {
				file = file[dataDirStrLen:]
			}
			file = file[0 : len(file)-3]
			files = append(files, file)
		}
		return nil
	})

	body, err := s.render(c, "index", struct {
		Files []string
	}{
		Files: files,
	})
	if err != nil {
		return err
	}

	c.Response.Body = body
	return nil
}

func (s *Server) showContent(c *bono.Context, next bono.Next) error {
	content, err := ioutil.ReadFile(s.getMarkdownFile(c.Request.URL.Path))
	if err != nil {
		return c.Redirect("?do=update")
	}

	markdown := blackfriday.MarkdownCommon(content)

	body, err := s.render(c, "read", struct {
		Content template.HTML
	}{
		Content: template.HTML(string(markdown)),
	})

	if err != nil {
		return err
	}
	c.Response.Body = body

	return nil
}

func (s *Server) updateContent(c *bono.Context, next bono.Next) error {
	c.Request.ParseForm()

	filename := s.getMarkdownFile(c.Request.URL.Path)

	content := c.Request.Form.Get("content")
	os.MkdirAll(filepath.Dir(filename), 0755)
	ioutil.WriteFile(filename, []byte(content), 0644)

	c.Set("success", "Content saved")

	body, err := s.render(c, "update", struct {
		Content string
	}{
		Content: content,
	})
	if err != nil {
		return err
	}
	c.Response.Body = body

	return nil
}

func (s *Server) updateContentForm(c *bono.Context, next bono.Next) error {
	contentByte, _ := ioutil.ReadFile(s.getMarkdownFile(c.Request.URL.Path))

	body, err := s.render(c, "update", struct {
		Content string
	}{
		Content: string(contentByte),
	})
	if err != nil {
		return err
	}
	c.Response.Body = body
	return nil
}

func (s *Server) render(c *bono.Context, name string, data interface{}, _withLayout ...bool) ([]byte, error) {
	withLayout := true
	if len(_withLayout) > 0 {
		withLayout = _withLayout[0]
	}

	innerName := name
	innerData := data
	if withLayout {
		body, err := s.render(c, innerName, innerData, false)
		if err != nil {
			return nil, err
		}
		mainBody := string(body)

		name = "layout"
		data = struct {
			Title string
			Main  template.HTML
		}{
			Title: "Wiki",
			Main:  template.HTML(mainBody),
		}
	}

	filename := "templates/" + name + ".html"

	var content []byte
	var err error

	if s.debug {
		content, err = ioutil.ReadFile(filename)
	} else {
		content, err = Asset(filename)
	}
	if err != nil {
		return nil, err
	}

	helperShowAlerts := func() template.HTML {
		alert := c.Get("success")
		if alert != nil {
			buffer := bytes.NewBuffer([]byte{})
			buffer.WriteString(`<div class="alert alert-success alert-dismissible" role="alert">
        <button type="button" class="close" data-dismiss="alert" aria-label="Close"><span aria-hidden="true">&times;</span></button>
        `)
			buffer.WriteString(alert.(string))
			buffer.WriteString("</div>")
			return template.HTML(buffer.String())
		}
		return ""
	}

	helperUri := func() string {
		return c.Request.URL.Path
	}

	helperIsRead := func() bool {
		return c.Request.URL.Query().Get("do") == ""
	}

	helperIsUpdate := func() bool {
		return c.Request.URL.Query().Get("do") == "update"
	}

	t, err := template.New(name).Funcs(template.FuncMap{
		"uri":        helperUri,
		"isRead":     helperIsRead,
		"isUpdate":   helperIsUpdate,
		"showAlerts": helperShowAlerts,
	}).Parse(string(content))
	if err != nil {
		return nil, err
	}

	w := bytes.NewBuffer([]byte{})
	t.Execute(w, data)

	return w.Bytes(), nil
}
