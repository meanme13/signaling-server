package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	rdb *redis.Client
	ctx = context.Background()
)

func Init(addr, password string, db int) error {
	rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return err
	}
	return nil
}

func GetClient() *redis.Client {
	return rdb
}

func SetKey(key string, value string, ttl time.Duration) error {
	c, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return rdb.Set(c, key, value, ttl).Err()
}

func GetKey(key string) (string, error) {
	c, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return rdb.Get(c, key).Result()
}

func DeleteKey(key string) error {
	c, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return rdb.Del(c, key).Err()
}

func Ctx() context.Context {
	return ctx
}
