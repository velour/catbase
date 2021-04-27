package last

import (
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestNextNoonBeforeNoon(t *testing.T) {
	log.Logger = log.Logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	t0 := time.Date(2021, 04, 27, 10, 10, 0, 0, time.Local)
	t1 := nextNoon(t0)
	expected := 1*time.Hour + 50*time.Minute
	assert.Equal(t, expected, t1)
}

func TestNextNoonAfterNoon(t *testing.T) {
	log.Logger = log.Logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	t0 := time.Date(2021, 04, 27, 14, 15, 0, 0, time.Local)
	t1 := nextNoon(t0)
	expected := 21*time.Hour + 45*time.Minute
	assert.Equal(t, expected, t1)
}
