package babbler

import (
	"fmt"
	"strings"
)

func (p *BabblerPlugin) initializeBabbler(tokens []string) (string, bool) {
	who := tokens[3]
	_, err := p.getOrCreateBabbler(who)
	if err != nil {
		return "babbler initialization failed.", true
	}
	return "okay.", true
}

func (p *BabblerPlugin) addToBabbler(babblerName, whatWasSaid string) (string, bool) {
	babblerId, err := p.getOrCreateBabbler(babblerName)
	if err == nil {
		if p.WithGoRoutines {
			go p.addToMarkovChain(babblerId, whatWasSaid)
		} else {
			p.addToMarkovChain(babblerId, whatWasSaid)
		}
	}
	return "", false
}

func (p *BabblerPlugin) getBabble(tokens []string) (string, bool) {
	who := tokens[0]
	_, err := p.getBabbler(who)

	if err != nil {
		if err == NO_BABBLER {
			// return fmt.Sprintf("%s babbler not found.", who), true
			return "", false
		}
	} else {

		var saying string
		if len(tokens) == 2 {
			saying, err = p.babble(who)
		} else {
			saying, err = p.babbleSeed(who, tokens[2:])
		}

		if err != nil {
			if err == SAID_NOTHING {
				return fmt.Sprintf("%s hasn't said anything yet.", who), true
			} else if err == NEVER_SAID {
				return fmt.Sprintf("%s never said '%s'", who, strings.Join(tokens[2:], " ")), true
			}
		} else if saying != "" {
			return saying, true
		}
	}
	return "", false
}

func (p *BabblerPlugin) getBabbleWithSuffix(tokens []string) (string, bool) {
	who := tokens[0]
	_, err := p.getBabbler(who)

	if err != nil {
		if err == NO_BABBLER {
			// return fmt.Sprintf("%s babbler not found.", who), true
			return "", false
		}
	} else {

		saying, err := p.babbleSeedSuffix(who, tokens[2:])

		if err != nil {
			if err == SAID_NOTHING {
				return fmt.Sprintf("%s hasn't said anything yet.", who), true
			} else if err == NEVER_SAID {
				return fmt.Sprintf("%s never said '%s'", who, strings.Join(tokens[2:], " ")), true
			}
		} else if saying != "" {
			return saying, true
		}
	}
	return "", false
}

func (p *BabblerPlugin) getBabbleWithBookends(start, end []string) (string, bool) {
	who := start[0]
	_, err := p.getBabbler(who)

	if err != nil {
		if err == NO_BABBLER {
			// return fmt.Sprintf("%s babbler not found.", who), true
			return "", false
		}
	} else {

		saying, err := p.babbleSeedBookends(who, start[2:], end)

		if err != nil {
			if err == SAID_NOTHING {
				return fmt.Sprintf("%s hasn't said anything yet.", who), true
			} else if err == NEVER_SAID {
				seeds := append(start[2:], "...")
				seeds = append(seeds, end...)
				return fmt.Sprintf("%s never said '%s'", who, strings.Join(seeds, " ")), true
			}
		} else if saying != "" {
			return saying, true
		}
	}
	return "", false
}

func (p *BabblerPlugin) batchLearn(tokens []string) (string, bool) {
	who := tokens[3]
	babblerId, err := p.getOrCreateBabbler(who)
	if err != nil {
		return "batch learn failed.", true
	}

	body := strings.Join(tokens[4:], " ")
	body = strings.ToLower(body)

	for _, a := range strings.Split(body, ".") {
		for _, b := range strings.Split(a, "!") {
			for _, c := range strings.Split(b, "?") {
				for _, d := range strings.Split(c, "\n") {
					trimmed := strings.TrimSpace(d)
					if trimmed != "" {
						p.addToMarkovChain(babblerId, trimmed)
					}
				}
			}
		}
	}
	return "phew that was tiring.", true
}

func (p *BabblerPlugin) merge(tokens []string) (string, bool) {
	if tokens[3] != "into" {
		return "try using 'merge babbler [x] into [y]'", true
	}

	who := tokens[2]
	into := tokens[4]

	if who == into {
		return "that's annoying. stop it.", true
	}

	whoBabbler, err := p.getBabbler(who)
	if err != nil {
		if err == NO_BABBLER {
			return fmt.Sprintf("%s babbler not found.", who), true
		}
		return "merge failed.", true
	}
	intoBabbler, err := p.getOrCreateBabbler(into)
	if err != nil {
		return "merge failed.", true
	}

	err = p.mergeBabblers(intoBabbler, whoBabbler, into, who)
	if err != nil {
		return "merge failed.", true
	}

	return "mooooiggged", true
}
