package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	"github.com/pawalt/homelab/golinks/pkg/config"
)

//go:embed templates/*
var f embed.FS

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, config.CONNECTION_URL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v\n", err)
	}

	router := gin.Default()
	templ := template.Must(template.New("").ParseFS(f, "templates/*.tmpl"))
	router.SetHTMLTemplate(templ)

	redirects, err := refreshRedirects(conn)
	if err != nil {
		log.Fatalf("error intializing redirects: %v\n", err)
	}

	// refresh redirects every 5 min
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for {
			select {
			case <-ticker.C:
				redirects, err = refreshRedirects(conn)
				if err != nil {
					log.Printf("error refreshing redirects: %v\n", err)
				}
			}
		}
	}()

	router.GET("/_/hosts", func(c *gin.Context) {
		redirects, err = getRedirects(c.Request.Context(), conn)
		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("err refreshing redirects: %v", err))
		}

		c.HTML(http.StatusOK, "hosts.tmpl", gin.H{
			"redirects": redirects,
		})
	})

	router.POST("/_/hosts", func(c *gin.Context) {
		ctx := c.Request.Context()

		sources := c.PostFormArray("sources[]")
		targets := c.PostFormArray("targets[]")

		if len(sources) != len(targets) {
			c.String(http.StatusBadRequest, "srcs != targets")
			return
		}

		if len(sources) != 0 {
			tx, err := conn.Begin(ctx)
			if err != nil {
				c.String(http.StatusBadRequest, fmt.Sprintf("err starting txn: %v", err))
				return
			}

			_, err = tx.Exec(ctx, "TRUNCATE redirects")
			if err != nil {
				c.String(http.StatusBadRequest, fmt.Sprintf("err truncating: %v", err))
				return
			}

			insertCmd := `INSERT INTO redirects (source, target) VALUES `

			jawns := make([]string, 0, len(sources))
			for i := range sources {
				source := sources[i]
				target := targets[i]
				jawns = append(jawns, fmt.Sprintf(`('%s', '%s')`, source, target))
			}

			insertCmd += strings.Join(jawns, ",")
			_, err = tx.Exec(ctx, insertCmd)
			if err != nil {
				c.String(http.StatusBadRequest, fmt.Sprintf("err inserting: %v", err))
				return
			}

			err = tx.Commit(ctx)
			if err != nil {
				c.String(http.StatusBadRequest, fmt.Sprintf("err committing: %v", err))
				return
			}
		}

		newRedirects, err := getRedirects(ctx, conn)
		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("err getting redirects: %v", err))
			return
		}

		err = writeRedirects(newRedirects)
		if err != nil {
			c.String(http.StatusInternalServerError, "failure writing redirects: %v", err)
			return
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

		c.Redirect(http.StatusFound, redirects[longestPrefix]+retainedPath)
	})

	router.Run(":80")
}
