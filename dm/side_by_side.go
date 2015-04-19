package dm

import (
	"bytes"
	"fmt"
	"io"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
)

// Format for side-by-side display, where B is primary (i.e. its lines
// appear exactly once and in order), and A is secondary (its lines may
// appear out of order, and possibly more than once).
// The purpose of this is to ease debugging (though testing should be
// done differently).
//
// Intended format:
//
// AAA aaaaaaaaa C bbbbbbbbb BBB
//
// Where AAA and BBB are the line numbers in their files, aaaa and bbbb are
// the contents of the line.
// If the line wraps, the line number will be the double quote character,
// meaning ditto.
// The C character (code) in the middle will represent the kind of change:
//   = means lines are the same
//   ~ means lines are the same after normalization
//   ! means lines are different
//   < means lines inserted in A
//   > means lines inserted in B
//   M means a move is detected

// TODO Maybe handle non-ASCII files, i.e. multi-byte characters, which display
// as a single character, but are multiple bytes of input, which throws off
// width calculations/alignment.

// Inputs to display process, unrelated to the actual files.
type SideBySideConfig struct {
	// How many columns (mono-spaced characters) does the output 'device' have?
	DisplayColumns int

	DisplayLineNumbers bool
	//	displayEntireFiles bool  // TODO
	WrapLongLines bool // Wrap (vs. truncate) long lines.

	SpacesPerTab int // Defaults to 8
}

var DefaultSideBySideConfig = SideBySideConfig{
	DisplayColumns:     80,
	DisplayLineNumbers: true,
	SpacesPerTab:       8,
	WrapLongLines:      true,
}

// TODO Measure width of longest line in each file so that we can decide to
// give more space to one file than the other IFF they require different widths.

type sideBySideState struct {
	cfg SideBySideConfig

	// The files being compared/displayed.
	aFile, bFile *File
	// // The exact and approximate matches, moves and copies, and differences.
	pairs []*BlockPair

	aDigitColumns, aOutputColumns int
	bDigitColumns, bOutputColumns int

	aDigitOffset, aOutputOffset int
	bDigitOffset, bOutputOffset int
	codeOffset                  int

	lineBuf    []byte
	lineBuffer *bytes.Buffer
	w          io.Writer

	lineFormat string
}

func (p *SideBySideConfig) lineToOutputBufs(line []byte, numColumns int) (bufs [][]byte) {
	var curBuf []byte
	bytesOutput := 0
	stop := false
	doOutput := func(b byte) {
		if len(curBuf) >= numColumns {
			bufs = append(bufs, curBuf)
			stop = !p.WrapLongLines
			curBuf = make([]byte, 0, numColumns)
		}
		curBuf = append(curBuf, b)
		bytesOutput++
	}
	for _, b := range line {
		// Only printable ASCII for now, plus tabs.
		if 32 <= b && b <= 126 {
			doOutput(b)
		} else if b == '\t' {
			bo := bytesOutput + 1
			nextTabStop := bo + (p.SpacesPerTab - bo%p.SpacesPerTab)
			for bytesOutput < nextTabStop {
				doOutput(' ')
			}
		} else if b == '\n' || b == '\r' {
			// Suppress
		} else {
			doOutput(176) // Based on code page 437 on windows, this is a gray block.
		}
		if stop {
			return bufs[0:1]
		}
	}
	if len(curBuf) > 0 {
		bufs = append(bufs, curBuf)
	}
	return bufs
}

// The C character (code) in the middle will represent the kind of change:
//   = means lines are the same
//   ~ means lines are the same after normalization
//   ! means lines are different
//   < means lines inserted in A
//   > means lines inserted in B
//   M means a move is detected, of exact lines
//   m means a move is detected, with normalization

func (state *sideBySideState) getCodeForBlockPair(pair *BlockPair) byte {
	if pair.IsMatch {
		if pair.IsMove {
			return 'M'
		} else {
			return '='
		}
	}
	if pair.IsNormalizedMatch {
		if pair.IsMove {
			return 'm'
		} else {
			return '~'
		}
	}
	if pair.ALength <= 0 {
		if pair.BLength <= 0 {
			glog.Fatalf("Invalid BlockPair: %v", *pair)
		}
		return '>'
	} else if pair.BLength <= 0 {
		return '<'
	} else {
		return '!'
	}
}

func selectOutputBuf(bufs [][]byte, n, cols int) (buf []byte) {
	if n < len(bufs) {
		buf = bufs[n]
	} else {
		buf = make([]byte, 0, cols)
	}
	// Pad short bufs
	for len(buf) < cols {
		buf = append(buf, ' ')
	}
	return
}

func (state *sideBySideState) outputABLines(aIndex, bIndex int, code string) {
	var aBufs, bBufs [][]byte
	if aIndex >= 0 {
		aBytes := state.aFile.GetLineBytes(aIndex)
		aBufs = state.cfg.lineToOutputBufs(aBytes, state.aOutputColumns)
	}
	if bIndex >= 0 {
		bBytes := state.bFile.GetLineBytes(bIndex)
		bBufs = state.cfg.lineToOutputBufs(bBytes, state.bOutputColumns)
	}

	limit := maxInt(1, maxInt(len(aBufs), len(bBufs))) // If both are blank, want at least 1.

	glog.Infof("outputABLines: %d, %d, %s;  #aBufs %d; #bBufs %d; limit %d",
		aIndex, bIndex, code, len(aBufs), len(bBufs), limit)

	for n := 0; n < limit; n++ {
		aBuf := selectOutputBuf(aBufs, n, state.aOutputColumns)
		bBuf := selectOutputBuf(bBufs, n, state.bOutputColumns)
		if state.cfg.DisplayLineNumbers {
			var aLineNo, bLineNo string
			if n == 0 {
				if aIndex >= 0 {
					aLineNo = fmt.Sprintf("%d", aIndex+1)
				}
				if bIndex >= 0 {
					bLineNo = fmt.Sprintf("%d", bIndex+1)
				}
			} else {
				if aIndex >= 0 {
					aLineNo = "\""
				}
				if bIndex >= 0 {
					bLineNo = "\""
				}
			}
			fmt.Fprintf(state.w, state.lineFormat, aLineNo, aBuf, code, bBuf, bLineNo)
		} else {
			fmt.Fprintf(state.w, state.lineFormat, aBuf, code, bBuf)
		}
	}
}

func (state *sideBySideState) outputBlockPair(pair *BlockPair) {

	glog.Infof("outputBlockPair: %v", *pair)

	code := string([]byte{state.getCodeForBlockPair(pair)})
	limit := maxInt(pair.ALength, pair.BLength)
	for i := 0; i < limit; i++ {
		var aIndex, bIndex int
		if i < pair.ALength {
			aIndex = pair.AIndex + i
		} else {
			aIndex = -1
		}
		if i < pair.BLength {
			bIndex = pair.BIndex + i
		} else {
			bIndex = -1
		}
		state.outputABLines(aIndex, bIndex, code)
	}
}

func FormatSideBySide(aFile, bFile *File, pairs []*BlockPair, aIsPrimary bool,
	w io.Writer, config SideBySideConfig) error {
	pairs = append([]*BlockPair(nil), pairs...)
	if aIsPrimary {
		SortBlockPairsByAIndex(pairs)
	} else {
		SortBlockPairsByBIndex(pairs)
	}

	state := &sideBySideState{
		cfg:   config,
		aFile: aFile,
		bFile: bFile,
		pairs: pairs,
		w:     w,
	}

	// Subtract space for the code character and a space on either side.
	availableOutputColumns := state.cfg.DisplayColumns - 3

	if state.cfg.DisplayLineNumbers {
		state.aDigitColumns = DigitCount(maxInt(2, aFile.GetLineCount()))
		state.bDigitColumns = DigitCount(maxInt(2, bFile.GetLineCount()))
		availableOutputColumns -= (state.aDigitColumns + state.bDigitColumns + 2)
	} else {
		state.aDigitColumns = 0
		state.aDigitOffset = -1 // Intended to cause an OOBE if used.
		state.bDigitColumns = 0
		state.bDigitOffset = -1 // Intended to cause an OOBE if used.
	}

	state.aOutputColumns = maxInt(availableOutputColumns/2, 10)
	state.bOutputColumns = state.aOutputColumns

	var totalColumns int
	if state.cfg.DisplayLineNumbers {
		state.aDigitOffset = 0
		state.aOutputOffset = 1 + state.aDigitColumns + state.aDigitOffset
		state.codeOffset = 1 + state.aOutputOffset + state.aOutputColumns
		state.bOutputOffset = 2 + state.aOutputOffset
		state.bDigitOffset = 1 + state.bOutputOffset + state.bOutputColumns
		totalColumns = state.bDigitOffset + state.bDigitColumns

		state.lineFormat = fmt.Sprintf("%%%ds %%s %%s %%s %%-%ds\n", state.aDigitColumns, state.bDigitColumns)
	} else {
		state.aOutputOffset = 0
		state.codeOffset = 1 + state.aOutputOffset + state.aOutputColumns
		state.bOutputOffset = 2 + state.aOutputOffset
		totalColumns = state.bOutputOffset + state.bOutputColumns

		state.lineFormat = "%s %s %s\n"
	}

	glog.Info(spew.Sdump(state))

	state.lineBuf = make([]byte, totalColumns)
	state.lineBuffer = bytes.NewBuffer(state.lineBuf)

	for _, pair := range pairs {
		state.outputBlockPair(pair)
	}

	// TODO Calculate how much width we need for line numbers (based on number
	// of digits required for largest line number, 1-based, so that is number
	// of lines in the larger file)
	// TODO Calculate how much width we assign to each file, leaving room for
	// leading digits (might not be the same if we have an odd number of chars
	// available).
	// TODO Consider issue related to multibyte runes.

	return nil
}