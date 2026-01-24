package ba_logo

import (
	"LanMei/internal/bot/utils/llog"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

var baseurl = "https://lab.nulla.top/ba-logo"

func GetBALOGO(left, right string) string {
	// 1) 进程/渲染配置：视口固定、禁用 / 软件渲染兜底
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),

		// 如页面使用 WebGL，优先试“不开 --disable-gpu”；要纯软件渲染可以用 swiftshader：
		// chromedp.Flag("use-angle", "swiftshader"),
		// chromedp.Flag("use-gl", "swiftshader"),

		// 统一视口（与命令行 --window-size 等效）
		chromedp.WindowSize(1440, 900),
	)
	alloc, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(alloc)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 2) 初始动作：导航 + 视口 DPR
	var dataURL string
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(baseurl),
		chromedp.WaitVisible(`canvas`, chromedp.ByQuery),
		emulation.SetDeviceMetricsOverride(1440, 900, 1.0, false),

		// 3) 读取基线（改变前的画布指纹），用于后续对比，避免盲等
		chromedp.EvaluateAsDevTools(`(() => {
			const c = document.querySelector('canvas'); 
			return c ? c.toDataURL('image/png').slice(0,128) : '';
		})()`, &dataURL),
	})
	if err != nil {
		llog.Error(err.Error())
		return ""
	}
	baseline := dataURL // 先存一份

	// 4) 安全注入文本：使用 strconv.Quote 防止引号/换行等字符把 JS 搞坏
	jsSetInputs := fmt.Sprintf(`(() => {
		const ins = document.querySelectorAll('input[type="text"]');
		if (ins.length < 2) return "notfound";
		ins[0].value = %s;
		ins[0].dispatchEvent(new Event('input', {bubbles:true}));
		ins[1].value = %s;
		ins[1].dispatchEvent(new Event('input', {bubbles:true}));
		return "ok";
	})()`, strconv.Quote(left), strconv.Quote(right))

	var setResult string
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.EvaluateAsDevTools(jsSetInputs, &setResult),
	}); err != nil || setResult != "ok" {
		if err != nil {
			llog.Error("set inputs error: ", err.Error())
		} else {
			llog.Error("set inputs failed: ", setResult)
		}
		return ""
	}

	// 5) 轮询等待 Canvas 真的变化（最多 ~5s），避免硬睡
	var finalDataURL string
	waitCanvasChanged := chromedp.ActionFunc(func(ctx context.Context) error {
		deadline := time.Now().Add(5 * time.Second)
		for {
			if time.Now().After(deadline) {
				return fmt.Errorf("canvas not updated before timeout")
			}
			var cur string
			if err := chromedp.EvaluateAsDevTools(`(() => {
				const c = document.querySelector('canvas'); 
				return c ? c.toDataURL('image/png') : '';
			})()`, &cur).Do(ctx); err != nil {
				return err
			}
			if cur != "" && cur != "null" && cur[:min(128, len(cur))] != baseline {
				finalDataURL = cur
				return nil
			}
			time.Sleep(150 * time.Millisecond)
		}
	})

	if err := chromedp.Run(ctx, waitCanvasChanged); err != nil {
		llog.Error("wait change: ", err.Error())
		return ""
	}

	// 6) 拿到 dataURL，拆前缀
	parts := strings.SplitN(finalDataURL, ",", 2)
	if len(parts) != 2 || parts[1] == "" {
		llog.Error("未获取到: ", finalDataURL)
		return ""
	}

	// parts[0] 形如 "data:image/png;base64"
	return parts[1]
}

// 小工具：取最小值，避免切片越界
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
