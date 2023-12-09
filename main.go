package main

import (
	"log"
	"os"

	"git.sr.ht/~mmohammadi9812/gpaste/controller"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	router *gin.Engine
)

func setup() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	go (func(){
		err = controller.Init()
		if err != nil {
			log.Fatal(err)
		}
	})()

	router = getGin()
}

func main() {
	setup()

	port, ok := os.LookupEnv("GPASTE_PORT")
	if !ok {
		port = "3000"
	}

	if err := router.Run(":" + port); err != nil {
		log.Printf("error while running server: %v\n", err)
	}

	defer controller.Close()
}
