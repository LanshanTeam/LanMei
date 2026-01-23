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

func (m *DBManagerImpl) StaticWords(ctx context.Context, words map[string]int64, groupId string) {
	key := fmt.Sprintf("wordcloud:%v", groupId)
	for k, v := range words {
		err := m.cacheDB.client.HIncrBy(ctx, key, k, v).Err()
		if err != nil {
			llog.Error("添加词条失败！", k, v)
			continue
		}
	}
}

func (m *DBManagerImpl) GetWords(ctx context.Context, groupId string) map[string]int64 {
	key := fmt.Sprintf("wordcloud:%v", groupId)
	res, err := m.cacheDB.client.HGetAll(ctx, key).Result()
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

func (m *DBManagerImpl) HasJrlp(ctx context.Context, groupId int64, userId int64) bool {
	key := fmt.Sprintf("jrlp:%v:%v:%v:qq", groupId, time.Now().Format("2006-01-02"), userId)
	res, err := m.cacheDB.client.Exists(ctx, key).Result()
	if err != nil {
		return false
	}
	if res == 0 {
		return false
	}
	return true
}

func (m *DBManagerImpl) SetJrlp(ctx context.Context, groupId int64, userId int64, lpId int64, msg string) {
	qqkey := fmt.Sprintf("jrlp:%v:%v:%v:qq", groupId, time.Now().Format("2006-01-02"), userId)
	err := m.cacheDB.client.Set(ctx, qqkey, lpId, 24*time.Hour).Err()
	if err != nil {
		llog.Error("设置今日老婆失败！", err)
	}
	msgKey := fmt.Sprintf("jrlp:%v:%v:%v:msg", groupId, time.Now().Format("2006-01-02"), userId)
	err = m.cacheDB.client.Set(ctx, msgKey, msg, 24*time.Hour).Err()
	if err != nil {
		llog.Error("设置今日老婆消息失败！", err)
	}
}

func (m *DBManagerImpl) GetJrlp(ctx context.Context, groupId int64, userId int64) (int64, string) {
	qqkey := fmt.Sprintf("jrlp:%v:%v:%v:qq", groupId, time.Now().Format("2006-01-02"), userId)
	res, err := m.cacheDB.client.Get(ctx, qqkey).Result()
	if err != nil {
		llog.Error("获取今日老婆失败！", err)
		return 0, ""
	}
	msgKey := fmt.Sprintf("jrlp:%v:%v:%v:msg", groupId, time.Now().Format("2006-01-02"), userId)
	msg, err := m.cacheDB.client.Get(ctx, msgKey).Result()
	if err != nil {
		llog.Error("获取今日老婆消息失败！", err)
		return 0, ""
	}
	ans, _ := strconv.ParseInt(res, 10, 64)
	return ans, msg
}
