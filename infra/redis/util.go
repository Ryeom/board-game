package redisutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Ryeom/board-game/log"
	"github.com/redis/go-redis/v9"
	"time"
)

const ForeverTTL = "-1"

func AddString(target string, key string, expireTime string, value string) {
	rdb := Client[target]
	if rdb == nil {
		return
	}

	ctx := context.Background()
	var err error

	if expireTime == ForeverTTL {
		err = rdb.Set(ctx, key, value, 0).Err()
	} else {
		ttl, parseErr := time.ParseDuration(expireTime + "s")
		if parseErr != nil {
			log.Logger.Errorf("Invalid expire time format: %s", expireTime)
			return
		}
		err = rdb.Set(ctx, key, value, ttl).Err()
	}

	if err != nil {
		log.Logger.Errorf("String Insert ERROR expire time : %s Error : %v", expireTime, err)
	}
}

func AddDefaultValue(target string, values map[string]interface{}) {
	rdb := Client[target]
	if rdb == nil {
		return
	}

	for key, value := range values {
		if IsExist(target, key) {
			continue
		}
		switch v := value.(type) {
		case string:
			AddString(target, key, ForeverTTL, v)
		case map[string]string:
			AddHash(target, key, v)
		}
	}
}

func IsExist(target string, key string) bool {
	rdb := Client[target]
	if rdb == nil {
		return false
	}
	ctx := context.Background()
	exists, err := rdb.Exists(ctx, key).Result()
	if err != nil || exists == 0 {
		return false
	}
	return true
}

func AddList(target string, key, value string) {
	rdb := Client[target]
	if rdb == nil {
		return
	}
	ctx := context.Background()
	err := rdb.RPush(ctx, key, value).Err()
	if err != nil {
		log.Logger.Errorf("List Insert ERROR : %v", err)
	}
}

func AddExpire(target string, key string, ttl int) {
	rdb := Client[target]
	if rdb == nil {
		return
	}
	ctx := context.Background()
	err := rdb.Expire(ctx, key, time.Duration(ttl)*time.Second).Err()
	if err != nil {
		log.Logger.Errorf("EXPIRE Insert ERROR : %v", err)
	}
}

func AddHash(target string, key string, value interface{}) {
	rdb := Client[target]
	if rdb == nil {
		return
	}
	ctx := context.Background()
	err := rdb.HSet(ctx, key, value).Err()
	if err != nil {
		log.Logger.Errorf("HASH Insert ERROR : %v", err)
	}
}

func RemoveList(target string, key, value string) bool {
	rdb := Client[target]
	if rdb == nil {
		return false
	}
	ctx := context.Background()
	err := rdb.LRem(ctx, key, 1, value).Err()
	if err != nil {
		log.Logger.Error(err)
		return false
	}
	return true
}

func GetString(target string, key string) (string, error) {
	rdb := Client[target] //
	if rdb == nil {
		return "", errors.New(fmt.Sprintf("redis client not found for target: %s", target))
	}
	ctx := context.Background()
	str, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		log.Logger.Errorf("Redis GetString ERROR for key %s in target %s: %v", key, target, err)
		return "", err
	}
	return str, nil
}

func GetExpireTime(target string, key string) int {
	rdb := Client[target]
	if rdb == nil {
		return -9
	}
	ctx := context.Background()
	ttl, err := rdb.TTL(ctx, key).Result()
	if err != nil {
		log.Logger.Error(err)
		return -9
	}
	return int(ttl.Seconds())
}

func GetList(target string, key string) []string {
	rdb := Client[target]
	if rdb == nil {
		return nil
	}
	ctx := context.Background()
	value, err := rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		log.Logger.Error(err)
	}
	return value
}

func GetType(target string, key string) string {
	rdb := Client[target]
	if rdb == nil {
		return ""
	}
	ctx := context.Background()
	typeStr, err := rdb.Type(ctx, key).Result()
	if err != nil {
		log.Logger.Error(err)
	}
	return typeStr
}

func ScanKeyList(target string, keyPattern string) []string {
	rdb := Client[target]
	if rdb == nil {
		return nil
	}
	ctx := context.Background()
	var (
		cursor uint64
		result []string
	)
	for {
		keys, nextCursor, err := rdb.Scan(ctx, cursor, keyPattern, 1000).Result()
		if err != nil {
			return result
		}
		result = append(result, keys...)
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
	return result
}

func GetHash(target string, key string) map[string]string {
	rdb := Client[target]
	if rdb == nil {
		return nil
	}
	ctx := context.Background()
	obj, err := rdb.HGetAll(ctx, key).Result()
	if err != nil {
		log.Logger.Error(err)
	}
	return obj
}

func GetSet(target string, key string) []string {
	rdb := Client[target]
	if rdb == nil {
		return nil
	}
	ctx := context.Background()
	obj, err := rdb.SMembers(ctx, key).Result()
	if err != nil {
		log.Logger.Error(err)
	}
	return obj
}

// UpdateHashField updates a single field of a Redis hash.
func UpdateHashField(target string, key string, field string, value interface{}) {
	rdb := Client[target]
	if rdb == nil {
		return
	}
	ctx := context.Background()
	err := rdb.HSet(ctx, key, field, value).Err()
	if err != nil {
		log.Logger.Errorf("HSet ERROR: %v", err)
	}
}

// GetHashField retrieves a single field from a Redis hash.
func GetHashField(target string, key string, field string) string {
	rdb := Client[target]
	if rdb == nil {
		return ""
	}
	ctx := context.Background()
	val, err := rdb.HGet(ctx, key, field).Result()
	if err != nil {
		log.Logger.Errorf("HGet ERROR: %v", err)
		return ""
	}
	return val
}

func Delete(target string, key string) error {
	rdb := Client[target] //
	if rdb == nil {
		return errors.New(fmt.Sprintf("redis client not found for target: %s", target)) // errors 임포트 추가
	}
	ctx := context.Background()
	err := rdb.Del(ctx, key).Err()
	if err != nil {
		log.Logger.Errorf("Redis Delete ERROR for key %s in target %s: %v", key, target, err)
		return err
	}
	return nil
}

func SaveJSON(target string, key string, value interface{}, ttl time.Duration) error {
	rdb := Client[target] //
	if rdb == nil {
		log.Logger.Errorf("SaveJSON - Redis client not found for target: %s", target) // 에러 로깅 추가
		return errors.New(fmt.Sprintf("redis client not found for target: %s", target))
	}
	ctx := context.Background()
	data, err := json.Marshal(value)
	if err != nil {
		log.Logger.Errorf("JSON Marshal ERROR: %v", err)
		return err
	}
	err = rdb.Set(ctx, key, data, ttl).Err()
	if err != nil {
		log.Logger.Errorf("Redis Set ERROR: %v", err)
		return err
	}
	return nil
}

func GetJSON(target string, key string, dest interface{}) bool {
	rdb := Client[target]
	if rdb == nil {
		log.Logger.Errorf("GetJSON - Redis client not found for target: %s", target)
		return false
	}
	ctx := context.Background()
	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false
		}
		log.Logger.Errorf("Redis Get ERROR for key %s: %v", key, err)
		return false
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		log.Logger.Errorf("JSON Unmarshal ERROR for key %s: %v", key, err)
		return false
	}
	return true
}

func AddSet(target string, key string, members ...interface{}) error {
	rdb := Client[target]
	if rdb == nil {
		return errors.New(fmt.Sprintf("redis client not found for target: %s", target))
	}
	ctx := context.Background()
	err := rdb.SAdd(ctx, key, members...).Err()
	if err != nil {
		log.Logger.Errorf("Redis SAdd ERROR for key %s in target %s: %v", key, target, err)
		return err
	}
	return nil
}

func GetSetMembers(target string, key string) ([]string, error) {
	rdb := Client[target]
	if rdb == nil {
		return nil, errors.New(fmt.Sprintf("redis client not found for target: %s", target))
	}
	ctx := context.Background()
	members, err := rdb.SMembers(ctx, key).Result()
	if err != nil {
		log.Logger.Errorf("Redis SMembers ERROR for key %s in target %s: %v", key, target, err)
		return nil, err
	}
	return members, nil
}

func RemoveSetMembers(target string, key string, members ...interface{}) error {
	rdb := Client[target]
	if rdb == nil {
		return errors.New(fmt.Sprintf("redis client not found for target: %s", target))
	}
	ctx := context.Background()
	err := rdb.SRem(ctx, key, members...).Err()
	if err != nil {
		log.Logger.Errorf("Redis SRem ERROR for key %s in target %s: %v", key, target, err)
		return err
	}
	return nil
}

func SetStringWithTTL(target string, key string, value string, ttl time.Duration) error {
	rdb := Client[target] //
	if rdb == nil {
		return errors.New(fmt.Sprintf("redis client not found for target: %s", target))
	}
	ctx := context.Background()
	err := rdb.Set(ctx, key, value, ttl).Err()
	if err != nil {
		log.Logger.Errorf("Redis SetWithTTL ERROR for key %s in target %s: %v", key, target, err)
		return err
	}
	return nil
}

func RPushList(target string, key string, values ...interface{}) error {
	rdb := Client[target]
	if rdb == nil {
		return errors.New(fmt.Sprintf("redis client not found for target: %s", target))
	}
	ctx := context.Background()
	err := rdb.RPush(ctx, key, values...).Err()
	if err != nil {
		log.Logger.Errorf("Redis RPushList ERROR for key %s in target %s: %v", key, target, err)
		return err
	}
	return nil
}

func LRangeList(target string, key string, start, stop int64) ([]string, error) {
	rdb := Client[target]
	if rdb == nil {
		return nil, errors.New(fmt.Sprintf("redis client not found for target: %s", target))
	}
	ctx := context.Background()
	result, err := rdb.LRange(ctx, key, start, stop).Result()
	if err != nil {
		log.Logger.Errorf("Redis LRangeList ERROR for key %s in target %s: %v", key, target, err)
		return nil, err
	}
	return result, nil
}

func LTrimList(target string, key string, start, stop int64) error {
	rdb := Client[target]
	if rdb == nil {
		return errors.New(fmt.Sprintf("redis client not found for target: %s", target))
	}
	ctx := context.Background()
	err := rdb.LTrim(ctx, key, start, stop).Err()
	if err != nil {
		log.Logger.Errorf("Redis LTrimList ERROR for key %s in target %s: %v", key, target, err)
		return err
	}
	return nil
}
