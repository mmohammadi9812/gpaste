package main

import (
	"log"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
)

func customRenderer() multitemplate.Renderer {
	r := multitemplate.New()
	r.AddFromFiles("form", "web/base.html", "web/userform.html")
	r.AddFromFiles("index", "web/base.html", "web/index.html")
	r.AddFromFiles("error", "web/base.html", "web/error.html")
	r.AddFromFiles("text", "web/base.html", "web/text.html")

	return r
}

func getGin() *gin.Engine {
	r := gin.Default()

	// set trusted proxies for local development only
	r.ForwardedByClientIP = true
	err := r.SetTrustedProxies([]string{"127.0.0.1", "192.168.1.2", "10.0.0.0/8"})
	if err != nil {
		log.Fatal(err)
	}

	// serve static files (for now, it's only css)
	r.StaticFile("/styles.css", "./web/styles.css")
	r.StaticFile("/text-paste.css", "./web/text-paste.css")
	r.StaticFile("/userform.css", "./web/userform.css")
	r.StaticFile("/favicon.ico", "./web/favicon.ico")

	r.HTMLRender = customRenderer()

	// we don't need extra slashes
	r.RemoveExtraSlash = true

	r.GET("/error.html", ErrorHandler)

	r.GET("/login.html", LoginHandler)

	r.GET("/signup.html", SignUpHandler)

	// controller paths
	r.GET("/", IndexHandler)
	api := r.Group("/api")
	{
		api.POST("/create", CreateHandler)
	}

	r.GET("/:key", KeyHandler)

	return r
}
