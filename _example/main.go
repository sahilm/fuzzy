package main

import (
	"fmt"
	"log"

	"io/ioutil"
	"strings"

	"time"

	"github.com/jroimartin/gocui"
	"github.com/sahilm/fuzzy"
)

var filenamesBytes []byte
var err error

var filenames []string

var g *gocui.Gui

func main() {
	filenamesBytes, err = ioutil.ReadFile("../testdata/ue4_filenames.txt")
	if err != nil {
		panic(err)
	}

	filenames = strings.Split(string(filenamesBytes), "\n")

	g, err = gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Cursor = true
	g.Mouse = false

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("finder", gocui.KeyArrowRight, gocui.ModNone, switchToMainView); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("main", gocui.KeyArrowLeft, gocui.ModNone, switchToSideView); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func switchToSideView(g *gocui.Gui, view *gocui.View) error {
	if _, err := g.SetCurrentView("finder"); err != nil {
		return err
	}
	return nil
}

func switchToMainView(g *gocui.Gui, view *gocui.View) error {
	if _, err := g.SetCurrentView("main"); err != nil {
		return err
	}
	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("finder", -1, 0, 80, 10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Wrap = true
		v.Editable = true
		v.Frame = true
		v.Title = "Type pattern here. Press -> or <- to switch between panes"
		if _, err := g.SetCurrentView("finder"); err != nil {
			return err
		}
		v.Editor = gocui.EditorFunc(finder)
	}
	if v, err := g.SetView("main", 79, 0, maxX, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		fmt.Fprintf(v, "%s", filenamesBytes)
		v.Editable = false
		v.Wrap = true
		v.Frame = true
		v.Title = "list of all files"
	}

	if v, err := g.SetView("results", -1, 3, 79, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = false
		v.Wrap = true
		v.Frame = true
		v.Title = "Search Results"
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func finder(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch {
	case ch != 0 && mod == 0:
		v.EditWrite(ch)
		g.Update(func(gui *gocui.Gui) error {
			results, err := g.View("results")
			if err != nil {
				// handle error
			}
			results.Clear()
			t := time.Now()
			matches := fuzzy.Find(strings.TrimSpace(v.ViewBuffer()), filenames)
			elapsed := time.Since(t)
			fmt.Fprintf(results, "found %v matches in %v\n", len(matches), elapsed)
			for _, match := range matches {
				for i := 0; i < len(match.Str); i++ {
					if contains(i, match.MatchedIndexes) {
						fmt.Fprintf(results, fmt.Sprintf("\033[1m%s\033[0m", string(match.Str[i])))
					} else {
						fmt.Fprintf(results, string(match.Str[i]))
					}

				}
				fmt.Fprintln(results, "")
			}
			return nil
		})
	case key == gocui.KeySpace:
		v.EditWrite(' ')
	case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
		v.EditDelete(true)
		g.Update(func(gui *gocui.Gui) error {
			results, err := g.View("results")
			if err != nil {
				// handle error
			}
			results.Clear()
			t := time.Now()
			matches := fuzzy.Find(strings.TrimSpace(v.ViewBuffer()), filenames)
			elapsed := time.Since(t)
			fmt.Fprintf(results, "found %v matches in %v\n", len(matches), elapsed)
			for _, match := range matches {
				for i := 0; i < len(match.Str); i++ {
					if contains(i, match.MatchedIndexes) {
						fmt.Fprintf(results, fmt.Sprintf("\033[1m%s\033[0m", string(match.Str[i])))
					} else {
						fmt.Fprintf(results, string(match.Str[i]))
					}
				}
				fmt.Fprintln(results, "")
			}
			return nil
		})
	case key == gocui.KeyDelete:
		v.EditDelete(false)
		g.Update(func(gui *gocui.Gui) error {
			results, err := g.View("results")
			if err != nil {
				// handle error
			}
			results.Clear()
			t := time.Now()
			matches := fuzzy.Find(strings.TrimSpace(v.ViewBuffer()), filenames)
			elapsed := time.Since(t)
			fmt.Fprintf(results, "found %v matches in %v\n", len(matches), elapsed)
			for _, match := range matches {
				for i := 0; i < len(match.Str); i++ {
					if contains(i, match.MatchedIndexes) {
						fmt.Fprintf(results, fmt.Sprintf("\033[1m%s\033[0m", string(match.Str[i])))
					} else {
						fmt.Fprintf(results, string(match.Str[i]))
					}
				}
				fmt.Fprintln(results, "")
			}
			return nil
		})
	case key == gocui.KeyInsert:
		v.Overwrite = !v.Overwrite
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
