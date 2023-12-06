package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"git.sr.ht/~mmohammadi9812/gpaste/controller"
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

	if err := router.Run(":3000"); err != nil {
		log.Printf("error while running server: %v\n", err)
	}

	defer controller.Close()
}
