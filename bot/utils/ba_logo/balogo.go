package ba_logo

import (
	"LanMei/bot/utils/llog"
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

var baseurl = "https://lab.nulla.top/ba-logo"

func GetBALOGO(left, right string) string {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(1440, 900),
	)
	alloc, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(alloc)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var dataURL string
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(baseurl),
		chromedp.WaitVisible(`canvas`, chromedp.ByQuery),
		emulation.SetDeviceMetricsOverride(1440, 900, 1.0, false),
	}); err != nil {
		llog.Error(err.Error())
		return ""
	}

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
		chromedp.Evaluate(jsSetInputs, &setResult),
	}); err != nil || setResult != "ok" {
		if err != nil {
			llog.Error("set inputs error: ", err.Error())
		} else {
			llog.Error("set inputs failed: ", setResult)
		}
		return ""
	}

	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Evaluate(`(async () => {
		  const c = document.querySelector('canvas');
		  if (!c) return null;

		  try { if (document.fonts && document.fonts.ready) { await document.fonts.ready; } } catch {}

		  const ctx = c.getContext('2d');
		  function hash() {
			try {
			  const w = c.width|0, h = c.height|0;
			  if (!w || !h) return "";
			  const d = ctx.getImageData(0,0,w,h).data;
			  let hsh = 0;
			  for (let i=0; i<d.length; i+=128) hsh = (hsh*131 + d[i])|0;
			  return String(hsh);
			} catch(e) {
			  const u = c.toDataURL('image/png');
			  return u ? u.slice(0,256) : "";
			}
		  }

		  const stableNeeded = 10;
		  const deadline = performance.now() + 8000;
		  let prev = null, stable = 0;

		  await new Promise(r => requestAnimationFrame(r));

		  while (performance.now() < deadline) {
			const cur = hash();
			if (cur && cur === prev) {
			  if (++stable >= stableNeeded) {
				await new Promise(r => setTimeout(r, 1000)); // 稳定后再等 1s
				const u = c.toDataURL('image/png');
				return (u && u.startsWith('data:image/png;base64,')) ? u : null;
			  }
			} else {
			  stable = 0; prev = cur;
			}
			await new Promise(r => requestAnimationFrame(r));
		  }
		  return null;
		})()`, &dataURL),
	}); err != nil {
		llog.Error("wait stable evaluate:", err.Error())
		return ""
	}

	if dataURL == "" || dataURL == "null" || !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		var pngBytes []byte
		if err := chromedp.Run(ctx, chromedp.Tasks{
			chromedp.Screenshot(`canvas`, &pngBytes, chromedp.ByQuery),
		}); err != nil || len(pngBytes) == 0 {
			if err != nil {
				llog.Error("screenshot fallback error:", err.Error())
			} else {
				llog.Error("screenshot fallback empty")
			}
			return ""
		}
		return base64.StdEncoding.EncodeToString(pngBytes)
	}

	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 || parts[1] == "" {
		llog.Error("dataURL split error: ", dataURL[:min(128, len(dataURL))])
		return ""
	}
	return parts[1]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
