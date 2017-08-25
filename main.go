package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/src-d/go-git.v4/plumbing/object"

	git "gopkg.in/src-d/go-git.v4"
)

func main() {
	blank := regexp.MustCompile(`^\s*$`)
	path := flag.String("source", ".", "")
	extOpt := flag.String("extensions", ".rb,.js,.css,.html,.haml,.erb", "")

	flag.Parse()
	exts := strings.Split(*extOpt, ",")
	commit := getHeadCommit()

	var authors = make(map[string]int)
	total := 0.0

	for line := range blame(commit, files(*path, exts)) {
		if blank.MatchString(line.Text) {
			continue
		}

		authors[line.Author]++
		total++
		if math.Mod(total, 10.0) == 0 {
			printRank(authors, total)
		}

	}
	printRank(authors, total)
}

func getHeadCommit() *object.Commit {
	r, err := git.PlainOpen(".")
	if err != nil {
		log.Fatalf("Could not open respository %s : %s", ".", err)
	}
	head, err := r.Head()
	if err != nil {
		log.Fatalf("Could not get HEAD: %s", err)
	}

	commit, err := r.CommitObject(head.Hash())
	if err != nil {
		log.Fatalf("Could not get HEAD commit: %s", err)
	}
	return commit
}

func printRank(authors map[string]int, total float64) {
	fmt.Println("author\tlines\ttotal\t%")

	for author, lines := range authors {
		fmt.Printf("%s:\t%d\t%.1f\t%.1f%%\n", author, lines, total, (float64(lines)/total)*100)
	}
	fmt.Printf("\n")
}

func files(path string, acceptedExts []string) <-chan string {
	out := make(chan string)
	go func() {
		filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
			if file == ".git" {
				return filepath.SkipDir
			}

			if info.IsDir() {
				return nil
			}

			if !hasExtension(file, acceptedExts) {
				return nil
			}

			if err == nil {
				out <- file
			}
			return nil
		})
		close(out)
	}()
	return out
}

func hasExtension(file string, accepted []string) bool {
	fext := filepath.Ext(file)
	for _, e := range accepted {
		if fext == e {
			return true
		}
	}
	return false
}

func blame(commit *object.Commit, in <-chan string) <-chan *git.Line {
	out := make(chan *git.Line)
	var wg sync.WaitGroup

	wg.Add(4)
	go func() {
		wg.Wait()
		close(out)
	}()

	for i := 0; i < 4; i++ {
		go func(worker int) {
			defer wg.Done()
			for file := range in {
				blame, err := git.Blame(commit, file)

				if err != nil {
					fmt.Printf("file %s is not in the repository: %s\n", file, err)
				} else {
					for _, line := range blame.Lines {
						out <- line
					}
				}
			}
		}(i)
	}

	return out
}
