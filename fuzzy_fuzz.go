package fuzzy_fuzz

import (
	"github.com/sahilm/fuzzy"

	"github.com/golang/protobuf/proto"
)

func FuzzFuzzyFind(data []byte) int {
	args := &fuzzy.FindArgs{}
	err := proto.Unmarshal(data, args)
	if err != nil {
		return 0
	}
	matches := fuzzy.Find(*args.Pattern, args.Datas)
	for _, match := range matches {
		for i := 0; i < len(match.Str); i++ {
			for _, j := range match.MatchedIndexes {
				if j == i {
					//fmt.Printf("found %#+v\n", match)
					break
				}
			}
		}
	}
	return 1
}
