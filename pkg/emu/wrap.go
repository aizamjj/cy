package emu

func emptyLine(cols int) Line {
	line := make(Line, cols)
	for i := range line {
		line[i] = EmptyGlyph()
	}
	return line
}

func wrapLine(line Line, cols int) []Line {
	// We only want to wrap non-whitespace characters
	var length int = 0
	for i := len(line) - 1; i >= 0; i-- {
		glyph := line[i]
		if glyph.Char != ' ' || glyph.FG != DefaultFG || glyph.BG != DefaultBG {
			length = i + 1
			break
		}
	}

	result := make([]Line, 0)

	if length == 0 {
		return nil
	}

	numLines := length / cols
	if (length % cols) > 0 {
		numLines++
	}

	for i := 0; i < numLines; i++ {
		start := i * cols
		end := (i + 1) * cols

		if end <= length {
			result = append(result, line[start:end])
			continue
		}

		// It's the last line, split it up
		newLine := make(Line, cols)
		for j := start; j < end; j++ {
			if j < length {
				newLine[j-start] = line[j]
				continue
			}

			newLine[j-start] = EmptyGlyph()
		}
		result = append(result, newLine)
	}

	// Remove all existing attrWrap
	for _, line := range result {
		for _, glyph := range line {
			glyph.Mode ^= attrWrap
		}
	}

	// Mark attrWrap
	for i := 0; i < len(result)-1; i++ {
		result[i][cols-1].Mode = attrWrap
	}

	return result
}

// reflow recalculates the wrap point for all lines in `lines` and `history`.
func reflow(history, lines []Line, rows, cols int) ([]Line, []Line) {
	result := make([]Line, 0)

	var current Line = nil
	for _, line := range append(history, lines...) {
		// the line was wrapped originally, aggregate it
		wasWrapped := line[len(line)-1].Mode == attrWrap

		if current == nil {
			current = copyLine(line)
		} else {
			current = append(current, line...)
		}

		if wasWrapped {
			continue
		}

		result = append(result, wrapLine(current, cols)...)
		current = nil
	}

	if current != nil {
		result = append(result, wrapLine(current, cols)...)
	}

	// Remove trailing empty lines
	for i := len(result) - 1; i >= 0; i-- {
		if result[i] != nil {
			break
		}
		result = result[:i]
	}

	for i := range result {
		if result[i] != nil {
			continue
		}

		result[i] = emptyLine(cols)
	}

	numHistory := clamp(len(result)-rows, 0, len(result))
	newHistory := result[:numHistory]
	newLines := result[numHistory:]

	return newHistory, newLines
}