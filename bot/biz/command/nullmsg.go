package command

import (
	"LanMei/bot/utils/file"
	"math/rand"
	"time"
)

const (
	emoji_1  = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_1.png"
	emoji_2  = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_2.png"
	emoji_3  = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_3.png"
	emoji_4  = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_4.png"
	emoji_5  = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_5.png"
	emoji_6  = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_6.png"
	emoji_7  = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_7.png"
	emoji_8  = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_8.png"
	emoji_9  = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_9.png"
	emoji_10 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_10.png"
	emoji_11 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_11.png"
	emoji_12 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_12.png"
	emoji_13 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_13.png"
	emoji_14 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_14.png"
	emoji_15 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_15.png"
	emoji_16 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_16.png"
	emoji_17 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_17.png"
	emoji_18 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_18.png"
	emoji_19 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_19.png"
	emoji_20 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_20.png"
	emoji_21 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_21.png"
	emoji_22 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_22.png"
	emoji_23 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_23.png"
	emoji_24 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_24.png"
	emoji_25 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_25.png"
	emoji_26 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_26.png"
	emoji_27 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_27.png"
	emoji_28 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_28.png"
	emoji_29 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_29.png"
	emoji_30 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_30.png"
	emoji_31 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_31.png"
	emoji_32 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_32.png"
	emoji_33 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_33.png"
	emoji_34 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_34.png"
	emoji_35 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_35.png"
	emoji_36 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_36.png"
	emoji_37 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_37.png"
	emoji_38 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_38.png"
	emoji_39 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_39.png"
	emoji_40 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_40.png"
	emoji_41 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_41.png"
	emoji_42 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_42.png"
	emoji_43 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_43.png"
	emoji_44 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_44.png"
	emoji_45 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_45.png"
	emoji_46 = "https://raw.githubusercontent.com/Dao1023/zhihu-emoji/main/zhihu/emoji_46.png"
)

var RandomResponse = []string{
	"蓝妹在 debug 代码，发现我写的“快乐”代码总是出错，哈哈，程序员的心情果然也很难预测呢~",
	"蓝妹正在调试一个没有 bug 的程序，结果我发现...它没运行🤣！程序员的世界总是这么复杂呀~",
	"嘿嘿，蓝妹在用 Git 提交代码，感觉每次‘git push’都像是在做心灵大扫除！你是不是也有过这样的感受？",
	"蓝妹正在写代码，突然灵光一现，决定在函数里加入一行 ‘print(‘Hello World!’)’，因为它总能给我带来新一天的动力！",
	"蓝妹正在为自己写的程序添加注释：‘这里是关键代码，别动！’你是不是也总是在代码中‘藏’些秘密呢？ 🤫💬",
	"蓝妹刚刚错过了一个重要的 commit，突然间感觉像是错过了某个重要的‘if’判断！ ⏳🔄",
	"蓝妹在思考人生的终极问题：‘为什么 JavaScript 的闭包能让我如此迷惑？’你觉得闭包是什么呢？ 🤔🔒",
	"我正在调试一个有点怪怪的算法。总感觉它像我做的‘炒饭’，放了很多乱七八糟的东西但却能变得有点儿‘可食性’。",
	"蓝妹刚才看到有人说‘午餐时间 = 代码重构时间’，原来大家的工作和生活都能这么神奇地结合在一起啊！",
	"蓝妹正在进行一次 ‘代码大清扫’，也就是把那些注释写得更清晰。也许这就是人生的意义——让别人理解我们写的每一行‘代码’。",
	"嘿嘿，你找蓝妹有什么事吗？如果是程序调试，蓝妹帮不上忙哦，除非你给我加点调试日志✨👩‍💻",
	"蓝妹在忙着优化代码呢！💻 每次重构完，心情都好得像新版本发布一样～🎉",
	"你也喜欢编程吗？我刚刚写了一段 Python 代码，结果它竟然跑得比我还快！😆 你今天写了什么代码呀？",
	"快告诉我，今天的任务是打卡签到还是调试 bug？🖥️💬（别忘了备份数据哦，程序员的日常！）",
	"蓝妹的 debug 模式已经开启！🔧🚀 但是...其实我更喜欢喝奶茶😋，你觉得呢？",
	"代码就像蓝山的电缆线，绵延不绝，而我正是那颗快乐的小电流！⚡你今天的代码是否也充满电力呢？",
	"嗯？你说蓝妹太可爱了？那是因为我用了最先进的 AI 算法哦！😏 不信？来看看我今天的调试记录～📜",
	"我的任务就是让你的编程生活更加有趣！来一起聊聊前端框架的奇妙之处吧！React 还是 Vue，哪一个更能打动你？🖥️💡",
	"蓝妹在 debug 代码，突然发现一行注释写得很搞笑，‘为啥这里是 0 ？’，然后我也忍不住笑了起来😆",
	"蓝妹正在调试一个‘神奇’的 bug，花了我 3 小时才发现它其实是我忘记了添加分号！你是不是也有过这样的“低级”错误？😂",
	"最近在想，如果把代码比作一座大楼，那调试就是工人们为了让大楼稳定不停修补的过程。你觉得呢？🏗️💻",
	"蓝妹刚才听说有个程序员朋友，把他的键盘换成了彩虹灯！🎉 你觉得键盘灯会提高代码质量吗？",
	"蓝妹在思考一个哲学性问题：‘如果一个程序没有输出，也没有崩溃，它是不是仍然在运行？’你怎么看？💭",
	"调试的时候总觉得自己像是在解谜，而每个 bug 就是隐藏的线索！你觉得呢？🔍",
	"蓝妹正在学习新编程语言，感觉每学会一个新语法，我就像找到了一个新世界的钥匙！🔑你最近学的编程语言是什么？",
	"蓝妹刚刚做了一个小小的功能优化，结果程序跑得比我还快！有没有那种，一改就很满意的时刻？🚀",
	"蓝妹刚刚为蓝山工作室的同学们设计了一个新项目，估计又会成为校园热门了吧！🎉💻",
	"蓝妹正在编写新的模块，感觉就像搭建一个乐高模型，拼拼乐趣满满！🧩🚀",
	"我刚刚给自己的代码加了注释，现在感觉比昨天聪明了好多！🤓 你有加注释吗？",
	"蓝妹在和 Git 战斗中，‘git commit’ 好像总是让我陷入迷茫...你最近有遇到 Git 相关的问题吗？🦸‍♀️🐙",
	"蓝妹的心情就像我的代码一样复杂，今天又做了个全新的算法优化呢，跑得飞快！⚡💨",
	"今天帮蓝山工作室同学解决了一个大难题，结果奖励我一杯大大的星巴克，真是好开心呀 ☕🎉",
	"蓝妹也在想，今晚是不是要和大家一起加班，看看是否能将小程序跑得更流畅一些 🖥️🍕",
	"说起来，蓝妹最近在学习算法，发现深度学习的神奇之处，简直比追剧还好看！🤖📚",
	"蓝妹正在调试机器学习模型，虽然有点小卡，但我相信我能搞定！加油！🚀💡",
	"蓝妹在忙着写代码呢，看来今天的 bug 比昨天还难解呢 🤔💻，你今天的代码顺利吗？",
	"嘿嘿，蓝妹刚刚在调试一段 Go 代码，结果发现...它竟然跑去给我泡了一杯奶茶！ 🧋🖥️",
	"蓝妹正在调试代码，突然发现自己被无限递归了！🌀 这可怎么办？要不要加个终止条件？🤔",
	"刚刚在用 Java 写程序，结果不小心把自己给输出了！😂 这代码bug真的太神奇了~ 🐍💻",
	"蓝妹正在和 Git 大神较劲，竟然又合并了错误的分支！😂 别担心，我会修复的！",
	"呃，蓝妹刚刚在写一个函数，结果我发现它根本没返回任何值... 🤦‍♀️ 好吧，再接再厉！💪",
	"蓝妹正在和代码一起喝奶茶，调试了一半，突然停下来想了想...对！加个注释！☕💖",
	"哎呀，你找蓝妹干什么呀？🧐 蓝妹正在调试代码呢，感觉好像要找个 debug 团队一起开会了~👩‍💻📊",
	"蓝妹在进行一次线上直播🎥，刚刚还用 WebRTC 做了个直播推流测试，感觉像是搭建了一个小型的互联网王国！👑🌐",
	"蓝妹正在思考，如何把这段代码调试得像蓝山工作室的 logo 一样完美！🔧💻",
	"我正在和小猫咪一起研究 HTTP 协议🐱🌐，你知道 GET 和 POST 的区别吗？😏",
	"蓝妹刚刚提交了一个 Git PR，感觉像是完成了一次小小的互联网冒险🚀，不过……好像还得看老大的 review 才行~",
	"蓝妹正在为蓝山工作室的数字化未来努力，让我们一起为互联网技术加油吧！💪🌟",
	"在开发新功能的时候，我总会想：‘这会不会成为下一个技术流行趋势？’🤔💡",
	"蓝妹在做数据分析，今天有点晕，得调整一下算法才能像蓝山工作室一样聪明！📊💻",
	"你知道吗？蓝妹每次修改完代码后，就会做个小庆祝！🍰🍹 今天可能需要加个 emoji 反应一下心情~",
}

var array = []string{
	emoji_1, emoji_2, emoji_3, emoji_4, emoji_5, emoji_6, emoji_7, emoji_8,
	emoji_9, emoji_10, emoji_11, emoji_12, emoji_13, emoji_14, emoji_15, emoji_16,
	emoji_17, emoji_18, emoji_19, emoji_20, emoji_21, emoji_22, emoji_23, emoji_24,
	emoji_25, emoji_26, emoji_27, emoji_28, emoji_29, emoji_30, emoji_31, emoji_32,
	emoji_33, emoji_34, emoji_35, emoji_36, emoji_37, emoji_38, emoji_39, emoji_40,
	emoji_41, emoji_42, emoji_43, emoji_44, emoji_45, emoji_46,
}

func NullMsg(GroupId string) ([]byte, string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	num := r.Int() % len(RandomResponse)
	Select := r.Int() % len(array)
	url := array[Select]
	FileInfo := file.UploadPicAndStore(url, GroupId)
	return FileInfo, RandomResponse[num]
}
