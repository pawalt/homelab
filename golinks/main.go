package main

import (
	"embed"
	"github.com/gin-gonic/gin"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

//go:embed templates/*
var f embed.FS

func main () {
	router := gin.Default()
	templ := template.Must(template.New("").ParseFS(f, "templates/*.tmpl"))
	router.SetHTMLTemplate(templ)

	redirects, err := pullRedirects()
	if err != nil {
		log.Fatalln(err)
	}

	err = writeRedirects(redirects)
	if err != nil {
		log.Fatalln(err)
	}

	// refresh redirects every 5 min
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for {
			select {
			case <- ticker.C:
				redirects, err = pullRedirects()
				if err != nil {
					log.Printf("error refreshing redirects: %v", err)
				}
				err = writeRedirects(redirects)
				if err != nil {
					log.Printf("error writing redirects: %v", err)
				}
			}
		}
	}()


	router.GET("/_/hosts", func(c *gin.Context) {
		c.HTML(http.StatusOK, "hosts.tmpl", gin.H{
			"redirects": redirects,
		})
	})

	router.POST("/_/hosts", func(c *gin.Context) {
		sources := c.PostFormArray("sources[]")
		targets := c.PostFormArray("targets[]")

		if len(sources) != len(targets) {
			c.String(http.StatusBadRequest, "srcs != targets")
			return
		}

		newRedirects := make(map[string]string)
		for i, source := range sources {
			target := targets[i]

			if source == "" || target == "" {
				continue
			}

			newRedirects[source] = target
		}

		err = writeRedirects(newRedirects)
		if err != nil {
			c.String(http.StatusInternalServerError, "failure writing redirects: %v", err)
		}

		redirects = newRedirects

		c.HTML(http.StatusOK, "hosts.tmpl", gin.H{
			"redirects": redirects,
		})
	})

	router.NoRoute(func(c *gin.Context) {
		toMatch := c.Request.Host + c.Request.URL.Path
		toMatch = strings.Trim(toMatch, "/") + "/"

		if toMatch == "go/" {
			c.Redirect(http.StatusFound, "/_/hosts")
			return
		}

		longestPrefix := ""
		for rawPrefix := range redirects {
			newPrefix := rawPrefix + "/"
			if strings.HasPrefix(toMatch, newPrefix) {
				if len(rawPrefix) > len(longestPrefix) {
					longestPrefix = rawPrefix
				}
			}
		}

		if longestPrefix == "" {
			c.String(http.StatusNotFound, "not found")
			return
		}

		toStrip := c.Request.Host + c.Request.URL.String()
		sides := strings.SplitN(toStrip, longestPrefix, 2)
		retainedPath := ""
		if len(sides) == 2 {
			retainedPath = sides[1]
		}

		c.Redirect(http.StatusFound, redirects[longestPrefix] + retainedPath)
	})

	router.Run(":80")
}
