package main

import (
	"bono"
	"bytes"
	"github.com/russross/blackfriday"
	"html/template"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	app := bono.New()

	app.Use(func(c *bono.Context, next bono.Next) error {
		log.Printf("%s %s", c.Request.Method, c.Request.RequestURI)
		return next()
	})

	if os.Getenv("DEBUG") == "" {
		app.Use(func(c *bono.Context, next bono.Next) error {
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
		app.Use(bono.StaticMiddleware("www"))
	}

	app.Use(func(c *bono.Context, next bono.Next) error {
		if c.Request.URL.Query().Get("update") == "" {
			return showContent(c, next)
		} else {
			if c.Request.Method == "POST" {
				return updateContent(c, next)
			} else {
				return updateContentForm(c, next)
			}
		}
		return nil
	})

	log.Println("Running wiki")
	http.HandleFunc("/", app.Callback())
	http.ListenAndServe(":3000", nil)
	log.Println("End listening")
}

func getFile(uri string) string {
	file := uri
	if file == "/" {
		file = "/index"
	}
	file = file + ".md"
	return file
}

func showContent(c *bono.Context, next bono.Next) error {
	content, err := ioutil.ReadFile("files" + getFile(c.Request.URL.Path))
	if err != nil {
		return c.Redirect("?update=true")
	}

	markdown := blackfriday.MarkdownCommon(content)

	body, err := render(c, "read", map[string]interface{}{
		"content": template.HTML(string(markdown)),
	})
	if err != nil {
		return err
	}
	c.Response.Body = body

	return nil
}

func updateContent(c *bono.Context, next bono.Next) error {
	c.Request.ParseForm()

	filename := "files" + getFile(c.Request.URL.Path)

	content := c.Request.Form.Get("content")
	os.MkdirAll(filepath.Dir(filename), 0755)
	ioutil.WriteFile(filename, []byte(content), 0644)

	c.Set("success", "Content saved")

	body, err := render(c, "update", map[string]string{
		"content": content,
	})
	if err != nil {
		return err
	}
	c.Response.Body = body

	return nil
}

func updateContentForm(c *bono.Context, next bono.Next) error {
	contentByte, _ := ioutil.ReadFile("files" + getFile(c.Request.URL.Path))

	body, err := render(c, "update", map[string]interface{}{
		"content": string(contentByte),
	})
	if err != nil {
		return err
	}
	c.Response.Body = body
	return nil
}

func render(c *bono.Context, name string, data interface{}, _withLayout ...bool) ([]byte, error) {
	withLayout := true
	if len(_withLayout) > 0 {
		withLayout = _withLayout[0]
	}

	innerName := name
	innerData := data
	if withLayout {
		body, err := render(c, innerName, innerData, false)
		if err != nil {
			return nil, err
		}
		mainBody := string(body)

		name = "layout"
		data = map[string]interface{}{
			"Title": "Wiki",
			"Main":  template.HTML(mainBody),
		}
	}

	filename := "templates/" + name + ".html"

	var content []byte
	var err error

	if os.Getenv("DEBUG") == "" {
		content, err = Asset(filename)
	} else {
		content, err = ioutil.ReadFile(filename)
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
		return c.Request.URL.Query().Get("update") == ""
	}

	helperIsUpdate := func() bool {
		return c.Request.URL.Query().Get("update") != ""
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
