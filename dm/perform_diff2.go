package dm

import (
	"github.com/golang/glog"
)

func PerformDiff2(aFile, bFile *File, config DifferencerConfig) (pairs []*BlockPair) {
	defer glog.Flush()
	if aFile.LineCount() == 0 {
		if bFile.LineCount() == 0 {
			return nil // They're the same.
		}
		pair := &BlockPair{
			AIndex:  0,
			ALength: 0,
			BIndex:  0,
			BLength: bFile.LineCount(),
		}
		return append(pairs, pair)
	} else if bFile.LineCount() == 0 {
		pair := &BlockPair{
			AIndex:  0,
			ALength: aFile.LineCount(),
			BIndex:  0,
			BLength: 0,
		}
		return append(pairs, pair)
	}
	filePair := MakeFilePair(aFile, bFile)
	rootRangePair := filePair.FullFileRangePair()

	// Phase 1: Match ends

	var mase *middleAndSharedEnds
	middleRangePair := rootRangePair
	if config.MatchEnds {
		mase = FindMiddleAndSharedEnds(rootRangePair, config)
		if mase != nil {
			if mase.sharedEndsData.RangesAreEqual {
				return
			} else if mase.sharedEndsData.RangesAreApproximatelyEqual {
				glog.Info("PerformDiff2: files are identical after normalization")
				// TODO Calculate indentation changes.
				panic("TODO Calculate indentation changes. Make BlockPairs")
			}
			middleRangePair = mase.middleRangePair
		}
	}

	// Phase 2: LCS alignment.

	maxRareOccurrences := uint8(MaxInt(1, MinInt(255, config.MaxRareLineOccurrencesInFile)))
	normSim := MaxFloat32(0, MinFloat32(1, float32(config.LcsNormalizedSimilarity)))
	halfDelta := (1 - normSim) / 2
	sf := SimilarityFactors{
		MaxRareOccurrences: maxRareOccurrences,
		ExactRare:          1,
		NormalizedRare:     normSim,
		ExactNonRare:       1 - halfDelta,
		NormalizedNonRare:  MaxFloat32(0, normSim-halfDelta),
	}
	if !config.AlignNormalizedLines {
		sf.NormalizedRare = 0
		sf.NormalizedNonRare = 0
	}
	if config.AlignRareLines {
		sf.ExactNonRare = 0
		sf.NormalizedNonRare = 0
	}

	lcsData := PerformLCS(middleRangePair, config, sf)

	if false {
		if mase != nil {
			pairs = append(pairs, mase.sharedPrefixPairs...)
		}
		if lcsData != nil {
			pairs = append(pairs, lcsData.lcsPairs...)
		}
		if mase != nil {
			pairs = append(pairs, mase.sharedSuffixPairs...)
		}
		SortBlockPairsByAIndex(pairs)
		return
	}

	// TODO Phase 3: Small edit detection (nearly match gap in A with corresponding
	// gap in B)

	var middleBlockPairs BlockPairs
	if lcsData != nil {
		middleBlockPairs = append(middleBlockPairs, lcsData.lcsPairs...)
	}
	middleBlockPairs = PerformSmallEditDetectionInGaps(middleRangePair, middleBlockPairs, config)

	if false {
		if mase != nil {
			pairs = append(pairs, mase.sharedPrefixPairs...)
		}
		pairs = append(pairs, middleBlockPairs...)
		if mase != nil {
			pairs = append(pairs, mase.sharedSuffixPairs...)
		}
		SortBlockPairsByAIndex(pairs)
		return
	}

	// TODO Phase 4: move detection (match a gap in A with some gap(s) in B)

	numMatchedLines, _ := middleBlockPairs.CountLinesInPairs()
	passes := 0
	for {
		passes++
		var multipleCandidates bool
		middleBlockPairs, multipleCandidates = PerformMoveDetectionInGaps(middleRangePair, middleBlockPairs, config, sf)
		newNumMatchedLines, _ := middleBlockPairs.CountLinesInPairs()
		glog.Infof("Found %d moved lines on pass %d", newNumMatchedLines-numMatchedLines, passes)
		numMatchedLines = newNumMatchedLines
		if !multipleCandidates {
			glog.Infof("Finished looking for move candidates on pass %d", passes)
			break
		}
		glog.Info("There were multiple candidates for some gaps, repeating process")
	}

	if false {
		if mase != nil {
			pairs = append(pairs, mase.sharedPrefixPairs...)
		}
		pairs = append(pairs, middleBlockPairs...)
		if mase != nil {
			pairs = append(pairs, mase.sharedSuffixPairs...)
		}
		SortBlockPairsByAIndex(pairs)
		return
	}

	// TODO Phase 5: copy detection (match a gap in B with similar size region in B)

	//	rootDiffer.BaseRangesAreNotEmpty()

	//
	//
	//// Note that common prefix may overlap, as when comparing these two strings
	//// for common prefix and suffix: "ababababababa" and "ababa".
	//// Returns true if fully consumed.
	//func (p *simpleDiffer) MeasureCommonEnds(onlyExactMatches bool, maxRareOccurrences uint8) (rangesSame bool) {
	//	return p.baseRangePair.MeasureCommonEnds(onlyExactMatches, maxRareOccurrences)
	//}
	//
	pairs = nil
	if mase != nil {
		pairs = append(pairs, mase.sharedPrefixPairs...)
	}
	pairs = append(pairs, middleBlockPairs...)
	if mase != nil {
		pairs = append(pairs, mase.sharedSuffixPairs...)
	}
	SortBlockPairsByAIndex(pairs)
	return pairs
}