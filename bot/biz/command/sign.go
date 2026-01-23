package command

import (
	"LanMei/bot/biz/dao"
	"LanMei/bot/biz/model"
	"LanMei/bot/utils/llog"
	"context"
	"fmt"
	"math/rand"
	"time"

	zero "github.com/wdvxdr1123/ZeroBot"
)

// PingCommand ping 命令，用于测试
func PingCommand() string {
	return "pong!"
}

type Event struct {
	Template string   // 句子模板
	Persons  []string // 人物
	Acts     []string // 动作
	Point    int      // 积分
}

var negativeEvents = []Event{
	{
		Template: "你被%s狠狠地%s了一顿，扣除了%v积分",
		Persons:  []string{"同学", "舍友", "学长", "学姐", "朋友"},
		Acts:     []string{"欺负", "吐槽", "蛐蛐"},
	},
	{
		Template: "你在和%s的%s中败下阵来，扣除了%v积分",
		Persons:  []string{"同学", "舍友", "学长", "学姐", "朋友"},
		Acts:     []string{"辩论", "讨论"},
	},
	{
		Template: "%s在背后对你进行了%s，你损失了%v积分",
		Persons:  []string{"同学", "朋友"},
		Acts:     []string{"背刺", "吐槽", "打小报告", "挂校园墙"},
	},
}

var positiveEvents = []Event{
	{
		Template: "你和%s一起%s，获得了%v积分",
		Persons:  []string{"同学", "舍友", "学长", "学姐", "朋友"},
		Acts:     []string{"原神", "三国杀", "鸣潮", "三角洲", "打瓦", "Go", "打篮球", "学习", "讨论代码"},
	},
	{
		Template: "%s偷偷给你%s，心里暖暖的，获得了%v积分",
		Persons:  []string{"舍友", "朋友", "暗恋对象"},
		Acts:     []string{"塞了糖", "送早餐", "点了外卖"},
	},
	{
		Template: "你和%s在食堂一起%s，聊得很开心，获得了%v积分",
		Persons:  []string{"朋友", "舍友", "学长", "学姐"},
		Acts:     []string{"吃饭", "分享", "打饭"},
	},
}

// Sign 试试手气的命令处理
func Sign(ctx *zero.Ctx, qqId string, random bool) string {
	point := 5
	if random {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		src := r.Int() % 1001
		switch true {
		case src < 20:
			point = int(src%9) - 4 // 2% -4~4
		case src >= 20 && src < 800:
			point = int(src%4) + 4 // 78% 4~8
		case src >= 800 && src < 980:
			point = int(src%5) + 8 // 18% 8~13
		case src >= 980:
			point = int(src%5) + 11 // %2 11~16
		}
	}
	user := &model.User{
		QQId:     qqId,
		Username: ctx.Event.Sender.NickName,
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
		return fmt.Sprintf("\n今天已经签到过了，请明天再来吧~\n目前你积分为%v\n排名第%v位", user.Point, rank)
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
	response := fmt.Sprintf("\n签到成功，%v。\n目前你积分为%v\n排名第%d位", getEventByPoint(point), user.Point+int64(point), rank)
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

func getEventByPoint(point int) string {
	var events []Event
	if point >= 0 {
		events = positiveEvents
	} else {
		point = -point
		events = negativeEvents
	}

	event := events[rand.Intn(len(events))]
	person := event.Persons[rand.Intn(len(event.Persons))]
	act := event.Acts[rand.Intn(len(event.Acts))]

	eventStr := fmt.Sprintf(event.Template, person, act, point)
	return eventStr
}
