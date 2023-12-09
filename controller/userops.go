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
	Username string `form:"email" binding:"required"`
	Password string `form:"password" binding:"required"`
}

type SignupInput struct {
	Email string `form:"email" binding:"required"`
	Password string `form:"password" binding:"required"`
	FirstName string `form:"first-name"`
	LastName  string `form:"last-name"`
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
	err = kg.Session.Query("SELECT username FROM paste.User WHERE id = ? ALLOW FILTERING;", userid).Scan(&u)
	if err != nil {
		return "guest"
	}
	return u
}

func getUserIdFromName(username string) string {
	if username == "" {
		return ""
	}

	var id string
	err := kg.Session.Query("SELECT id FROM paste.User WHERE username = ? ALLOW FILTERING;", username).Scan(&id)
	if err != nil {
		return ""
	}

	return id
}

func authenticatUser(ctx *gin.Context, username string, passwd string) bool {
	var fetchedPassword string

	stmt := "SELECT password_hash FROM paste.User WHERE username = ? ALLOW FILTERING;"
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

	stmt := "SELECT first_name, last_name FROM paste.User WHERE username = ? ALLOW FILTERING;"
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
	var si SignupInput
	if err = ctx.ShouldBind(&si); err != nil {
		return
	}

	uid := gocql.MustRandomUUID()
	hash, err := passwdHash(si.Password)
	if err != nil {
		return
	}

	stmt := "INSERT INTO paste.User (id, username, password_hash, first_name, last_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)"
	err = kg.Session.Query(stmt, uid, si.Email, hash, si.FirstName, si.LastName, time.Now(), time.Now()).WithContext(ctx).Exec()
	if err != nil {
		return
	}

	return nil
}

type Paste struct {
	PasteType int
	Key       string
	Text      string
	S3Url     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func getUserPastes(ctx *gin.Context, username string) ([]Paste, error) {
	// TODO: paging
	// FIXME: change user_id in paste.Paste table to username
	userid := getUserIdFromName(username)
	if userid == "" {
		return []Paste{}, fmt.Errorf("couldn't find this username: %v", username)
	}
	stmt := "SELECT * FROM paste.Paste WHERE user_id = ? ALLOW FILTERING;"
	iter := kg.Session.Query(stmt, userid).WithContext(ctx).Iter()

	var pastes []Paste
	for {
		row := make(map[string]any)
		if !iter.MapScan(row) {
			break
		}

		pid := row["id"].(gocql.UUID)
		var Key string

		err := kg.Session.Query("SELECT key FROM paste.PasteKeys WHERE paste_id = ? ALLOW FILTERING;", pid).WithContext(ctx).Scan(&Key)
		if err != nil {
			return []Paste{}, fmt.Errorf("couldn't find associated key to the paste: %v", err.Error())
		}

		var (
			PasteType = row["ptype"].(int)
			Text      = row["ptext"].(string)
			S3Url     = row["s3_url"].(string)
			CreatedAt = row["created_at"].(time.Time)
			UpdatedAt = row["updated_at"].(time.Time)
			paste     = Paste{PasteType, Key, Text, S3Url, CreatedAt, UpdatedAt}
		)

		pastes = append(pastes, paste)
	}

	return pastes, nil
}
