/*
Package fuzzy provides fuzzy string matching optimized
for filenames and code symbols in the style of Sublime Text,
VSCode, IntelliJ IDEA et al.
*/
package fuzzy

import (
	"sort"
	"unicode"
	"unicode/utf8"
)

//go:generate go install google.golang.org/protobuf/proto
//go:generate go install google.golang.org/protobuf/cmd/protoc-gen-go
//go:generate protoc --go_out=. ./fuzzy.proto

// Match represents a matched string.
type Match struct {
	// The matched string.
	Str string
	// The index of the matched string in the supplied slice.
	Index int
	// The indexes of matched characters. Useful for highlighting matches.
	MatchedIndexes []int
	// Score used to rank matches
	Score int
}

const (
	firstCharMatchBonus            = 10
	caseSensitiveBonus             = 3
	matchFollowingSeparatorBonus   = 20
	camelCaseMatchBonus            = 20
	adjacentMatchBonus             = 5
	unmatchedLeadingCharPenalty    = -5
	maxUnmatchedLeadingCharPenalty = -15
	separators                     = `/-_ .\`
)

// Matches is a slice of Match structures.
type Matches []Match

func (a Matches) Len() int           { return len(a) }
func (a Matches) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Matches) Less(i, j int) bool { return a[i].Score >= a[j].Score }

// Source represents an abstract source of a list of strings. Source must be iterable type such as a slice.
// The source will be iterated over till Len() with String(i) being called for each element where i is the
// index of the element. You can find a working example in the README.
type Source interface {
	// The string to be matched at position i.
	String(i int) string
	// The length of the source. Typically is the length of the slice of things that you want to match.
	Len() int
}

// StringSource is a simple implementation of the Source interface.
type StringSource []string

func (ss StringSource) String(i int) string {
	return ss[i]
}

func (ss StringSource) Len() int { return len(ss) }

/*
Find looks up pattern in data and returns matches
in descending order of match quality. Match quality
is determined by a set of bonus and penalty rules.

The following types of matches apply a bonus:

* The first character in the pattern matches the first character in the match string.

* The matched character is camel cased.

* The matched character follows a separator such as an underscore character.

* The matched character is adjacent to a previous match.

Penalties are applied for every character in the search string that wasn't matched and all leading
characters up to the first match.
*/
func Find(in string, dictionary []string) Matches {
	if len(in) == 0 {
		return nil
	}

	runes := []rune(in)
	var matches Matches
	var matchedIndexes []int

	for i := 0; i < len(dictionary); i++ {
		match := Match{
			Str:            dictionary[i],
			Index:          i,
			MatchedIndexes: matchedIndexes,
			Score:          0,
		}
		if matchedIndexes == nil {
			match.MatchedIndexes = make([]int, 0, len(runes))
		}

		if match.Compare(runes) {
			matches = append(matches, match)
			matchedIndexes = nil
		} else {
			matchedIndexes = match.MatchedIndexes[:0] // Recycle match index slice
		}
	}
	sort.Stable(matches)

	return matches
}

// FindFrom is an alternative implementation of Find
// using a Source instead of a slice of strings.
func FindFrom(pattern string, dictionary Source) Matches {
	if len(pattern) == 0 {
		return nil
	}

	runes := []rune(pattern)
	var matches Matches
	var matchedIndexes []int

	dataLen := dictionary.Len()
	for i := 0; i < dataLen; i++ {
		match := Match{
			Str:            dictionary.String(i),
			Index:          i,
			MatchedIndexes: matchedIndexes,
			Score:          0,
		}
		if matchedIndexes == nil {
			match.MatchedIndexes = make([]int, 0, len(runes))
		}

		if match.Compare(runes) {
			matches = append(matches, match)
			matchedIndexes = nil
		} else {
			matchedIndexes = match.MatchedIndexes[:0] // Recycle match index slice
		}
	}
	sort.Stable(matches)

	return matches
}

// Compare computes the matching between input and target.
func Compare(in, target string) *Match {
	match := Match{
		Str:            target,
		Index:          0,
		MatchedIndexes: nil,
		Score:          0,
	}

	if match.Compare([]rune(in)) {
		return &match
	}

	return nil
}

// Compare computes the matching between input and target.
func (match *Match) Compare(inRunes []rune) bool {
	var score int
	inRunesIndex := 0
	bestScore := -1
	matchedIndex := -1
	currAdjacentMatchBonus := 0
	var last rune
	var lastIndex int
	nextTargetRune, nextSize := utf8.DecodeRuneInString(match.Str)
	var candidate rune
	var candidateSize int

	for i := 0; i < len(match.Str); i += candidateSize {
		candidate, candidateSize = nextTargetRune, nextSize
		if score := equalRunesFold(inRunes, inRunesIndex, candidate); score > 0 {
			if i == 0 {
				score += firstCharMatchBonus
			}
			if unicode.IsLower(last) && unicode.IsUpper(candidate) {
				score += camelCaseMatchBonus
			}
			if i != 0 && isSeparator(last) {
				score += matchFollowingSeparatorBonus
			}
			if len(match.MatchedIndexes) > 0 {
				lastMatch := match.MatchedIndexes[len(match.MatchedIndexes)-1]
				bonus := adjacentCharBonus(lastIndex, lastMatch, currAdjacentMatchBonus)
				score += bonus
				// adjacent matches are incremental and keep increasing based on previous adjacent matches
				// thus we need to maintain the current match bonus
				currAdjacentMatchBonus += bonus
			}
			if score > bestScore {
				bestScore = score
				matchedIndex = i
			}
		}

		var nextInRune rune
		if inRunesIndex < len(inRunes)-1 {
			nextInRune = inRunes[inRunesIndex+1]
		}

		if i+candidateSize < len(match.Str) {
			if match.Str[i+candidateSize] < utf8.RuneSelf { // Fast path for ASCII
				nextTargetRune, nextSize = rune(match.Str[i+candidateSize]), 1
			} else {
				nextTargetRune, nextSize = utf8.DecodeRuneInString(match.Str[i+candidateSize:])
			}
		} else {
			nextTargetRune, nextSize = 0, 0
		}

		// We apply the best score when we have the next match coming up or when the search string has ended.
		// Tracking when the next match is coming up allows us to exhaustively find the best match and not necessarily
		// the first match.
		// For example given the pattern "tk" and search string "The Black Knight", exhaustively matching allows us
		// to match the second k thus giving this string a higher extra.
		if matchedIndex > -1 {
			if extra := equalFold(nextInRune, nextTargetRune); extra > 0 {
				if len(match.MatchedIndexes) == 0 {
					penalty := matchedIndex * unmatchedLeadingCharPenalty
					bestScore += max(penalty, maxUnmatchedLeadingCharPenalty)
				}
				match.Score += bestScore + extra
				match.MatchedIndexes = append(match.MatchedIndexes, matchedIndex)
				bestScore = -1
				inRunesIndex++
			}
		}

		lastIndex = i
		last = candidate
	}

	// apply penalty for each unmatched character
	penalty := len(match.MatchedIndexes) - len(match.Str)
	match.Score += penalty

	return len(match.MatchedIndexes) == len(inRunes)
}

func equalRunesFold(runes []rune, index int, targetRune rune) (score int) {
	if index >= len(runes) {
		return 0
	}

	return equalFold(runes[index], targetRune)
}

// Taken from strings.EqualFold.
func equalFold(inRune, targetRune rune) (score int) {
	if inRune == targetRune {
		return caseSensitiveBonus
	}

	if targetRune == 0 {
		return 1
	}

	if isSeparator(inRune) && isSeparator(targetRune) {
		return 1
	}

	if inRune < targetRune {
		inRune, targetRune = targetRune, inRune
	}

	// Fast check for ASCII.
	if inRune < utf8.RuneSelf {
		// if targetRune is upper case. inRune must be lower case.
		if targetRune <= 'Z' && 'A' <= targetRune && inRune == targetRune+'a'-'A' {
			return 1
		}

		return 0
	}

	// General case. SimpleFold(x) returns the next equivalent rune > x
	// or wraps around to smaller values.
	r := unicode.SimpleFold(targetRune)
	for r != targetRune && r < inRune {
		r = unicode.SimpleFold(r)
	}

	if r == inRune {
		return 1
	}

	return 0
}

func adjacentCharBonus(i, lastMatch, currentBonus int) int {
	if lastMatch == i {
		return currentBonus*2 + adjacentMatchBonus
	}

	return 0
}

func isSeparator(s rune) bool {
	for _, sep := range separators {
		if s == sep {
			return true
		}
	}

	return false
}

func max(x, y int) int {
	if x > y {
		return x
	}

	return y
}
