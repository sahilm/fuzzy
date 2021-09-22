package fuzzy

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/protobuf/proto"
)

func FuzzFuzzyFind(data []byte) int {
	args := &FindArgs{
		Pattern: new(string),
		Datas:   []string{},
	}

	err := proto.Unmarshal(data, args)
	if err != nil {
		return 0
	}

	matches := Find(*args.Pattern, args.Datas)
	for _, match := range matches {
		for i := 0; i < len(match.Str); i++ {
			for _, j := range match.MatchedIndexes {
				if j == i {
					// fmt.Printf("found %#+v\n", match)
					break
				}
			}
		}
	}

	return 1
}

func TestFindWithUnicode(t *testing.T) {
	matches := Find("\U0001F41D", []string{"\U0001F41D"})
	if len(matches) != 1 {
		t.Errorf("got %v Matches; expected 1 match", len(matches))
	}

	best := BestMatch("\U0001F41D", []string{"\U0001F41D"})
	if best == nil {
		t.Error("got best=nil; expected 1 match")
	}
}

func TestFindWithCannedData(t *testing.T) {
	cases := []struct {
		pattern string
		data    []string
		matches []Match
	}{
		// first char bonus, camel case bonuses and unmatched chars penalty
		// (m = 10, n = 20, r = 20) - 18 unmatched chars = 32
		{
			"mnr", []string{"moduleNameResolver.ts"}, []Match{
				{
					Str:            "moduleNameResolver.ts",
					Index:          0,
					MatchedIndexes: []int{0, 6, 10},
					Score:          38,
				},
			},
		},
		{
			"mmt", []string{"mémeTemps"}, []Match{
				{
					Str:            "mémeTemps",
					Index:          0,
					MatchedIndexes: []int{0, 3, 5},
					Score:          29,
				},
			},
		},
		// ranking
		{
			"mnr", []string{"moduleNameResolver.ts", "my name is_Ramsey"}, []Match{
				{
					Str:            "my name is_Ramsey",
					Index:          1,
					MatchedIndexes: []int{0, 3, 11},
					Score:          42,
				},
				{
					Str:            "moduleNameResolver.ts",
					Index:          0,
					MatchedIndexes: []int{0, 6, 10},
					Score:          38,
				},
			},
		},
		// simple repeated pattern and adjacent match bonus
		{
			"aaa", []string{"aaa", "bbb"}, []Match{
				{
					Str:            "aaa",
					Index:          0,
					MatchedIndexes: []int{0, 1, 2},
					Score:          36,
				},
			},
		},
		// exhaustive matching
		{
			"tk", []string{"The Black Knight"}, []Match{
				{
					Str:            "The Black Knight",
					Index:          0,
					MatchedIndexes: []int{0, 10},
					Score:          20,
				},
			},
		},
		// any unmatched char in the pattern removes the whole match
		{
			"cats", []string{"cat"}, []Match{},
		},
		// empty patterns return no Matches
		{
			"", []string{"cat"}, []Match{},
		},
		// separator bonus
		{
			"abcx", []string{"abc\\x"}, []Match{
				{
					Str:            "abc\\x",
					Index:          0,
					MatchedIndexes: []int{0, 1, 2, 4},
					Score:          57,
				},
			},
		},
	}
	for _, c := range cases {
		t.Run("Find="+c.pattern, func(t *testing.T) {
			matches := Find(c.pattern, c.data)
			if len(matches) != len(c.matches) {
				t.Errorf("got %v Matches; expected %v match", len(matches), len(c.matches))
			}
			if diff := pretty.Compare(c.matches, matches); diff != "" {
				t.Errorf("%v", diff)
			}
		})

		t.Run("Best="+c.pattern, func(t *testing.T) {
			best := BestMatch(c.pattern, c.data)
			if best == nil && len(c.matches) > 0 {
				t.Errorf("got best=%v ; expected %v match", best, len(c.matches))
			}
			if best != nil && len(c.matches) == 0 {
				t.Errorf("got best=%v ; expected %v match", best, len(c.matches))
			}
			if best != nil && len(c.matches) > 0 {
				if diff := pretty.Compare(c.matches[0], best); diff != "" {
					t.Errorf("%v", diff)
				}
			}
		})
	}
}

type employee struct {
	name string
}

type employees []employee

func (e employees) String(i int) string {
	return e[i].name
}

func (e employees) Len() int {
	return len(e)
}

func TestFindFromSource(t *testing.T) {
	emps := employees{
		{
			name: "Alice",
		},
		{
			name: "Bob",
		},
		{
			name: "Allie",
		},
	}

	want := Matches{
		Match{
			Str:            "Allie",
			Index:          2,
			MatchedIndexes: []int{0, 1},
			Score:          16,
		}, Match{
			Str:            "Alice",
			Index:          0,
			MatchedIndexes: []int{0, 1},
			Score:          16,
		},
	}

	t.Run("FindFrom", func(t *testing.T) {
		got := FindFrom("al", emps)
		if diff := pretty.Compare(want, got); diff != "" {
			t.Errorf("%v", diff)
		}
	})
}

func TestFindWithRealworldData(t *testing.T) {
	t.Run("with unreal 4 file names", func(t *testing.T) {
		cases := []struct {
			pattern    string
			numMatches int
			filenames  []string
		}{
			{
				"ue4", 4, []string{
					"UE4Game.cpp",
					"UE4Build.cs",
					"UE4Game.Build.cs",
					"UE4BuildUtils.cs",
				},
			},
			{
				"lll", 3, []string{
					"LogFileLogger.cs",
					"LockFreeListImpl.h",
					"LevelExporterLOD.h",
				},
			},
			{
				"aes", 3, []string{
					"AES.h",
					"AES.cpp",
					"ActiveSound.h",
				},
			},
		}

		bytes, err := ioutil.ReadFile("testdata/ue4_filenames.txt")
		if err != nil {
			t.Fatal(err)
		}

		filenames := strings.Split(string(bytes), "\n")

		for _, c := range cases {
			now := time.Now()
			matches := Find(c.pattern, filenames)
			elapsed := time.Since(now)

			if matches == nil || len(matches) < c.numMatches {
				t.Errorf("Got matches=%v ; want at least %v", len(matches), c.numMatches)

				continue
			}

			t.Logf("Matching '%v' in Unreal 4 found %v Matches in %v\n", c.pattern, len(matches), elapsed)

			foundfilenames := make([]string, 0)
			for i := 0; i < c.numMatches; i++ {
				foundfilenames = append(foundfilenames, matches[i].Str)
			}
			if diff := pretty.Compare(c.filenames, foundfilenames); diff != "" {
				t.Errorf("%v", diff)
			}

			now = time.Now()
			best := BestMatch(c.pattern, filenames)
			elapsed = time.Since(now)
			t.Logf("Best '%v' in Unreal 4 in %v\n", c.pattern, elapsed)
			if best == nil {
				t.Error("Got best=nil ; expected a match")
			}
		}
	})

	t.Run("with linux kernel file names", func(t *testing.T) {
		cases := []struct {
			pattern    string
			numMatches int
			filenames  []string
		}{

			{
				"make", 4, []string{
					"make",
					"makelst",
					"Makefile",
					"Makefile",
				},
			},
			{
				"alsa", 4, []string{
					"alsa.h",
					"alsa.c",
					"aw2-alsa.c",
					"cx88-alsa.c",
				},
			},
		}

		bytes, err := ioutil.ReadFile("testdata/linux_filenames.txt")
		if err != nil {
			t.Fatal(err)
		}

		filenames := strings.Split(string(bytes), "\n")

		for _, c := range cases {
			now := time.Now()
			matches := Find(c.pattern, filenames)
			elapsed := time.Since(now)
			t.Logf("Matching '%v' in linux kernel found %v Matches in %v\n", c.pattern, len(matches), elapsed)

			foundfilenames := make([]string, 0)
			if len(matches) < c.numMatches {
				t.Fatal("Too few Matches")
			}
			for i := 0; i < c.numMatches; i++ {
				foundfilenames = append(foundfilenames, matches[i].Str)
			}
			if diff := pretty.Compare(c.filenames, foundfilenames); diff != "" {
				t.Errorf("%v", diff)
			}

			now = time.Now()
			best := BestMatch(c.pattern, filenames)
			elapsed = time.Since(now)
			t.Logf("Best '%v' in Unreal 4 in %v\n", c.pattern, elapsed)
			if best == nil {
				t.Error("Got best=nil ; expected a match")
			}
		}
	})
}

func BenchmarkFind(b *testing.B) {
	b.Run("with unreal 4 (~16K files)", func(b *testing.B) {
		bytes, err := ioutil.ReadFile("testdata/ue4_filenames.txt")
		if err != nil {
			b.Fatal(err)
		}
		filenames := strings.Split(string(bytes), "\n")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Find("lll", filenames)
		}
	})

	b.Run("with linux kernel (~60K files)", func(b *testing.B) {
		bytes, err := ioutil.ReadFile("testdata/linux_filenames.txt")
		if err != nil {
			b.Fatal(err)
		}
		filenames := strings.Split(string(bytes), "\n")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Find("alsa", filenames)
		}
	})
}

func BenchmarkBest(b *testing.B) {
	b.Run("with unreal 4 (~16K files)", func(b *testing.B) {
		bytes, err := ioutil.ReadFile("testdata/ue4_filenames.txt")
		if err != nil {
			b.Fatal(err)
		}
		filenames := strings.Split(string(bytes), "\n")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			BestMatch("lll", filenames)
		}
	})

	b.Run("with linux kernel (~60K files)", func(b *testing.B) {
		bytes, err := ioutil.ReadFile("testdata/linux_filenames.txt")
		if err != nil {
			b.Fatal(err)
		}
		filenames := strings.Split(string(bytes), "\n")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			BestMatch("alsa", filenames)
		}
	})
}

type testCase struct{ source, want string }

func initDictionary(kind string) []string {
	switch kind {
	default:
		return []string{
			/*0*/ "Limit Book",
			/*1*/ "Order Book by Limit",
			/*2*/ "Full Book",
			/*3*/ "Full Order Book",
			/*4*/ "BinanceJersey",
			/*5*/ "Binance Jersey",
			"LILI_BOBO", "limik-tobo", "LimikTobo", "LIMIK-BOTO", "TILIM KOOB", "tilim-koob", "tilimkoob",
			"LUFL KOBO", "LUFLBOKO", "lufl.kobo", "lufl boko", "LuflKobo", "Lufl Kobo", "LufL-KoBo", "LufL KooB",
			"King Gizzard", "The Lizard Wizard", "Lizzard Wizzard",
		}
	case "lower":
		return []string{
			/*0*/ "limit book",
			/*1*/ "order book by limit",
			/*2*/ "full book",
			/*3*/ "full order book",
			/*4*/ "binancejersey",
			/*5*/ "binance jersey",
			"lili_bobo", "limik-tobo", "limiktobo", "limik-boto", "tilim koob", "tilim-koob", "tilimkoob",
			"lufl kobo", "luflboko", "lufl.kobo", "lufl boko", "luflkobo", "lufl kobo", "lufl-kobo", "lufl koob",
			"king gizzard", "the lizard wizard", "lizzard wizzard",
		}
	case "upper":
		return []string{
			/*0*/ "LIMIT BOOK",
			/*1*/ "ORDER BOOK BY LIMIT",
			/*2*/ "FULL BOOK",
			/*3*/ "FULL ORDER BOOK",
			/*4*/ "BINANCEJERSEY",
			/*5*/ "BINANCE JERSEY",
			"LILI_BOBO", "LIMIK-TOBO", "LIMIKTOBO", "LIMIK-BOTO", "TILIM KOOB", "TILIM-KOOB", "TILIMKOOB",
			"LUFL KOBO", "LUFLBOKO", "LUFL.KOBO", "LUFL BOKO", "LUFLKOBO", "LUFL KOBO", "LUFL-KOBO", "LUFL KOOB",
			"KING GIZZARD", "THE LIZARD WIZARD", "LIZZARD WIZZARD",
		}
	}
}

func initTestCases(dictionary []string) []testCase {
	return []testCase{
		{source: "limit", want: dictionary[0]},
		{source: "LImit", want: dictionary[0]},
		{source: "Limit", want: dictionary[0]},
		{source: "Book by", want: dictionary[1]},
		{source: "Full", want: dictionary[2]},
		{source: "ul Boo", want: dictionary[2]},
		{source: "ul ord", want: dictionary[3]},
		{source: "FullBook", want: dictionary[2]},
		{source: "fullbook", want: dictionary[2]},
		{source: "full-book", want: dictionary[2]},
		{source: "full.book", want: dictionary[2]},
		{source: "full/book", want: dictionary[2]},
		{source: "FULL_BOOK", want: dictionary[2]},
		{source: "LimitBook", want: dictionary[0]},
		{source: "limit-book", want: dictionary[0]},
		{source: "LIMIT_BOOK", want: dictionary[0]},
		{source: "LIMIT.BOOK", want: dictionary[0]},
		{source: "LIMIT/BOOK", want: dictionary[0]},
		{source: "BINANCE_JERSEY", want: dictionary[5]},
	}
}

func testFind(t *testing.T, source, want string, dico []string) {
	t.Helper()

	matches := Find(source, dico)

	if matches == nil {
		t.Errorf("in=%q got=nil want=%q", source, want)

		return
	}

	if len(matches) == 0 {
		t.Errorf("in=%q got=empty want=%q", source, want)

		return
	}

	if got := dico[matches[0].Index]; got != want {
		t.Errorf("in=%q got=%q want=%q", source, matches[0].Index, want)
		t.Logf("matches=%+v", matches)
	}
}

func testBest(t *testing.T, source, want string, dico []string) {
	t.Helper()

	best := BestMatch(source, dico)

	if best == nil {
		t.Errorf("in=%q got=nil want=%q", source, want)

		return
	}

	if got := dico[best.Index]; got != want {
		t.Errorf("in=%q got={index:%v str:%q} want=%q", source, best.Index, dico[best.Index], want)
		t.Logf("best=%+v", best)
	}
}

func TestUpperLowerCases(t *testing.T) {
	dictionaries := map[string][]string{
		"vanilla": initDictionary("vanilla"),
		"lower":   initDictionary("lower"),
		"upper":   initDictionary("upper"),
	}

	for kind, dictionary := range dictionaries {
		cases := initTestCases(dictionary)

		for _, c := range cases {
			t.Run(kind+"/Find="+c.source, func(t *testing.T) {
				testFind(t, c.source, c.want, dictionary)
			})
			t.Run(kind+"/find="+c.source, func(t *testing.T) {
				testFind(t, strings.ToLower(c.source), c.want, dictionary)
			})
			t.Run(kind+"/FIND="+c.source, func(t *testing.T) {
				testFind(t, strings.ToUpper(c.source), c.want, dictionary)
			})

			t.Run(kind+"/Best="+c.source, func(t *testing.T) {
				testBest(t, c.source, c.want, dictionary)
			})
			t.Run(kind+"/best="+c.source, func(t *testing.T) {
				testBest(t, strings.ToLower(c.source), c.want, dictionary)
			})
			t.Run(kind+"/BEST="+c.source, func(t *testing.T) {
				testBest(t, strings.ToUpper(c.source), c.want, dictionary)
			})
		}
	}
}

func TestMatch_Compare(t *testing.T) {
	cases := []struct {
		source string
		target string
		want   bool
	}{
		{source: "Full Book", target: "FULL_BOOK", want: true},
		{source: "Full Book", target: "full-book", want: true},
		{source: "Full Book", target: "full.book", want: true},
		{source: "Full Book", target: "full/book", want: true},
		/* TODO fail
		{source: "full book", target: "FullBook", want: true},
		{source: "Full Book", target: "fullbook", want: true},
		{source: "Full Book", target: "FullBook", want: true},
		{source: "FULL BOOK", target: "FullBook", want: true},
		*/
		{source: "FULL_BOOK", target: "Full Book", want: true},
		{source: "FULL_BOOK", target: "full.book", want: true},
		{source: "full-book", target: "Full Book", want: true},
		/* TODO fail
		{source: "full-book", target: "FullBook", want: true},
		*/
		{source: "full.book", target: "Full Book", want: true},
		{source: "full.book", target: "FULL_BOOK", want: true},
		{source: "full.book", target: "full/book", want: true},
		{source: "full/book", target: "Full Book", want: true},
		{source: "full/book", target: "full.book", want: true},
		{source: "fullbook", target: "Full Book", want: true},
		{source: "FullBook", target: "full book", want: true},
		{source: "FullBook", target: "Full Book", want: true},
		{source: "FullBook", target: "FULL BOOK", want: true},
		{source: "FullBook", target: "full-book", want: true},
		{source: "Limit Book", target: "LIMIT_BOOK", want: true},
		{source: "Limit Book", target: "limit-book", want: true},
		{source: "Limit Book", target: "LIMIT.BOOK", want: true},
		{source: "Limit Book", target: "LIMIT/BOOK", want: true},
		/* TODO fail
		{source: "Limit Book", target: "LimitBook", want: true},
		*/
		{source: "LIMIT_BOOK", target: "Limit Book", want: true},
		{source: "limit-book", target: "Limit Book", want: true},
		{source: "LIMIT.BOOK", target: "Limit Book", want: true},
		{source: "LIMIT/BOOK", target: "Limit Book", want: true},
		{source: "LimitBook", target: "Limit Book", want: true},
	}

	for _, c := range cases {
		t.Run(c.source+"=="+c.target, func(t *testing.T) {
			match := Match{
				Str:            c.target,
				Index:          0,
				MatchedIndexes: nil,
				Score:          0,
			}

			got := match.Compare([]rune(c.source))

			if got != c.want {
				t.Errorf("in=%q target=%q got=%v want=%v", c.source, c.target, got, c.want)
				t.Logf("match=%+v", match)
			}
		})
	}
}

func rStr(r rune) string {
	s := string(r)

	if r < ' ' || !utf8.ValidRune(r) {
		s = fmt.Sprintf("%#x", r)
	}

	return s
}

func Test_equalFold_range(t *testing.T) {
	for r1 := rune(0); r1 < rune(9999); r1++ {
		r2 := r1 + 'A' - 'a'

		name := string(r1) + "==" + string(r2)
		t.Run(name, func(t *testing.T) {
			r3 := unicode.SimpleFold(r2)
			for r3 != r2 && r3 < r1 {
				r3 = unicode.SimpleFold(r3)
			}

			want := 0
			if r3 == r1 {
				want = 1
			}

			if testEqualFold(t, r1, r2, want) {
				testEqualFold(t, r2, r1, want)
			}
		})
	}
}

func Test_equalFold(t *testing.T) {
	cases := []struct {
		sr   rune
		tr   rune
		want int
	}{
		{'', -0x19, 0},
		{0x8, -0x18, 0},
		{'a', 'a', caseSensitiveBonus},
		{'-', '-', caseSensitiveBonus},
		{'3', '3', caseSensitiveBonus},
		{'*', '*', caseSensitiveBonus},
		{'R', 'R', caseSensitiveBonus},
		{' ', 'a', 0},
		{'a', 'A', 1},
		{'Z', 'z', 1},
		{'a', 'z', 0},
		{'A', 'z', 0},
		{'"', 'z', 0},
		{'#', 'A', 0},
		{'$', '9', 0},
		{'(', '@', 0},
		{'*', '1', 0},
		{'-', '←', 0},
		{'-', '_', 1},
		{'.', '↑', 0},
		// {')', '\\', 1},
		// {'+', '`', 1},
		// {'&', '_', 1},
		// {'%', ' ', 1},
		// {'!', ' ', 1},
		// {',', '^', 1},
		// {'/', '~', 1},
		// {'\'', ']', 1},
	}

	for _, c := range cases {
		name := string(c.sr) + "==" + string(c.tr)
		t.Run(name, func(t *testing.T) {
			if testEqualFold(t, c.sr, c.tr, c.want) {
				testEqualFold(t, c.tr, c.sr, c.want)
			}
		})
	}
}

func testEqualFold(t *testing.T, sr, tr rune, want int) bool {
	t.Helper()
	if got := equalFold(sr, tr); got != want {
		t.Errorf("equalFold(%v %v) = %v, want %v", rStr(sr), rStr(tr), got, want)

		return false
	}

	return true
}
