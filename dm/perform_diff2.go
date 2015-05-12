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

	// Phase 4: move detection (match a gap in A with some gap(s) in B)

	numMatchedLines, _ := middleBlockPairs.CountLinesInPairs()
	middleBlockPairs = PerformMoveDetectionInGaps(middleRangePair, middleBlockPairs, config, sf)
	newNumMatchedLines, _ := middleBlockPairs.CountLinesInPairs()
	glog.Infof("Found %d moved or copied lines", newNumMatchedLines-numMatchedLines)

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

	// TODO Phase 5: copy detection (match a gap in B with similar size region anywhere in file A)

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

	// Phase 6: common & normalized matches (grow unique line matches forward, then backwards).












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

// Extend matches forward, then backward. A line in A may be matched up with
// multiple lines in B, but only if the BlockPairs of those B lines have
// different MoveIds.

func MatchesExtender(filePair FilePair, blockPairs BlockPairs) {
	matchedBIndices := MakeIntervalSet()
	insertBIndices := func(pairs BlockPairs) {
		for _, pair := range pairs {
			matchedBIndices.InsertInterval(pair.BIndex, pair.BBeyond())
		}
	}
	containsAnyBIndices := func(pairs BlockPairs) bool {
		for _, pair := range pairs {
			if matchedBIndices.ContainsSome(pair.BIndex, pair.BBeyond()) {
				return true
			}
		}
		return false
	}



}





