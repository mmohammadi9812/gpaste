package controller

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)


var (
	wg sync.WaitGroup
)

func getIdFromKey(ctx *gin.Context, key string) (id gocql.UUID, err error) {
	var strid string
	strid, err = rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		// key was not in cache
		err = kg.Session.Query("SELECT paste_id FROM paste.PasteKeys WHERE key = ?;", key).Scan(&id)
		if err != nil {
			return gocql.UUID{}, err
		}
	} else if err != nil {
		return gocql.UUID{}, err
	} else {
		id, err = gocql.ParseUUID(strid)
		if err != nil {
			return gocql.UUID{}, err
		}
	}
	return id, nil
}

func getPasteFromId(id gocql.UUID) (map[string]interface{}, error) {
	var s3url, ptext, userid string
	// var userid gocql.UUID
	var createdat time.Time

	err := kg.Session.Query("SELECT ptext, s3_url, user_id, created_at FROM paste.Paste WHERE id = ?", id).Scan(&ptext, &s3url, &userid, &createdat)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("error on scanning paste queries: %v", err)
	}

	return map[string]interface{}{
		"ptext":      ptext,
		"s3_url":     s3url,
		"user_id":    userid,
		"created_at": createdat,
	}, nil
}

func saveContent(ctx *gin.Context, dataType int) {
	var form any
	switch dataType {
	case PasteText:
		var tf TextForm
		if err := ctx.Bind(&tf); err != nil {
			log.Fatal(err)
		}
		form = tf
	case PasteImage:
		var imf ImageForm
		if err := ctx.ShouldBind(&imf); err != nil {
			log.Fatal(err)
		}
		form = imf
	default:
		log.Fatalln("saveContent was called with wrong anguments")
	}

	key, err := kg.GetKey()
	if err != nil {
		log.Fatal(err)
	}

	uuid := gocql.MustRandomUUID()

	done := make(chan bool)

	go (func() {
		err = rdb.Set(ctx, key, uuid.String(), 0).Err()
		if err != nil {
			done <- false // Indicate insertion failure
			return
		}

		err = kg.Session.Query("INSERT INTO paste.PasteKeys (key, paste_id, expires_at) VALUES (?, ?, ?)",
			key, uuid, nil).WithContext(ctx).Exec()

		if err != nil {
			done <- false
			return
		}

		var values = []any{uuid, dataType}
		switch dataType {
		case PasteText:
			// TODO: upload texts > 10KB to local S3
			values = append(values, form.(TextForm).Text, nil)
		case PasteImage:
			objectURL, err := putToS3(ctx, form.(ImageForm).Image)
			if err != nil {
				ctx.Set("reason", err.Error())
				ctx.Redirect(http.StatusFound, "/error.html")
			}
			values = append(values, nil, objectURL)
		}
		values = append(values, nil, time.Now(), time.Now())
		err = kg.Session.Query("INSERT INTO paste.Paste (id, ptype, ptext, s3_url, user_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			values...).WithContext(ctx).Exec()
		if err != nil {
			done <- false
			return
		}

		done <- true
	})()

	success := <-done // Wait for signal and check for success

	if success {
		ctx.Redirect(http.StatusFound, fmt.Sprintf("/%s", key))
	} else {
		ctx.Set("reason", err.Error())
		ctx.Redirect(http.StatusFound, "/error.html")
	}
}

func Init() (err error) {
	rpass := os.Getenv("REDIS_PASSWORD")
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: rpass,
		DB:       0,
	})

	accessKeyID := os.Getenv("S3_ACCESSKEY")
	secretAccessKey := os.Getenv("S3_SECRETKEY")
	mc, err = minio.New(S3_ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		return err
	}

	wg.Add(1)
	go (func() {
		kg.Init()
		wg.Done()
	})()

	wg.Wait()

	return nil
}

func Close() {
	fmt.Print("Closing redis & key generation service ...")
	rdb.Close()
	kg.Close()
}
