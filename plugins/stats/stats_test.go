package stats

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

var dbPath = "test.db"

func TestJSON(t *testing.T) {
	expected := 5
	b, err := json.Marshal(expected)
	assert.Nil(t, err)
	t.Logf("%+v", expected)
	t.Log(string(b))
}

func TestValueConversion(t *testing.T) {
	expected := value(5)

	b, err := expected.Bytes()
	assert.Nil(t, err)

	t.Log(string(b))

	actual, err := valueFromBytes(b)
	assert.Nil(t, err)

	assert.Equal(t, actual, expected)
}

func rmDB(t *testing.T) {
	err := os.Remove(dbPath)
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatal(err)
	}
}

func TestWithDB(t *testing.T) {
	rmDB(t)

	t.Run("TestDBReadWrite", func(t *testing.T) {
		bucket := "testBucket"
		key := "testKey"

		expected := stats{stat{
			bucket,
			key,
			1,
		}}

		err := expected.toDB(dbPath)
		assert.Nil(t, err)

		actual, err := statFromDB(dbPath, bucket, key)
		assert.Nil(t, err)

		assert.Equal(t, actual, expected[0])

	})

	rmDB(t)

	t.Run("TestDBAddStatInLoop", func(t *testing.T) {
		bucket := "testBucket"
		key := "testKey"
		expected := value(25)

		statPack := stats{stat{
			bucket,
			key,
			5,
		}}

		for i := 0; i < 5; i++ {
			err := statPack.toDB(dbPath)
			assert.Nil(t, err)
		}

		actual, err := statFromDB(dbPath, bucket, key)
		assert.Nil(t, err)

		assert.Equal(t, actual.val, expected)
	})

	rmDB(t)

	t.Run("TestDBAddStats", func(t *testing.T) {
		bucket := "testBucket"
		key := "testKey"
		expected := value(5)

		statPack := stats{
			stat{
				bucket,
				key,
				1,
			},
			stat{
				bucket,
				key,
				1,
			},
			stat{
				bucket,
				key,
				1,
			},
			stat{
				bucket,
				key,
				1,
			},
			stat{
				bucket,
				key,
				1,
			},
		}

		err := statPack.toDB(dbPath)
		assert.Nil(t, err)

		actual, err := statFromDB(dbPath, bucket, key)
		assert.Nil(t, err)

		assert.Equal(t, actual.val, expected)
	})

	rmDB(t)
}

func makeMessage(payload string) msg.Message {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return msg.Message{
		User:    &user.User{Name: "tester"},
		Channel: "test",
		Body:    payload,
		Command: isCmd,
	}
}

func testUserCounter(t *testing.T, count int) {
	expected := value(count)
	mb := bot.NewMockBot()
	mb.Cfg.Stats.DBPath = dbPath
	s := New(mb)
	assert.NotNil(t, s)

	for i := 0; i < count; i++ {
		s.Message(makeMessage("test"))
	}

	_, err := os.Stat(dbPath)
	assert.Nil(t, err)

	stat, err := statFromDB(mb.Config().Stats.DBPath, "user", "tester")
	assert.Nil(t, err)
	actual := stat.val
	assert.Equal(t, actual, expected)
}

func TestMessages(t *testing.T) {
	_, err := os.Stat(dbPath)
	assert.NotNil(t, err)

	t.Run("TestOneUserCounter", func(t *testing.T) {
		count := 5
		expected := value(count)
		mb := bot.NewMockBot()
		mb.Cfg.Stats.DBPath = dbPath
		s := New(mb)
		assert.NotNil(t, s)

		for i := 0; i < count; i++ {
			s.Message(makeMessage("test"))
		}

		_, err := os.Stat(dbPath)
		assert.Nil(t, err)

		stat, err := statFromDB(mb.Config().Stats.DBPath, "user", "tester")
		assert.Nil(t, err)
		actual := stat.val
		assert.Equal(t, actual, expected)
	})

	rmDB(t)

	t.Run("TestTenUserCounter", func(t *testing.T) {
		count := 5
		expected := value(count)
		mb := bot.NewMockBot()
		mb.Cfg.Stats.DBPath = dbPath
		s := New(mb)
		assert.NotNil(t, s)

		for i := 0; i < count; i++ {
			s.Message(makeMessage("test"))
		}

		_, err := os.Stat(dbPath)
		assert.Nil(t, err)

		stat, err := statFromDB(mb.Config().Stats.DBPath, "user", "tester")
		assert.Nil(t, err)
		actual := stat.val
		assert.Equal(t, actual, expected)
	})

	rmDB(t)

	t.Run("TestChannelCounter", func(t *testing.T) {
		count := 5
		expected := value(count)
		mb := bot.NewMockBot()
		mb.Cfg.Stats.DBPath = dbPath
		s := New(mb)
		assert.NotNil(t, s)

		for i := 0; i < count; i++ {
			s.Message(makeMessage("test"))
		}

		_, err := os.Stat(dbPath)
		assert.Nil(t, err)

		stat, err := statFromDB(mb.Config().Stats.DBPath, "channel", "test")
		assert.Nil(t, err)
		actual := stat.val
		assert.Equal(t, actual, expected)
	})

	rmDB(t)

	t.Run("TestSightingCounter", func(t *testing.T) {
		count := 5
		expected := value(count)
		mb := bot.NewMockBot()

		mb.Cfg.Stats.DBPath = dbPath
		mb.Cfg.Stats.Sightings = []string{"user", "nobody"}

		s := New(mb)
		assert.NotNil(t, s)

		for i := 0; i < count; i++ {
			s.Message(makeMessage("user sighting"))
		}

		_, err := os.Stat(dbPath)
		assert.Nil(t, err)

		stat, err := statFromDB(mb.Config().Stats.DBPath, "sighting", "user")
		assert.Nil(t, err)
		actual := stat.val
		assert.Equal(t, actual, expected)
	})

	rmDB(t)

	t.Run("TestSightingCounterNoResults", func(t *testing.T) {
		count := 5
		expected := value(0)
		mb := bot.NewMockBot()

		mb.Cfg.Stats.DBPath = dbPath
		mb.Cfg.Stats.Sightings = []string{}

		s := New(mb)
		assert.NotNil(t, s)

		for i := 0; i < count; i++ {
			s.Message(makeMessage("user sighting"))
		}

		_, err := os.Stat(dbPath)
		assert.Nil(t, err)

		stat, err := statFromDB(mb.Config().Stats.DBPath, "sighting", "user")
		assert.Nil(t, err)
		actual := stat.val
		assert.Equal(t, actual, expected)
	})

	rmDB(t)
}
