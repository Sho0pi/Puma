package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

const (
	emptyString = ""
	wordListLength = 100
)

var (
	maxDepth uint
	extensionsList string
	dictPath string
	websiteUrl string
)

func init() {
	flag.UintVar(&maxDepth, "-maxDepth", 3, "The directory maxDepth to search in.")

	flag.StringVar(&dictPath, "d", emptyString, "The path to the dictionary.")
	flag.StringVar(&extensionsList, "e", "html,php,js", "The extensions you wanna search for.")
	flag.StringVar(&websiteUrl, "url", emptyString, "The url to the website.")

}

func isFileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err){
		return false
	}
	return !info.IsDir()
}

func checkSite() {
	_, err := http.Get(websiteUrl)
	if err != nil {
		log.Fatal("Cant connect to the website.", err)
	}
}

func checkInput() {
	if maxDepth == 0 || dictPath == emptyString || extensionsList == emptyString || websiteUrl == emptyString{
		flag.PrintDefaults()
		os.Exit(1)
	}
	if !isFileExists(dictPath){
		flag.PrintDefaults()
		os.Exit(1)
	}


}

func generateExtensions() []string {
	fileNames := strings.Split(extensionsList, ",")
	var extensions []string
	for _, fileName := range fileNames {
		extensions = append(extensions, "." + fileName)
	}
	return extensions
}

func main() {
	flag.Parse()
	checkInput()

	checkSite()

	extensions := generateExtensions()

	words := make(chan string, wordListLength)
	found := make(chan string, wordListLength)

	var wg sync.WaitGroup

	// Only one worker is running at start.
	wg.Add(1)
	go worker(0, websiteUrl, extensions, words, found, &wg)
	////////////////////////////////////////

	dictFile, err := os.Open(dictPath)
	if err != nil {
		log.Fatal("Not able to open the file", err)
	}
	defer dictFile.Close()

	wordsScanner := bufio.NewScanner(dictFile)
	go func() {
		for wordsScanner.Scan() {
			words <- wordsScanner.Text()
		}
		close(words)
	}()

	go func() {
		for url := range found {
			fmt.Println(url)
		}
	}()

	wg.Wait()
	close(found)
	return

}

func buildUrl(baseUrl, word, extension string) string {
	var builder strings.Builder
	_, _ = fmt.Fprintf(&builder, "%s%s%s", baseUrl, word, extension)

	return builder.String()
}

func worker(depth uint, url string, extensions []string, words <-chan string, found chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()

	for word := range words {
		// Checks if any extensions are valid
		for _, extension := range extensions {
			newUrl := buildUrl(url, word, extension)
			resp, err := http.Get(newUrl)
			if err != nil {
				continue
			}
			if resp.StatusCode == 200 {
				found <- newUrl
			}
			resp.Body.Close()
		}
		// Checks if subdirectory exists
		if depth < maxDepth {
			newUrl := buildUrl(url, word, "/")
			resp, err := http.Get(newUrl)
			if err != nil {
				continue
			}
			if resp.StatusCode == 200 {
				found <- newUrl

				newWords := make(chan string, wordListLength)

				dictFile, err:= os.Open(dictPath)
				if err != nil {
					log.Fatal("Not Able to open the file", err)
				}
				defer dictFile.Close()
				wordlist := bufio.NewScanner(dictFile)
				go func() {
					for wordlist.Scan() {
						newWords<- wordlist.Text()
					}
					close(newWords)
				}()
				wg.Add(1)
				go worker(depth+1, newUrl, extensions, newWords, found, wg)
			}
			resp.Body.Close()
		}
	}

}
