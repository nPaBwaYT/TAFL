package main

import (
	"bufio"
	"fmt"
	"os"
	"unicode"
)

// Состояния минимального ДКА. Номер константы совпадает с индексом
// состояния q0…q11 в отчёте (q2 = q⊥ — поглощающее состояние отказа).
const (
	qStart         = iota // q0  = {A0, A2}:  ждём первый символ локальной части
	qLocal                // q1  = {A1}:      читаем локальную часть
	qDead                 // q⊥  (q2):        поглощающее состояние отказа
	qAfterAt              // q3  = {A3, A5}:  после @ или «-» в домене ждём \w
	qDomain               // q4  = {A4}:      читаем часть домена
	qAfterDot             // q5  = {A6, A10}: после «.» или «-» в домене ждём \w
	qEmailEnd             // q6  = {A7}:      адрес можно завершить (одна \w после точки)
	qEmailWord            // q7  = {A8}:      адрес можно завершить, либо \w начала новый адрес
	qSpaces               // q8  = {A9}:      после адреса прочитаны пробелы
	qAfterFinalDot        // q9  = {A11, A14}: точка — разделитель домена или списка
	qAfterSep             // q10 = {A12}:     прочитан разделитель списка [;,.]
	qAfterHyphen          // q11 = {A13}:     «-» после ≥ 2 символов \w в конце домена
)

// stateNames — имена состояний для вывода трассы, как в отчёте.
var stateNames = [...]string{"q0", "q1", "q⊥", "q3", "q4", "q5", "q6", "q7", "q8", "q9", "q10", "q11"}

// isWord приближённо воспроизводит смысл \w в исходном .NET-паттерне:
// буквы, десятичные цифры, комбинируемые знаки и знаки соединения (включая _).
func isWord(r rune) bool {
	return unicode.IsLetter(r) ||
		unicode.IsDigit(r) ||
		unicode.Is(unicode.Mn, r) ||
		unicode.Is(unicode.Pc, r)
}

// isSpace соответствует классу \s: пробельные символы.
func isSpace(r rune) bool {
	return unicode.IsSpace(r)
}

// nextState выполняет один шаг автомата по текущему состоянию и символу.
// Если для пары «состояние + символ» переход не предусмотрен, уходим в qDead.
func nextState(state int, r rune) int {
	w := isWord(r)
	s := isSpace(r)

	switch state {
	case qStart:
		if w {
			return qLocal
		}
		return qDead

	case qLocal:
		if w {
			return qLocal
		}
		switch r {
		case '+', '-', '.':
			return qStart // после знака должна идти непустая группа \w+
		case '@':
			return qAfterAt
		default:
			return qDead
		}

	case qDead:
		return qDead

	case qAfterAt:
		if w {
			return qDomain
		}
		return qDead

	case qDomain:
		if w {
			return qDomain
		}
		switch r {
		case '-':
			return qAfterAt
		case '.':
			return qAfterDot
		default:
			return qDead
		}

	case qAfterDot:
		if w {
			return qEmailEnd
		}
		return qDead

	case qEmailEnd:
		if w {
			return qEmailWord
		}
		if s {
			return qSpaces
		}
		switch r {
		case '-':
			return qAfterDot
		case '.':
			return qAfterFinalDot
		case ';', ',':
			return qAfterSep
		default:
			return qDead
		}

	case qEmailWord:
		if w {
			return qEmailWord
		}
		if s {
			return qSpaces
		}
		switch r {
		case '+':
			return qStart
		case '-':
			return qAfterHyphen
		case '.':
			return qAfterFinalDot
		case '@':
			return qAfterAt
		case ';', ',':
			return qAfterSep
		default:
			return qDead
		}

	case qSpaces:
		if w {
			return qLocal
		}
		if s {
			return qSpaces
		}
		if r == '.' || r == ';' || r == ',' {
			return qAfterSep
		}
		return qDead

	case qAfterFinalDot:
		if w {
			return qEmailWord
		}
		if s {
			return qAfterSep
		}
		return qDead

	case qAfterSep:
		if w {
			return qLocal
		}
		if s {
			return qAfterSep
		}
		return qDead

	case qAfterHyphen:
		if w {
			return qEmailWord
		}
		return qDead

	default:
		return qDead
	}
}

// isFinal проверяет, является ли состояние заключительным (q6–q10).
func isFinal(state int) bool {
	switch state {
	case qEmailEnd, qEmailWord, qSpaces, qAfterFinalDot, qAfterSep:
		return true
	default:
		return false
	}
}

// Run обрабатывает всю цепочку и возвращает последовательность
// пройденных состояний (трассу) и результат распознавания.
func Run(input string) ([]int, bool) {
	state := qStart
	trace := []int{state}
	for _, r := range input {
		state = nextState(state, r)
		trace = append(trace, state)
		if state == qDead {
			break // из q⊥ выхода нет, дальше читать бессмысленно
		}
	}
	return trace, isFinal(state)
}

// Accepts возвращает true, если цепочка принадлежит языку.
func Accepts(input string) bool {
	_, ok := Run(input)
	return ok
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	// Увеличиваем стандартный лимит длины строки для длинных списков адресов.
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	fmt.Println("Проверка строк конечным автоматом. Для выхода введите exit.")
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if input == "exit" {
			break
		}

		trace, ok := Run(input)
		fmt.Print("Трасса: ")
		for i, q := range trace {
			if i > 0 {
				fmt.Print(" → ")
			}
			fmt.Print(stateNames[q])
		}
		fmt.Println()
		if ok {
			fmt.Println("Цепочка принадлежит языку.")
		} else {
			fmt.Println("Цепочка не принадлежит языку.")
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка чтения ввода: %v\n", err)
		os.Exit(1)
	}
}
