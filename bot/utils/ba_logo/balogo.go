package ba_logo

import (
	"LanMei/bot/utils/llog"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

var baseurl = "https://lab.nulla.top/ba-logo"

func GetBALOGO(left, right string) string {

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)
	alloc, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(alloc)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var dataURL string
	// 抓包发现前端源码是监听html dom，修改后canvas直接绘图，不类似于react的事件驱动
	// 直接狠狠改，狠狠导出<canvas>到资源URL
	// 学的前端还真有点用hhh
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(`https://lab.nulla.top/ba-logo`),
		chromedp.WaitVisible(`canvas`, chromedp.ByQuery),
		chromedp.EvaluateAsDevTools(fmt.Sprintf(`(() => {
	  const ins = document.querySelectorAll('input[type="text"]');
	  if (ins.length < 2) return "notfound";
	  ins[0].value = "%s";
	  ins[0].dispatchEvent(new Event('input', {bubbles:true}));
	  ins[1].value = "%s";
	  ins[1].dispatchEvent(new Event('input', {bubbles:true}));
	  return "ok";
	})()`, left, right), nil),
		chromedp.Sleep(1 * time.Second),
		chromedp.EvaluateAsDevTools(`(() => {
	  const c = document.querySelector('canvas');
	  return c ? c.toDataURL('image/png') : null;
	})()`, &dataURL),
	})

	if err != nil {
		llog.Error(err.Error())
		return ""
	}
	if dataURL == "" || dataURL == "null" {
		llog.Error("未获取到:", dataURL)
		return ""
	}

	parts := strings.SplitN(dataURL, ",", 2)

	fmt.Println(parts[1])

	return parts[1]
}
