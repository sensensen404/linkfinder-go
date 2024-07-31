package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

var regexStr = `(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;| *()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{3,}(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:[\?|#][^"|']{0,}|)))(?:"|')`

func main() {
	fileFlag := flag.String("f", "", "Input file")
	dirFlag := flag.String("d", "", "Input directory")
	outputFlag := flag.String("o", "", "Output file")
	flag.Parse()

	if *fileFlag == "" && *dirFlag == "" {
		fmt.Println("Usage: linkfinder-go -f <file> or -d <directory>")
		os.Exit(1)
	}

	var files []string
	if *fileFlag != "" {
		files = append(files, *fileFlag)
	}
	if *dirFlag != "" {
		err := filepath.Walk(*dirFlag, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			fmt.Printf("Error reading directory: %v\n", err)
			os.Exit(1)
		}
	}

	var output *os.File
	var err error
	if *outputFlag != "" {
		output, err = os.Create(*outputFlag)
		if err != nil {
			fmt.Printf("Error creating output file %s: %v\n", *outputFlag, err)
			os.Exit(1)
		}
		defer output.Close()
	}

	uniqueMatches := make(map[string]bool)
	for _, file := range files {
		processFile(file, output, uniqueMatches)
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

func processFile(filePath string, output *os.File, uniqueMatches map[string]bool) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", filePath, err)
		return
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("Error reading file %s: %v\n", filePath, err)
		return
	}

	matches := findURLs(string(content))
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
