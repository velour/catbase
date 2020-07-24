package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cdipaolo/goml/base"
	"github.com/cdipaolo/goml/text"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type logEntry struct {
	Who    string
	Author string
	Body   string
	Emojy  string
}

type logs []logEntry

type emojySet map[string]bool

type MetaData struct {
	NClasses  uint8
	ClassList []string
}

func main() {
	log.Logger = log.With().Caller().Stack().Logger()
	log.Logger = log.Logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	logDir := flag.String("path", "", "path to logs")
	outFile := flag.String("out", "emojy.model", "path to store model")

	flag.Parse()
	if *logDir == "" {
		fmt.Fprintf(os.Stderr, "You must provide a log path.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	logs, classes := prepLogs(*logDir)
	model, meta := bayes(logs, classes)
	err := model.PersistToFile(*outFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to save model")
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to save model")
	}
	err = ioutil.WriteFile(*outFile+".json", metaJSON, 0666)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to save model")
	}
}

var re = regexp.MustCompile(`(?i)^\[.+\] <(?P<Who>[[:punct:][:alnum:]]+)> reacted to (?P<Author>[[:punct:][:alnum:]]+): (?P<Body>.+) with :(?P<Emojy>[[:punct:][:alnum:]]+):$`)

func prepLogs(path string) (logs, emojySet) {
	entries := logs{}
	emojies := emojySet{}
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		tmp, err := ioutil.ReadFile(path)
		content := string(tmp)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(content, "\n") {
			if strings.Contains(line, "unknown event") {
				continue
			}
			if !re.MatchString(line) {
				continue
			}
			entry := parseEntry(line)
			emojies[entry.Emojy] = true
			log.Debug().
				Interface("entry", entry).
				Str("line", line).
				Msgf("Found emojy reaction entry")
			entries = append(entries, entry)
		}
		return nil
	})
	if err != nil {
		log.Fatal().Msgf("Error walking: %s", err)
	}
	return entries, emojies
}

func parseEntry(content string) logEntry {
	out := logEntry{}
	subs := re.FindStringSubmatch(content)
	if len(subs) == 0 {
		return out
	}
	for i, n := range re.SubexpNames() {
		switch n {
		case "Who":
			out.Who = subs[i]
		case "Author":
			out.Author = subs[i]
		case "Body":
			out.Body = subs[i]
		case "Emojy":
			out.Emojy = subs[i]
		}
	}
	return out
}

func bayes(logs logs, classes emojySet) (*text.NaiveBayes, MetaData) {
	// create the channel of data and errors
	stream := make(chan base.TextDatapoint, 100)
	errors := make(chan error)

	nClasses := uint8(len(classes))

	classMap := map[string]uint8{}
	classList := []string{}
	for k, _ := range classes {
		classList = append(classList, k)
		classMap[k] = uint8(len(classList) - 1)
	}

	log.Debug().Strs("classList", classList).Interface("classMap", classMap).Int("nLogs", len(logs)).Msgf("about to train")

	// make a new NaiveBayes model with
	// 2 classes expected (classes in
	// datapoints will now expect {0,1}.
	// in general, given n as the classes
	// variable, the model will expect
	// datapoint classes in {0,...,n-1})
	//
	// Note that the model is filtering
	// the text to omit anything except
	// words and numbers (and spaces
	// obviously)
	model := text.NewNaiveBayes(stream, nClasses, base.OnlyWordsAndNumbers)
	go model.OnlineLearn(errors)

	for _, l := range logs {
		stream <- base.TextDatapoint{
			X: l.Body,
			Y: classMap[l.Emojy],
		}
	}

	close(stream)
	for {
		err := <-errors
		if err != nil {
			log.Error().Err(err).Msg("Error passed")
		} else {
			// training is done!
			break
		}
	}
	// now you can predict like normal
	in := "Should work properly once that number of documents increases."
	class := model.Predict(in) // 0
	emojy := classList[class]
	log.Debug().Msgf("Class prediction for %s: %v", in, emojy)

	meta := MetaData{
		NClasses:  nClasses,
		ClassList: classList,
	}

	return model, meta
}
