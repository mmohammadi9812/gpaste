package main

import (
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
)

const (
	PasteText = iota
	PasteImage
)

// TODO: https://github.com/envoyproxy/ratelimit or https://github.com/sethvargo/go-limiter

type Form struct {
	Text string `form:"text"`
}

var (
	kgs KGS
	rdb *redis.Client
	wg  sync.WaitGroup
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	wg.Add(1)
	go (func() {
		kgs.Init()
		wg.Done()
	})()
	defer kgs.Close()

	r := getGin()

	if err := r.Run(":3000"); err != nil {
		log.Fatal(err)
	}

	wg.Wait()
}
