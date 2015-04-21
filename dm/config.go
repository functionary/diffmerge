package dm

import (
	"flag"
)

// TODO Create a generator that generates the go source for CreateFlags,
// using a struct field tag to provide the name (else use a normalized
// form of the field name), and another tag for the default value (if not
// the zero value for the type), and uses the field comment as the usage
// text.

// Guides the process of producing a diff between two file ranges.
// For example, we may use this to represent the ranges between the common
// prefix and suffix of two files.

type DifferencerConfig struct {
	// Before computing the alignment between lines of two files, should
	// the common prefix and suffix be identified, reducing the number of
	// lines being aligned by the more general technique? (Improves the
	// alignment of inserted functions in C-like languages, as the trailing
	// curly braces get matched to the correct function more often.)
	matchEnds bool

	// When matching the common prefix and suffix, after matching full lines,
	// should common normalized prefix and suffix lines be matched?
	matchNormalizedEnds bool

	// When computing an alignment between files, should lines be normalized
	// before comparing (i.e. compare hashes of normalized lines, not of full
	// lines).
	alignNormalizedLines bool

	// When computing an alignment between files, should unique/rare lines be
	// used for computing the alignment, or all lines?
	alignRareLines bool

	// When deciding which lines are rare in a region being aligned, how many
	// times may a line appear (actually, how many times may its hash appear)
	// and still be considered rare?
	maxRareLineOccurrencesInRange int

	// When selecting rare lines, discard those lines whose hash, after
	// normalization, appears more than this many times. If 0, not applied.
	maxRareLineOccurrencesInFile int

	// When deciding which lines are rare in two regions being aligned,
	// must those lines appear the same number of times in each region?
	requireSameRarity bool

	// When computing an alignment between files, should blocks of moved lines
	// be detected (i.e. detect re-ordering of paragraphs/functions).
	detectBlockMoves bool

	// When computing the longest common subsequence of two file ranges,
	// how similar are two normalized lines to be considered, where 0 is
	// completely dissimilar, and 1 is equal.
	lcsNormalizedSimilarity float64

	// When computing an LCS alignment between files, should longer equal lines
	// be weighted more heavily that short lines?
	lengthWeightedSimilarity bool

	// When doing alignment (initial or move/copy detection), omit from
	// consideration the lines that are probably common (e.g. "/*" or "}").
	omitProbablyCommonLines bool
}

func (p *DifferencerConfig) CreateFlags(f *flag.FlagSet) {
	f.BoolVar(
		&p.matchEnds, "match-ends", true, `
		Before computing the alignment between lines of two files, should
		the common prefix and suffix be identified, reducing the number of
		lines being aligned by the more general technique? (Improves the
		alignment of inserted functions in C-like languages, as the trailing
		curly braces get matched to the correct function more often.)
		`)

	f.BoolVar(
		&p.matchNormalizedEnds, "match-normalized-ends", true, `
		When matching the common prefix and suffix, after matching full lines,
		should common normalized prefix and suffix lines be matched?
		`)

	f.BoolVar(
		&p.alignNormalizedLines, "align-normalized-lines", true, `
		When computing an alignment between files, should lines be normalized
		before comparing (i.e. compare hashes of normalized lines, not of full
		lines).
		`)

	f.BoolVar(
		&p.alignRareLines, "align-rare-lines", true, `
		When computing an alignment between files, should unique/rare lines be
		used for computing the alignment, or all lines?
		`)

	f.IntVar(
		&p.maxRareLineOccurrencesInRange, "max-rare-line-occurrences-in-range", 1, `
		When deciding which lines are rare in a region being aligned, how many
		times may a line appear (actually, how many times may its hash appear)
		and still be considered rare?
		`)

	f.IntVar(
		&p.maxRareLineOccurrencesInFile, "max-rare-line-occurrences-in-file", 3, `
		When selecting rare lines, discard those lines whose hash, after
		normalization, appears more than this many times. If 0, not applied.
		`)

	f.BoolVar(
		&p.requireSameRarity, "require-same-rarity", true, `
		When deciding which lines are rare in two regions being aligned,
		must those lines appear the same number of times in each region?
		`)

	f.BoolVar(
		&p.detectBlockMoves, "detect-block-moves", true, `
		When computing an alignment between files, should blocks of moved lines
		be detected (i.e. detect re-ordering of paragraphs/functions).
		`)

	f.Float64Var(
		&p.lcsNormalizedSimilarity, "lcs-normalized-similarity", 0.5, `
		When computing the longest common subsequence (LCS) of two file ranges,
	  how similar are two normalized lines to be considered, where 0 is
	  completely dissimilar, and 1 is equal.
		`)

	f.BoolVar(
		&p.lengthWeightedSimilarity, "length-weighted-similarity", true, `
		When computing an LCS alignment between files, should longer equal lines
		be weighted more heavily that short lines?
		`)

	f.BoolVar(
		&p.omitProbablyCommonLines, "omit-probably-common-lines", true, `
		When doing alignment (initial or move/copy detection), omit from
		consideration the lines that are probably common (e.g. "/*" or "}").
		`)
}

