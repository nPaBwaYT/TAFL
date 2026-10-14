package main

import (
	"regexp"
	"testing"
)

// Исходный паттерн. В Go (RE2) класс \w — только ASCII, поэтому
// сравнение ведётся на ASCII-алфавите: a, пробел, + - . @ ; ,
var pattern = regexp.MustCompile(`^((\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*)\s*[;,.]{0,1}\s*)+$`)

// TestAgainstRegex перебирает ВСЕ цепочки длины 0…7 над алфавитом
// из 8 представителей классов и сравнивает ответ автомата с регулярным выражением.
func TestAgainstRegex(t *testing.T) {
	alpha := []rune("a +-.@;,")
	var buf []rune
	count := 0
	var rec func(depth int)
	rec = func(depth int) {
		s := string(buf)
		count++
		if Accepts(s) != pattern.MatchString(s) {
			t.Fatalf("расхождение на %q: автомат=%v, regex=%v", s, Accepts(s), pattern.MatchString(s))
		}
		if depth == 7 {
			return
		}
		for _, r := range alpha {
			buf = append(buf, r)
			rec(depth + 1)
			buf = buf[:len(buf)-1]
		}
	}
	rec(0)
	t.Logf("проверено цепочек: %d", count)
}
