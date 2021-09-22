/*
Package fuzzy provides fuzzy string matching optimized
for filenames and code symbols in the style of Sublime Text,
VSCode, IntelliJ IDEA et al.
*/
package fuzzy

import (
	"sort"
	"strings"
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
	enableFasterCode               = true
	firstCharMatchBonus            = 10 // 16
	caseSensitiveBonus             = 1  // 3
	penaltyUnmatched               = 1  // 2
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

// stringSource is a simple implementation of the Source interface.
type stringSource []string

func (ss stringSource) String(i int) string { return ss[i] }
func (ss stringSource) Len() int            { return len(ss) }

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
func Find(source string, dictionary []string) Matches {
	return FindFrom(source, stringSource(dictionary))
}

// BestMatch is an optimized version of Find()
// assuming input is not empty and returning the best match.
func BestMatch(source string, dictionary []string) *Match {
	return BestMatchFrom(source, stringSource(dictionary))
}

// FindFrom is an alternative implementation of Find
// using a Source instead of a slice of strings.
func FindFrom(source string, dictionary Source) (matches Matches) {
	if source == "" {
		return nil
	}

	matchedIndexes := make([]int, 0, len(source))

	dicLen := dictionary.Len()
	for i := 0; i < dicLen; i++ {
		match := Match{
			Str:            dictionary.String(i),
			Index:          i,
			MatchedIndexes: matchedIndexes,
			Score:          0,
		}

		if match.Compare([]rune(source)) {
			matches = append(matches, match)
			matchedIndexes = make([]int, 0, len(source))
		} else {
			matchedIndexes = match.MatchedIndexes[:0] // Recycle match index slice
		}
	}

	sort.Stable(matches)

	return matches
}

// BestMatchFrom is an optimized version of FindFrom()
// assuming input is not empty and returning the best match.
func BestMatchFrom(source string, dictionary Source) *Match {
	best := &Match{
		Str:            "",
		Index:          0,
		MatchedIndexes: make([]int, 0, len(source)),
		Score:          -1,
	}

	match := &Match{
		Str:            "",
		Index:          0,
		MatchedIndexes: make([]int, 0, len(source)),
		Score:          0,
	}

	dicLen := dictionary.Len()
	for i := 0; i < dicLen; i++ {
		match.Str = dictionary.String(i)
		match.MatchedIndexes = match.MatchedIndexes[:0] // Recycle match index slice
		match.Score = 0

		if match.Compare([]rune(source)) && match.Score > best.Score {
			best, match = match, best
			best.Index = i
		}
	}

	if best.Score < 0 {
		return nil
	}

	return best
}

// Compare computes the matching between two strings: source and target.
func Compare(source, target string) *Match {
	match := Match{
		Str:            target,
		Index:          0,
		MatchedIndexes: nil,
		Score:          0,
	}

	if match.Compare([]rune(source)) {
		return &match
	}

	return nil
}

// Compare computes the matching between input and target.
func (match *Match) Compare(sourceRunes []rune) bool {
	sourceIndex := 0
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
		if score := equalRuneFold(sourceRunes, sourceIndex, candidate); score > 0 {
			score = 0
			if i == 0 {
				score = firstCharMatchBonus
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

		var nextSourceRune rune
		if sourceIndex < len(sourceRunes)-1 {
			nextSourceRune = sourceRunes[sourceIndex+1]
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
		// to match the second k thus giving this string a higher score.
		if matchedIndex > -1 {
			if extra := zeroOrFold(nextSourceRune, nextTargetRune); extra > 0 {
				if len(match.MatchedIndexes) == 0 {
					penalty := matchedIndex * unmatchedLeadingCharPenalty
					bestScore += max(penalty, maxUnmatchedLeadingCharPenalty)
				}
				match.Score += bestScore // + extra
				match.MatchedIndexes = append(match.MatchedIndexes, matchedIndex)
				bestScore = -1
				sourceIndex++
			}
		}

		lastIndex = i
		last = candidate
	}

	// apply penalty for each unmatched character
	penalty := (len(match.MatchedIndexes) - len(match.Str)) * penaltyUnmatched
	match.Score += penalty

	return len(match.MatchedIndexes) == len(sourceRunes)
}

func equalRuneFold(runes []rune, index int, targetRune rune) (score int) {
	if index >= len(runes) {
		return 0
	}

	return equalFold(runes[index], targetRune)
}

func zeroOrFold(sr, tr rune) (score int) {
	if tr == 0 {
		if sr == 0 {
			return 1
		}

		return 1
	}

	if sr == 0 {
		return 0
	}

	return equalFold(sr, tr)
}

func equalFold(tr, sr rune) (score int) {
	if enableFasterCode {
		return equalFoldNew(tr, sr)
	}

	return equalFoldOld(tr, sr)
}

// Taken from strings.EqualFold.
func equalFoldOld(tr, sr rune) (score int) {
	if tr == sr {
		return caseSensitiveBonus
	}

	if tr < sr {
		tr, sr = sr, tr
	}

	// Fast check for ASCII.
	if tr < utf8.RuneSelf {
		if isSeparator(tr) && isSeparator(sr) {
			return 1
		}

		// if targetRune is upper case. sourceRune must be lower case.
		if sr <= 'Z' && 'A' <= sr && tr == sr+'a'-'A' {
			return 1
		}

		return 0
	}

	// General case. SimpleFold(x) returns the next equivalent rune > x
	// or wraps around to smaller values.
	r := unicode.SimpleFold(sr)
	for r != sr && r < tr {
		r = unicode.SimpleFold(r)
	}

	if r == tr {
		return 1
	}

	return 0
}

func equalFoldNew(tr, sr rune) (score int) {
	if tr == sr {
		return caseSensitiveBonus
	}

	if tr < sr {
		tr, sr = sr, tr
	}

	// Fast check for ASCII.
	if tr < utf8.RuneSelf {
		if tr >= 'a' {
			if tr <= 'z' {
				return equalLowerUpperCase(tr, sr)
			}
		} else if '0' <= tr && tr <= 'Z' {
			return 0
		}

		return fastPunctuationCheck(sr)
	}

	// General case. SimpleFold(x) returns the next equivalent rune > x
	// or wraps around to smaller values.
	r := unicode.SimpleFold(sr)
	for r != sr && r < tr {
		r = unicode.SimpleFold(r)
	}

	if r == tr {
		return 1
	}

	return 0
}

// if tr is lower case. sr must be upper case.
func equalLowerUpperCase(tr, sr rune) (score int) {
	if tr == sr+'a'-'A' {
		return 1
	}

	return 0
}

// assumption: r is already in the lower part of the ASCII table.
func fastPunctuationCheck(r rune) (score int) {
	if r > 'Z' {
		if r < 'a' {
			return 1
		}
	} else if r < '0' {
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

func isSeparator(r rune) bool {
	if enableFasterCode {
		return strings.IndexByte(separators, byte(r)) >= 0
	}

	for _, sep := range separators {
		if r == sep {
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
