package main

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"
	"time"
)

var (
	ApiPort             string
	TwitterToken        string
	PostgresUrl         string
	ServerTz            string
	ServerlessSenderUrl string
	RssPollingRate      time.Duration
	TwitterPollingRate  time.Duration
	AlmanaxPollingRate  time.Duration
	SendBatchEnabled    bool
)

func ReadEnvs() {
	gopath := getEnv("GOPATH", ".")
	envDir := getEnv("ENV_DIR", gopath+"/src/github.com/dofusdude/ankama-discord-hooks")
	err := godotenv.Load(envDir + "/.env")
	if err != nil {
		log.Println("Could not find .env file, loading from env variables")
	}

	ApiPort = getEnv("API_PORT", "3000")
	if RssPollingRate, err = time.ParseDuration(getEnv("RSS_POLLING_RATE", "5m")); err != nil {
		log.Fatal("could not convert RSS_POLLING_RATE", err)
	}
	if TwitterPollingRate, err = time.ParseDuration(getEnv("TWITTER_POLLING_RATE", "5m")); err != nil {
		log.Fatal("could not convert TWITTER_POLLING_RATE", err)
	}
	if AlmanaxPollingRate, err = time.ParseDuration(getEnv("ALMANAX_POLLING_RATE", "1m")); err != nil {
		log.Fatal("could not convert ALMANAX_POLLING_RATE", err)
	}
	TwitterToken = getEnv("TWITTER_TOKEN", "undefined")
	PostgresUrl = getEnv("POSTGRES_URL", "")
	if PostgresUrl == "" {
		log.Fatal("POSTGRES_URL is not defined.")
	}
	ServerTz = getEnv("SERVER_TZ", "Europe/Berlin")
}

func TruncateText(s string, max int) string {
	if max > len(s) {
		return s
	}
	return s[:strings.LastIndex(s[:max], " ")] + " ..."
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func sliceContains[T comparable](s []T, str T) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// based on string only https://gist.github.com/bgadrian/cb8b9344d9c66571ef331a14eb7a2e80
// rewritten to be generic

type Set[T comparable] struct {
	list map[T]struct{} //empty structs occupy 0 memory
}

func (s *Set[T]) Has(v T) bool {
	_, ok := s.list[v]
	return ok
}

func (s *Set[T]) Add(v T) {
	s.list[v] = struct{}{}
}

func (s *Set[T]) Remove(v T) {
	delete(s.list, v)
}

func (s *Set[T]) Clear() {
	s.list = make(map[T]struct{})
}

func (s *Set[T]) Size() int {
	return len(s.list)
}

func NewSet[T comparable]() *Set[T] {
	s := &Set[T]{}
	s.list = make(map[T]struct{})
	return s
}

func (s *Set[T]) Slice() []T {
	var res []T
	for v := range s.list {
		res = append(res, v)
	}
	return res
}

// fp

func Map[T, U any](data []T, f func(T) U) []U {

	res := make([]U, 0, len(data))

	for _, e := range data {
		res = append(res, f(e))
	}

	return res
}
