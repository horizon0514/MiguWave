package lyrics

import (
	"encoding/binary"
	"errors"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

type Line struct {
	At    time.Duration
	Text  string
	Timed bool
	Words []Word
}

type Word struct {
	At   time.Duration
	Dur  time.Duration
	Text string
}

type Document struct {
	Lines  []Line
	Plain  bool
	Title  string
	Artist string
	Album  string
	By     string
}

var (
	lrcStampRE = regexp.MustCompile(`^\[(\d{1,3}):(\d{1,2})(?:[.:](\d{1,3}))?\]`)
	lrcMetaRE  = regexp.MustCompile(`^\[([A-Za-z]+):(.*)\]$`)
	spaceRE    = regexp.MustCompile(`\s+`)
)

func ParseLRC(raw string) Document {
	raw = strings.TrimPrefix(raw, "\ufeff")
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	var doc Document
	var timed, plain []Line
	var offset time.Duration
	for _, rawLine := range strings.Split(raw, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.EqualFold(line, "@migu music@") {
			continue
		}
		if meta := lrcMetaRE.FindStringSubmatch(line); meta != nil {
			value := cleanDisplay(meta[2])
			switch strings.ToLower(meta[1]) {
			case "ti":
				doc.Title = value
			case "ar":
				doc.Artist = value
			case "al":
				doc.Album = value
			case "by":
				doc.By = value
			case "offset":
				if ms, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
					offset = time.Duration(ms) * time.Millisecond
				}
			}
			continue
		}

		stamps, text := parseLRCStamps(line)
		text = cleanLyricText(text)
		if len(stamps) == 0 {
			if text != "" {
				plain = append(plain, Line{Text: text})
			}
			continue
		}
		for _, at := range stamps {
			if shifted := at + offset; shifted > 0 {
				at = shifted
			} else {
				at = 0
			}
			timed = append(timed, Line{At: at, Text: text, Timed: true})
		}
	}
	if len(timed) > 0 {
		sort.SliceStable(timed, func(i, j int) bool { return timed[i].At < timed[j].At })
		doc.Lines = timed
		return doc
	}
	doc.Plain = true
	doc.Lines = plain
	return doc
}

var (
	mrcStampRE = regexp.MustCompile(`^\[(-?\d+),(-?\d+)\]`)
	mrcWordRE  = regexp.MustCompile(`\((-?\d+),(-?\d+)\)`)
)

// DecryptMRC turns Migu's hex ciphertext into the UTF-16LE MRC document.
// The payload uses the same TEA variant as Migu's mobile lyric reader.
func DecryptMRC(raw []byte) (string, error) {
	text := strings.TrimSpace(strings.TrimPrefix(string(raw), "\ufeff"))
	if text == "" {
		return "", errors.New("empty MRC")
	}
	if strings.HasPrefix(text, "[") {
		return text, nil
	}
	for _, r := range text {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return "", errors.New("not MRC hex")
		}
	}
	if len(text) < 32 {
		return "", errors.New("short MRC")
	}
	data := mrcLongArray(text)
	key := mrcKeyLongs("karakal@123Qcomyidongtiantianhaoting")
	mrcTEADecrypt(data, key)
	buf := make([]byte, 0, len(data)*8)
	for _, value := range data {
		var word [8]byte
		binary.LittleEndian.PutUint64(word[:], uint64(value))
		buf = append(buf, word[:]...)
	}
	for len(buf) > 0 && buf[len(buf)-1] == '0' {
		buf = buf[:len(buf)-1]
	}
	if len(buf)%2 == 1 {
		buf = buf[:len(buf)-1]
	}
	units := make([]uint16, len(buf)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(buf[i*2:])
	}
	out := string(utf16.Decode(units))
	out = strings.TrimLeft(out, "\ufeff")
	out = strings.TrimRight(out, "\x00")
	if !strings.Contains(out, "[") {
		return "", errors.New("invalid MRC payload")
	}
	return out, nil
}

func mrcLongArray(hexText string) []int64 {
	out := make([]int64, len(hexText)/16)
	for i := range out {
		var value uint64
		for _, c := range hexText[i*16 : i*16+16] {
			var nibble uint64
			switch {
			case c >= '0' && c <= '9':
				nibble = uint64(c - '0')
			case c >= 'a' && c <= 'f':
				nibble = uint64(c-'a') + 10
			default:
				nibble = uint64(c-'A') + 10
			}
			value = value<<4 | nibble
		}
		out[i] = int64(value)
	}
	return out
}

func mrcKeyLongs(key string) []int64 {
	for len(key) < 32 {
		key += "0"
	}
	buf := make([]byte, len(key)*2)
	for i := range key {
		buf[i*2] = key[i]
	}
	count := (len(buf) + 7) / 8
	out := make([]int64, count)
	for i := 0; i < count-1; i++ {
		out[i] = int64(binary.LittleEndian.Uint64(buf[i*8:]))
	}
	var tail [8]byte
	copy(tail[:], buf[(count-1)*8:])
	out[count-1] = int64(binary.LittleEndian.Uint64(tail[:]))
	return out
}

func mrcTEADecrypt(data, key []int64) {
	const delta int64 = 2654435769
	n := len(data)
	if n == 0 {
		return
	}
	var value0 int64 = data[0]
	for sum := (int64(6) + int64(52/n)) * delta; sum != 0; sum -= delta {
		mix := int64(3) & (sum >> 2)
		i := int64(n)
		for {
			i--
			if i <= 0 {
				break
			}
			prev := data[i-1]
			value0 = data[i] - (((value0 ^ sum) + (prev ^ key[int((3&i)^mix)])) ^
				(((prev >> 5) ^ (value0 << 2)) + ((value0 >> 3) ^ (prev << 4))))
			data[i] = value0
		}
		last := data[n-1]
		value0 = data[0] - (((key[int((i&3)^mix)] ^ last) + (value0 ^ sum)) ^
			(((last >> 5) ^ (value0 << 2)) + ((value0 >> 3) ^ (last << 4))))
		data[0] = value0
	}
}

func ParseMRC(raw string) Document {
	raw = strings.TrimPrefix(raw, "\ufeff")
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	var doc Document
	var timed, plain []Line
	for _, rawLine := range strings.Split(raw, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if meta := lrcMetaRE.FindStringSubmatch(line); meta != nil {
			value := cleanDisplay(meta[2])
			switch strings.ToLower(meta[1]) {
			case "ti":
				doc.Title = value
			case "ar":
				doc.Artist = value
			case "al":
				doc.Album = value
			case "by":
				doc.By = value
			}
			continue
		}
		stamps := mrcStampRE.FindStringSubmatch(line)
		if stamps == nil {
			if text := cleanLyricText(line); text != "" {
				plain = append(plain, Line{Text: text})
			}
			continue
		}
		startMS, _ := strconv.Atoi(stamps[1])
		durationMS, _ := strconv.Atoi(stamps[2])
		rest := line[len(stamps[0]):]
		words, text := parseMRCWords(rest)
		at := time.Duration(max(0, startMS)) * time.Millisecond
		if text == "" && len(words) == 0 {
			continue
		}
		timed = append(timed, Line{
			At:    at,
			Text:  text,
			Timed: true,
			Words: words,
		})
		_ = durationMS
	}
	if len(timed) > 0 {
		sort.SliceStable(timed, func(i, j int) bool { return timed[i].At < timed[j].At })
		doc.Lines = timed
		return doc
	}
	doc.Plain = true
	doc.Lines = plain
	return doc
}

func parseMRCWords(text string) ([]Word, string) {
	matches := mrcWordRE.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil, cleanLyricText(text)
	}
	words := make([]Word, 0, len(matches))
	var plain strings.Builder
	prev := 0
	for _, match := range matches {
		part := text[prev:match[0]]
		startMS, _ := strconv.Atoi(text[match[2]:match[3]])
		durationMS, _ := strconv.Atoi(text[match[4]:match[5]])
		if part != "" {
			part = strings.ReplaceAll(html.UnescapeString(part), "\u00a0", " ")
			plain.WriteString(part)
			words = append(words, Word{
				At:   time.Duration(max(0, startMS)) * time.Millisecond,
				Dur:  time.Duration(max(0, durationMS)) * time.Millisecond,
				Text: part,
			})
		}
		prev = match[1]
	}
	if tail := text[prev:]; tail != "" {
		tail = strings.ReplaceAll(html.UnescapeString(tail), "\u00a0", " ")
		plain.WriteString(tail)
		words = append(words, Word{Text: tail})
	}
	return words, cleanLyricText(plain.String())
}

func WordCount(lines []Line) int {
	count := 0
	for _, line := range lines {
		count += len(line.Words)
	}
	return count
}

func parseLRCStamps(line string) ([]time.Duration, string) {
	var stamps []time.Duration
	rest := line
	for strings.HasPrefix(rest, "[") {
		m := lrcStampRE.FindStringSubmatch(rest)
		if m == nil {
			break
		}
		minutes, _ := strconv.Atoi(m[1])
		seconds, _ := strconv.Atoi(m[2])
		var fraction time.Duration
		switch len(m[3]) {
		case 1:
			ms, _ := strconv.Atoi(m[3])
			fraction = time.Duration(ms) * 100 * time.Millisecond
		case 2:
			cs, _ := strconv.Atoi(m[3])
			fraction = time.Duration(cs) * 10 * time.Millisecond
		case 3:
			ms, _ := strconv.Atoi(m[3])
			fraction = time.Duration(ms) * time.Millisecond
		}
		stamps = append(stamps, time.Duration(minutes)*time.Minute+
			time.Duration(seconds)*time.Second+fraction)
		rest = strings.TrimLeft(rest[len(m[0]):], " ")
	}
	return stamps, rest
}

func CleanText(text string) string {
	return cleanLyricText(text)
}

func cleanLyricText(text string) string {
	text = html.UnescapeString(strings.TrimSpace(text))
	text = strings.ReplaceAll(text, "\u00a0", " ")
	return strings.TrimSpace(text)
}

func cleanDisplay(value string) string {
	value = html.UnescapeString(strings.TrimSpace(value))
	value = spaceRE.ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

// StripHeader drops the credit block many Migu LRC files put before
// the first lyric: the "title - artist" line, the 作词/作曲/编曲 credits
// and the copyright notice. Otherwise the page opens on the credits.
func StripHeader(lines []Line, title, artist string) []Line {
	first := -1
	for i, line := range lines {
		if strings.TrimSpace(line.Text) != "" {
			first = i
			break
		}
	}
	if first < 0 {
		return lines
	}
	// Migu closes the credit block with a copyright notice, so it is the
	// surest place to cut, even when a credit line uses a role we do not
	// know. Only trust it in the header, after a few credit-looking lines.
	creditsBefore := 0
	for i := first; i < len(lines) && i < first+60; i++ {
		text := strings.TrimSpace(lines[i].Text)
		if text == "" {
			continue
		}
		if isCopyrightNotice(text) && creditsBefore >= 3 {
			return trimTrailingLyricCredits(trimLyricGap(lines[i+1:]), title, artist)
		}
		if isLyricCredit(text, title, artist, i == first) {
			creditsBefore++
		}
	}
	start, credits := 0, 0
	for i := first; i < len(lines); i++ {
		text := strings.TrimSpace(lines[i].Text)
		if text == "" {
			continue
		}
		if !isLyricCredit(text, title, artist, i == first) {
			break
		}
		credits++
		start = i + 1
	}
	if credits == 0 {
		return lines
	}
	return trimTrailingLyricCredits(trimLyricGap(lines[start:]), title, artist)
}

func trimLyricGap(lines []Line) []Line {
	for len(lines) > 0 && strings.TrimSpace(lines[0].Text) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1].Text) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func trimTrailingLyricCredits(lines []Line, title, artist string) []Line {
	end := len(lines)
	for end > 0 {
		text := strings.TrimSpace(lines[end-1].Text)
		if text != "" && !isLyricCredit(text, title, artist, false) {
			break
		}
		end--
	}
	return lines[:end]
}

func isCopyrightNotice(text string) bool {
	if !strings.Contains(text, "著作权") && !strings.Contains(text, "翻唱翻录") {
		return false
	}
	return strings.HasPrefix(text, "「") || strings.Contains(text, "未经") ||
		strings.Contains(text, "许可")
}

var lyricCreditRoles = []string{
	"作词", "作曲", "编曲", "制作人", "配唱", "吉他", "贝司", "贝斯",
	"钢琴", "弦乐", "伴唱", "和声", "和音", "录音", "混音", "母带", "监制",
	"统筹", "出品", "发行", "编辑", "制作", "词", "曲",
	"策划", "鸣谢", "封面", "设计",
	"arranger", "producer", "composer", "lyricist", "mixing",
	"mastering", "guitar", "bass", "piano", "strings", "vocals",
	"recorded", "studio", "op", "sp",
}

func isLyricCredit(text, title, artist string, first bool) bool {
	if first {
		if title != "" && text == title {
			return true
		}
		if strings.Contains(text, " - ") &&
			((title != "" && strings.Contains(text, title)) ||
				(artist != "" && strings.Contains(text, artist))) {
			return true
		}
	}
	if strings.Contains(text, "著作权") ||
		strings.Contains(text, "未经著作权人") {
		return true
	}
	if strings.HasPrefix(text, "「") && strings.HasSuffix(text, "」") &&
		(strings.Contains(text, "许可") || strings.Contains(text, "翻唱")) {
		return true
	}
	at := strings.IndexAny(text, "：:")
	if at <= 0 || at > 80 {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(text[:at]))
	if strings.HasPrefix(role, "op(") || strings.HasPrefix(role, "sp(") {
		return true
	}
	for _, want := range lyricCreditRoles {
		if role == want || strings.HasPrefix(role, want) ||
			strings.HasSuffix(role, want) {
			return true
		}
		if len([]rune(want)) < 2 || strings.ContainsAny(role, "，。！？、；") {
			continue
		}
		if strings.Contains(role, want) {
			return true
		}
	}
	return false
}

// IndexAt is the last non-empty lyric line at or before position.
func IndexAt(lines []Line, position time.Duration) int {
	index := -1
	for i, line := range lines {
		if line.At > position {
			break
		}
		if strings.TrimSpace(line.Text) != "" {
			index = i
		}
	}
	return index
}
