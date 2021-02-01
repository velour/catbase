package babbler

import (
	"fmt"
	"strings"
)

func (p *BabblerPlugin) initializeBabbler(who string) string {
	_, err := p.getOrCreateBabbler(who)
	if err != nil {
		return "babbler initialization failed."
	}
	return "okay."
}

func (p *BabblerPlugin) addToBabbler(babblerName, whatWasSaid string) {
	babblerID, err := p.getOrCreateBabbler(babblerName)
	if err == nil {
		if p.WithGoRoutines {
			go p.addToMarkovChain(babblerID, whatWasSaid)
		} else {
			p.addToMarkovChain(babblerID, whatWasSaid)
		}
	}
}

func (p *BabblerPlugin) getBabble(who string, tokens []string) string {
	_, err := p.getBabbler(who)

	if err != nil {
		if err == NO_BABBLER {
			// return fmt.Sprintf("%s babbler not found.", who), true
			return ""
		}
	} else {

		var saying string
		if len(tokens) == 0 {
			saying, err = p.babble(who)
		} else {
			saying, err = p.babbleSeed(who, tokens)
		}

		if err != nil {
			if err == SAID_NOTHING {
				return fmt.Sprintf("%s hasn't said anything yet.", who)
			} else if err == NEVER_SAID {
				return fmt.Sprintf("%s never said '%s'", who, strings.Join(tokens, " "))
			}
		} else if saying != "" {
			return saying
		}
	}
	return ""
}

func (p *BabblerPlugin) getBabbleWithSuffix(who string, tokens []string) string {
	_, err := p.getBabbler(who)

	if err != nil {
		if err == NO_BABBLER {
			// return fmt.Sprintf("%s babbler not found.", who), true
			return ""
		}
	} else {

		saying, err := p.babbleSeedSuffix(who, tokens)

		if err != nil {
			if err == SAID_NOTHING {
				return fmt.Sprintf("%s hasn't said anything yet.", who)
			} else if err == NEVER_SAID {
				return fmt.Sprintf("%s never said '%s'", who, strings.Join(tokens, " "))
			}
		} else if saying != "" {
			return saying
		}
	}
	return ""
}

func (p *BabblerPlugin) getBabbleWithBookends(who string, start, end []string) string {
	_, err := p.getBabbler(who)

	if err != nil {
		if err == NO_BABBLER {
			// return fmt.Sprintf("%s babbler not found.", who), true
			return ""
		}
	} else {

		saying, err := p.babbleSeedBookends(who, start, end)

		if err != nil {
			if err == SAID_NOTHING {
				return fmt.Sprintf("%s hasn't said anything yet.", who)
			} else if err == NEVER_SAID {
				seeds := append(start[1:], "...")
				seeds = append(seeds, end...)
				return fmt.Sprintf("%s never said '%s'", who, strings.Join(seeds, " "))
			}
		} else if saying != "" {
			return saying
		}
	}
	return ""
}

func (p *BabblerPlugin) batchLearn(tokens []string) string {
	who := tokens[3]
	babblerId, err := p.getOrCreateBabbler(who)
	if err != nil {
		return "batch learn failed."
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
	return "phew that was tiring."
}

func (p *BabblerPlugin) merge(who, into string) string {
	if who == into {
		return "that's annoying. stop it."
	}

	whoBabbler, err := p.getBabbler(who)
	if err != nil {
		if err == NO_BABBLER {
			return fmt.Sprintf("%s babbler not found.", who)
		}
		return "merge failed."
	}
	intoBabbler, err := p.getOrCreateBabbler(into)
	if err != nil {
		return "merge failed."
	}

	err = p.mergeBabblers(intoBabbler, whoBabbler, into, who)
	if err != nil {
		return "merge failed."
	}

	return "mooooiggged"
}
