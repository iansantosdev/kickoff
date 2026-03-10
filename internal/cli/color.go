package cli

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// color wraps text with an ANSI color code.
func color(code, text string) string {
	return code + text + colorReset
}

// bold wraps text with ANSI bold.
func bold(text string) string {
	return colorBold + text + colorReset
}

// dim wraps text with ANSI dim.
func dim(text string) string {
	return colorDim + text + colorReset
}
