package check

import (
	"strings"

	"github.com/errata-ai/regexp2"
	"github.com/jdkato/twine/strcase"

	"github.com/errata-ai/vale/v3/internal/core"
)

var reNumberList = regexp2.MustCompileStd(`\d+\.`)

var varToFunc = map[string]func(string, *regexp2.Regexp) (string, bool){
	"$lower": lower,
	"$upper": upper,
}

var readabilityMetrics = []string{
	"Gunning Fog",
	"Coleman-Liau",
	"Flesch-Kincaid",
	"SMOG",
	"Automated Readability",
}

func wasIndicator(indicators []string) strcase.IndicatorFunc {
	return func(word string, idx int) bool {
		if core.HasAnySuffix(word, indicators) {
			return true
		} else if idx == 0 && reNumberList.MatchStringStd(word) {
			return true
		}
		return false
	}
}

func isMatch(r *regexp2.Regexp, s string) bool {
	// TODO: `r.String() != ""`?
	//
	// Should we ensure that empty regexes == `nil`?
	if r == nil || r.String() == "" {
		return false
	}
	match, err := r.MatchString(s)
	if err != nil {
		return false
	}
	return match
}

func lower(s string, re *regexp2.Regexp) (string, bool) {
	expected := strings.ToLower(s)
	return expected, s == expected || isMatch(re, s)
}

func upper(s string, re *regexp2.Regexp) (string, bool) {
	expected := strings.ToUpper(s)
	return expected, s == expected || isMatch(re, s)
}

func title(s string, except *regexp2.Regexp, tc *strcase.TitleConverter, threshold float64) (string, bool) {
	count := 0.0
	words := 0.0

	ps := `[\p{N}\p{L}*]+[^\s]*`
	if except != nil && except.String() != "" {
		ps = except.String() + "|" + ps
	}
	re := regexp2.MustCompileStd(ps)

	expected := tc.Convert(s)
	expectedTokens := re.FindAllString(expected, -1)

	extent := len(expectedTokens)
	for i, word := range re.FindAllString(s, -1) {
		if i >= extent { //nolint:gocritic
			// TODO: Look into this more.
			//
			// The problem is that `prose/transform` uses a different split
			// regex than `makeExceptions`, and we can't change the latter due
			// to https://github.com/errata-ai/vale/pull/253.
			//
			// In theory, this works because the only we'd find ourselves in
			// this situation is if the would-be entry at `expected[i]` is
			// listed as an exception, but it doesn't feel like a great
			// solution.
			count++
		} else if word == expectedTokens[i] || isMatch(except, word) {
			count++
		} else if word == strings.ToUpper(word) {
			count++
		}
		words++
	}

	return expected, (count / words) >= threshold
}

func sentence(s string, except *regexp2.Regexp, sc *strcase.SentenceConverter, threshold float64) (string, bool) {
	count := 0.0
	words := 0.0

	ps := `[\p{N}\p{L}*]+[^\s]*`
	if except != nil && except.String() != "" {
		ps = except.String() + "|" + ps
	}
	re := regexp2.MustCompileStd(ps)

	expected := sc.Convert(s)
	expectedTokens := re.FindAllString(expected, -1)

	extent := len(expectedTokens)
	for i, w := range re.FindAllString(s, -1) {
		if i >= extent { //nolint:gocritic
			// NOTE: See the comment in `title` above.
			count++
		} else if w == expectedTokens[i] || isMatch(except, w) {
			count++
		} else if i == 0 && w != strings.Title(strings.ToLower(w)) { //nolint:staticcheck
			return expected, false
		}
		words++
	}

	return expected, (count / words) >= threshold
}
