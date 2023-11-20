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

func IndexHandler(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "index.html", nil)
}

func ErrorHandler(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "error.html", nil)
}

func CreateHandler(ctx *gin.Context) {
	var f Form
	if err := ctx.Bind(&f); err != nil {
		ctx.Redirect(http.StatusFound, "/error.html")
	}

	key, err := kgs.GetKey()
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

		err = kgs.Session.Query("INSERT INTO paste.PasteKeys (key, paste_id, expires_at) VALUES (?, ?, ?)",
			key, uuid, nil).WithContext(ctx).Exec()

		if err != nil {
			done <- false
			return
		}

		err = kgs.Session.Query("INSERT INTO paste.Paste (id, ptype, ptext, s3_url, user_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			uuid, PasteText, f.Text, nil, nil, time.Now(), time.Now()).WithContext(ctx).Exec()
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
		ctx.HTML(http.StatusOK, "error.html", nil)
	}

}

func getIdFromKey(ctx *gin.Context, key string) (id gocql.UUID, err error) {
	var strid string
	strid, err = rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		// key was not in cache
		err = kgs.Session.Query("SELECT paste_id FROM paste.PasteKeys WHERE key = ?;", key).Scan(&id)
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

	err := kgs.Session.Query("SELECT ptext, s3_url, user_id, created_at FROM paste.Paste WHERE id = ?", id).Scan(&ptext, &s3url, &userid, &createdat)
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

func getUsernameFromId(id string) string {
	if id == "" {
		return "guest"
	}
	userid, err := gocql.ParseUUID(id)
	if err != nil {
		return "guest"
	}
	var u string
	err = kgs.Session.Query("SELECT username FROM paste.User WHERE id = ?", userid).Scan(&u)
	if err != nil {
		return "guest"
	}
	return u
}

func KeyHandler(ctx *gin.Context) {
	key := ctx.Param("key")
	// redis needs context
	id, err := getIdFromKey(ctx, key)
	if err != nil {
		log.Fatal(err)
	}

	err = kgs.Session.Query("SELECT id FROM paste.Paste WHERE id = ?", id).Exec()
	if err != nil {
		log.Fatal(err)
	}

	values, err := getPasteFromId(id)
	if err != nil {
		log.Fatal(err)
	}

	userid, ok := values["user_id"]
	if !ok {
		userid = ""
	}
	username := getUsernameFromId(userid.(string))

	// convert created_at field to human readable text
	layout := "2006-01-02 15:04:05"

	ctx.HTML(http.StatusOK, "text.html", gin.H{
		"key":      key,
		"content":  values["ptext"],
		"username": username,
		"date":     values["created_at"].(time.Time).Format(layout),
	})
}
