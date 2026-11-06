package lyrics

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type unit struct {
	text   string
	weight float64
	pause  float64
}

// WordsFor returns timed words for a lyric line. MRC lines already carry
// per-word timing; plain LRC lines are interpolated up to the next line.
func WordsFor(line Line, nextAt time.Duration) []Word {
	if len(line.Words) > 0 {
		return line.Words
	}
	return interpolate(line, nextAt)
}

func interpolate(line Line, nextAt time.Duration) []Word {
	text := strings.TrimSpace(line.Text)
	if text == "" {
		return nil
	}
	end := nextAt
	if end <= line.At {
		estimate := time.Duration(max(2600, utf8.RuneCountInString(text)*190)) * time.Millisecond
		end = line.At + estimate
	}
	available := end - line.At
	if available <= 0 {
		return nil
	}
	units := tokenize(text)
	if len(units) == 0 {
		return nil
	}
	total := 0.0
	for _, unit := range units {
		total += unit.weight + unit.pause
	}
	if total <= 0 {
		return nil
	}
	words := make([]Word, 0, len(units))
	cursor := line.At
	for _, unit := range units {
		dur := time.Duration(float64(available) * unit.weight / total)
		if dur < 0 {
			dur = 0
		}
		words = append(words, Word{At: cursor, Dur: dur, Text: unit.text})
		cursor += dur + time.Duration(float64(available)*unit.pause/total)
	}
	return words
}

func tokenize(text string) []unit {
	var units []unit
	var word strings.Builder
	flush := func(pause float64) {
		if word.Len() == 0 {
			return
		}
		value := word.String()
		weight := float64(utf8.RuneCountInString(value)) * 0.55
		weight = max(1.1, weight)
		units = append(units, unit{text: value, weight: weight, pause: pause})
		word.Reset()
	}
	for _, r := range text {
		switch {
		case unicode.IsSpace(r):
			if word.Len() > 0 {
				word.WriteRune(r)
				flush(0.15)
			} else {
				units = append(units, unit{text: string(r), weight: 0.3})
			}
		case isSungRune(r):
			flush(0)
			units = append(units, unit{text: string(r), weight: 1})
		case isWordRune(r):
			word.WriteRune(r)
		default:
			pause := lyricPause(r)
			if word.Len() > 0 {
				word.WriteRune(r)
				flush(pause)
			} else {
				units = append(units, unit{text: string(r), weight: 0.35, pause: pause})
			}
		}
	}
	flush(0)
	return units
}

func isSungRune(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || r == '’' || r == '-'
}

func lyricPause(r rune) float64 {
	if strings.ContainsRune("，。！？；：、,.!?;:", r) {
		return 1.05
	}
	if strings.ContainsRune("…—～()（）[]【】《》<>“”‘’\"", r) {
		return 0.55
	}
	return 0.35
}

// LineEnd estimates where a lyric line ends based on the next timed line.
func LineEnd(lines []Line, index int) time.Duration {
	if index < 0 || index >= len(lines) {
		return 0
	}
	for i := index + 1; i < len(lines); i++ {
		if lines[i].At > lines[index].At {
			return lines[i].At
		}
	}
	text := strings.TrimSpace(lines[index].Text)
	return lines[index].At + time.Duration(max(2600, utf8.RuneCountInString(text)*190))*time.Millisecond
}

// StagePair returns the current non-empty line and the next one to stage.
func StagePair(lines []Line, active int) (current, next int) {
	current, next = -1, -1
	start := active
	if start < 0 {
		start = 0
	}
	for i := start; i < len(lines); i++ {
		if strings.TrimSpace(lines[i].Text) != "" {
			current = i
			break
		}
	}
	if current < 0 {
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i].Text) != "" {
				current = i
				break
			}
		}
	}
	for i := current + 1; i >= 0 && i < len(lines); i++ {
		if strings.TrimSpace(lines[i].Text) != "" {
			next = i
			break
		}
	}
	return current, next
}
