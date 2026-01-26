# LanMei

基于 ZeroBot 和 LLoneBot 实现的 qq 机器人蓝妹

## 功能

### 默认自带插件

- ping，测试机器人是否正常工作，返回 pong!

- 签到系统，支持签到/试试手气两种方式，可以通过 `/排名` 或者 `/rank` 查看签到积分排名前十。

- 今日运势，资源来自于[丸子鱼版蓝妹](https://github.com/LanshanTeam/Lanmei-QQbot-V2)。

- 抽塔罗牌，资源来自于[丸子鱼版蓝妹](https://github.com/LanshanTeam/Lanmei-QQbot-V2)。

- 词云，功能来自于[冉神版的蓝妹](https://github.com/jizizr/LanMei)的词云，通过 rust 实现，如需本地部署，需要先执行 `cargo build --release`

- 猫猫/哈基米，获得一张猫猫/哈基米状态码图片。

- 每日一句，获得一句每日一句的句子。

- BA-LOGO，生成一张BA风格的自定义LOGO图片

- 今日老婆，从近期活跃的成员中选取一位成员作为老婆，每日刷新。

### AI 聊天

- 通过连接飞书表格集成 RAG 知识库，支持实时更新。

- 支持以群聊为单位的短暂记忆和长期记忆，长期记忆通过 LLM 压缩事件实现。

- 自判定回复，通过 judgeModel 和 plannerModel 判定是否可以介入群聊讨论。

- 意图分析，让回复更准确。

- 利用 [openserp](https://github.com/karust/openserp) 实现多步网络检索，提高回复的消息实时性。