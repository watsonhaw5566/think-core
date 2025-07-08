package think

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/think-go/tg/config"
	"net/http"
	"sync"
	"time"
)

var (
	rdbInstance = sync.Map{}
)

type RSource struct {
	Addr     string // 地址
	Password string // 密码
	DB       int    // 索引
	PoolSize int    // 最大连接池
}

type tRdb struct {
	instance *redis.Client
	ctx      context.Context
}

func RDb(source ...RSource) *tRdb {
	config := RSource{
		Addr:     config.Config.GetRedisSource("default.addr").String(),
		Password: config.Config.GetRedisSource("default.password").String(),
		DB:       int(config.Config.GetRedisSource("default.db").Int()),
		PoolSize: int(config.Config.GetRedisSource("default.poolSize").Int()),
	}
	if len(source) > 0 {
		config = RSource{
			Addr:     source[0].Addr,
			Password: source[0].Password,
			DB:       source[0].DB,
			PoolSize: source[0].PoolSize,
		}
	}
	key := fmt.Sprintf("%s:%v", config.Addr, config.DB)
	if ins, ok := rdbInstance.Load(key); ok {
		return &tRdb{
			instance: ins.(*redis.Client),
			ctx:      context.Background(),
		}
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	})
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		panic(Exception{
			StateCode: http.StatusInternalServerError,
			ErrorCode: ErrorCode.RedisError,
			Message:   "Redis连接异常",
			Error:     err,
		})
	}
	rdbInstance.Store(key, rdb)
	return &tRdb{
		instance: rdb,
		ctx:      ctx,
	}
}

// ----字符串----

// Get 获取存储在键上的字符串值
func (db *tRdb) Get(key string) *redis.StringCmd {
	return db.instance.Get(db.ctx, key)
}

// Set 设置键的字符串值
func (db *tRdb) Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return db.instance.Set(db.ctx, key, value, expiration)
}

// Incr 将键的整数值增加一
func (db *tRdb) Incr(key string) *redis.IntCmd {
	return db.instance.Incr(db.ctx, key)
}

// Decr 将键的整数值减少一
func (db *tRdb) Decr(key string) *redis.IntCmd {
	return db.instance.Decr(db.ctx, key)
}

// MGet 获取多个键的值
func (db *tRdb) MGet(keys ...string) *redis.SliceCmd {
	return db.instance.MGet(db.ctx, keys...)
}

// MSet 同时设置多个键的值
func (db *tRdb) MSet(values ...interface{}) *redis.StatusCmd {
	return db.instance.MSet(db.ctx, values...)
}

// Del 删除一个或多个键
func (db *tRdb) Del(keys ...string) *redis.IntCmd {
	return db.instance.Del(db.ctx, keys...)
}

// ----列表----

// LRange 获取列表中指定范围内的元素
func (db *tRdb) LRange(key string, start, end int64) *redis.StringSliceCmd {
	return db.instance.LRange(db.ctx, key, start, end)
}

// LPush 将一个或多个值插入到列表头部
func (db *tRdb) LPush(key string, values ...interface{}) *redis.IntCmd {
	return db.instance.LPush(db.ctx, key, values...)
}

// RPush 将一个或多个值插入到列表尾部
func (db *tRdb) RPush(key string, values ...interface{}) *redis.IntCmd {
	return db.instance.RPush(db.ctx, key, values...)
}

// LPop 移除并返回列表的第一个元素
func (db *tRdb) LPop(key string) *redis.StringCmd {
	return db.instance.LPop(db.ctx, key)
}

// RPop 移除并返回列表的最后一个元素
func (db *tRdb) RPop(key string) *redis.StringCmd {
	return db.instance.RPop(db.ctx, key)
}

// ----集合----

// SAdd 向集合添加一个或多个成员
func (db *tRdb) SAdd(key string, members ...interface{}) *redis.IntCmd {
	return db.instance.SAdd(db.ctx, key, members...)
}

// SMembers 返回集合中的所有成员
func (db *tRdb) SMembers(key string) *redis.StringSliceCmd {
	return db.instance.SMembers(db.ctx, key)
}

// SIsMember 判断成员是否存在于集合中
func (db *tRdb) SIsMember(key string, member interface{}) *redis.BoolCmd {
	return db.instance.SIsMember(db.ctx, key, member)
}

// SRem 移除集合中的一个或多个成员
func (db *tRdb) SRem(key string, members ...interface{}) *redis.IntCmd {
	return db.instance.SRem(db.ctx, key, members...)
}

// ----有序集合----

// ZAdd 向有序集合添加一个或多个成员，或者更新其分数
func (db *tRdb) ZAdd(key string, members ...redis.Z) *redis.IntCmd {
	return db.instance.ZAdd(db.ctx, key, members...)
}

// ZRange 返回有序集合中指定范围的成员
func (db *tRdb) ZRange(key string, start, end int64) *redis.StringSliceCmd {
	return db.instance.ZRange(db.ctx, key, start, end)
}

// ZRevRange 按分数从高到低返回有序集合中指定范围的成员
func (db *tRdb) ZRevRange(key string, start, end int64) *redis.StringSliceCmd {
	return db.instance.ZRevRange(db.ctx, key, start, end)
}

// ZRem 移除有序集合中的一个或多个成员
func (db *tRdb) ZRem(key string, members ...interface{}) *redis.IntCmd {
	return db.instance.ZRem(db.ctx, key, members...)
}

// ----哈希表----

// HGet 获取存储在哈希表中指定字段的值
func (db *tRdb) HGet(key, field string) *redis.StringCmd {
	return db.instance.HGet(db.ctx, key, field)
}

// HSet 设置哈希表中字段的值
func (db *tRdb) HSet(key, field string, value interface{}) *redis.IntCmd {
	return db.instance.HSet(db.ctx, key, field, value)
}

// HGetAll 获取哈希表中所有的字段和值
func (db *tRdb) HGetAll(key string) *redis.MapStringStringCmd {
	return db.instance.HGetAll(db.ctx, key)
}

// HDel 删除哈希表中的一个或多个字段
func (db *tRdb) HDel(key string, fields ...string) *redis.IntCmd {
	return db.instance.HDel(db.ctx, key, fields...)
}

// ----发布/订阅----

// Publish 向指定频道发送消息
func (db *tRdb) Publish(channel string, message interface{}) *redis.IntCmd {
	return db.instance.Publish(db.ctx, channel, message)
}

// Subscribe 订阅一个或多个频道
func (db *tRdb) Subscribe(channel ...string) *redis.PubSub {
	return db.instance.Subscribe(db.ctx, channel...)
}

// Eval 执行 Lua 脚本
func (db *tRdb) Eval(script string, keys []string, args ...interface{}) *redis.Cmd {
	return db.instance.Eval(db.ctx, script, keys, args...)
}
