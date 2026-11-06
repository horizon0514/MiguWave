package lyrics

import (
	"strings"
	"testing"
	"time"
)

func TestParseLRC(t *testing.T) {
	raw := "\ufeff[ti:晴天]\r\n[ar:周杰伦]\r\n[al:叶惠美]\r\n[by:测试]\r\n[offset:500]\r\n" +
		"[00:01.00]故事的小黄花\r\n[00:28.50][00:30.00]从出生那年就飘着\r\n"
	doc := ParseLRC(raw)
	if doc.Plain {
		t.Fatal("timed LRC was parsed as plain text")
	}
	if doc.Title != "晴天" || doc.Artist != "周杰伦" || doc.Album != "叶惠美" || doc.By != "测试" {
		t.Fatalf("metadata = %+v", doc)
	}
	if len(doc.Lines) != 3 {
		t.Fatalf("lines = %+v", doc.Lines)
	}
	if doc.Lines[0].At != 1500*time.Millisecond || doc.Lines[0].Text != "故事的小黄花" {
		t.Fatalf("first line = %+v", doc.Lines[0])
	}
	if doc.Lines[1].At != 29*time.Second || doc.Lines[2].At != 30500*time.Millisecond {
		t.Fatalf("multiple timestamps = %+v", doc.Lines)
	}
}

func TestParseLRCPlainText(t *testing.T) {
	doc := ParseLRC("[ti:测试]\n第一行\n\n第二行 &amp; 结尾\n")
	if !doc.Plain || doc.Title != "测试" {
		t.Fatalf("doc = %+v", doc)
	}
	if len(doc.Lines) != 2 || doc.Lines[0].Text != "第一行" || doc.Lines[1].Text != "第二行 & 结尾" {
		t.Fatalf("lines = %+v", doc.Lines)
	}
}

func TestParseMRC(t *testing.T) {
	raw := "[ti:晴天]\r\n[ar:周杰伦]\r\n" +
		"[0,0]作(0,0)词(0,0)：(0,0)周(0,0)杰(0,0)伦(0,0)\r\n" +
		"[27566,2095]故(27566,299)事(27865,199)的(28064,250)小(28314,549)黄(28863,449)花(29312,349)\r\n" +
		"[30459,2494]从(30459,349)出(30808,349)生(31157,499)那(31656,300)年(31956,249)就(32205,200)飘(32405,249)着(32654,299)\r\n"
	doc := ParseMRC(raw)
	if doc.Plain || doc.Title != "晴天" || doc.Artist != "周杰伦" {
		t.Fatalf("doc = %+v", doc)
	}
	if len(doc.Lines) != 3 {
		t.Fatalf("lines = %+v", doc.Lines)
	}
	line := doc.Lines[1]
	if line.At != 27566*time.Millisecond || line.Text != "故事的小黄花" {
		t.Fatalf("line = %+v", line)
	}
	if len(line.Words) != 6 {
		t.Fatalf("words = %+v", line.Words)
	}
	if line.Words[0].At != 27566*time.Millisecond || line.Words[0].Dur != 299*time.Millisecond ||
		line.Words[0].Text != "故" {
		t.Fatalf("first word = %+v", line.Words[0])
	}
}

func TestStripHeader(t *testing.T) {
	title := "一半（《折月亮》影视剧片尾曲）"
	lines := []Line{
		{Text: title + " - 段奥娟"},
		{Text: "作词：四季逗/潘超"},
		{Text: "作曲：宋天祥/李喆"},
		{Text: "制作人：宋天祥"},
		{Text: "总策划：王锋"},
		{Text: "「 未经著作权人许可不得翻唱翻录或使用」"},
		{At: 12 * time.Second, Text: "时间上的灰尘", Timed: true},
		{At: 18 * time.Second, Text: "每一步都很轻缓", Timed: true},
	}
	got := StripHeader(lines, title, "段奥娟")
	if len(got) != 2 || got[0].Text != "时间上的灰尘" || got[1].Text != "每一步都很轻缓" {
		t.Fatalf("stripped = %+v", got)
	}

	lyrics := []Line{
		{At: 29 * time.Second, Text: "故事的小黄花", Timed: true},
		{At: 32 * time.Second, Text: "从出生那年就飘着", Timed: true},
	}
	got = StripHeader(lyrics, "晴天", "周杰伦")
	if len(got) != 2 || got[0].Text != "故事的小黄花" {
		t.Fatalf("plain lyrics changed = %+v", got)
	}
}

func TestStripBilingualLyricCredits(t *testing.T) {
	lines := []Line{
		{Text: "月下签 - 蒲熠星"},
		{Text: "作词：蒲熠星/郑眠眠"},
		{Text: "音乐监制Music Executive Director：朴桑松@壹橙Music"},
		{Text: "和声Background Vocals：陈小同@壹橙Music"},
		{Text: "和音：杜婧荧"},
		{Text: "混音/母带后期工程师Mixing/Mastering Engineer：钟焕楷@壹橙Music"},
		{Text: "封面设计Cover Designer：敬尧"},
		{At: 17 * time.Second, Text: "推开山门月光落在脚边", Timed: true},
		{At: 21 * time.Second, Text: "折起签文一篇折起谎言", Timed: true},
		{At: 200 * time.Second, Text: "特别鸣谢Special Thanks：蒲熠星工作室", Timed: true},
		{At: 202 * time.Second, Text: "OP(Original Publisher)：咪咕音乐", Timed: true},
	}
	got := StripHeader(lines, "月下签", "蒲熠星")
	if len(got) != 2 || got[0].Text != "推开山门月光落在脚边" {
		t.Fatalf("stripped = %+v", got)
	}
}

func TestWordsForInterpolatesLine(t *testing.T) {
	line := Line{At: 10 * time.Second, Text: "你好 world!", Timed: true}
	words := WordsFor(line, 14*time.Second)
	if len(words) < 4 {
		t.Fatalf("words = %+v", words)
	}
	if words[0].Text != "你" || words[0].At != 10*time.Second {
		t.Fatalf("first word = %+v", words[0])
	}
	if last := words[len(words)-1]; last.At >= 14*time.Second {
		t.Fatalf("last word runs past line end: %+v", last)
	}
	var joined strings.Builder
	for _, word := range words {
		joined.WriteString(word.Text)
	}
	if got := joined.String(); got != "你好 world!" {
		t.Fatalf("joined = %q", got)
	}
}

func TestStagePair(t *testing.T) {
	lines := []Line{
		{At: 1 * time.Second, Text: "第一句", Timed: true},
		{At: 2 * time.Second, Text: "", Timed: true},
		{At: 3 * time.Second, Text: "第二句", Timed: true},
		{At: 4 * time.Second, Text: "第三句", Timed: true},
	}
	current, next := StagePair(lines, 2)
	if current != 2 || next != 3 {
		t.Fatalf("pair = %d/%d", current, next)
	}
	current, next = StagePair(lines, -1)
	if current != 0 || next != 2 {
		t.Fatalf("leading pair = %d/%d", current, next)
	}
}

func TestIndexAt(t *testing.T) {
	lines := []Line{
		{At: 1 * time.Second, Text: "第一句", Timed: true},
		{At: 2 * time.Second, Text: "", Timed: true},
		{At: 3 * time.Second, Text: "第二句", Timed: true},
	}
	if got := IndexAt(lines, 2500*time.Millisecond); got != 0 {
		t.Fatalf("index before next line = %d", got)
	}
	if got := IndexAt(lines, 4*time.Second); got != 2 {
		t.Fatalf("index after last line = %d", got)
	}
}
