<img src="assets/search-gopher-1.png" alt="gopher looking for stuff">  <img src="assets/search-gopher-2.png" alt="gopher found stuff">

# fuzzy

[![Build Status](https://travis-ci.org/teal-finance/fuzzy.svg?branch=master)](https://travis-ci.org/teal-finance/fuzzy)
[![Documentation](https://godoc.org/github.com/teal-finance/fuzzy?status.svg)](https://godoc.org/github.com/teal-finance/fuzzy)

Go library that provides fuzzy string matching optimized for filenames and code symbols in the style of Sublime Text,
VSCode, IntelliJ IDEA et al. This library is external dependency-free. It only depends on the Go standard library.

## Features

- Intuitive matching. Quality is determined by:
  - The first character in the pattern matches the first character in the match string.
  - The matched character is camel cased.
  - The matched character follows a separator such as an underscore character.
  - The matched character is adjacent to a previous match.
  - Favor case sensitive matching: if two similar targets mathes, the one respecting the input case has a higher score.
  - Insensitive to the punctuations `/-_ .\` thus "BTC/USD" matches "BTC-USD".

- `Find()` and `FindFrom()` return result in descending order of match quality.

- `BestMatch()` and `BestMatchFrom()` return the matching having the highest score.

- Speed. Matches are returned in milliseconds. It's perfect for interactive search boxes.

- The positions of matches are returned. Allows you to highlight matching characters.

- Unicode aware.

## Demo

Here is a [demo](example/main.go) of matching various patterns against ~16K files from the Unreal Engine 4 codebase.

![demo](assets/demo.gif)

Run the demo:

```
cd example
go get github.com/jroimartin/gocui
go run main.go
```

## Usage

The following example prints out matches with the matched chars in bold.

```go
package main

import (
    "fmt"

    "github.com/teal-finance/fuzzy"
)

func main() {
    const bold = "\033[1m%s\033[0m"
    pattern := "mnr"
    data := []string{"game.cpp", "moduleNameResolver.ts", "my name is_Ramsey"}

    matches := fuzzy.Find(pattern, data)

    for _, match := range matches {
        for i := 0; i < len(match.Str); i++ {
            if contains(i, match.MatchedIndexes) {
                fmt.Print(fmt.Sprintf(bold, string(match.Str[i])))
            } else {
                fmt.Print(string(match.Str[i]))
            }
        }
        fmt.Println()
    }
}

func contains(needle int, haystack []int) bool {
    for _, i := range haystack {
        if needle == i {
            return true
        }
    }
    return false
}
```

If the data you want to match isn't a slice of strings, you can use `FindFrom` by implementing
the provided `Source` interface. Here's an example:

```go
package main

import (
    "fmt"

    "github.com/teal-finance/fuzzy"
)

type employee struct {
    name string
    age  int
}

type employees []employee

func (e employees) String(i int) string {
    return e[i].name
}

func (e employees) Len() int {
    return len(e)
}

func main() {
    emps := employees{
        {
            name: "Alice",
            age:  45,
        },
        {
            name: "Bob",
            age:  35,
        },
        {
            name: "Allie",
            age:  35,
        },
    }
    results := fuzzy.FindFrom("al", emps)
    for _, r := range results {
        fmt.Println(emps[r.Index])
    }
}
```

Check out the [godoc](https://pkg.go.dev/github.com/teal-finance/fuzzy) for detailed documentation.

## Installation

`go get github.com/teal-finance/fuzzy`

## Speed

The benchmark includes:
1. the [forked project](https://github.com/isacikgoz/fuzzy) by @isacikgoz using Go channel (⚠️ the channel overhead slows down this bench),
2. the [original project](https://github.com/sahilm/fuzzy) from @sahilm,
3. the current repo, which is twice as fast as the original,
4. the memory-optimized `BestMatch()`, 25% faster than `Find()`.

```
$ go test -count 6 -benchmem -run=^$ -bench . github.com/teal-finance/fuzzy

goos: linux
goarch: amd64
pkg: github.com/teal-finance/fuzzy
cpu: AMD Ryzen 9 3900X 12-Core Processor

BenchmarkUnrealFiles/isacikgoz.Find-24    86   13483431 ns/op   151874 B/op   898 allocs/op
BenchmarkUnrealFiles/isacikgoz.Find-24    87   13620413 ns/op   151875 B/op   898 allocs/op
BenchmarkUnrealFiles/isacikgoz.Find-24    90   13537883 ns/op   151873 B/op   898 allocs/op
BenchmarkUnrealFiles/isacikgoz.Find-24    90   13608595 ns/op   151864 B/op   898 allocs/op
BenchmarkUnrealFiles/isacikgoz.Find-24    93   13468849 ns/op   151872 B/op   898 allocs/op
BenchmarkUnrealFiles/isacikgoz.Find-24   100   13583070 ns/op   151875 B/op   898 allocs/op

BenchmarkUnrealFiles/sahilm.Find-24      139    8147677 ns/op   151752 B/op   896 allocs/op
BenchmarkUnrealFiles/sahilm.Find-24      140    7849173 ns/op   151752 B/op   896 allocs/op
BenchmarkUnrealFiles/sahilm.Find-24      148    7483526 ns/op   151752 B/op   896 allocs/op
BenchmarkUnrealFiles/sahilm.Find-24      150    7708139 ns/op   151752 B/op   896 allocs/op
BenchmarkUnrealFiles/sahilm.Find-24      156    8012760 ns/op   151752 B/op   896 allocs/op
BenchmarkUnrealFiles/sahilm.Find-24      157    7671143 ns/op   151752 B/op   896 allocs/op

BenchmarkUnrealFiles/teal.Find-24        256    4403617 ns/op   151752 B/op   896 allocs/op
BenchmarkUnrealFiles/teal.Find-24        258    4568313 ns/op   151752 B/op   896 allocs/op
BenchmarkUnrealFiles/teal.Find-24        282    4592112 ns/op   151752 B/op   896 allocs/op
BenchmarkUnrealFiles/teal.Find-24        286    4675680 ns/op   151752 B/op   896 allocs/op
BenchmarkUnrealFiles/teal.Find-24        314    4102624 ns/op   151752 B/op   896 allocs/op
BenchmarkUnrealFiles/teal.Find-24        324    4030270 ns/op   151752 B/op   896 allocs/op

BenchmarkUnrealFiles/teal.BestMatch-24   374    2683358 ns/op      200 B/op     5 allocs/op
BenchmarkUnrealFiles/teal.BestMatch-24   376    2748506 ns/op      200 B/op     5 allocs/op
BenchmarkUnrealFiles/teal.BestMatch-24   381    2807797 ns/op      200 B/op     5 allocs/op
BenchmarkUnrealFiles/teal.BestMatch-24   381    2910482 ns/op      200 B/op     5 allocs/op
BenchmarkUnrealFiles/teal.BestMatch-24   382    2844940 ns/op      200 B/op     5 allocs/op
BenchmarkUnrealFiles/teal.BestMatch-24   390    2819916 ns/op      200 B/op     5 allocs/op

BenchmarkLinuxFiles/isacikgoz.Find-24     36   29633575 ns/op    73632 B/op   368 allocs/op
BenchmarkLinuxFiles/isacikgoz.Find-24     39   29405028 ns/op    73634 B/op   368 allocs/op
BenchmarkLinuxFiles/isacikgoz.Find-24     39   29617157 ns/op    73632 B/op   368 allocs/op
BenchmarkLinuxFiles/isacikgoz.Find-24     39   29922710 ns/op    73634 B/op   368 allocs/op
BenchmarkLinuxFiles/isacikgoz.Find-24     40   29664292 ns/op    73648 B/op   368 allocs/op
BenchmarkLinuxFiles/isacikgoz.Find-24     51   29403815 ns/op    73637 B/op   368 allocs/op

BenchmarkLinuxFiles/sahilm.Find-24        70   15967443 ns/op    73520 B/op   366 allocs/op
BenchmarkLinuxFiles/sahilm.Find-24        70   16319718 ns/op    73520 B/op   366 allocs/op
BenchmarkLinuxFiles/sahilm.Find-24        72   16534890 ns/op    73520 B/op   366 allocs/op
BenchmarkLinuxFiles/sahilm.Find-24        74   16189509 ns/op    73520 B/op   366 allocs/op
BenchmarkLinuxFiles/sahilm.Find-24        79   14999524 ns/op    73520 B/op   366 allocs/op
BenchmarkLinuxFiles/sahilm.Find-24        84   16200886 ns/op    73520 B/op   366 allocs/op

BenchmarkLinuxFiles/teal.Find-24         148    7276280 ns/op    73520 B/op   366 allocs/op
BenchmarkLinuxFiles/teal.Find-24         148    7726970 ns/op    73520 B/op   366 allocs/op
BenchmarkLinuxFiles/teal.Find-24         157    7592565 ns/op    73520 B/op   366 allocs/op
BenchmarkLinuxFiles/teal.Find-24         159    8123690 ns/op    73520 B/op   366 allocs/op
BenchmarkLinuxFiles/teal.Find-24         164    7334939 ns/op    73520 B/op   366 allocs/op
BenchmarkLinuxFiles/teal.Find-24         176    7197796 ns/op    73520 B/op   366 allocs/op

BenchmarkLinuxFiles/teal.BestMatch-24    177    6481958 ns/op      200 B/op     5 allocs/op
BenchmarkLinuxFiles/teal.BestMatch-24    178    6249032 ns/op      200 B/op     5 allocs/op
BenchmarkLinuxFiles/teal.BestMatch-24    178    6256040 ns/op      200 B/op     5 allocs/op
BenchmarkLinuxFiles/teal.BestMatch-24    183    6444893 ns/op      200 B/op     5 allocs/op
BenchmarkLinuxFiles/teal.BestMatch-24    190    6282288 ns/op      200 B/op     5 allocs/op
BenchmarkLinuxFiles/teal.BestMatch-24    198    6010804 ns/op      200 B/op     5 allocs/op
```

Matching a pattern against ~60K files from the Linux kernel takes about 30ms.

The function `BestMatch()` is an memory-optimized version of `Find()` returning only the best match.

## Contributing

Everyone is welcome to contribute. Please send pull request or open an issue.

## Credits

- [@ericpauley](https://github.com/ericpauley) & [@lunixbochs](https://github.com/lunixbochs) contributed Unicode awareness and various performance optimisations.

- The algorithm is based of the awesome work of [forrestthewoods](https://github.com/forrestthewoods/lib_fts/blob/master/code/fts_fuzzy_match.js).
See [this](https://blog.forrestthewoods.com/reverse-engineering-sublime-text-s-fuzzy-match-4cffeed33fdb#.d05n81yjy)
blog post for details of the algorithm.

- The artwork is by my lovely wife Sanah. It's based on the Go Gopher.

- The Go gopher was designed by Renee French (<http://reneefrench.blogspot.com/>).
The design is licensed under the Creative Commons 3.0 Attributions license.

## License

The MIT License (MIT)

Copyright (c) 2017-2021 Sahil Muthoo and some other contributors  
Copyright (c) 2021      Teal.Finance contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
