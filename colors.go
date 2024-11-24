package main

const (
	colorBlack  = "\033[0;30m"
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[0;33m"
	colorBlue   = "\033[0;34m"
	colorPurple = "\033[0;35m"
	colorCyan   = "\033[0;36m"
	colorWhite  = "\033[0;37m"

	colorBrightBlack  = "\033[1;30m"
	colorBrightRed    = "\033[1;31m"
	colorBrightGreen  = "\033[1;32m"
	colorBrightYellow = "\033[1;33m"
	colorBrightBlue   = "\033[1;34m"
	colorBrightPurple = "\033[1;35m"
	colorBrightCyan   = "\033[1;36m"
	colorBrightWhite  = "\033[1;37m"

	colorNone = "\033[0m"
)

const (
	typeColorDone  = iota
	typeColorWarn  = iota
	typeColorInfo  = iota
	typeColorError = iota
)

func GetPrefix(prefix, colorOfPrefix string, t int) string {
	pref := colorOfPrefix + "[" + prefix + "]" + colorNone

	if t == typeColorDone {
		pref += colorBrightGreen + " done: " + colorNone
	} else if t == typeColorWarn {
		pref += colorBrightYellow + " warn: " + colorNone
	} else if t == typeColorError {
		pref += colorBrightRed + " err: " + colorNone
	} else if t == typeColorInfo {
		pref += " info: "
	}

	return pref
}
