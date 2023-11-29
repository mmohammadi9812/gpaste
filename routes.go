package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"git.sr.ht/~mmohammadi9812/gpaste/controller"
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

	// serve static files
	r.StaticFS("/assets", http.Dir("web"))
	r.StaticFile("/favicon.ico", "./web/favicon.ico")

	r.HTMLRender = customRenderer()

	// we don't need extra slashes
	r.RemoveExtraSlash = true


	// controller paths
	r.GET("/", controller.IndexHandler)
	r.GET("/error.html", controller.ErrorHandler)
	r.GET("/login.html", controller.LoginHandler)
	r.GET("/signup.html", controller.SignUpHandler)

	api := r.Group("/api")
	{
		create := api.Group("/create")
		{
			create.POST("/text", controller.TextHandler)
			create.POST("/image", controller.ImageHandler)
		}
	}

	r.GET("/:key", controller.KeyHandler)

	return r
}
