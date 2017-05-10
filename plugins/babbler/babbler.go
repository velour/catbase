// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package babbler

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

var (
	NO_BABBLER = errors.New("babbler not found")
	SAID_NOTHING = errors.New("hasn't said anything yet")
	NEVER_SAID = errors.New("never said that")
)


type BabblerPlugin struct {
	Bot    bot.Bot
	db     *sqlx.DB
	config *config.Config
}

func New(bot bot.Bot) *BabblerPlugin {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if bot.DBVersion() == 1 {
		if _, err := bot.DB().Exec(`create table if not exists babblers (
			id integer primary key,
			babbler string
		);`); err != nil {
			log.Fatal(err)
		}

		if _, err := bot.DB().Exec(`create table if not exists babblerWords (
			id integer primary key,
			word string
		);`); err != nil {
			log.Fatal(err)
		}

		if _, err := bot.DB().Exec(`create table if not exists babblerNodes (
			id integer primary key,
			babblerId integer,
			wordId integer,
			root integer,
			rootFrequency integer
		);`); err != nil {
			log.Fatal(err)
		}

		if _, err := bot.DB().Exec(`create table if not exists babblerArcs (
			id integer primary key,
			fromWordId integer,
			toWordId interger,
			frequency integer
		);`); err != nil {
			log.Fatal(err)
		}
	}

	plugin := &BabblerPlugin{
		Bot:    bot,
		db:     bot.DB(),
		config: bot.Config(),
	}

	return plugin
}

func (p *BabblerPlugin) Message(message msg.Message) bool {
	lowercase := strings.ToLower(message.Body)
	tokens := strings.Fields(lowercase)
	numTokens := len(tokens)

	saidSomething := false
	saidWhat := ""

	if numTokens >= 2 && tokens[1] == "says" {
		saidWhat, saidSomething = p.getBabble(tokens)
	} else if len(tokens) == 4 && strings.Index(lowercase, "initialize babbler for ") == 0 {
		saidWhat, saidSomething = p.initializeBabbler(tokens)
	} else if strings.Index(lowercase, "batch learn for ") == 0 {
		saidWhat, saidSomething = p.batchLearn(tokens)
	} else if len(tokens) == 5 && strings.Index(lowercase, "merge babbler") == 0 {
		saidWhat, saidSomething = p.merge(tokens)
	} else {
		//this should always return "", false
		saidWhat, saidSomething = p.addToBabbler(message.User.Name, lowercase)
	}

	if saidSomething {
		p.Bot.SendMessage(message.Channel, saidWhat)
	}
	return saidSomething
}

func (p *BabblerPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "initialize babbler for seabass\n\nseabass says")
}

func (p *BabblerPlugin) Event(kind string, message msg.Message) bool {
	return false
}

func (p *BabblerPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *BabblerPlugin) RegisterWeb() *string {
	return nil
}

func (p *BabblerPlugin) makeBabbler(babbler string) (int64, error) {
	res, err := p.db.Exec(`insert into babblers (babbler) values (?);`, babbler)
	if err == nil {
		id, _ := res.LastInsertId()
		return id, nil
	}
	return -1, err
}

func (p *BabblerPlugin) getBabbler(babbler string) (int64, error) {
	id := int64(-1)
	err := p.db.Get(&id, `select id from babblers where babbler = ?`, babbler)
	if err == sql.ErrNoRows {
		return -1, NO_BABBLER
	}
	return id, err
}

func (p *BabblerPlugin) getOrCreateBabbler(babbler string) (int64, error) {
	id, err := p.getBabbler(babbler)
	if err != nil {
		id, err = p.makeBabbler(babbler)
		if err != nil {
			return id, err
		}
		query := fmt.Sprintf(`select tidbit from factoid where fact like '%s quotes';`, babbler)
		rows, err := p.db.Query(query)
		if err != nil {
			//we'll just ignore this but the actual creation succeeded previously
			return id, nil
		}
		defer rows.Close()

		tidbits := []string{}
		for rows.Next() {
			var tidbit string
			err := rows.Scan(&tidbit)
			if err != nil {
				return id, err
			}
			tidbits = append(tidbits, tidbit)
		}

		for _, tidbit := range tidbits {
			p.addToMarkovChain(id, tidbit)
		}

	}
	return id, err
}

func (p *BabblerPlugin) getWordId(word string) (int64, error) {
	id := int64(-1)
	if err := p.db.Get(&id, `select id from babblerWords where word = ?`, word); err == sql.ErrNoRows {
		return -1, NEVER_SAID
	} else {
		if err != nil {
			log.Print(err)
		}
		return id, err
	}
}

func (p *BabblerPlugin) createNewWord(word string) (int64, error) {
	if res, err := p.db.Exec(`insert into babblerWords (word) values (?);`, word); err == nil {
		id, _ := res.LastInsertId()
		return id, nil
	} else {
		if err != nil {
			log.Print(err)
		}
		return -1, err
	}
}

func (p *BabblerPlugin) getOrCreateWord(word string) (int64, error) {
	if id, err := p.getWordId(word); err == NEVER_SAID {
		return p.createNewWord(word)
	} else {
		return id, err
	}
}

func (p *BabblerPlugin) getWordNodeId(babblerId int64, word string) (int64, error) {
	var err error
	wordId := int64(-1)
	if wordId, err = p.getWordId(word); err != nil {
		return -1, err
	}
	id := int64(-1)
	if err = p.db.Get(&id, `select id from babblerNodes where babblerId = ? and wordId = ?`, babblerId, wordId); err == sql.ErrNoRows {
		return -1, NEVER_SAID
	} else {
		if err != nil {
			log.Print(err)
		}
		return id, err
	}
}

func (p *BabblerPlugin) createWordNodeId(babblerId int64, word string) (int64, error) {
	var err error
	wordId := int64(-1)
	if wordId, err = p.getOrCreateWord(word); err != nil {
		log.Print(err)
		return -1, err
	}
	id := int64(-1)
	res, err := p.db.Exec(`insert into babblerNodes (babblerId, wordId, root, rootFrequency) values (?, ?, 0, 0)`, babblerId, wordId)
	if err == sql.ErrNoRows {
		return -1, NEVER_SAID
	} else if err == nil {
		id, _ = res.LastInsertId()
	}
	if err != nil {
		log.Print(err)
	}
	return id, err
}

func (p *BabblerPlugin) getOrCreateWordNode(babblerId int64, word string) (int64, error) {
	if id, err := p.getWordNodeId(babblerId, word); err == NEVER_SAID {
		return p.createWordNodeId(babblerId, word)
	} else {
		if err != nil {
			log.Print(err)
		}
		return id, err
	}
}

func (p *BabblerPlugin) incrementRootWordFrequency(babblerId int64, word string) (int64, error) {
	var err error
	id := int64(-1)
	if id, err = p.getOrCreateWordNode(babblerId, word); err != nil {
		return -1, err
	}

	if _, err = p.db.Exec(`update babblerNodes set rootFrequency = rootFrequency + 1, root = 1 where babblerId = ? and wordId = ?;`, babblerId, id); err != nil {
		if err != nil {
			log.Print(err)
		}
		return -1, err
	}
	return id, nil
}

func (p *BabblerPlugin) getWordArcHelper(fromWordId, toWordId int64) (int64, error) {
	id := int64(-1)
	err := p.db.Get(&id, `select id from babblerArcs where fromWordId = ? and toWordId = ?`, fromWordId, toWordId)
	if err == sql.ErrNoRows {
		return -1, NEVER_SAID
	}
	if err != nil {
		log.Print(err)
	}
	return id, err
}

func (p *BabblerPlugin) incrementWordArc(fromWordId, toWordId int64) (int64, error) {
	res, err := p.db.Exec(`update babblerArcs set frequency = frequency + 1 where fromWordId = ? and toWordId = ?`, fromWordId, toWordId)
	affectedRows := int64(0)
	if err == nil {
		affectedRows, _ = res.RowsAffected()
	}

	if err != nil || affectedRows == 0 {
		res, err = p.db.Exec(`insert into babblerArcs (fromWordId, toWordId, frequency) values (?, ?, 1);`, fromWordId, toWordId)
		if err != nil {
			log.Print(err)
			return -1, err
		}
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (p *BabblerPlugin) incrementFinalWordArcHelper(wordId int64) (int64, error) {
	return p.incrementWordArc(wordId, -1)
}

func (p *BabblerPlugin) incrementWordArcHelper(babblerId, fromWordId int64, toWord string) (int64, error) {
	toWordId, err := p.getOrCreateWordNode(babblerId, toWord)
	if err != nil {
		return -1, err
	}
	_, err = p.incrementWordArc(fromWordId, toWordId)
	if err != nil {
		return -1, err
	}
	return toWordId, nil
}

func (p *BabblerPlugin) addToMarkovChain(babblerId int64, phrase string) {
	words := strings.Fields(strings.ToLower(phrase))

	if len(words) <= 0 {
		return
	}

	id, err := p.incrementRootWordFrequency(babblerId, words[0])
	if err != nil {
		log.Print(err)
		return
	}

	for i := 1; i < len(words); i++ {
		id, err = p.incrementWordArcHelper(babblerId, id, words[i])
		if err != nil {
			log.Print(err)
			return
		}
	}

	_, err = p.incrementFinalWordArcHelper(id)
}

func (p *BabblerPlugin) getWeightedRootWord(babblerId int64) (int64, string, error) {
	query := fmt.Sprintf("select id, wordId, rootFrequency from babblerNodes where babblerId = %d and root = 1", babblerId)
	rows, err := p.db.Query(query)
	if err != nil {
		log.Print(err)
		return -1, "", err
	}
	defer rows.Close()

	idToWordId := map[int64]int64{}
	idToFreq := map[int64]int64{}
	total := int64(0)

	for rows.Next() {
		var id int64
		var wordId int64
		var rootFrequency int64
		err = rows.Scan(&id, &wordId, &rootFrequency)
		if err != nil {
			log.Print(err)
			return -1, "", err
		}

		total += rootFrequency
		idToFreq[id] = rootFrequency
		idToWordId[id] = wordId
	}

	if total == 0 {
		return -1, "", SAID_NOTHING
	}

	which := rand.Int63n(total)
	total = 0
	for id, freq := range idToFreq {
		if total+freq >= which {
			var word string
			err := p.db.Get(&word, `select word from babblerWords where id = ?`, idToWordId[id])
			if err != nil {
				log.Print(err)
				return -1, "", err
			}
			return id, word, nil
		}
		total += freq
	}
	log.Fatalf("shouldn't happen")
	return -1, "", errors.New("failed to find weighted root word")
}

func (p *BabblerPlugin) getWeightedNextWord(fromWordId int64) (int64, string, error) {
	query := fmt.Sprintf("select toWordId, frequency from babblerArcs where fromWordId = %d;", fromWordId)
	rows, err := p.db.Query(query)
	if err != nil {
		log.Print(err)
		return -1, "", err
	}
	defer rows.Close()

	idToFreq := map[int64]int64{}
	total := int64(0)

	for rows.Next() {
		var toWordId int64
		var frequency int64
		err = rows.Scan(&toWordId, &frequency)
		if err != nil {
			log.Print(err)
			return -1, "", err
		}
		total += frequency
		idToFreq[toWordId] = frequency
	}

	if total == 0 {
		return -1, "", errors.New("missing arcs")
	}

	which := rand.Int63n(total)
	total = 0
	for id, freq := range idToFreq {
		if total+freq >= which {
			if id < 0 {
				return -1, "", nil
			}

			var wordId int64
			if err := p.db.Get(&wordId, `select wordId from babblerNodes where id = ?`, id); err != nil {
				log.Print(err)
				return -1, "", err
			}
			var word string
			if err := p.db.Get(&word, `select word from babblerWords where id = ?`, wordId); err != nil {
				log.Print(err)
				return -1, "", err
			}

			return id, word, nil
		}
		total +=freq
	}
	log.Fatalf("shouldn't happen")
	return -1, "", errors.New("failed to find weighted next word")
}

func (p *BabblerPlugin) babble(who string) (string, error) {
	return p.babbleSeed(who, []string{})
}

func (p *BabblerPlugin) babbleSeed(babbler string, seed []string) (string, error) {
	babblerId, err := p.getBabbler(babbler)
	if err != nil {
		return "", nil
	}

	words := seed

	var curWordId int64
	if len(seed) == 0 {
		id, word, err := p.getWeightedRootWord(babblerId)
		if err != nil {
			return "", err
		}
		curWordId = id
		words = append(words, word)
	} else {
		id, err := p.getWordNodeId(babblerId, seed[0])
		if err != nil {

			return "", err
		}
		curWordId = id
		for i := 1; i < len(seed); i++ {
			nextWordId, err := p.getWordNodeId(babblerId, seed[i])
			if err != nil {
				return "", err
			}
			_, err = p.getWordArcHelper(curWordId, nextWordId)
			if err != nil {
				return "", err
			}
			curWordId = nextWordId
		}
	}

	for {
		id, word, err := p.getWeightedNextWord(curWordId)
		if err != nil {
			return "", err
		}
		if id < 0 {
			break
		}
		words = append(words, word)
		curWordId = id
	}

	return strings.TrimSpace(strings.Join(words, " ")), nil
}

func (p *BabblerPlugin) mergeBabblers(intoId, otherId int64, intoName, otherName string) error {
	intoStringId, err := p.getOrCreateWordNode(intoId, "<" + intoName + ">")
	if err != nil {
		return err
	}
	otherStringId, err := p.getOrCreateWordNode(otherId, "<" + otherName + ">")
	if err != nil {
		return err
	}

	log.Print(intoStringId)
	log.Print(otherStringId)

	mapping := map[int64]int64{}

	query := fmt.Sprintf("select wordId, root, rootFrequency from babblerNodes where babblerId = %d;", otherId)
	rows, err := p.db.Query(query)
	if err != nil {
		log.Print(err)
		return err
	}
	defer rows.Close()

	type Word struct {
		Id int64
		Word string
		Root int64
		RootFrequency int64
	}

	words := []Word{}

	for rows.Next() {
		word := Word{}
		err = rows.Scan(&word.Id, &word.Root, &word.RootFrequency)
		if err != nil {
			log.Print(err)
			return err
		}
		words = append(words, word)
	}

	for _, word := range words {

		if word.Id == otherStringId {
			log.Print("HERE")
			word.Id = intoStringId
		}

		var res sql.Result

		if word.Root > 0 {
			res, err = p.db.Exec(`update babblerNodes set rootFrequency = rootFrequency + ?, root = 1 where babblerId = ? and wordId = ? output id	;`, word.RootFrequency, intoId, word.Id)
		} else {
			res, err = p.db.Exec(`update babblerNodes set rootFrequency = rootFrequency + ? where babblerId = ? and wordId = ? output id;`, word.RootFrequency, intoId, word.Id)
		}

		rowsAffected := int64(-1)
		if err == nil {
			rowsAffected, _ = res.RowsAffected()
		}

		wordId := int64(-1)
		if err != nil || rowsAffected == 0 {
			res, err = p.db.Exec(`insert into babblerNodes (babblerId, wordId, root, rootFrequency) values (?,?,?,?) ;`, intoId, word.Id, word.Root, word.RootFrequency)
			if err != nil {
				log.Print(err)
				return err
			}
		}

		wordId, _ = res.LastInsertId()

		if word.Id == intoStringId {
			word.Id = otherStringId
		}
		mapping[word.Id] = wordId
	}

	type Arc struct {
		ToWordId int64
		Frequency int64
	}

	for lookup, newArcStart := range mapping {
		query = fmt.Sprintf("select toWordId, frequency from babblerArcs where fromWordId = %d;", lookup)
		rows, err := p.db.Query(query)
		if err != nil {
			return err
		}
		defer rows.Close()

		arcs := []Arc{}

		for rows.Next() {
			var arc Arc
			err = rows.Scan(&arc.ToWordId, &arc.Frequency)
			if err != nil {
				return err
			}
			arcs = append(arcs, arc)
		}

		for _, arc := range arcs {
			newArcEnd := int64(-1) //handle end arcs
			if arc.ToWordId >= 0 {
				newArcEnd = mapping[arc.ToWordId]
			}

			res, err := p.db.Exec(`update babblerArcs set frequency = frequency + ? where fromWordId = ? and toWordId = ?`, arc.Frequency, newArcStart, newArcEnd)
			rowsAffected := int64(0)
			if err == nil {
				rowsAffected, _ = res.RowsAffected()
			}
			if err != nil || rowsAffected == 0 {
				_, err = p.db.Exec(`insert into babblerArcs (fromWordId, toWordId, frequency) values (?, ?, ?);`, newArcStart, newArcEnd, arc.Frequency)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
