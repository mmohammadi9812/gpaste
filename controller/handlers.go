package controller

import (
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"git.sr.ht/~mmohammadi9812/gpaste/kgs"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"

	jwt "github.com/appleboy/gin-jwt/v2"
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

func IndexPage(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "index", nil)
}

func ErrorPage(ctx *gin.Context) {
	reason, ok := ctx.Get("reason")
	var obj any = nil
	if ok {
		obj = gin.H{
			"reason": reason.(string),
		}
	}

	ctx.HTML(http.StatusOK, "error", obj)
}

func SignUpPage(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "form", gin.H{
		"action": "signup",
		"title":  "SignUp",
	})
}

func LoginPage(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "form", gin.H{
		"action": "login",
		"title":  "Login",
	})
}

func CreateUserHandler(ctx *gin.Context) {
	if err := saveUser(ctx); err != nil {
		ctx.Set("reason", err.Error())
		ctx.Redirect(http.StatusOK, "/error.html")
	} else {
		ctx.Redirect(http.StatusOK, "/login.html")
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

func DashboardHandler(ctx *gin.Context) {
	claims := jwt.ExtractClaims(ctx)
	username := claims[identityKey]
	log.Println(username)

	// FIXME: render dashboard
	panic("unimplemtnted")
}
