package utils

var colorCodeMap = map[string]string{
	";bk;": "\033[90m", // Bright Black
	";r;":  "\033[91m", // Bright Red
	";g;":  "\033[92m", // Bright Green
	";y;":  "\033[93m", // Bright Yellow
	";b;":  "\033[94m", // Bright Blue
	";db;": "\033[34m", // Dark Blue
	";m;":  "\033[95m", // Bright Magenta
	";c;":  "\033[96m", // Bright Cyan
	";w;":  "\033[97m", // Bright White
	";x;":  "\033[0m",  // Reset
}

// ColorText replaces special color code strings (e.g., ";r;" for Bright Red)
// with their corresponding ANSI terminal color codes and returns
// the modified string.
func ColorText(text string) string {
	return ReplaceAll(text, colorCodeMap)
}
