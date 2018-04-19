package utils

import (
	"html"
	"regexp"
	"strings"
)

const singleSpace = " "

var (
	interactiveGraphicsRegex = regexp.MustCompile(`(?s)<div[\s]*class="interactive-comp">(.*?)</div>`)
	pullTagRegex             = regexp.MustCompile(`(?s)<pull-quote(\s|>).*?</pull-quote>`)
	nbspRegex                = regexp.MustCompile(`&nbsp;`)
	scriptRegex              = regexp.MustCompile(`(?i)(?s)<script[^>]*>(.*?)</script>`)
	tagRegex                 = regexp.MustCompile(`<[^>]*>`)
	embedRegex               = regexp.MustCompile(`embed\d+`)
	squaredCaptionRegex      = regexp.MustCompile(`\[/?caption[^]]*]`)
	duplicateWhiteSpaceRegex = regexp.MustCompile(`\s+`)
)

type textTransformer func(string) string

func TransformText(text string, transformers ...textTransformer) string {
	current := text
	for _, transformer := range transformers {
		current = transformer(current)
	}
	return current
}

func InteractiveGraphicsMarkupTagRemover(input string) string {
	return interactiveGraphicsRegex.ReplaceAllString(input, "")

}

func PullTagTransformer(input string) string {
	return pullTagRegex.ReplaceAllString(input, "")
}

func HtmlEntityTransformer(input string) string {
	text := nbspRegex.ReplaceAllString(input, " ")
	return html.UnescapeString(text)
}

func ScriptTagRemover(input string) string {
	return scriptRegex.ReplaceAllString(input, "")
}

func TagsRemover(input string) string {
	return tagRegex.ReplaceAllString(input, "")
}

func OuterSpaceTrimmer(input string) string {
	return strings.TrimSpace(input)
}

func Embed1Replacer(input string) string {
	return embedRegex.ReplaceAllString(input, "")
}

func SquaredCaptionReplacer(input string) string {
	return squaredCaptionRegex.ReplaceAllString(input, "")

}

func DuplicateWhiteSpaceRemover(input string) string {
	return duplicateWhiteSpaceRegex.ReplaceAllString(input, singleSpace)
}
