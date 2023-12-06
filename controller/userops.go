package controller

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"golang.org/x/crypto/bcrypt"
)

type LoginInput struct {
	Username  string `form:"email" binding:"required"`
	Password  string `form:"password" binding:"required"`
}

type UserProfile struct {
	Username  string
	FirstName string
	LastName  string
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

func authenticatUser(ctx *gin.Context, username string, passwd string) bool {
	var fetchedPassword string

	stmt := "SELECT password_hash FROM paste.User WHERE username = ?"
	err := kg.Session.Query(stmt, username).WithContext(ctx).Scan(&fetchedPassword)
	if err == gocql.ErrNotFound {
		return false
	} else if err != nil {
		log.Printf("error in selecting password_hash from database, you might wanna check: %v\n", err.Error())
		return false
	}

	pwh, err := passwdHash(passwd)
	if err != nil {
		log.Printf("error in hashing inputed password: %v\n", err.Error())
		return false
	}


	return fetchedPassword == pwh
}

func getProfileFromUsername(ctx *gin.Context, username string) (UserProfile, error) {
	var firstName, lastName string

	stmt := "SELECT first_name, last_name FROM paste.User WHERE username = ?"
	err := kg.Session.Query(stmt, username).WithContext(ctx).Scan(&firstName, &lastName)
	if err != nil {
		return UserProfile{}, fmt.Errorf("query failed: %v", err.Error())
	}

	return UserProfile{
		Username:  username,
		FirstName: firstName,
		LastName:  lastName,
	}, nil
}

func passwdHash(passwd string) (string, error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashBytes), nil
}

func saveUser(ctx *gin.Context) (err error) {
	var l LoginInput
	if err = ctx.Bind(&l); err != nil {
		return
	}

	uid := gocql.MustRandomUUID()
	hash, err := passwdHash(l.Password)
	if err != nil {
		return
	}

	err = kg.Session.Query("INSERT INTO Paste.User (id, username, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		uid, l.Username, hash, time.Now(), time.Now()).WithContext(ctx).Exec()
	if err != nil {
		return
	}

	return nil
}
