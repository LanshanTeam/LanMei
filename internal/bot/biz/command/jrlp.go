package command

import (
	"LanMei/internal/bot/biz/dao"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/bytedance/sonic"
	zero "github.com/wdvxdr1123/ZeroBot"
)

type MemberInfo struct {
	GroupID         int64  `json:"group_id"`
	UserID          int64  `json:"user_id"`
	Nickname        string `json:"nickname"`
	Card            string `json:"card"`
	CardOrNickname  string `json:"card_or_nickname"`
	Sex             string `json:"sex"`
	Age             int    `json:"age"`
	Area            string `json:"area"`
	Level           string `json:"level"`
	QQLevel         int    `json:"qq_level"`
	JoinTime        int64  `json:"join_time"`
	LastSentTime    int64  `json:"last_sent_time"`
	TitleExpireTime int64  `json:"title_expire_time"`

	Unfriendly     bool `json:"unfriendly"`
	CardChangeable bool `json:"card_changeable"`
	IsRobot        bool `json:"is_robot"`

	ShutUpTimestamp int64  `json:"shut_up_timestamp"`
	Role            string `json:"role"`
	Title           string `json:"title"`
}

var jrlpTemplate = "你今天的老婆是——\n\n【%s】"

func JrlpCommand(ctx *zero.Ctx, qqId string) (int64, string) {
	if dao.DBManager.HasJrlp(context.Background(), ctx.Event.GroupID, ctx.Event.Sender.ID) {
		lpId, msg := dao.DBManager.GetJrlp(context.Background(), ctx.Event.GroupID, ctx.Event.Sender.ID)
		return lpId, msg
	}

	json := ctx.GetGroupMemberList(ctx.Event.GroupID)
	var memberInfoResp = []MemberInfo{}
	sonic.UnmarshalString(json.Raw, &memberInfoResp)
	memberInfoResp = Skip(memberInfoResp, ctx.Event.Sender.ID)
	if len(memberInfoResp) == 0 {
		return 0, "抱歉，群里暂时没有合适的成员成为你的老婆哦~"
	}
	lp := memberInfoResp[rand.Intn(len(memberInfoResp))]
	msg := fmt.Sprintf(jrlpTemplate, lp.CardOrNickname)
	dao.DBManager.SetJrlp(context.Background(), ctx.Event.GroupID, ctx.Event.Sender.ID, lp.UserID, msg)
	return lp.UserID, msg
}

func Skip(list []MemberInfo, qqId int64) []MemberInfo {
	result := []MemberInfo{}
	for _, member := range list {
		if member.IsRobot || member.LastSentTime < time.Now().AddDate(0, 0, -3).Unix() || member.UserID == qqId {
			continue
		}
		result = append(result, member)
	}
	return result
}
