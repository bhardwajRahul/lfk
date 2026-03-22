package app

// nextWordStart returns the column of the next word start (vim 'w' motion).
// Returns n (past end) when no next word exists on this line, signaling cross-line needed.
func nextWordStart(line string, col int) int {
	runes := []rune(line)
	n := len(runes)
	if n == 0 || col >= n-1 {
		return n
	}
	i := col
	// Skip current word characters.
	for i < n && !isWordBoundary(runes[i]) {
		i++
	}
	// Skip whitespace/punctuation.
	for i < n && isWordBoundary(runes[i]) {
		i++
	}
	if i >= n {
		return n
	}
	return i
}

// wordEnd returns the column of the current/next word end (vim 'e' motion).
// Returns n (past end) when no next word end exists on this line, signaling cross-line needed.
func wordEnd(line string, col int) int {
	runes := []rune(line)
	n := len(runes)
	if n == 0 || col >= n-1 {
		return n
	}
	i := col + 1
	// Skip whitespace/punctuation.
	for i < n && isWordBoundary(runes[i]) {
		i++
	}
	if i >= n {
		return n
	}
	// Move to end of word.
	for i < n-1 && !isWordBoundary(runes[i+1]) {
		i++
	}
	return i
}

// prevWordStart returns the column of the previous word start (vim 'b' motion).
// Returns -1 when no previous word exists on this line, signaling cross-line needed.
func prevWordStart(line string, col int) int {
	runes := []rune(line)
	n := len(runes)
	if n == 0 || col <= 0 {
		return -1
	}
	if col >= n {
		col = n
	}
	i := col - 1
	// Skip whitespace/punctuation.
	for i > 0 && isWordBoundary(runes[i]) {
		i--
	}
	// Move to start of word.
	for i > 0 && !isWordBoundary(runes[i-1]) {
		i--
	}
	return i
}

// isWordBoundary returns true if the rune is whitespace or punctuation (non-word character).
func isWordBoundary(r rune) bool {
	return r == ' ' || r == '\t' || r == '.' || r == ':' || r == ',' || r == ';' ||
		r == '/' || r == '-' || r == '_' || r == '"' || r == '\'' || r == '(' || r == ')' ||
		r == '[' || r == ']' || r == '{' || r == '}'
}

// nextWORDStart returns the column of the next WORD start (vim 'W' motion).
// WORDs are whitespace-delimited (only spaces and tabs are boundaries).
// Returns n (past end) when no next WORD exists on this line, signaling cross-line needed.
func nextWORDStart(line string, col int) int {
	runes := []rune(line)
	n := len(runes)
	if n == 0 || col >= n-1 {
		return n
	}
	i := col
	// Skip current WORD characters (non-whitespace).
	for i < n && runes[i] != ' ' && runes[i] != '\t' {
		i++
	}
	// Skip whitespace.
	for i < n && (runes[i] == ' ' || runes[i] == '\t') {
		i++
	}
	if i >= n {
		return n
	}
	return i
}

// prevWORDStart returns the column of the previous WORD start (vim 'B' motion).
// WORDs are whitespace-delimited (only spaces and tabs are boundaries).
// Returns -1 when no previous WORD exists on this line, signaling cross-line needed.
func prevWORDStart(line string, col int) int {
	runes := []rune(line)
	n := len(runes)
	if n == 0 || col <= 0 {
		return -1
	}
	if col >= n {
		col = n
	}
	i := col - 1
	// Skip whitespace.
	for i > 0 && (runes[i] == ' ' || runes[i] == '\t') {
		i--
	}
	// Move to start of WORD (non-whitespace).
	for i > 0 && runes[i-1] != ' ' && runes[i-1] != '\t' {
		i--
	}
	return i
}

// WORDEnd returns the column of the current/next WORD end (vim 'E' motion).
// WORDs are whitespace-delimited (only spaces and tabs are boundaries).
// Returns n (past end) when no next WORD end exists on this line, signaling cross-line needed.
func WORDEnd(line string, col int) int {
	runes := []rune(line)
	n := len(runes)
	if n == 0 || col >= n-1 {
		return n
	}
	i := col + 1
	// Skip whitespace.
	for i < n && (runes[i] == ' ' || runes[i] == '\t') {
		i++
	}
	if i >= n {
		return n
	}
	// Move to end of WORD (non-whitespace).
	for i < n-1 && runes[i+1] != ' ' && runes[i+1] != '\t' {
		i++
	}
	return i
}

// firstNonWhitespace returns the column of the first non-space/tab character (vim '^' motion).
func firstNonWhitespace(line string) int {
	for i, r := range []rune(line) {
		if r != ' ' && r != '\t' {
			return i
		}
	}
	return 0
}

// countLines counts newline characters.
func countLines(s string) int {
	n := 0
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return n
}
