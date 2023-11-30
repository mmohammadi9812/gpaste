package controller

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"git.sr.ht/~mmohammadi9812/gpaste/kgs"
	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

const (
	S3_ENDPOINT = "localhost:9001"
	BucketName  = "images"
	BucketLoc   = "us-east-1"
)

const (
	PasteText = iota
	PasteImage
)

var (
	kg  kgs.KGS
	rdb *redis.Client
	mc  *minio.Client
)

// TODO: change log.Fatal lines, not all errors are fatal

type TextForm struct {
	Text       string `form:"text"`
	Language   string `form:"language"`
	Expiration int32  `form:"expiration"`
}

type ImageForm struct {
	Image      *multipart.FileHeader `form:"image" binding:"-"`
	Expiration float32               `form:"expiration"`
}

func IndexHandler(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "index", nil)
}

func ErrorHandler(ctx *gin.Context) {
	reason, ok := ctx.Get("reason")
	var obj any = nil
	if ok {
		obj = gin.H{
			"reason": reason.(string),
		}
	}

	ctx.HTML(http.StatusOK, "error", obj)
}

func SignUpHandler(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "form", gin.H{
		"action": "signup",
		"title":  "SignUp",
	})
}

func LoginHandler(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "form", gin.H{
		"action": "login",
		"title":  "Login",
	})
}

func CreateUsetHandler(ctx *gin.Context) {
	if err := saveUser(ctx); err != nil {
		ctx.Set("reason", err.Error())
		ctx.Redirect(http.StatusOK, "/error.html")
	} else {
		ctx.Redirect(http.StatusOK, "/login.html")
	}
}

func GetUserHandler(ctx *gin.Context) {
	// FIXME: getUser is not implemented actually
	if err := getUser(ctx); err != nil {
		ctx.Set("reason", err.Error())
		ctx.Redirect(http.StatusOK, "/error.html")
	} else {
		ctx.Redirect(http.StatusOK, "/")
	}
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

func TextHandler(ctx *gin.Context) {
	saveContent(ctx, PasteText)
}

func ImageHandler(ctx *gin.Context) {
	saveContent(ctx, PasteImage)
}

func KeyHandler(ctx *gin.Context) {
	key := ctx.Param("key")
	// redis needs context
	id, err := getIdFromKey(ctx, key)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: do we need this?
	err = kg.Session.Query("SELECT id FROM paste.Paste WHERE id = ?", id).Exec()
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

	ctx.HTML(http.StatusOK, "view", gin.H{
		"key":      key,
		"content":  values["ptext"],
		"s3_url":   values["s3_url"],
		"username": username,
		"date":     values["created_at"].(time.Time).Format(layout),
	})
}
