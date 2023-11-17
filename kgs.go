package main

import (
	"errors"
	"github.com/dgraph-io/ristretto"
	"github.com/gocql/gocql"
	"log"
	"math/rand"
	"slices"
	"strings"
)

const (
	keyLength = 6
	keyChars  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
)

type KGS struct {
	cluster *gocql.ClusterConfig
	session *gocql.Session
	cache   *ristretto.Cache

	lastPrefix string
}

func (k *KGS) Init() {
	var err error

	k.cluster = gocql.NewCluster("localhost:9042")
	k.cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: "cassandra",
		Password: "cassandra",
	}
	k.cluster.Keyspace = "paste"
	k.cluster.Consistency = gocql.One
	k.session, err = k.cluster.CreateSession()
	if err != nil {
		log.Fatal(err)
	}

	k.cache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e3,
		MaxCost:     1 << 4,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatal(err)
	}

	k.lastPrefix = ""
}

func (k *KGS) Close() {
	k.session.Close()
	k.cache.Close()
}

func (k *KGS) CQuery(stmt string, values ...interface{}) error {
	if k.session.Closed() {
		return errors.New("current session is closed")
	}

	return k.session.Query(stmt, values...).Exec()
}

// GenerateKeyRange generates random key prefix
func (k *KGS) GenerateKeyRange() (string, gocql.UUID, error) {
	var prefix strings.Builder
	for i := 0; i < keyLength-1; i++ {
		prefix.WriteByte(keyChars[rand.Intn(len(keyChars))])
	}

	// Insert key range and lock row
	id := gocql.MustRandomUUID()
	err := k.session.Query("INSERT INTO key_ranges (id, prefix, used) VALUES (?, ?, false)",
		id, prefix.String()).Exec()
	if err != nil {
		return "", gocql.UUID{}, err
	}

	// Return generated prefix
	return prefix.String(), id, nil
}

// GetKey retrieves an unused key from the cache or generates a new key range.
func (k *KGS) GetKey() (string, error) {
	keys, ok := k.cache.Get(k.lastPrefix)
	if !ok || len(keys.([]string)) == 64 {
		var (
			id gocql.UUID
			prefix string
		)

		err := k.session.Query("SELECT id, prefix FROM key_ranges WHERE used = false LIMIT 1 ALLOW FILTERING;").Scan(&id, &prefix)
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
		err = k.session.Query("UPDATE key_ranges SET used = true WHERE id = ?", id.String()).Exec()
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
		if slices.Contains(keys.([]string), sb.String()) {
			sb = strings.Builder{}
			sb.WriteString(k.lastPrefix)
			continue
		} else {
			res = sb.String()
			keys = append(keys.([]string), res)
			k.cache.Set(k.lastPrefix, keys.([]string), int64(len(keys.([]string))))
			break
		}
	}

	if res == "" {
		log.Fatal("Couldn't generate a key")
	}

	return res, nil
}
