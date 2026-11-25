package nativeui

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/feeds"
	"windows/internal/store"
)

func savePNG(t *testing.T, dir, name string, tt *ui.Tester) {
	t.Helper()
	img := tt.Image()
	if img == nil {
		t.Fatalf("nil image for %s", name)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode %s: %v", name, err)
	}
}

func TestNativeUIRenderSnapshots(t *testing.T) {
	outDir := os.Getenv("PLANET_SNAPSHOT_DIR")
	if outDir == "" {
		t.Skip("set PLANET_SNAPSHOT_DIR to dump rendered PNG snapshots")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", outDir, err)
	}

	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{
			{ID: "s1", Name: "山野通信", Created: now.AddDate(0, -1, 0), LastPublished: &now},
			{ID: "s2", Name: "设计观察", Created: now.AddDate(0, -2, 0)},
			{ID: "s3", Name: "生活里的小记录", Created: now.AddDate(0, -3, 0)},
		},
		articles: []store.Article{
			{
				ID: "a1", SiteID: "s1", Title: "一封写给秋天的信",
				Content: "傍晚沿河走了半小时。风从芦苇间穿过去，水面把天色留得比岸边久一点。\n\n## 记下来的几句话\n\n- 先把草稿留在本地，不必急着公开\n- 确认排版无误后，再一键发布到 IPFS\n\n```go\nfmt.Println(\"Hello Planet\")\n```",
				Created: now.AddDate(0, 0, -2), Modified: now.AddDate(0, 0, -1),
			},
			{
				ID: "a2", SiteID: "s2", Title: "少一点装饰，多一点内容",
				Content: "界面里的安静感，来自克制的层级与清晰的留白。",
				Created: now.AddDate(0, 0, -4), Modified: now.AddDate(0, 0, -3),
			},
		},
		drafts: []store.Draft{
			{
				ID: "d1", SiteID: "s1", Title: "把散落的想法，慢慢写成一篇文章",
				Content: "傍晚收工后翻了翻这周记下的几条短句。\n\n- 先把草稿留在本地，不必急着公开\n- 配上一两张随手拍的照片或插图\n- 确认排版无误后，再一键发布到 IPFS\n\n写作工具最好的状态，就是打开就能直接开始写。",
				Created: now.AddDate(0, 0, -1), Modified: now,
			},
		},
		subscriptions: []store.Subscription{
			{
				URL: "https://river.example/rss",
				Feed: feeds.Feed{
					Title: "沿途手记",
					Items: []feeds.Item{
						{
							Key:       "sub-1",
							Title:     "在河边留一页空白",
							Published: "2026/11/19",
							Content:   "<p>傍晚沿河走了半小时。风从芦苇间穿过去，水面把天色留得比岸边久一点。回家先记下这段光线，剩下的想法等明天再写。</p><p>散步时不用急着找一个题目。把记得的声音和颜色写下来，就已经足够了。</p>",
						},
						{
							Key:       "sub-2",
							Title:     "窗边的第一杯茶",
							Published: "2026/11/18",
							Content:   "<p>雨还没停，先给自己泡一杯茶。</p>",
						},
					},
				},
			},
		},
		activity: []store.DailyActivity{
			{Date: now.Format("2006-01-02"), Articles: 2, Drafts: 1},
			{Date: now.AddDate(0, 0, -1).Format("2006-01-02"), Articles: 1, Drafts: 2},
			{Date: now.AddDate(0, 0, -3).Format("2006-01-02"), Articles: 3, Drafts: 0},
		},
	}

	app := New(backend)
	tt := ui.NewTester(app.View, 1360, 860)
	// Simulate Windows hidden titlebar right inset (min/max/close buttons ~138px).
	tt.SetTitleBar(ui.TitleBar{Height: 42, Right: 138})
	tt.Frame()
	savePNG(t, outDir, "01-dashboard.png", tt)

	app.selectSite("s1")
	tt.Frame()
	savePNG(t, outDir, "02-site-reader.png", tt)

	app.openDraft(backend.drafts[0])
	tt.Frame()
	savePNG(t, outDir, "03-editor.png", tt)

	app.openSubscriptionView(viewSubscription, "https://river.example/rss")
	tt.Frame()
	savePNG(t, outDir, "04-subscription.png", tt)

	// Dark mode site reader snapshot.
	app.selectSite("s1")
	tt.SetDark(true)
	tt.Frame()
	savePNG(t, outDir, "05-site-reader-dark.png", tt)

	// Light mode macSheet snapshot.
	tt.SetDark(false)
	app.createSiteOpen = true
	tt.Frame()
	savePNG(t, outDir, "06-sheet-create-site.png", tt)
	app.createSiteOpen = false

	// IPFS status popover snapshot.
	app.openIPFSModal()
	tt.Frame()
	savePNG(t, outDir, "07-ipfs-popover.png", tt)
}
