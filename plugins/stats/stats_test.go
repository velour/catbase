package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestJSON(t *testing.T) {
	expected := 5
	b, err := json.Marshal(expected)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%+v", expected)
	t.Log(string(b))
}

func TestValueConversion(t *testing.T) {
	expected := value(5)

	b, err := expected.Bytes()
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(string(b))

	actual, err := valueFromBytes(b)
	if err != nil {
		t.Error(err)
		return
	}

	if fmt.Sprintf("%+v", actual) != fmt.Sprintf("%+v", expected) {
		t.Errorf("Did not get equivalent objects: %+v != %+v", actual, expected)
	}
}

func rmDB(t *testing.T, dbPath string) {
	err := os.Remove(dbPath)
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatal(err)
	}
}

func TestWithDB(t *testing.T) {
	dbPath := "test.db"
	rmDB(t, dbPath)

	t.Run("TestDBReadWrite", func(t *testing.T) {
		bucket := "testBucket"
		key := "testKey"

		expected := stats{stat{
			bucket,
			key,
			1,
		}}

		err := expected.toDB(dbPath)
		if err != nil {
			t.Fatalf("Error writing to DB: %s", err)
		}

		actual, err := statFromDB(dbPath, bucket, key)
		if err != nil {
			t.Fatalf("Error reading DB: %s", err)
		}

		if actual != expected[0] {
			t.Fatalf("%+v != %+v", actual, expected)
		}

	})

	rmDB(t, dbPath)

	t.Run("TestDBAddStatInLoop", func(t *testing.T) {
		bucket := "testBucket"
		key := "testKey"
		expected := value(25)

		os.Remove(dbPath)
		defer func() {
			err := os.Remove(dbPath)
			if err != nil {
				t.Fatal(err)
			}
		}()

		statPack := stats{stat{
			bucket,
			key,
			5,
		}}

		for i := 0; i < 5; i++ {
			err := statPack.toDB(dbPath)
			if err != nil {
				t.Fatalf("Error writing to DB: %s", err)
			}
		}

		actual, err := statFromDB(dbPath, bucket, key)
		if err != nil {
			t.Fatalf("Error reading DB: %s", err)
		}

		if actual.val != expected {
			t.Fatalf("%+v != %+v", actual.val, expected)
		}
	})

	rmDB(t, dbPath)

	t.Run("TestDBAddStats", func(t *testing.T) {
		bucket := "testBucket"
		key := "testKey"
		expected := value(5)

		os.Remove(dbPath)
		defer func() {
			err := os.Remove(dbPath)
			if err != nil {
				t.Fatal(err)
			}
		}()

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
		if err != nil {
			t.Fatalf("Error writing to DB: %s", err)
		}

		actual, err := statFromDB(dbPath, bucket, key)
		if err != nil {
			t.Fatalf("Error reading DB: %s", err)
		}

		if actual.val != expected {
			t.Fatalf("%+v != %+v", actual.val, expected)
		}
	})

	rmDB(t, dbPath)
}
