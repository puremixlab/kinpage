// Package templatecatalog lists the template identities understood by Kinpage.
package templatecatalog

import "slices"

type Builtin struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Build       int    `json:"build"`
}

var upstream = []Builtin{
	{"plain", "Plain", "简洁的文字博客", 132},
	{"gamedb", "8-bit", "像素游戏风格", 50},
	{"grid", "Grid", "适合图片的网格布局", 99},
	{"platinum", "Platinum", "经典桌面窗口风格", 20},
	{"sepia", "Sepia", "暖色调的微博客", 90},
}

var kinpage = []Builtin{
	{"gazette", "Gazette", "头条与分栏的杂志版式", 1},
	{"folio", "Folio", "以封面为主的作品集", 1},
	{"notebook", "Notebook", "按主题浏览的知识笔记", 1},
	{"dispatch", "Dispatch", "适合连载与通讯的阅读版式", 1},
	{"revision", "Revision", "按时间浏览的项目更新", 1},
	{"airwaves", "Airwaves", "可在首页播放的播客与音频日志", 1},
	{"waypoint", "Waypoint", "图文并排的旅行手记", 1},
	{"linkshelf", "Linkshelf", "带短评的链接收藏", 1},
}

func Builtins() []Builtin { return slices.Concat(upstream, kinpage) }

func Kinpage() []Builtin { return slices.Clone(kinpage) }

func IsKinpage(name string) bool {
	return slices.ContainsFunc(kinpage, func(item Builtin) bool { return item.Name == name })
}

func SupportsName(name string) bool {
	return IsKinpage(name) || slices.ContainsFunc(upstream, func(item Builtin) bool { return item.Name == name })
}
