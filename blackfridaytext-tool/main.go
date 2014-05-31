package main

import (
	"github.com/gholt/blackfridaytext"
	"io/ioutil"
	"os"
)

func main() {
    width := 0 // Use the default width (from terminal or 79)
	color := true
	if len(os.Args) == 2 && os.Args[1] == "--no-color" {
		color = false
	}
	markdown, _ := ioutil.ReadAll(os.Stdin)
	metadata, output := blackfridaytext.MarkdownToText(markdown, width, color)
	for _, item := range metadata {
		name, value := item[0], item[1]
		os.Stdout.WriteString(name)
		os.Stdout.WriteString(":\n    ")
		os.Stdout.WriteString(value)
		os.Stdout.WriteString("\n")
	}
	os.Stdout.WriteString("\n")
	os.Stdout.Write(output)
	os.Stdout.WriteString("\n")
}
