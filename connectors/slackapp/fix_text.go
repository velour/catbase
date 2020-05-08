package slackapp

import (
	"unicode/utf8"

	"github.com/slack-go/slack"
)

// fixText strips all of the Slack-specific annotations from message text,
// replacing it with the equivalent display form.
// Currently it:
// • Replaces user mentions like <@U124356> with @ followed by the user's nick.
// This uses the lookupUser function, which must map U1243456 to the nick.
// • Replaces user mentions like <U123456|nick> with the user's nick.
// • Strips < and > surrounding links.
//
// This was directly bogarted from velour/chat with emoji conversion removed.
func fixText(findUser func(id, defaultName string) (string, *slack.User), text string) string {
	var output []rune
	for len(text) > 0 {
		r, i := utf8.DecodeRuneInString(text)
		text = text[i:]
		switch {
		case r == '<':
			var tag []rune
			for {
				r, i := utf8.DecodeRuneInString(text)
				text = text[i:]
				switch {
				case r == '>':
					if t, ok := fixTag(findUser, tag); ok {
						output = append(output, t...)
						break
					}
					fallthrough
				case len(text) == 0:
					output = append(output, '<')
					output = append(output, tag...)
					output = append(output, r)
				default:
					tag = append(tag, r)
					continue
				}
				break
			}
		default:
			output = append(output, r)
		}
	}
	return string(output)
}

func fixTag(findUser func(string, string) (string, *slack.User), tag []rune) ([]rune, bool) {
	switch {
	case hasPrefix(tag, "@U"):
		if i := indexRune(tag, '|'); i >= 0 {
			return tag[i+1:], true
		}
		if findUser != nil {
			u, _ := findUser(string(tag[1:]), "unknown")
			return []rune(u), true
		}
		return tag, true

	case hasPrefix(tag, "#C"):
		if i := indexRune(tag, '|'); i >= 0 {
			return append([]rune{'#'}, tag[i+1:]...), true
		}

	case hasPrefix(tag, "http"):
		if i := indexRune(tag, '|'); i >= 0 {
			tag = tag[:i]
		}
		return tag, true
	}

	return nil, false
}

func hasPrefix(text []rune, prefix string) bool {
	for _, r := range prefix {
		if len(text) == 0 || text[0] != r {
			return false
		}
		text = text[1:]
	}
	return true
}

func indexRune(text []rune, find rune) int {
	for i, r := range text {
		if r == find {
			return i
		}
	}
	return -1
}
