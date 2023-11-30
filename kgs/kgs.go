package kgs

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"os"
	"slices"
	"strings"

	"github.com/gocql/gocql"
	"github.com/redis/go-redis/v9"
)

const (
	keyLength = 6
	keyChars  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
)

type KGS struct {
	cluster *gocql.ClusterConfig
	Session *gocql.Session
	cache   *redis.Client

	lastPrefix string
}

func (k *KGS) Init() {
	var err error

	cuser, cpass := os.Getenv("CASSANDRA_USER"), os.Getenv("CASSANDRA_PASSWORD")
	k.cluster = gocql.NewCluster("localhost:9042")
	k.cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: cuser,
		Password: cpass,
	}
	k.cluster.Keyspace = "paste"
	k.cluster.Consistency = gocql.Quorum

	k.Session, err = k.cluster.CreateSession()
	if err != nil {
		log.Fatal(err)
	}

	rpass := os.Getenv("REDIS_PASSWORD")
	k.cache = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: rpass,
		DB:       1,
	})
	if err != nil {
		log.Fatal(err)
	}

	k.lastPrefix = ""
	log.Println("Key generation service started ...")
}

func (k *KGS) Close() {
	log.Println("Closing key generation service ...")
	k.Session.Close()
	k.cache.Close()
	os.Exit(0)
}

// GenerateKeyRange generates random key prefix
func (k *KGS) GenerateKeyRange() (string, gocql.UUID, error) {
	var prefix strings.Builder
	for i := 0; i < keyLength-1; i++ {
		prefix.WriteByte(keyChars[rand.Intn(len(keyChars))])
	}

	// Insert key range and lock row
	id := gocql.MustRandomUUID()
	err := k.Session.Query("INSERT INTO paste.KeyRanges (id, prefix, used) VALUES (?, ?, false)",
		id, prefix.String()).Exec()
	if err != nil {
		return "", gocql.UUID{}, err
	}

	// Return generated prefix
	return prefix.String(), id, nil
}

// GetKey retrieves an unused key from the cache or generates a new key range.
func (k *KGS) GetKey() (string, error) {
	var (
		exists int64
		err error
		keys []string
	)

	exists, err = k.cache.Exists(context.Background(), k.lastPrefix).Result()
	if exists == 1 {
		keys, err = k.cache.LRange(context.Background(), k.lastPrefix, 0, -1).Result()
	}
	if exists == 0 || err != nil || len(keys) == 64 {
		var (
			id     gocql.UUID
			prefix string
		)

		err := k.Session.Query("SELECT id, prefix FROM paste.KeyRanges WHERE used = false LIMIT 1 ALLOW FILTERING;").Scan(&id, &prefix)
		if err != nil {
			if errors.Is(err, gocql.ErrNotFound) {
				// Generate new key range
				prefix, id, err = k.GenerateKeyRange()
				if err != nil {
					return "", err
				}
			} else {
				return "", err
			}
		}

		k.lastPrefix = prefix
		keys = make([]string, 1)

		// Mark key range as used
		err = k.Session.Query("UPDATE paste.KeyRanges SET used = true WHERE id = ?", id.String()).Exec()
		if err != nil {
			return "", err
		}
	}

	var sb strings.Builder
	sb.WriteString(k.lastPrefix)

	var res = ""

	for i := 0; i < len(keyChars); i++ {
		lastByte := keyChars[rand.Intn(len(keyChars))]
		sb.WriteByte(lastByte)
		if slices.Contains(keys, sb.String()) {
			sb = strings.Builder{}
			sb.WriteString(k.lastPrefix)
			continue
		} else {
			res = sb.String()
			err = k.cache.RPush(context.Background(), k.lastPrefix, res).Err()
			if err != nil {
				return "", err
			}
			break
		}
	}

	if res == "" {
		log.Fatal("Couldn't generate a key")
	}

	return res, nil
}
