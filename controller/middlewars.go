package controller

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	jwt "github.com/appleboy/gin-jwt/v2"
)

const identityKey = "sub"

func AuthMiddleware() (*jwt.GinJWTMiddleware, error) {
	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm: "User API",
		Key: []byte(os.Getenv("JWT_SECRET_KEY")),
		Timeout: time.Hour,
		MaxRefresh: time.Hour,
		IdentityKey: identityKey,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*UserProfile); ok {
				return jwt.MapClaims{
					identityKey: v.Username,
				}
			}
			return jwt.MapClaims{}
		},
		// TODO: The project readme claims this function is optional, but still not sure, check more to be sure
		// TODO: https://github.com/appleboy/gin-jwt/blob/b074f91d263e14a4c675575cc9f95b4179434190/README.md?plain=1#L313
		IdentityHandler: func(ctx *gin.Context) interface{} {
			claims := jwt.ExtractClaims(ctx)
			username := claims[identityKey].(string)
			u, err := getProfileFromUsername(ctx, username)
			if err != nil {
				return nil
			}
			return &u
		},
		Authenticator: func(ctx *gin.Context) (interface{}, error) {
			var l LoginInput
			if err := ctx.ShouldBind(&l); err != nil {
				return "", jwt.ErrMissingLoginValues
			}
			userName := l.Username
			passWd := l.Password

			if !authenticatUser(ctx, userName, passWd) {
				return nil, jwt.ErrFailedAuthentication
			}

			u, err := getProfileFromUsername(ctx, userName)
			if err != nil {
				return nil, jwt.ErrFailedAuthentication
			}

			return u, nil
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			// FIXME:
			panic("unimpleted!")
		},
	})

	if err != nil {
		log.Fatal(err)
	}

	return authMiddleware, nil
}

