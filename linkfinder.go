package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

var (
	regexStr   = `(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;| *()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{3,}(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:[\?|#][^"|']{0,}|)))(?:"|')`
	fileFlag   string
	dirFlag    string
	urlFlag    string
	listFlag   string
	outputFlag string
)

func init() {
	flag.StringVar(&fileFlag, "f", "", "Input file")
	flag.StringVar(&dirFlag, "d", "", "Input directory")
	flag.StringVar(&urlFlag, "u", "", "Input URL")
	flag.StringVar(&listFlag, "l", "", "Input URL list file")
	flag.StringVar(&outputFlag, "o", "", "Output file")
}

func main() {
	flag.Parse()

	validateFlags()

	uniqueMatches := make(map[string]bool)

	switch {
	case fileFlag != "":
		processFile(fileFlag, uniqueMatches)
	case dirFlag != "":
		processDirectory(dirFlag, uniqueMatches)
	case urlFlag != "":
		processURL(urlFlag, uniqueMatches)
	case listFlag != "":
		processURLList(listFlag, uniqueMatches)
	}

	writeOutput(uniqueMatches)
}

func validateFlags() {
	inputFlags := []string{fileFlag, dirFlag, urlFlag, listFlag}
	activeFlags := 0
	for _, flag := range inputFlags {
		if flag != "" {
			activeFlags++
		}
	}

	if activeFlags != 1 {
		fmt.Println("Usage: linkfinder-go -f <file> or -d <directory> or -u <url> or -l <url list file>")
		os.Exit(1)
	}
}

func processFile(filePath string, uniqueMatches map[string]bool) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file %s: %v\n", filePath, err)
		return
	}
	processContent(string(content), uniqueMatches)
}

func processDirectory(dirPath string, uniqueMatches map[string]bool) {
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			processFile(path, uniqueMatches)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		os.Exit(1)
	}
}

func processURL(url string, uniqueMatches map[string]bool) {
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	router := setupRouter(browser)
	go router.Run()

	page, err := browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		fmt.Println("browser.Page err:", err)
		page.Close()
	}

	html, err := page.HTML()
	if err != nil {
		fmt.Printf("Error getting HTML from URL %s: %v\n", url, err)
		return
	}

	processContent(html, uniqueMatches)

	router.Stop()
}

func processURLList(listPath string, uniqueMatches map[string]bool) {
	file, err := os.Open(listPath)
	if err != nil {
		fmt.Printf("Error opening URL list file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		processURL(scanner.Text(), uniqueMatches)
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading URL list file: %v\n", err)
		os.Exit(1)
	}
}

func processContent(content string, uniqueMatches map[string]bool) {
	matches := findURLs(content)
	for _, match := range matches {
		uniqueMatches[match] = true
	}
}

func findURLs(content string) []string {
	re := regexp.MustCompile(regexStr)
	matchGroups := re.FindAllStringSubmatch(content, -1)

	uniqueMatches := make(map[string]bool)
	for _, group := range matchGroups {
		for _, match := range group[1:] {
			if match != "" {
				uniqueMatches[match] = true
			}
		}
	}

	var result []string
	for match := range uniqueMatches {
		result = append(result, match)
	}
	return result
}

func writeOutput(uniqueMatches map[string]bool) {
	var output *os.File
	var err error
	if outputFlag != "" {
		output, err = os.Create(outputFlag)
		if err != nil {
			fmt.Printf("Error creating output file %s: %v\n", outputFlag, err)
			os.Exit(1)
		}
		defer output.Close()
	}

	if output != nil {
		for match := range uniqueMatches {
			_, err := output.WriteString(match + "\n")
			if err != nil {
				fmt.Printf("Error writing to output file: %v\n", err)
				return
			}
		}
	} else {
		for match := range uniqueMatches {
			fmt.Println(match)
		}
	}
}

func setupRouter(browser *rod.Browser) *rod.HijackRouter {
	router := browser.HijackRequests()
	router.MustAdd("*", func(ctx *rod.Hijack) {
		hijackTraffic(ctx)
	})
	return router
}

func hijackTraffic(ctx *rod.Hijack) {
	requestType := ctx.Request.Type()
	if requestType == proto.NetworkResourceTypeFont || requestType == proto.NetworkResourceTypeMedia || requestType == proto.NetworkResourceTypeImage {
		ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
		return
	}

	ctx.ContinueRequest(&proto.FetchContinueRequest{})

	if err := ctx.LoadResponse(http.DefaultClient, true); err != nil {
		fmt.Printf("Error loading response: %v\n", err)
		return
	}

	bodyBytes := ctx.Response.Payload().Body

	content := string(bodyBytes)

	matches := findURLs(content)

	for _, match := range matches {
		uniqueMatches[match] = true
	}
}

var uniqueMatches = make(map[string]bool)
