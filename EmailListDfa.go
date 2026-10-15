package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"
)

// ─── Состояния ────────────────────────────────────────────────────────────────

// State — состояние минимального ДКА. Номер константы совпадает с индексом
// состояния q0…q11 в отчёте (q2 = q⊥ — поглощающее состояние отказа).
type State int

const (
	qStart         State = iota // q0  = {A0, A2}:   ждём первый символ локальной части
	qLocal                      // q1  = {A1}:       читаем локальную часть
	qDead                       // q⊥  (q2):         поглощающее состояние отказа
	qAfterAt                    // q3  = {A3, A5}:   после @ или «-» в домене ждём \w
	qDomain                     // q4  = {A4}:       читаем часть домена
	qAfterDot                   // q5  = {A6, A10}:  после «.» или «-» в домене ждём \w
	qEmailEnd                   // q6  = {A7}:       адрес можно завершить (одна \w после точки)
	qEmailWord                  // q7  = {A8}:       адрес можно завершить, либо \w начала новый адрес
	qSpaces                     // q8  = {A9}:       после адреса прочитаны пробелы
	qAfterFinalDot              // q9  = {A11, A14}: точка — разделитель домена или списка
	qAfterSep                   // q10 = {A12}:      прочитан разделитель списка [;,.]
	qAfterHyphen                // q11 = {A13}:      «-» после ≥ 2 символов \w в конце домена
)

// String возвращает имя состояния, как в отчёте.
func (q State) String() string {
	return []string{"q0", "q1", "q⊥", "q3", "q4", "q5", "q6", "q7", "q8", "q9", "q10", "q11"}[q]
}

// ─── Входной алфавит ──────────────────────────────────────────────────────────

// Symbol — класс входного символа. Автомат работает не с отдельными
// рунами, а с классами: все символы одного класса ведут себя одинаково.
type Symbol int

const (
	Word   Symbol = iota // \w: буквы, цифры, _ и т. п.
	Space                // \s: пробельные символы
	Plus                 // +
	Hyphen               // -
	Dot                  // .
	At                   // @
	Sep                  // ; или ,
	Other                // всё остальное — переходов по нему нет нигде
)

// classify относит руну к классу входного алфавита.
func classify(r rune) Symbol {
	switch {
	case isWord(r):
		return Word
	case unicode.IsSpace(r):
		return Space
	}
	switch r {
	case '+':
		return Plus
	case '-':
		return Hyphen
	case '.':
		return Dot
	case '@':
		return At
	case ';', ',':
		return Sep
	default:
		return Other
	}
}

// isWord приближённо воспроизводит смысл \w в исходном .NET-паттерне:
// буквы, десятичные цифры, комбинируемые знаки и знаки соединения (включая _).
func isWord(r rune) bool {
	return unicode.IsLetter(r) ||
		unicode.IsDigit(r) ||
		unicode.Is(unicode.Mn, r) ||
		unicode.Is(unicode.Pc, r)
}

// ─── Автомат ──────────────────────────────────────────────────────────────────

// DFA — распознаватель на основе детерминированного конечного автомата
// M = (Q, Σ, δ, q0, F). Помимо описания самого M он хранит текущее
// состояние устройства управления: автомат читает цепочку по одному символу
// и после каждого такта можно посмотреть, где он находится.
type DFA struct {
	start State                      // q0 — начальное состояние
	dead  State                      // q⊥ — куда ведут отсутствующие в δ переходы
	final map[State]bool             // F  — заключительные состояния
	delta map[State]map[Symbol]State // δ  — функция переходов

	current State // текущее состояние
}

// NewEmailListDFA создаёт распознаватель списков адресов электронной почты:
//
//	^((\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*)\s*[;,.]{0,1}\s*)+$
//
// Автомат сразу стоит в начальном состоянии.
func NewEmailListDFA() *DFA {
	a := &DFA{
		start: qStart,
		dead:  qDead,
		final: map[State]bool{
			qEmailEnd:      true,
			qEmailWord:     true,
			qSpaces:        true,
			qAfterFinalDot: true,
			qAfterSep:      true,
		},
		delta: map[State]map[Symbol]State{
			// Локальная часть: \w+([-+.]\w+)*
			qStart: {Word: qLocal},
			qLocal: {Word: qLocal, Plus: qStart, Hyphen: qStart, Dot: qStart, At: qAfterAt},

			// Домен: \w+([-.]\w+)*\.\w+([-.]\w+)*
			qAfterAt:     {Word: qDomain},
			qDomain:      {Word: qDomain, Hyphen: qAfterAt, Dot: qAfterDot},
			qAfterDot:    {Word: qEmailEnd},
			qEmailEnd:    {Word: qEmailWord, Space: qSpaces, Hyphen: qAfterDot, Dot: qAfterFinalDot, Sep: qAfterSep},
			qEmailWord:   {Word: qEmailWord, Space: qSpaces, Plus: qStart, Hyphen: qAfterHyphen, Dot: qAfterFinalDot, At: qAfterAt, Sep: qAfterSep},
			qAfterHyphen: {Word: qEmailWord},

			// Разделитель между адресами: \s*[;,.]{0,1}\s*
			qSpaces:        {Word: qLocal, Space: qSpaces, Dot: qAfterSep, Sep: qAfterSep},
			qAfterFinalDot: {Word: qEmailWord, Space: qAfterSep},
			qAfterSep:      {Word: qLocal, Space: qAfterSep},

			// qDead переходов не имеет: из него автомат никуда не уходит.
		},
	}
	a.Reset()
	return a
}

// Reset возвращает автомат в начальное состояние перед чтением новой цепочки.
func (a *DFA) Reset() {
	a.current = a.start
}

// State возвращает текущее состояние автомата.
func (a *DFA) State() State {
	return a.current
}

// Accepting сообщает, находится ли автомат в заключительном состоянии,
// то есть принял бы он цепочку, если бы она закончилась прямо сейчас.
func (a *DFA) Accepting() bool {
	return a.final[a.current]
}

// Rejected сообщает, что автомат попал в q⊥ и цепочку уже не примет.
func (a *DFA) Rejected() bool {
	return a.current == a.dead
}

// Read выполняет один такт: читает символ r и переходит
// в δ(текущее, класс(r)). Возвращает новое состояние.
func (a *DFA) Read(r rune) State {
	next, ok := a.delta[a.current][classify(r)]
	if !ok {
		next = a.dead
	}
	a.current = next
	return next
}

// Run сбрасывает автомат, подаёт ему цепочку посимвольно и возвращает
// последовательность пройденных состояний (трассу) и результат распознавания.
func (a *DFA) Run(input string) ([]State, bool) {
	a.Reset()
	trace := []State{a.State()}
	for _, r := range input {
		trace = append(trace, a.Read(r))
		if a.Rejected() {
			break // из q⊥ выхода нет, дальше читать бессмысленно
		}
	}
	return trace, a.Accepting()
}

// Accepts возвращает true, если цепочка принадлежит языку.
func (a *DFA) Accepts(input string) bool {
	_, ok := a.Run(input)
	return ok
}

// ─── Диалог с пользователем ───────────────────────────────────────────────────

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	// Увеличиваем стандартный лимит длины строки для длинных списков адресов.
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	dfa := NewEmailListDFA()

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

		trace, ok := dfa.Run(input)
		fmt.Println("Трасса:", formatTrace(trace))
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

// formatTrace печатает трассу в виде «q0 → q1 → …».
func formatTrace(trace []State) string {
	names := make([]string, len(trace))
	for i, q := range trace {
		names[i] = q.String()
	}
	return strings.Join(names, " → ")
}
