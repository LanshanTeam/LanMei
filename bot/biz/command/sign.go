package command

import (
	"LanMei/bot/biz/dao"
	"LanMei/bot/biz/model"
	"LanMei/bot/utils/llog"
	"context"
	"fmt"
	"math/rand"
	"time"
)

// PingCommand ping 命令，用于测试
func PingCommand() string {
	return "pong!"
}

// Sign 试试手气的命令处理
func Sign(qqId string, random bool) string {
	point := 5
	if random {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		src := r.Int31() % 1000
		addition := r.Intn(rand.Int())%5 - 2
		switch true {
		case src < 10:
			point = int(src%7+1) + addition
		case src >= 10 && src < 800:
			point = int(src%8+3) + addition
		case src >= 800 && src < 980:
			point = int(src%12+3) + addition
		case src >= 980:
			point = int(src%20+2) + addition
		}
	}
	user := &model.User{
		QQId: qqId,
	}
	err := dao.DBManager.MarkAsSigned(context.Background(), qqId)
	if err != nil {
		llog.Error("签到失败，原因为：", err)
		err = dao.DBManager.GetUserDefine(user)
		if err != nil {
			llog.Debug("获取用户信息失败！")
		}
		rank, err := dao.DBManager.GetUserRank(user)
		if err != nil {
			llog.Debug("查询排名失败！")
			rank = -1
		}
		return fmt.Sprintf("\n今天已经签到过了，明天再来吧\n目前你积分为%v\n排名第%v位", user.Point, rank)
	}
	err = dao.DBManager.AddPoint(user, point)
	if err != nil {
		llog.Error("签到失败：", err)
		return fmt.Sprintln("出错了，详见日志")
	}
	rank, err := dao.DBManager.GetUserRank(user)
	if err != nil {
		llog.Debug("查询排名失败！")
		rank = -1
	}
	response := fmt.Sprintf("\n签到成功，获得%v积分\n目前你积分为%v\n排名第%d位", point, user.Point+int64(point), rank)
	return response
}

func Rank() string {
	users, err := dao.DBManager.RankList()
	if err != nil {
		llog.Error("排行榜查询错误：", err)
		return "排行榜查询失败，详情请见日志。"
	}
	response := "\n群内积分排行榜前10名："
	for i, x := range users {
		response += fmt.Sprintf("\n第%v名：%v，总积分：%v", i+1, x.Username, x.Point)
	}
	return response
}

func SetName(qqId string, username string) string {
	err := dao.DBManager.SetName(qqId, username)
	if err != nil {
		llog.Error("更新失败：", err)
		return "更新昵称失败，详情见日志"
	}
	return fmt.Sprintf("你的昵称已成功更新为：%v", username)
}
