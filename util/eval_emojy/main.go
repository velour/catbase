package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/cdipaolo/goml/base"
	"github.com/cdipaolo/goml/text"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type MetaData struct {
	NClasses  uint8
	ClassList []string
}

func main() {
	log.Logger = log.With().Caller().Stack().Logger()
	log.Logger = log.Logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	jsonPath := flag.String("path", "", "path to model JSON")

	flag.Parse()
	if *jsonPath == "" {
		fmt.Fprintf(os.Stderr, "You must provide a model path.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	input := strings.Join(flag.Args(), " ")

	f, err := ioutil.ReadFile(*jsonPath)
	if err != nil {
		log.Fatal().Err(err).Msgf("error reading json")
	}
	var meta MetaData
	err = json.Unmarshal(f, &meta)
	if err != nil {
		log.Fatal().Err(err).Msgf("error reading json")
	}

	stream := make(chan base.TextDatapoint, 100)
	//errors := make(chan error)
	model := text.NewNaiveBayes(stream, meta.NClasses, base.OnlyWordsAndNumbers)
	err = model.RestoreFromFile(strings.TrimSuffix(*jsonPath, ".json"))
	if err != nil {
		log.Fatal().Err(err).Msgf("error reading json")
	}

	class, prob := model.Probability(input)
	emojy := meta.ClassList[class]
	fmt.Printf("%s: %s (%.2f)\n", input, emojy, prob)
}
