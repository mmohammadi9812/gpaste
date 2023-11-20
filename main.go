package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/ratelimit"
)

const (
	PasteText = iota
	PasteImage
)

const RateLimit = 25

type Form struct {
	Text string `form:"text"`
}

var (
	kgs KGS
	rdb *redis.Client
	limit ratelimit.Limiter
)

func leakBucket() gin.HandlerFunc {
	prev := time.Now()
	return func(ctx *gin.Context) {
		now := limit.Take()
		log.Printf("%v", now.Sub(prev))
		prev = now
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	kgs.Init()
	defer kgs.Close()

	limit = ratelimit.New(RateLimit)

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
	r.StaticFile("/text-paste.css", "./web/text-paste.css")
	r.StaticFile("/favicon.ico", "./web/favicon.ico")

	// we don't need extra slashes
	r.RemoveExtraSlash = true

	r.GET("/error.html", ErrorHandler)

	// controller paths
	r.GET("/", IndexHandler)
	api := r.Group("/api")
	{
		api.Use(leakBucket())
		api.POST("/create", CreateHandler)
	}

	r.GET("/:key", KeyHandler)

	if err = r.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
