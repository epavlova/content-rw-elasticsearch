package main

import (
	"html"
	"regexp"
	"strings"
)

var interactiveGraphicsRegex = regexp.MustCompile(`(?s)<div[\s]*class="interactive-comp">(.*?)</div>`)
var pullTagRegex = regexp.MustCompile(`(?s)<pull-quote(\s|>).*?</pull-quote>`)
var nbspRegex = regexp.MustCompile(`&nbsp;`)
var scriptRegex = regexp.MustCompile(`(?i)(?s)<script[^>]*>(.*?)</script>`)
var tagRegex = regexp.MustCompile(`<[^>]*>`)
var embedRegex = regexp.MustCompile(`embed\d+`)
var squaredCaptionRegex = regexp.MustCompile(`\[/?caption[^]]*]`)
var duplicateWhiteSpaceRegex = regexp.MustCompile(`\s+`)

type textTransformer func(string) string

func transformText(text string, transformers ...textTransformer) string {
	current := text
	for _, transformer := range transformers {
		current = transformer(current)
	}
	return current
}

func interactiveGraphicsMarkupTagRemover(input string) string {
	return interactiveGraphicsRegex.ReplaceAllString(input, "")

}
func pullTagTransformer(input string) string {
	return pullTagRegex.ReplaceAllString(input, "")
}
func htmlEntityTransformer(input string) string {
	text := nbspRegex.ReplaceAllString(input, " ")
	return html.UnescapeString(text)
}

func scriptTagRemover(input string) string {
	return scriptRegex.ReplaceAllString(input, "")
}

func tagsRemover(input string) string {
	return tagRegex.ReplaceAllString(input, "")
}
func outerSpaceTrimmer(input string) string {
	return strings.TrimSpace(input)
}

func embed1Replacer(input string) string {
	return embedRegex.ReplaceAllString(input, "")
}
func squaredCaptionReplacer(input string) string {
	return squaredCaptionRegex.ReplaceAllString(input, "")

}
func duplicateWhiteSpaceRemover(input string) string {
	return duplicateWhiteSpaceRegex.ReplaceAllString(input, " ")
}
