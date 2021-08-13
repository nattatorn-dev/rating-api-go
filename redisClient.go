package main

import (
	"encoding/json"
	"fmt"
	"time"

	"log"

	"github.com/go-redis/redis"
)

var (
	client = &redisClient{}
)

type redisClient struct {
	c *redis.Client
}

func handleOnConnect(conn *redis.Conn) error {
	err := conn.ClientSetName("on_connect").Err()
	if err != nil {
		log.Print("[Redis]: connect failed")
		return err
	}
	log.Print("[Redis]: connected")
	return nil
}

func initialize() *redisClient {
	redisUrl := GetEnv("REDIS_URL", "localhost:6379")

	c := redis.NewClient(&redis.Options{
		Addr:       redisUrl,
		MaxRetries: 1,
		OnConnect:  handleOnConnect,
	})

	if err := c.Ping().Err(); err != nil {
		panic("Unable to connect to redis " + err.Error())
	}

	client.c = c
	return client
}

func (client *redisClient) getKey(key string, value interface{}) error {
	val, err := client.c.Get(key).Result()
	if err == redis.Nil || err != nil {
		return err
	}
	err = json.Unmarshal([]byte(val), &value)
	if err != nil {
		return err
	}
	return nil
}

func (client *redisClient) setKey(key string, value interface{}, expiration time.Duration) error {
	cacheEntry, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = client.c.Set(key, cacheEntry, expiration).Err()
	if err != nil {
		return err
	}
	return nil
}

func (client *redisClient) setHMSet(key string, hashes map[string]interface{}) error {
	err := client.c.HMSet(key, hashes).Err()
	if err != nil {
		return err
	}
	return nil
}

func (client *redisClient) getHMKey(key string) (hashes map[string]string, err error) {
	val, err := client.c.HGetAll(key).Result()
	if err == redis.Nil || err != nil {
		return nil, err
	}

	return val, nil
}

func (client *redisClient) setSortKey(key string, score float64, member string) (int64, error) {
	val, err := client.c.ZAdd(key, redis.Z{
		Score:  score,
		Member: member,
	}).Result()

	if err != nil {
		return 0, err
	}

	return val, nil
}

func (client *redisClient) setSortIncrementBy(key string, increment float64, member string) (float64, error) {
	val, err := client.c.ZIncrBy(key, increment, member).Result()

	if err != nil {
		return 0, err
	}

	return val, nil
}

func (client *redisClient) setIncrement(key string) (int64, error) {
	val, err := client.c.Incr(key).Result()

	if err != nil {
		return 0, err
	}

	return val, nil
}

func (client *redisClient) getSortScore(key string, member string) (score float64, err error) {
	val, err := client.c.ZScore(key, member).Result()

	if err != nil {
		return 0, err
	}

	return val, nil
}

type Rank struct {
	Score  float64
	Member string
}

func (client *redisClient) getRank(key string, size int64) (rank []Rank, err error) {
	ranks, err := client.c.ZRevRangeByScoreWithScores(key, redis.ZRangeBy{
		Max:   "+inf",
		Min:   "-inf",
		Count: size,
	}).Result()

	rankMapped := []Rank{}
	for _, rank := range ranks {
		rankModel := Rank{
			Score:  rank.Score,
			Member: fmt.Sprintf("%v", rank.Member),
		}
		rankMapped = append(rankMapped, rankModel)
	}
	if err != nil {
		return nil, err
	}

	return rankMapped, nil
}

func (client *redisClient) HealthCheck() error {
	err := client.c.Ping().Err()
	if err != nil {
		return err
	}

	return nil
}
