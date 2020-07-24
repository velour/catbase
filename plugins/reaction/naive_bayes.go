package reaction

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/cdipaolo/goml/base"
	"github.com/cdipaolo/goml/text"
	"github.com/rs/zerolog/log"
)

type MetaData struct {
	NClasses  uint8
	ClassList []string
}

type bayesReactor struct {
	model *text.NaiveBayes
	meta  MetaData
}

func newBayesReactor(jsonPath string) *bayesReactor {
	reactor := &bayesReactor{}
	f, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		log.Error().Err(err).Msgf("error reading json")
		return reactor
	}
	var meta MetaData
	err = json.Unmarshal(f, &meta)
	if err != nil {
		log.Error().Err(err).Msgf("error reading json")
		return reactor
	}
	reactor.meta = meta

	stream := make(chan base.TextDatapoint, 100)
	//errors := make(chan error)
	model := text.NewNaiveBayes(stream, meta.NClasses, base.OnlyWordsAndNumbers)
	err = model.RestoreFromFile(strings.TrimSuffix(jsonPath, ".json"))
	if err != nil {
		log.Error().Err(err).Msgf("error reading json")
		return reactor
	}
	reactor.model = model

	return reactor
}

// React returns an emojy and probability given an input
func (b *bayesReactor) React(input string) (string, float64) {
	if b.model == nil {
		return "", 0.0
	}
	class, prob := b.model.Probability(input)
	emojy := b.meta.ClassList[class]
	return emojy, prob
}
