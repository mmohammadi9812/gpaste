package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/redis/go-redis/v9"
)

const (
	PasteText = iota
	PasteImage
)

type Form struct {
	Text string `form:"text"`
}

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	var kgs KGS
	kgs.Init()
	defer kgs.Close()

	r := gin.Default()
	r.LoadHTMLGlob("web/*") // Load html files

	// set trusted proxies for local development only
	r.ForwardedByClientIP = true
	err := r.SetTrustedProxies([]string{"127.0.0.1", "192.168.1.2", "10.0.0.0/8"})
	if err != nil {
		log.Fatal(err)
	}

	// serve static files (for now, it's only css)
	r.StaticFile("/styles.css", "./web/styles.css")

	// controller paths
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	r.POST("/create", func(c *gin.Context) {
		fmt.Println("/create post 1st line")
		var f Form
		if err := c.Bind(&f); err != nil {
			c.Redirect(http.StatusFound, "/error.html")
		}

		key, err := kgs.GetKey()
		if err != nil {
			log.Fatal(err)
		}

		err = rdb.Set(c, key, f.Text, 0).Err()
		if err != nil {
			log.Fatal(err)
		}

		go (func() {
			err = kgs.CQuery("INSERT INTO paste.Paste (id, ptype, ptext, s3_url, user_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
				gocql.MustRandomUUID(), PasteText, f.Text, nil, nil, time.Now().Unix(), time.Now().Unix())
			if err != nil {
				log.Fatal(err)
			}
		})()
	})
	if err = r.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
