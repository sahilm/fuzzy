package fuzzy_test

import (
	"os"
	"testing"

	"github.com/sahilm/fuzzy"

	"strings"

	"fmt"
	"time"

	"github.com/kylelemons/godebug/pretty"
)

func TestFindWithUnicode(t *testing.T) {
	matches := fuzzy.Find("\U0001F41D", []string{"\U0001F41D"})
	if len(matches) != 1 {
		t.Errorf("got %v Matches; expected 1 match", len(matches))
	}
}

func TestFindWithCannedData(t *testing.T) {
	cases := []struct {
		pattern string
		data    []string
		matches []fuzzy.Match
	}{
		// first char bonus, camel case bonuses and unmatched chars penalty
		// (m = 10, n = 20, r = 20) - 18 unmatched chars = 32
		{
			"mnr", []string{"moduleNameResolver.ts"}, []fuzzy.Match{
			{
				Str:            "moduleNameResolver.ts",
				Index:          0,
				MatchedIndexes: []int{0, 6, 10},
				Score:          32,
			},
		},
		},
		{
			"mmt", []string{"mémeTemps"}, []fuzzy.Match{
			{
				Str:            "mémeTemps",
				Index:          0,
				MatchedIndexes: []int{0, 3, 5},
				Score:          23,
			},
		},
		},
		// ranking
		{
			"mnr", []string{"moduleNameResolver.ts", "my name is_Ramsey"}, []fuzzy.Match{
			{
				Str:            "my name is_Ramsey",
				Index:          1,
				MatchedIndexes: []int{0, 3, 11},
				Score:          36,
			},
			{
				Str:            "moduleNameResolver.ts",
				Index:          0,
				MatchedIndexes: []int{0, 6, 10},
				Score:          32,
			},
		},
		},
		// simple repeated pattern and adjacent match bonus
		{
			"aaa", []string{"aaa", "bbb"}, []fuzzy.Match{
			{
				Str:            "aaa",
				Index:          0,
				MatchedIndexes: []int{0, 1, 2},
				Score:          30,
			},
		},
		},
		// exhaustive matching
		{
			"tk", []string{"The Black Knight"}, []fuzzy.Match{
			{
				Str:            "The Black Knight",
				Index:          0,
				MatchedIndexes: []int{0, 10},
				Score:          16,
			},
		},
		},
		// any unmatched char in the pattern removes the whole match
		{
			"cats", []string{"cat"}, []fuzzy.Match{},
		},
		// empty patterns return no Matches
		{
			"", []string{"cat"}, []fuzzy.Match{},
		},
		// separator bonus
		{
			"abcx", []string{"abc\\x"}, []fuzzy.Match{
			{
				Str:            "abc\\x",
				Index:          0,
				MatchedIndexes: []int{0, 1, 2, 4},
				Score:          49,
			},
		},
		},
	}
	for _, c := range cases {
		matches := fuzzy.Find(c.pattern, c.data)
		if len(matches) != len(c.matches) {
			t.Errorf("got %v Matches; expected %v match", len(matches), len(c.matches))
		}
		if diff := pretty.Compare(c.matches, matches); diff != "" {
			t.Errorf("%v", diff)
		}
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
	want := fuzzy.Matches{
		{
			Str:            "Allie",
			Index:          2,
			MatchedIndexes: []int{0, 1},
			Score:          12,
		}, {
			Str:            "Alice",
			Index:          0,
			MatchedIndexes: []int{0, 1},
			Score:          12,
		},
	}
	got := fuzzy.FindFrom("al", emps)
	if diff := pretty.Compare(want, got); diff != "" {
		t.Errorf("%v", diff)
	}
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

		bytes, err := os.ReadFile("testdata/ue4_filenames.txt")
		if err != nil {
			t.Fatal(err)
		}

		filenames := strings.Split(string(bytes), "\n")

		for _, c := range cases {
			now := time.Now()
			matches := fuzzy.Find(c.pattern, filenames)
			elapsed := time.Since(now)
			fmt.Printf("Matching '%v' in Unreal 4... found %v Matches in %v\n", c.pattern, len(matches), elapsed)
			foundfilenames := make([]string, 0)
			for i := 0; i < c.numMatches; i++ {
				foundfilenames = append(foundfilenames, matches[i].Str)
			}
			if diff := pretty.Compare(c.filenames, foundfilenames); diff != "" {
				t.Errorf("%v", diff)
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

		bytes, err := os.ReadFile("testdata/linux_filenames.txt")
		if err != nil {
			t.Fatal(err)
		}

		filenames := strings.Split(string(bytes), "\n")

		for _, c := range cases {
			now := time.Now()
			matches := fuzzy.Find(c.pattern, filenames)
			elapsed := time.Since(now)
			fmt.Printf("Matching '%v' in linux kernel... found %v Matches in %v\n", c.pattern, len(matches), elapsed)
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
		}
	})

}

func BenchmarkFind(b *testing.B) {
	b.Run("with unreal 4 (~16K files)", func(b *testing.B) {
		bytes, err := os.ReadFile("testdata/ue4_filenames.txt")
		if err != nil {
			b.Fatal(err)
		}
		filenames := strings.Split(string(bytes), "\n")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fuzzy.Find("lll", filenames)
		}
	})

	b.Run("with linux kernel (~60K files)", func(b *testing.B) {
		bytes, err := os.ReadFile("testdata/linux_filenames.txt")
		if err != nil {
			b.Fatal(err)
		}
		filenames := strings.Split(string(bytes), "\n")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fuzzy.Find("alsa", filenames)
		}
	})
}
