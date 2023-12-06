package main

import (
	"log"
	"net/http"

	"git.sr.ht/~mmohammadi9812/gpaste/controller"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
)

func customRenderer() multitemplate.Renderer {
	r := multitemplate.New()
	r.AddFromFiles("form", "web/base.html", "web/userform.html")
	r.AddFromFiles("index", "web/base.html", "web/index.html")
	r.AddFromFiles("error", "web/base.html", "web/error.html")
	r.AddFromFiles("view", "web/base.html", "web/paste-view.html")

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

	authMiddleware, err := controller.AuthMiddleware()
	if err != nil {
		log.Fatalf("error while creating auth middleware: %v\n", err.Error())
	}

	// controller paths
	r.GET("/", controller.IndexPage)
	r.GET("/error.html", controller.ErrorPage)
	r.GET("/login.html", controller.LoginPage)
	r.GET("/signup.html", controller.SignUpPage)

	api := r.Group("/api")
	{
		create := api.Group("/create")
		{
			create.POST("/text", controller.TextHandler)
			create.POST("/image", controller.ImageHandler)
		}

		api.POST("/signup", controller.CreateUserHandler)
		api.POST("/login", authMiddleware.LoginHandler)
		api.GET("/refresh_token", authMiddleware.RefreshHandler)
		api.GET("/logout", authMiddleware.LogoutHandler)

		dash := api.Group("/dashboard")
		dash.Use(authMiddleware.MiddlewareFunc())
		{
			dash.GET("/:username", controller.DashboardHandler)
		}
	}

	r.GET("/:key", controller.KeyHandler)

	return r
}
