// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package babbler

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

var (
	NO_BABBLER   = errors.New("babbler not found")
	SAID_NOTHING = errors.New("hasn't said anything yet")
	NEVER_SAID   = errors.New("never said that")
)

type BabblerPlugin struct {
	Bot            bot.Bot
	db             *sqlx.DB
	WithGoRoutines bool

	handlers bot.HandlerTable
}

type Babbler struct {
	BabblerId int64  `db:"id"`
	Name      string `db:"babbler"`
}

type BabblerWord struct {
	WordId int64  `db:"id"`
	Word   string `db:"word"`
}

type BabblerNode struct {
	NodeId        int64 `db:"id"`
	BabblerId     int64 `db:"babblerId"`
	WordId        int64 `db:"wordId"`
	Root          int64 `db:"root"`
	RootFrequency int64 `db:"rootFrequency"`
}

type BabblerArc struct {
	ArcId      int64 `db:"id"`
	FromNodeId int64 `db:"fromNodeId"`
	ToNodeId   int64 `db:"toNodeId"`
	Frequency  int64 `db:"frequency"`
}

func New(b bot.Bot) *BabblerPlugin {
	if _, err := b.DB().Exec(`create table if not exists babblers (
			id integer primary key,
			babbler string
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if _, err := b.DB().Exec(`create table if not exists babblerWords (
			id integer primary key,
			word string
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if _, err := b.DB().Exec(`create table if not exists babblerNodes (
			id integer primary key,
			babblerId integer,
			wordId integer,
			root integer,
			rootFrequency integer
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if _, err := b.DB().Exec(`create table if not exists babblerArcs (
			id integer primary key,
			fromNodeId integer,
			toNodeId interger,
			frequency integer
		);`); err != nil {
		log.Fatal().Err(err)
	}

	plugin := &BabblerPlugin{
		Bot:            b,
		db:             b.DB(),
		WithGoRoutines: true,
	}

	plugin.createNewWord("")

	plugin.register()

	return plugin
}

func (p *BabblerPlugin) register() {
	p.handlers = bot.HandlerTable{
		bot.HandlerSpec{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`(?i)^(?P<who>\S+) says-bridge (?P<start>.+)\|(?P<end>.+)$`),
			Handler: func(r bot.Request) bool {
				who := r.Values["who"]
				start := strings.Fields(strings.ToLower(r.Values["start"]))
				end := strings.Fields(strings.ToLower(r.Values["end"]))
				return p.sayIt(r, p.getBabbleWithBookends(who, start, end))
			}},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`(?i)^(?P<who>\S+) says-tail (?P<what>.*)$`),
			Handler: func(r bot.Request) bool {
				who := r.Values["who"]
				what := strings.Fields(strings.ToLower(r.Values["what"]))
				return p.sayIt(r, p.getBabbleWithSuffix(who, what))
			}},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`(?i)^(?P<who>\S+) says-middle-out (?P<what>.*)$`),
			Handler: func(r bot.Request) bool {
				who := r.Values["who"]
				what := strings.ToLower(r.Values["what"])
				tokens := strings.Fields(what)
				saidSomething := false
				saidWhat := ""

				saidWhatStart := p.getBabbleWithSuffix(who, tokens)
				saidSomethingStart := saidWhatStart != ""
				neverSaidLooksLike := fmt.Sprintf("%s never said", who)
				if !saidSomethingStart || strings.HasPrefix(saidWhatStart, neverSaidLooksLike) {
					saidSomething = saidSomethingStart
					saidWhat = saidWhatStart
				} else {
					saidWhatEnd := p.getBabble(who, tokens)
					saidSomethingEnd := saidWhatEnd != ""
					saidSomething = saidSomethingStart && saidSomethingEnd
					if saidSomething {
						saidWhat = saidWhatStart + strings.TrimPrefix(saidWhatEnd, what)
					}
				}
				return p.sayIt(r, saidWhat)
			}},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: false, Regex: regexp.MustCompile(`(?i)^(?P<who>\S+) (says (?P<what>.*)?|says)$`),
			Handler: func(r bot.Request) bool {
				who := r.Values["who"]
				what := strings.Fields(strings.ToLower(r.Values["what"]))
				return p.sayIt(r, p.getBabble(who, what))
			}},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`(?i)^initialize babbler for (?P<who>\S+)$`),
			Handler: func(r bot.Request) bool {
				who := r.Values["who"]
				return p.sayIt(r, p.initializeBabbler(who))
			}},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`(?i)^merge babbler (?P<from>\S+) into (?P<to>\S+)$`),
			Handler: func(r bot.Request) bool {
				from, to := r.Values["from"], r.Values["to"]
				return p.sayIt(r, p.merge(from, to))
			}},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`.*`),
			Handler: func(r bot.Request) bool {
				p.addToBabbler(r.Msg.User.Name, strings.ToLower(r.Msg.Body))
				return false
			}},
	}
	p.Bot.RegisterTable(p, p.handlers)
	p.Bot.Register(p, bot.Help, p.help)
}

func (p *BabblerPlugin) sayIt(r bot.Request, what string) bool {
	if what != "" {
		p.Bot.Send(r.Conn, bot.Message, r.Msg.Channel, what)
	}
	return what != ""
}

func (p *BabblerPlugin) help(c bot.Connector, kind bot.Kind, msg msg.Message, args ...any) bool {
	commands := []string{
		"initialize babbler for seabass",
		"merge babbler drseabass into seabass",
		"seabass says ...",
		"seabass says-tail ...",
		"seabass says-middle-out ...",
		"seabass says-bridge ... | ...",
	}
	p.Bot.Send(c, bot.Message, msg.Channel, strings.Join(commands, "\n\n"))
	return true
}

func (p *BabblerPlugin) makeBabbler(name string) (*Babbler, error) {
	res, err := p.db.Exec(`insert into babblers (babbler) values (?);`, name)
	if err == nil {
		id, err := res.LastInsertId()
		if err != nil {
			log.Error().Err(err)
			return nil, err
		}
		return &Babbler{
			BabblerId: id,
			Name:      name,
		}, nil
	}
	return nil, err
}

func (p *BabblerPlugin) getBabbler(name string) (*Babbler, error) {
	var bblr Babbler
	err := p.db.QueryRowx(`select * from babblers where babbler = ? LIMIT 1;`, name).StructScan(&bblr)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Error().Msg("failed to find babbler")
			return nil, NO_BABBLER
		}
		log.Error().Err(err).Msg("encountered problem in babbler lookup")
		return nil, err
	}
	return &bblr, nil
}

func (p *BabblerPlugin) getOrCreateBabbler(name string) (*Babbler, error) {
	babbler, err := p.getBabbler(name)
	if err == NO_BABBLER {
		babbler, err = p.makeBabbler(name)
		if err != nil {
			log.Error().Err(err)
			return nil, err
		}

		rows, err := p.db.Queryx(fmt.Sprintf("select tidbit from factoid where fact like '%s quotes';", babbler.Name))
		if err != nil {
			log.Error().Err(err)
			return babbler, nil
		}
		defer rows.Close()

		tidbits := []string{}
		for rows.Next() {
			var tidbit string
			err := rows.Scan(&tidbit)

			log.Debug().Str("tidbit", tidbit)

			if err != nil {
				log.Error().Err(err)
				return babbler, err
			}
			tidbits = append(tidbits, tidbit)
		}

		for _, tidbit := range tidbits {
			if err = p.addToMarkovChain(babbler, tidbit); err != nil {
				log.Error().Err(err)
			}
		}
	}
	return babbler, err
}

func (p *BabblerPlugin) getWord(word string) (*BabblerWord, error) {
	var w BabblerWord
	err := p.db.QueryRowx(`select * from babblerWords where word = ? LIMIT 1;`, word).StructScan(&w)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NEVER_SAID
		}
		return nil, err
	}
	return &w, nil
}

func (p *BabblerPlugin) createNewWord(word string) (*BabblerWord, error) {
	res, err := p.db.Exec(`insert into babblerWords (word) values (?);`, word)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	return &BabblerWord{
		WordId: id,
		Word:   word,
	}, nil
}

func (p *BabblerPlugin) getOrCreateWord(word string) (*BabblerWord, error) {
	if w, err := p.getWord(word); err == NEVER_SAID {
		return p.createNewWord(word)
	} else {
		if err != nil {
			log.Error().Err(err)
		}
		return w, err
	}
}

func (p *BabblerPlugin) getBabblerNode(babbler *Babbler, word string) (*BabblerNode, error) {
	w, err := p.getWord(word)
	if err != nil {
		return nil, err
	}

	var node BabblerNode
	err = p.db.QueryRowx(`select * from babblerNodes where babblerId = ? and wordId = ? LIMIT 1;`, babbler.BabblerId, w.WordId).StructScan(&node)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NEVER_SAID
		}
		return nil, err
	}
	return &node, nil
}

func (p *BabblerPlugin) createBabblerNode(babbler *Babbler, word string) (*BabblerNode, error) {
	w, err := p.getOrCreateWord(word)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}

	res, err := p.db.Exec(`insert into babblerNodes (babblerId, wordId, root, rootFrequency) values (?, ?, 0, 0)`, babbler.BabblerId, w.WordId)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}

	return &BabblerNode{
		NodeId:        id,
		WordId:        w.WordId,
		Root:          0,
		RootFrequency: 0,
	}, nil
}

func (p *BabblerPlugin) getOrCreateBabblerNode(babbler *Babbler, word string) (*BabblerNode, error) {
	node, err := p.getBabblerNode(babbler, word)
	if err != nil {
		return p.createBabblerNode(babbler, word)
	}
	return node, nil
}

func (p *BabblerPlugin) incrementRootWordFrequency(babbler *Babbler, word string) (*BabblerNode, error) {
	node, err := p.getOrCreateBabblerNode(babbler, word)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	_, err = p.db.Exec(`update babblerNodes set rootFrequency = rootFrequency + 1, root = 1 where id = ?;`, node.NodeId)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	node.RootFrequency += 1
	return node, nil
}

func (p *BabblerPlugin) getBabblerArc(fromNode, toNode *BabblerNode) (*BabblerArc, error) {
	var arc BabblerArc
	err := p.db.QueryRowx(`select * from babblerArcs where fromNodeId = ? and toNodeId = ?;`, fromNode.NodeId, toNode.NodeId).StructScan(&arc)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NEVER_SAID
		}
		return nil, err
	}
	return &arc, nil
}

func (p *BabblerPlugin) incrementWordArc(fromNode, toNode *BabblerNode) (*BabblerArc, error) {
	res, err := p.db.Exec(`update babblerArcs set frequency = frequency + 1 where fromNodeId = ? and toNodeId = ?;`, fromNode.NodeId, toNode.NodeId)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}

	affectedRows := int64(0)
	if err == nil {
		affectedRows, _ = res.RowsAffected()
	}

	if affectedRows == 0 {
		res, err = p.db.Exec(`insert into babblerArcs (fromNodeId, toNodeId, frequency) values (?, ?, 1);`, fromNode.NodeId, toNode.NodeId)
		if err != nil {
			log.Error().Err(err)
			return nil, err
		}
	}
	return p.getBabblerArc(fromNode, toNode)
}

func (p *BabblerPlugin) incrementFinalWordArcHelper(babbler *Babbler, node *BabblerNode) (*BabblerArc, error) {
	nextNode, err := p.getOrCreateBabblerNode(babbler, " ")
	if err != nil {
		return nil, err
	}
	return p.incrementWordArc(node, nextNode)
}

func (p *BabblerPlugin) addToMarkovChain(babbler *Babbler, phrase string) error {
	words := strings.Fields(strings.ToLower(phrase))

	if len(words) <= 0 {
		return nil
	}

	curNode, err := p.incrementRootWordFrequency(babbler, words[0])
	if err != nil {
		log.Error().Err(err)
		return err
	}

	for i := 1; i < len(words); i++ {
		nextNode, err := p.getOrCreateBabblerNode(babbler, words[i])
		if err != nil {
			log.Error().Err(err)
			return err
		}
		_, err = p.incrementWordArc(curNode, nextNode)
		if err != nil {
			log.Error().Err(err)
			return err
		}
		curNode = nextNode
	}

	_, err = p.incrementFinalWordArcHelper(babbler, curNode)
	return err
}

func (p *BabblerPlugin) getWeightedRootNode(babbler *Babbler) (*BabblerNode, *BabblerWord, error) {
	rows, err := p.db.Queryx(`select * from babblerNodes where babblerId = ? and root = 1;`, babbler.BabblerId)
	if err != nil {
		log.Error().Err(err)
		return nil, nil, err
	}
	defer rows.Close()

	rootNodes := []*BabblerNode{}
	total := int64(0)

	for rows.Next() {
		var node BabblerNode
		err = rows.StructScan(&node)
		if err != nil {
			log.Error().Err(err)
			return nil, nil, err
		}
		rootNodes = append(rootNodes, &node)
		total += node.RootFrequency
	}

	if len(rootNodes) == 0 {
		return nil, nil, SAID_NOTHING
	}

	which := rand.Int63n(total)
	total = 0
	for _, node := range rootNodes {
		total += node.RootFrequency
		if total >= which {
			var w BabblerWord
			err := p.db.QueryRowx(`select * from babblerWords where id = ? LIMIT 1;`, node.WordId).StructScan(&w)
			if err != nil {
				log.Error().Err(err)
				return nil, nil, err
			}
			return node, &w, nil
		}

	}
	log.Fatal().Msg("failed to find weighted root word")
	return nil, nil, nil
}

func (p *BabblerPlugin) getWeightedNextWord(fromNode *BabblerNode) (*BabblerNode, *BabblerWord, error) {
	rows, err := p.db.Queryx(`select * from babblerArcs where fromNodeId = ?;`, fromNode.NodeId)
	if err != nil {
		log.Error().Err(err)
		return nil, nil, err
	}
	defer rows.Close()

	arcs := []*BabblerArc{}
	total := int64(0)
	for rows.Next() {
		var arc BabblerArc
		err = rows.StructScan(&arc)
		if err != nil {
			log.Error().Err(err)
			return nil, nil, err
		}
		arcs = append(arcs, &arc)
		total += arc.Frequency
	}

	if len(arcs) == 0 {
		return nil, nil, errors.New("missing arcs")
	}

	which := rand.Int63n(total)
	total = 0
	for _, arc := range arcs {

		total += arc.Frequency

		if total >= which {
			var node BabblerNode
			err := p.db.QueryRowx(`select * from babblerNodes where id = ? LIMIT 1;`, arc.ToNodeId).StructScan(&node)
			if err != nil {
				log.Error().Err(err)
				return nil, nil, err
			}

			var w BabblerWord
			err = p.db.QueryRowx(`select * from babblerWords where id = ? LIMIT 1;`, node.WordId).StructScan(&w)
			if err != nil {
				log.Error().Err(err)
				return nil, nil, err
			}
			return &node, &w, nil
		}

	}
	log.Fatal().Msg("failed to find weighted next word")
	return nil, nil, nil
}

func (p *BabblerPlugin) getWeightedPreviousWord(toNode *BabblerNode) (*BabblerNode, *BabblerWord, bool, error) {
	rows, err := p.db.Queryx(`select * from babblerArcs where toNodeId = ?;`, toNode.NodeId)
	if err != nil {
		log.Error().Err(err)
		return nil, nil, false, err
	}
	defer rows.Close()

	arcs := []*BabblerArc{}
	total := int64(0)
	for rows.Next() {
		var arc BabblerArc
		err = rows.StructScan(&arc)
		if err != nil {
			log.Error().Err(err)
			return nil, nil, false, err
		}
		arcs = append(arcs, &arc)
		total += arc.Frequency
	}

	if len(arcs) == 0 {
		return nil, nil, true, nil
	}

	which := rand.Int63n(total + toNode.RootFrequency)

	//terminate the babble
	if which >= total {
		return nil, nil, true, nil
	}

	total = 0
	for _, arc := range arcs {

		total += arc.Frequency

		if total >= which {
			var node BabblerNode
			err := p.db.QueryRowx(`select * from babblerNodes where id = ? LIMIT 1;`, arc.FromNodeId).StructScan(&node)
			if err != nil {
				log.Error().Err(err)
				return nil, nil, false, err
			}

			var w BabblerWord
			err = p.db.QueryRowx(`select * from babblerWords where id = ? LIMIT 1;`, node.WordId).StructScan(&w)
			if err != nil {
				log.Error().Err(err)
				return nil, nil, false, err
			}
			return &node, &w, false, nil
		}
	}
	log.Fatal().Msg("failed to find weighted previous word")
	return nil, nil, false, nil
}

func (p *BabblerPlugin) verifyPhrase(babbler *Babbler, phrase []string) (*BabblerNode, *BabblerNode, error) {
	curNode, err := p.getBabblerNode(babbler, phrase[0])
	if err != nil {
		log.Error().Err(err)
		return nil, nil, err
	}
	firstNode := curNode
	for i := 1; i < len(phrase); i++ {
		nextNode, err := p.getBabblerNode(babbler, phrase[i])
		if err != nil {
			log.Error().Err(err)
			return nil, nil, err
		}
		_, err = p.getBabblerArc(curNode, nextNode)
		if err != nil {
			log.Error().Err(err)
			return nil, nil, err
		}
		curNode = nextNode
	}

	return firstNode, curNode, nil
}

func (p *BabblerPlugin) babble(who string) (string, error) {
	return p.babbleSeed(who, []string{})
}

func (p *BabblerPlugin) babbleSeed(babblerName string, seed []string) (string, error) {
	babbler, err := p.getBabbler(babblerName)
	if err != nil {
		log.Error().Err(err)
		return "", nil
	}

	words := seed

	var curNode *BabblerNode
	var curWord *BabblerWord
	if len(seed) == 0 {
		curNode, curWord, err = p.getWeightedRootNode(babbler)
		if err != nil {
			log.Error().Err(err)
			return "", err
		}
		words = append(words, curWord.Word)
	} else {
		_, curNode, err = p.verifyPhrase(babbler, seed)
		if err != nil {
			log.Error().Err(err)
			return "", err
		}
	}

	for {
		curNode, curWord, err = p.getWeightedNextWord(curNode)
		if err != nil {
			log.Error().Err(err)
			return "", err
		}
		if curWord.Word == " " {
			break
		}
		words = append(words, curWord.Word)

		if len(words) >= 250 {
			break
		}
	}

	return strings.TrimSpace(strings.Join(words, " ")), nil
}

func (p *BabblerPlugin) mergeBabblers(intoBabbler, otherBabbler *Babbler, intoName, otherName string) error {
	intoNode, err := p.getOrCreateBabblerNode(intoBabbler, "<"+intoName+">")
	if err != nil {
		log.Error().Err(err)
		return err
	}
	otherNode, err := p.getOrCreateBabblerNode(otherBabbler, "<"+otherName+">")
	if err != nil {
		log.Error().Err(err)
		return err
	}

	mapping := map[int64]*BabblerNode{}

	rows, err := p.db.Queryx("select * from babblerNodes where babblerId = ?;", otherBabbler.BabblerId)
	if err != nil {
		log.Error().Err(err)
		return err
	}
	defer rows.Close()

	nodes := []*BabblerNode{}

	for rows.Next() {
		var node BabblerNode
		err = rows.StructScan(&node)
		if err != nil {
			log.Error().Err(err)
			return err
		}
		nodes = append(nodes, &node)
	}

	for _, node := range nodes {
		var res sql.Result

		if node.NodeId == otherNode.NodeId {
			node.WordId = intoNode.WordId
		}

		if node.Root > 0 {
			res, err = p.db.Exec(`update babblerNodes set rootFrequency = rootFrequency + ?, root = 1 where babblerId = ? and wordId = ?;`, node.RootFrequency, intoBabbler.BabblerId, node.WordId)
			if err != nil {
				log.Error().Err(err)
			}
		} else {
			res, err = p.db.Exec(`update babblerNodes set rootFrequency = rootFrequency + ? where babblerId = ? and wordId = ?;`, node.RootFrequency, intoBabbler.BabblerId, node.WordId)
			if err != nil {
				log.Error().Err(err)
			}
		}

		rowsAffected := int64(-1)
		if err == nil {
			rowsAffected, _ = res.RowsAffected()
		}

		if err != nil || rowsAffected == 0 {
			res, err = p.db.Exec(`insert into babblerNodes (babblerId, wordId, root, rootFrequency) values (?,?,?,?) ;`, intoBabbler.BabblerId, node.WordId, node.Root, node.RootFrequency)
			if err != nil {
				log.Error().Err(err)
				return err
			}
		}

		var updatedNode BabblerNode
		err = p.db.QueryRowx(`select * from babblerNodes where babblerId = ? and wordId = ? LIMIT 1;`, intoBabbler.BabblerId, node.WordId).StructScan(&updatedNode)
		if err != nil {
			log.Error().Err(err)
			return err
		}

		mapping[node.NodeId] = &updatedNode
	}

	for oldNodeId, newNode := range mapping {
		rows, err := p.db.Queryx("select * from babblerArcs where fromNodeId = ?;", oldNodeId)
		if err != nil {
			return err
		}
		defer rows.Close()

		arcs := []*BabblerArc{}

		for rows.Next() {
			var arc BabblerArc
			err = rows.StructScan(&arc)
			if err != nil {
				log.Error().Err(err)
				return err
			}
			arcs = append(arcs, &arc)
		}

		for _, arc := range arcs {
			_, err := p.incrementWordArc(newNode, mapping[arc.ToNodeId])
			if err != nil {
				return err
			}
		}
	}

	return err
}

func (p *BabblerPlugin) babbleSeedSuffix(babblerName string, seed []string) (string, error) {
	babbler, err := p.getBabbler(babblerName)
	if err != nil {
		log.Error().Err(err)
		return "", nil
	}

	firstNode, curNode, err := p.verifyPhrase(babbler, seed)
	if err != nil {
		log.Error().Err(err)
		return "", err
	}

	words := []string{}
	var curWord *BabblerWord
	var shouldTerminate bool
	curNode = firstNode
	for {
		curNode, curWord, shouldTerminate, err = p.getWeightedPreviousWord(curNode)
		if err != nil {
			log.Error().Err(err)
			return "", err
		}

		if shouldTerminate {
			break
		}

		words = append(words, curWord.Word)

		if len(words) >= 250 {
			break
		}
	}

	for i := 0; i < len(words)/2; i++ {
		index := len(words) - (i + 1)
		words[i], words[index] = words[index], words[i]
	}

	words = append(words, seed...)

	return strings.TrimSpace(strings.Join(words, " ")), nil
}

func (p *BabblerPlugin) getNextArcs(babblerNodeId int64) ([]*BabblerArc, error) {
	arcs := []*BabblerArc{}
	rows, err := p.db.Queryx(`select * from babblerArcs where fromNodeId = ?;`, babblerNodeId)
	if err != nil {
		log.Error().Err(err)
		return arcs, err
	}
	defer rows.Close()

	for rows.Next() {
		var arc BabblerArc
		err = rows.StructScan(&arc)
		if err != nil {
			log.Error().Err(err)
			return []*BabblerArc{}, err
		}
		arcs = append(arcs, &arc)
	}
	return arcs, nil
}

func (p *BabblerPlugin) getBabblerNodeById(nodeId int64) (*BabblerNode, error) {
	var node BabblerNode
	err := p.db.QueryRowx(`select * from babblerNodes where id = ? LIMIT 1;`, nodeId).StructScan(&node)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	return &node, nil
}

func shuffle(a []*BabblerArc) {
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

func (p *BabblerPlugin) babbleSeedBookends(babblerName string, start, end []string) (string, error) {
	babbler, err := p.getBabbler(babblerName)
	if err != nil {
		log.Error().Err(err)
		return "", nil
	}

	_, startWordNode, err := p.verifyPhrase(babbler, start)
	if err != nil {
		log.Error().Err(err)
		return "", err
	}

	endWordNode, _, err := p.verifyPhrase(babbler, end)
	if err != nil {
		log.Error().Err(err)
		return "", err
	}

	type searchNode struct {
		babblerNodeId int64
		previous      *searchNode
	}

	open := []*searchNode{{startWordNode.NodeId, nil}}
	closed := map[int64]*searchNode{startWordNode.NodeId: open[0]}
	goalNodeId := int64(-1)

	for i := 0; i < len(open) && i < 1000; i++ {
		cur := open[i]

		arcs, err := p.getNextArcs(cur.babblerNodeId)
		if err != nil {
			return "", err
		}
		//add a little randomization in through child ordering
		shuffle(arcs)

		for _, arc := range arcs {
			if _, ok := closed[arc.ToNodeId]; !ok {
				child := &searchNode{arc.ToNodeId, cur}
				open = append(open, child)
				closed[arc.ToNodeId] = child

				if arc.ToNodeId == endWordNode.NodeId {
					goalNodeId = cur.babblerNodeId
					//add a little randomization in through maybe searching beyond this solution?
					if rand.Intn(4) == 0 {
						break
					}
				}
			}
		}
	}

	if goalNodeId == -1 {
		return "", errors.New("couldn't find path")
	} else if closed[goalNodeId].previous == nil {
		seeds := append(start, end...)
		return strings.Join(seeds, " "), nil
	}

	words := []string{}

	curSearchNode := closed[goalNodeId]

	for {
		cur, err := p.getBabblerNodeById(curSearchNode.babblerNodeId)
		if err != nil {
			log.Error().Err(err)
			return "", err
		}
		var w BabblerWord
		err = p.db.QueryRowx(`select * from babblerWords where id = ? LIMIT 1;`, cur.WordId).StructScan(&w)
		if err != nil {
			log.Error().Err(err)
			return "", err
		}
		words = append(words, w.Word)

		curSearchNode = closed[curSearchNode.previous.babblerNodeId]

		if curSearchNode.previous == nil {
			break
		}
	}

	for i := 0; i < len(words)/2; i++ {
		index := len(words) - (i + 1)
		words[i], words[index] = words[index], words[i]
	}

	words = append(start, words...)
	words = append(words, end...)

	return strings.Join(words, " "), nil
}
