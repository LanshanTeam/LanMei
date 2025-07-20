package dao

import (
	"LanMei/bot/utils/llog"
	"context"
	"fmt"
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
	key := fmt.Sprintf("%v:%v", qqId, date)
	Exist, err := m.cacheDB.client.SetNX(ctx, key, "1", 24*time.Hour).Result()
	llog.Info("", Exist, err)
	if !Exist {
		return fmt.Errorf("今日已签到")
	}
	return nil
}
