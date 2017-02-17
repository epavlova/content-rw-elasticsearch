package main

import (
	"html"
	"regexp"
	"strings"
)

type TextTransformer func(string) string

func transformText(text string, transformers ...TextTransformer) string {
	current := text
	for _, transformer := range transformers {
		current = transformer(current)
	}
	return current
}

func interactiveGraphicsMarkupTagRemover(input string) string {
	return regexp.MustCompile("(?s)<div[\\s]*class=\"interactive-comp\">(.*?)</div>").ReplaceAllString(input, "")

}
func pullTagTransformer(input string) string {
	return regexp.MustCompile("(?s)<pull-quote(\\s|>).*?</pull-quote>").ReplaceAllString(input, "")
}
func htmlEntityTransformer(input string) string {
	text := regexp.MustCompile("&nbsp;").ReplaceAllString(input, " ")
	return html.UnescapeString(text)
}

func scriptTagRemover(input string) string {
	return regexp.MustCompile("(?i)(?s)<script[^>]*>(.*?)</script>").ReplaceAllString(input, "")
}

func tagsRemover(input string) string {
	return regexp.MustCompile("<[^>]*>").ReplaceAllString(input, "")
}
func outerSpaceTrimmer(input string) string {
	return strings.TrimSpace(input)
}

func embed1Replacer(input string) string {
	return regexp.MustCompile("embed\\d+").ReplaceAllString(input, "")
}
func squaredCaptionReplacer(input string) string {
	return regexp.MustCompile("\\[/?caption[^]]*]").ReplaceAllString(input, "")

}
func duplicateWhiteSpaceRemover(input string) string {
	return regexp.MustCompile("\\s+").ReplaceAllString(input, " ")
}
