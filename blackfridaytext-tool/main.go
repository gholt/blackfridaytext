package main

import (
	"github.com/gholt/blackfridaytext"
	"io/ioutil"
	"os"
)

func main() {
	color := true
	if len(os.Args) == 2 && os.Args[1] == "--no-color" {
		color = false
	}
	markdown, _ := ioutil.ReadAll(os.Stdin)
	metadata, output := blackfridaytext.MarkdownToText(markdown, color)
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
