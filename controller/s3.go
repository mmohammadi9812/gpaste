package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

var (
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
