package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Email    string `form:"email" binding:"required"`
	Password string `form:"password" binding:"required"`
}

var (
	wg           sync.WaitGroup
	BucketPolicy = map[string]any{
		"Version": "2012-10-17",
		"Statement": []any{
			map[string]any{
				"Action":    [1]string{"s3:GetObject"},
				"Effect":    "Allow",
				"Principal": "*",
				"Resource":  [1]string{fmt.Sprintf("arn:aws:s3:::%s/*", BucketName)},
			},
		},
	}
)

func ensureBucket(ctx context.Context) (err error) {
	err = mc.MakeBucket(ctx, BucketName, minio.MakeBucketOptions{Region: BucketLoc})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, err := mc.BucketExists(ctx, BucketName)
		if !exists || err != nil {
			return err
		}
	} else {
		// set lock for new bucket
		objectRetentionMode := minio.Compliance
		lockValidity := uint(30)
		lockUnit := minio.Days
		err = mc.SetObjectLockConfig(ctx, BucketName, &objectRetentionMode, &lockValidity, &lockUnit)
		if err != nil {
			return err
		}

		// set policy for new bucket
		policy, err := json.Marshal(BucketPolicy)
		if err != nil {
			return err
		}

		err = mc.SetBucketPolicy(ctx, BucketName, string(policy))
		if err != nil {
			return err
		}
	}

	return nil
}

func getObjectUrl(objectName string) (string, error) {
	return url.JoinPath(fmt.Sprintf("http://%s", S3_ENDPOINT), BucketName, objectName)
}

func putToS3(ctx *gin.Context, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	if err = ensureBucket(ctx); err != nil {
		return "", err
	}

	uid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	objectName := uid.String() + "-" + file.Filename

	// Upload the file with PutObject
	_, err = mc.PutObject(ctx, BucketName, objectName, src, file.Size, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return "", err
	}

	return getObjectUrl(objectName)
}

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

func getUsernameFromId(id string) string {
	if id == "" {
		return "guest"
	}
	userid, err := gocql.ParseUUID(id)
	if err != nil {
		return "guest"
	}
	var u string
	err = kg.Session.Query("SELECT username FROM paste.User WHERE id = ?", userid).Scan(&u)
	if err != nil {
		return "guest"
	}
	return u
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

func passwdHash(passwd string) (string, error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashBytes), nil
}

func saveUser(ctx *gin.Context) (err error) {
	var u User
	if err = ctx.Bind(&u); err != nil {
		return
	}

	uid := gocql.MustRandomUUID()
	hash, err := passwdHash(u.Password)
	if err != nil {
		return
	}

	err = kg.Session.Query("INSERT INTO Paste.User (id, username, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		uid, u.Email, hash, time.Now(), time.Now()).WithContext(ctx).Exec()
	if err != nil {
		return
	}

	return nil
}

func getUser(ctx *gin.Context) (err error) {
	var u User
	if err = ctx.Bind(&u); err != nil {
		return
	}

	//TODO: finish this function
	return fmt.Errorf("not implemented")
}
