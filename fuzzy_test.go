package fuzzy

import (
	"testing"

	"io/ioutil"
	"strings"

	"fmt"
	"time"

	"github.com/kylelemons/godebug/pretty"
)

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
					initialized:    true,
					score:          32,
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
					initialized:    true,
					score:          36,
				},
				{
					Str:            "moduleNameResolver.ts",
					Index:          0,
					MatchedIndexes: []int{0, 6, 10},
					initialized:    true,
					score:          32,
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
					initialized:    true,
					score:          30,
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
					initialized:    true,
					score:          16,
				},
			},
		},
		// any unmatched char in the pattern removes the whole match
		{
			"cats", []string{"cat"}, []Match{},
		},
		// empty patterns return no matches
		{
			"", []string{"cat"}, []Match{},
		},
	}
	for _, c := range cases {
		matches := Find(c.pattern, c.data)
		if len(matches) != len(c.matches) {
			t.Errorf("got %v matches; expected %v match", len(matches), len(c.matches))
		}
		if diff := pretty.Compare(c.matches, matches); diff != "" {
			t.Errorf("%v", diff)
		}
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
					"SVisualLoggerLogsList.h",
					"LockFreeListImpl.h",
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
			fmt.Printf("Matching '%v' in Unreal 4... found %v matches in %v\n", c.pattern, len(matches), elapsed)
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

		bytes, err := ioutil.ReadFile("testdata/linux_filenames.txt")
		if err != nil {
			t.Fatal(err)
		}

		filenames := strings.Split(string(bytes), "\n")

		for _, c := range cases {
			now := time.Now()
			matches := Find(c.pattern, filenames)
			elapsed := time.Since(now)
			fmt.Printf("Matching '%v' in linux kernel... found %v matches in %v\n", c.pattern, len(matches), elapsed)
			foundfilenames := make([]string, 0)
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
		bytes, err := ioutil.ReadFile("testdata/ue4_filenames.txt")
		if err != nil {
			b.Fatal(err)
		}
		filenames := strings.Split(string(bytes), "\n")

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

		for i := 0; i < b.N; i++ {
			Find("alsa", filenames)
		}
	})
}
