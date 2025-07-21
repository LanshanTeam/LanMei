package file

import (
	"LanMei/bot/config"
	"LanMei/bot/utils/llog"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
)

// 从万神写的蓝妹里面复制的
const (
	TheFool           = "https://i0.hdslb.com/bfs/article/df310acdd2ee8ea7ab953d8e5aadcb03a3a91e2b.png"
	TheMagician       = "https://i0.hdslb.com/bfs/article/93da381c563c2100930f98f017bf4dcfbe7692b6.png"
	TheHighPriestess  = "https://i0.hdslb.com/bfs/article/d7927d098960aaff7153c1b0e6c9f2386d6cd210.png"
	TheEmpress        = "https://i0.hdslb.com/bfs/article/5f059bfeafc5e77b8f87f83b4e657612ad016359.png"
	TheEmperor        = "https://i0.hdslb.com/bfs/article/8317599bd60de378ec69f4da552107cb77f24da6.png"
	TheHierophant     = "https://i0.hdslb.com/bfs/article/818a1629de5acb76b5d7838c0a192022134a9952.png"
	TheLovers         = "https://i0.hdslb.com/bfs/article/571cf1cd163ccf650d5acdeed366f14f19f07228.png"
	TheChariot        = "https://i0.hdslb.com/bfs/article/ea805a56e7983f4404099cc3a2f95c7f17baad9c.png"
	Strength          = "https://i0.hdslb.com/bfs/article/9d5e61270fd0d737439bf7b6ee830beea55d4387.png"
	TheHermit         = "https://i0.hdslb.com/bfs/article/9d906fadce00a399a11a98cbc30245036d7a8bd0.png"
	TheWheelOfFortune = "https://i0.hdslb.com/bfs/article/b4290041ab32c887e0247e22b2870db0b688dee4.png"
	Justice           = "https://i0.hdslb.com/bfs/article/aee2caf310c0e98cf3741a76ca49b82280599422.png"
	TheHangedMan      = "https://i0.hdslb.com/bfs/article/632f7ccf8a879bb6334e4fdb28298b0fec34f63b.png"
	Death             = "https://i0.hdslb.com/bfs/article/9220b3457b92594ba877365557f7dc2500277796.png"
	Temperance        = "https://i0.hdslb.com/bfs/article/37bc55a5205e99cf47b7c01a4800c5a34296f211.png"
	TheDevil          = "https://i0.hdslb.com/bfs/article/a8d974a92acdbb486d7fe1e997acfb69f18e6152.png"
	TheTower          = "https://i0.hdslb.com/bfs/article/fa11d2524c11361545f9edc1c37ae863c347d71b.png"
	TheStar           = "https://i0.hdslb.com/bfs/article/538110085aff8a95ccce5278a16fd8e9de6ce427.jpg"
	TheMoon           = "https://i0.hdslb.com/bfs/article/feffa2404471aa95279e42a370a5f0613da898ef.jpg"
	TheSun            = "https://i0.hdslb.com/bfs/article/6e9ee0d2d0c1f57fe221eac874ca889b545c8dcd.jpg"
	Judgement         = "https://i0.hdslb.com/bfs/article/8b501ec2089a0f9a8b30dfdb2f5e4a34df1531fc.jpg"
	TheWorld          = "https://i0.hdslb.com/bfs/article/57050824c91f20b4849fa141d17999201dbed765.jpg"
)

var _TheFool = [2]string{"憧憬自然的地方、毫无目的地前行、喜欢尝试挑战新鲜事物、四处流浪。明知是毫无意义的冒险，错误的选择及失败的结果，却一意孤行，盲目地追求梦想而完全忽略现实；好冒险、寻梦人、不拘泥于传统的观念、自由奔放、一切从基础出发、四处流浪。自由恋爱、不顾及他人看法、以独特的方式获得成功、轻易坠入爱河、浪漫多彩的爱情、独特的恋人、等待交往机会。工作上具冒险心、追求新奇。热衷于事业或学业、以独特的方式取得意外的收获、由于好奇心对当前的学业产生浓厚的兴趣、把握重点、寻求捷径、倾向于自由的工作氛围、适合艺术类工作或从事自由职业。健康状况佳。旅行有意外收获。美好的梦想。", "冒险的行动，追求可能性，重视梦想，无视物质的损失，离开家园，过于信赖别人，为出外旅行而烦恼。心情空虚、轻率的恋情、无法长久持续的融洽感、不安的爱情的旅程、对婚姻感到束缚、彼此忽冷忽热、不顾众人反对坠入爱河、为恋人的负心所伤、感情不专一。工作缺乏稳定性、无责任。成绩一落千丈、没有耐心、行事缺乏计划、经常迟到、猜题错误导致考试失利、考前突击无法为你带来太大的效果。因不安定的生活而生病。不能放心的旅行。不能下决心、怪癖。不切实际。"}
var _TheMagician = [2]string{"事情的开始，行动的改变，熟练的技术及技巧，贯彻我的意志，运用自然的力量来达到野心。", "意志力薄弱，起头难，走入错误的方向，知识不足，被骗和失败。"}
var _TheHighPriestess = [2]string{"开发出内在的神秘潜力，前途将有所变化的预言，深刻的思考，敏锐的洞察力，准确的直觉。", "过于洁癖，无知，贪心，目光短浅，自尊心过高，偏差的判断，有勇无谋，自命不凡。"}
var _TheEmpress = [2]string{"幸福，成功，收获，无忧无虑，圆满的家庭生活，良好的环境，美貌，艺术，与大自然接触，愉快的旅行，休闲。", "不活泼，缺乏上进心，散漫的生活习惯，无法解决的事情，不能看到成果，耽于享乐，环境险恶，与家人发生纠纷。"}
var _TheEmperor = [2]string{"光荣，权力，胜利，握有领导权，坚强的意志，达成目标，父亲的责任，精神上的孤单。", "幼稚，无力，独裁，撒娇任性，平凡，没有自信，行动力不足，意志薄弱，被支配。"}
var _TheHierophant = [2]string{"援助，同情，宽宏大量，可信任的人给予的劝告，良好的商量对象，得到精神上的满足，遵守规则，志愿者。信心十足，能正确理解事物本质，工作上外来压力过多，使你有被束缚的感觉。寻找新的工作方法，尽管会面对很大的阻力，但结果会证明这样做是值得的。爱情上屈从于他人的压力，只会按照对方的要求来盲目改变自己，自以为这是必要的付出，其实不过是被迫的选择。伴侣也不会对你保持忠诚，并很难满足双方真实的需要。", "错误的讯息，恶意的规劝，上当，援助被中断，愿望无法达成，被人利用，被放弃。事业上多了些灵活的态度，不再刻板遵循旧有的方式，勇于创新形成自己独特的理念，为自己的真实想法而活、而工作。感情上开始正视自己对感情的真实感受与做法，尽管依旧会听取对方的意见，但以不会全盘接受。当你感到无法接受对方的意见时，会及时与其沟通，找出改善关系的做法。"}
var _TheLovers = [2]string{"撮合，爱情，流行，兴趣，充满希望的未来，魅力，增加朋友。感情和肉体对爱的渴望，它暗示恋情将向彼此关系更亲密的方向发展。事业上将面临重大的抉择，它将关系到你的未来前途。", "禁不起诱惑，纵欲过度，反覆无常，友情变淡，厌倦，争吵，华丽的打扮，优柔寡断。感情上表现幼稚，对成长虽有期待与希望，却希望永远躲避危险，逃避责任。事业上总保持着很高的戒心，让人感到很不舒服，不愿同你合作。"}
var _TheChariot = [2]string{"努力而获得成功，胜利，克服障碍，行动力，自立，尝试，自我主张，年轻男子，交通工具，旅行运大吉。事业上显示出才能，办事卓有成效。自信而富理智的你将让客户更有信心，愿意与你共同合作。在感情上正在努力控制自己的情绪，而且控制得很好，这让你的感情发展得更顺利。", "争论失败，发生纠纷，阻滞，违返规则，诉诸暴力，顽固的男子，突然的失败，不良少年，挫折和自私自利。放弃以往在事业上所坚持的，结局将会更加完 美。感情上失去方向，你已经没有以往的冷静，这让对方在心中产生了不信任感，也许你要反省一下自己的所作所为了。"}
var _Strength = [2]string{"大胆的行动，有勇气的决断，新发展，大转机，异动，以意志力战胜困难，健壮的女人。\n在事业上你不断突破自我，上司和客户都对你有充分的信心，成就接踵而来。在爱情上，你将发展一段真正亲密的感情，你们全心投入，相互倾诉，丝毫没有距离感。", "胆小，输给强者，经不起诱惑，屈服在权威与常识之下，没有实践便告放弃，虚荣，懦弱，没有耐性。内心的恐惧使你畏首畏尾，进而遭遇事业的瓶颈，感到失去了自信。在爱情上患得患失，失去清醒的判断。"}
var _TheHermit = [2]string{"隐藏的事实，个别的行动，倾听他人的意见，享受孤独，自己的丢化，有益的警戒，年长者，避开危险，祖父，乡间生活。你在事业黄金时期引退，旁人都不了解这不过是你在为下一次黄金时期的到来进行休息。感情方面你将深刻思考自己在这段感情中的角色和地位，并探索彼此之间的关系。", "无视警，憎恨孤独，自卑，担心，幼稚思想，过于慎重导致失败，偏差，不宜旅行。在事业中过多的投入已经让你不愿面对其它事情，因而事业有了突破性的进展。在感情方面，用工作繁忙来逃避这段感情的发展，对伴侣态度冷淡，因为害怕感情的发展而在关键时刻退缩，使对方心寒。"}
var _TheWheelOfFortune = [2]string{"关键性的事件，有新的机会，因的潮流，环境的变化，幸运的开端，状况好转，问题解决，幸运之神降临。命运之轮正转到了你人生最低迷的时刻，也许你有些无法接受，但是若能以平常心来看待，这无疑是你成长的最好时机，需要认真面对。感情方面所受到的挫折近乎让你崩溃，然而你还在不断努力。虽然你面前是无数的荆棘，但坚持过去将是平坦的大道。你会发现以前所付出的无谓努力，而今反而成了你前进的动力，先前的付出终于有了回报。命运之轮是由命运女神转动的，所以你俩之前的风风雨雨都将过去，关系将进入稳定的发展阶段。", "边疆的不行，挫折，计划泡汤，障碍，无法修正方向，往坏处发展，恶性循环，中断。"}
var _Justice = [2]string{"公正、中立、诚实、心胸坦荡、表里如一、身兼二职、追求合理化、协调者、与法律有关、光明正大的交往、感情和睦。事业上你不会有其它太多的感觉，只是按照以前的计划认真地执行。你对感情生活相当满意，对于你的选择对方都是接受的态度。", "失衡、偏见、纷扰、诉讼、独断专行、问心有愧、无法两全、表里不一、男女性格不合、情感波折、无视社会道德的恋情。长时间的压抑使你在事业最关键的时刻倒下了，需要认真修整一番才能再次前进。感情上你一直忍让着，然而这次你却爆发了，开始指责对方的不是，你们的感情将会有很大的波折。"}
var _TheHangedMan = [2]string{"接受考验、行动受限、牺牲、不畏艰辛、不受利诱、有失必有得、吸取经验教训、浴火重生、广泛学习、奉献的爱。当牌面正立时，你的事业会有短暂的停顿，但你很清楚其中的原因，再次确认自己的目标，做好出发的准备。感情上同样需要反省的时间，你对爱情的牺牲对会给对方很大的触动，也会成为你们关系发展的催化剂。", "无谓的牺牲、骨折、厄运、不够努力、处于劣势、任性、利己主义者、缺乏耐心、受惩罚、逃避爱情、没有结果的恋情。当牌面倒立时，事业上缺乏远见，迷失了努力的目标。感情上你没有了为对方付出的念头，而对方对你的态度依旧，这使你更想逃避。你已经忽略了内心深处正确的判断力，这让你开始遇到很多失败。"}
var _Death = [2]string{"失败、接近毁灭、生病、失业、维持停滞状态、持续的损害、交易停止、枯燥的生活、别离、重新开始、双方有很深的鸿沟、恋情终止。事业上你将放弃一些得到的利益，并获得全新的发展机会。在感情上，你将会发生深刻的变化，将开始新的阶段，接受事实你们会有更加美好的旅程。", "抱有一线希望、起死回生、回心转意、摆脱低迷状态、挽回名誉、身体康复、突然改变计划、逃避现实、斩断情丝、与旧情人相逢。事业上你在试图“两全其美”，希望能够发生奇迹。在感情上，对方已经接受了改变，而你却在逃避现实，你俩的距离正在越来越大。"}
var _Temperance = [2]string{"单纯、调整、平顺、互惠互利、好感转为爱意、纯爱、深爱。你在事业上小心翼翼，因为处事理智让你的同事感到十分放心。当下你们的感情简简单单，一切都是这么的单纯、平静，正是因为彼此的沟通才让这段感情之路如此通畅。", "消耗、下降、疲劳、损失、不安、不融洽、爱情的配合度不佳。在事业上，你陷入了朝令夕改的怪圈，不妨效仿一下愚人勇往直前，或许能够取得更大的成功。感情上彼此虽然还在不断尝试着沟通，但每次之后总是感觉没有收获，正因为如此你们之间的距离才会越拉越大。"}
var _TheDevil = [2]string{"被束缚、堕落、生病、恶意、屈服、欲望的俘虏、不可抗拒的诱惑、颓废的生活、举债度日、不可告人的秘密、私密恋情。你将在事业中得到相当大的名声与财富，你心中的事业就是一切，财富就是你的目标。感情上你们开始被彼此束缚，却不希望改善这种关系，情愿忍受彼此的牵连和不满。", "逃离拘束、解除困扰、治愈病痛、告别过去、暂停、别离、拒绝诱惑、舍弃私欲、别离时刻、爱恨交加的恋情。事业上理性开始支配欲望，找到真正值得努力的目标。感情上开始尝试与对方进行沟通，这让你俩的感情更加牢固。"}
var _TheTower = [2]string{"破产、逆境、被开除、急病、致命的打击、巨大的变动、受牵连、信念崩溃、玩火自焚、纷扰不断、突然分离，破灭的爱。事业上的困难显而易见，回避不是办法，要勇于挑战，尽管它貌似强大。在感情方面，突然的改变让你陷入深深的痛苦中，接受改变可以让你或你们双方在未来的人生旅途中走得更好。", "困境、内讧、紧迫的状态、状况不佳、趋于稳定、骄傲自大将付出代价、背水一战、分离的预感、爱情危机。事业上开始有稳定的迹象，你不要盲目抵抗改变的发生，这只会导致更大的改变，无论你如何抵抗，改变终究会发生。在感情上双方的情绪终于平静下来，虽然沟通上还有些困难，但不会有太大的变化了，也许你做些让步，会让你们的感情更融洽。"}
var _TheStar = [2]string{"前途光明、充满希望、想象力、创造力、幻想、满足愿望、水准提高、理想的对象、美好的恋情。代表当你在事业上得到希望的能量时，前途会无比光明。在感情方面，你对自己很有信心，对两人的关系也抱有乐观的态度，相信自己能把握主动权，并努力追求对方，你们很可能就是命中注定的那一对。", "挫折、失望、好高骛远、异想天开、仓皇失措、事与愿违、工作不顺心、情况悲观、秘密恋情、缺少爱的生活。在事业上，你不要全部依靠别人的给予，因为你还有希望在心中燃烧，只有靠自己才有真正的发展动力。感情方面你俩无法彼此信任，感觉无法把自己托付给对方，也许你们退一步，都冷静一下就能找出解决问题的途径，因为答案就在你们的心中。"}
var _TheMoon = [2]string{"不安、迷惑、动摇、谎言、欺骗、鬼迷心窍、动荡的爱、三角关系。在事业上，你可能有些不满足，希望能够把自己内在的力量全使出来，于是你开始想要晚上的时间。感情方面，你很敏感害怕被伤害，尽管有伴侣的承诺，你仍然犹豫不决，甚至有逃避的想法。", "逃脱骗局、解除误会、状况好转、预知危险、等待、正视爱情的裂缝。在事业上，你因为外界的压力开始退缩了，并对自己的既定目标产生了怀疑。在感情上，你们之间的问题开始浮现，虽然有些痛，但是只要共同面对存在的困难，问题就解决一半了。"}
var _TheSun = [2]string{"活跃、丰富的生命力、充满生机、精力充沛、工作顺利、贵人相助、幸福的婚姻、健康的交际。事业上会有贵人相助，将会有更好的发展机遇。在感情方面，你们已经走出坎坷的感情之路，前面将是洒满歌声和欢乐的坦途，你们将开始规划未来的生活。", "消沉、体力不佳、缺乏连续性、意气消沉、生活不安、人际关系不好、感情波动、离婚。事业上竞争心太急切了，把对手都吓跑了，然而也让合作伙伴感到害怕，或许你该放松些。感情上两人间出现一些小变化，开始在乎对方的态度和自己的付出，这些怀疑也许都是没必要的。"}
var _Judgement = [2]string{"复活的喜悦、康复、坦白、好消息、好运气、初露锋芒、复苏的爱、重逢、爱的奇迹。当牌面正立时，事业上你超越了自我，在过去努力的基础上取得了成功。感情上双方都在认真学习和成长，虽然表面上的变化并不大，但内在的改变已经很大了。", "一蹶不振、幻灭、隐瞒、坏消息、无法决定、缺少目标、没有进展、消除、恋恋不舍。在事业上缺乏清晰的判断，试图用物质填充精神的空虚。在感情上，不断地回忆着过去的美好时光，不愿意去正视眼前的问题，你们的关系已经是貌合神离了。"}
var _TheWorld = [2]string{"完成、成功、完美无缺、连续不断、精神亢奋、拥有毕生奋斗的目标、完成使命、幸运降临、快乐的结束、模范情侣。在事业上因为努力工作，所以回报丰厚。感情上，你们在彼此的承诺中持续着美好的关系。", "未完成、失败、准备不足、盲目接受、一时不顺利、半途而废、精神颓废、饱和状态、合谋、态度不够融洽、感情受挫。在事业的路上有巨大的障碍，你精神不振，丧失了挑战的动力。感情上，你们不再重视承诺，只是盲目接受对方。彼此最好能沟通一下，不要让痛苦继续纠缠着你们。"}

var (
	Words = [][2]string{{_TheFool[0], _TheFool[1]}, {_TheMagician[0], _TheMagician[1]}, {_TheHighPriestess[0], _TheHighPriestess[1]}, {_TheEmpress[0], _TheEmpress[1]}, {_TheEmperor[0], _TheEmperor[1]}, {_TheHierophant[0], _TheHierophant[1]},
		{_TheLovers[0], _TheLovers[1]}, {_TheChariot[0], _TheChariot[1]}, {_Strength[0], _Strength[1]}, {_TheHermit[0], _TheHermit[1]}, {_TheWheelOfFortune[0], _TheWheelOfFortune[1]}, {_Justice[0], _Justice[1]}, {_TheHangedMan[0], _TheHangedMan[1]}, {_Death[0], _Death[1]},
		{_Temperance[0], _Temperance[1]}, {_TheDevil[0], _TheDevil[1]}, {_TheTower[0], _TheTower[1]}, {_TheTower[0], _TheTower[1]}, {_TheStar[0], _TheStar[1]}, {_TheMoon[0], _TheMoon[1]}, {_TheSun[0], _TheSun[1]}, {_Judgement[0], _Judgement[1]}, {_TheWorld[0], _TheWorld[1]}}
	Array = []string{TheFool, TheMagician, TheHighPriestess, TheEmpress, TheEmperor, TheHierophant, TheLovers, TheChariot, Strength, TheHermit, TheWheelOfFortune,
		Justice, TheHangedMan, Death, Temperance, TheDevil, TheTower, TheStar, TheMoon, TheSun, Judgement, TheWorld}
)

type FileUploaderImpl struct {
	api openapi.OpenAPI
}

type PicData struct {
	FileInfo []byte
	Expire   time.Time
}

var FileData *sync.Map
var FileExpire *sync.Map
var FileUploader *FileUploaderImpl

func InitFileUploader(api openapi.OpenAPI) {
	FileData = &sync.Map{}
	FileUploader = &FileUploaderImpl{
		api: api,
	}
}

// 上传文件，这里需要存储数据
func UploadPicAndStore(URL string, GroupId string) []byte {
	if Data, ok := FileData.Load(URL); ok {
		data := Data.(*PicData)
		if !time.Now().After(data.Expire) {
			return data.FileInfo
		}
	}
	msg := dto.RichMediaMessage{
		FileType:   1,
		URL:        URL,
		SrvSendMsg: false,
	}
	res, err := FileUploader.api.PostGroupMessage(context.Background(), GroupId, msg)
	if err != nil {
		llog.Error("上传文件请求失败：", err)
		return nil
	}
	data := &PicData{
		FileInfo: res.FileInfo,
		Expire:   time.Now().Add(time.Duration(res.TTL-1000) * time.Second),
	}
	FileData.Store(URL, data)
	llog.Info("加载图片成功：", URL)
	return res.FileInfo
}

// 不需要存储到哈希表的上传图片
func UploadPicToFiledata(url string, groupId string) []byte {
	msg := dto.RichMediaMessage{
		FileType:   1,
		URL:        url,
		SrvSendMsg: false,
	}

	res, err := FileUploader.api.PostGroupMessage(context.Background(), groupId, msg)
	if err != nil {
		llog.Error("上传文件请求失败：", err)
		return nil
	}
	return res.FileInfo
}

func UploadPicToUrl(picBase64 string) string {
	imageData, err := base64.StdEncoding.DecodeString(string(picBase64))
	if err != nil {
		return ""
	}
	picName := fmt.Sprintf("%v.png", uuid.NewString())
	picPath := "./data/wcloud/" + picName
	os.WriteFile(picPath, imageData, os.FileMode(os.O_CREATE))

	return fmt.Sprintf("https://%s/v1/file/%s", config.K.String("PublicIP"), picName)
}
