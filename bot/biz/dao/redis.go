package dao

import (
	"LanMei/bot/utils/llog"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisManagerImpl struct {
	client *redis.Client
}

func NewRedisManager(client *redis.Client) *RedisManagerImpl {
	return &RedisManagerImpl{client: client}
}

func (m *DBManagerImpl) MarkAsSigned(ctx context.Context, qqId string) error {
	date := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("sign:%v:%v", qqId, date)
	Exist, err := m.cacheDB.client.SetNX(ctx, key, "1", 24*time.Hour).Result()
	llog.Info("", Exist, err)
	if !Exist {
		return fmt.Errorf("今日已签到")
	}
	return nil
}

func (m *DBManagerImpl) MarkDayilyLuck(ctx context.Context, qqId string, sign int) int {
	date := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("daily_luck:%v:%v:", qqId, date)
	Exist, err := m.cacheDB.client.SetNX(ctx, key, sign, 24*time.Hour).Result()
	llog.Info("", Exist, err)
	if !Exist {
		str, _ := m.cacheDB.client.Get(ctx, key).Result()
		ans, _ := strconv.Atoi(str)
		return ans
	}
	return -1
}

func (m *DBManagerImpl) StaticWords(ctx context.Context, words map[string]int64) {
	for k, v := range words {
		err := m.cacheDB.client.HIncrBy(ctx, "wordcloud", k, v).Err()
		if err != nil {
			llog.Error("添加词条失败！", k, v)
			continue
		}
	}
}

func (m *DBManagerImpl) GetWords(ctx context.Context) map[string]int64 {
	res, err := m.cacheDB.client.HGetAll(ctx, "wordcloud").Result()
	if err != nil {
		llog.Error("查询词库失败")
		return nil
	}
	ans := make(map[string]int64)
	for k, v := range res {
		cnt, _ := strconv.ParseInt(v, 10, 64)
		ans[k] = cnt
	}
	return ans
}
