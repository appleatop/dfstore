package dfstore_test

import (
	"context"
	"log"

	"testing"

	"dfstore"

	"github.com/bobbae/q"
	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

var dataRows [][]string

func init() {
	dataRows = [][]string{
		{"title", "artist", "price", "year", "hardcover"},
		{"Blue Train", "John Coltrane", "56.99", "2018", "true"},
		{"Giant Steps", "John Coltrane", "63.99", "2019", "false"},
		{"Jeru", "Gerry Mulligan", "17.99", "2020", "false"},
		{"Sarah Vaughan", "Sarah Vaughan", "34.98", "2022", "true"},
	}
}

func TestDefault1(t *testing.T) {
	example1(t, "default")
}

func TestMemory1(t *testing.T) {
	example1(t, "memory")
}

func TestDocument1(t *testing.T) {
	example1(t, "document")
}

func TestParseCreateDB(t *testing.T) {
	dfs, err := dfstore.New(context.TODO(), "parse")
	if err != nil {
		t.Errorf("cannot get new dfstore, %v", err)
		return
	}

	defer dfs.Close()
	err = dfs.WriteRecords(dataRows)
	if err != nil {
		t.Errorf("cannot write, %v", err)
	}
}

func TestParseString1(t *testing.T) {
	dfs, err := dfstore.New(context.TODO(), "parse")
	if err != nil {
		t.Errorf("cannot get new dfstore, %v", err)
		return
	}

	defer dfs.Close()

	teststr := []string{
		//`([[artist]] == {"\"John\" Co\{ltt\}ran\[@@@ e\]"}) AND ([[year]] IN {"2018", "2019", "2022", "2023"})`,
		//`([[artist]] == {"\"John\" Co\{ltt\}ran\[@@@ e\]"}) AND ([[year]] != {"2018"})`,
		`(([[artist]] == {"John Coltrane"}) AND ([[year]] IN {"2018", "2019", "2022", "2023"})) OR ([[title]] != {"Blue Train"})`,
		`(([[title]] != {"Blue Train"}) AND ([[artist]] != {"John Coltrane"})) OR ([[hardcover]] == {"true"})) OR ([[year]] == {"2018"})`,
		// `([  [table.3.6]] == {"5"}) AND ([[color]	] != {"red", "blue"})`,
		// `([ [table.3.6]] == {"5"}) AND ([[color]	] != {{"red"}, {"blue"}})`,
		// `([ x [table.3.6]] == {"5"}) AND ([[color]	] != {"red", "blue"})`,
		// `([  [table.3.6]] == {"5"}) AND ([[color] x] != {"red", "blue"})`,
		// `([[artist]] == {"5"}) AND ([[color]] != {"red", "blue"})`,
		// `([[table.3.6]] == {"5"})`,
		`[[year]] != {"2018"}`,
		`[[year]] IN {"2018", "2022", "2020"}`,
		// `[[cost]] == [[income]]`,
	}

	columns := []string{"artist", "year", "title", "hardcover"}

	for _, condition := range teststr {
		res, err := dfs.ReadRecordsString(columns, []string{"ARTIST", "YEAR", "TITLE", "HARDCOVER?"}, condition, 20)
		if err != nil {
			t.Errorf("cannot read columns: %s condition: %s, %v", columns, condition, err)
		}
		q.Q(columns, condition, res)
		log.Println("read", res)
	}
}

func example1(t *testing.T, dbtype string) {
	dfs, err := dfstore.New(context.TODO(), dbtype)
	if err != nil {
		t.Errorf("cannot get new dfstore, %v", err)
		return
	}
	defer dfs.Close()

	err = dfs.WriteRecords(dataRows)
	if err != nil {
		t.Errorf("cannot write, %v", err)
	}
	// https://pkg.go.dev/github.com/go-gota/gota/dataframe#DataFrame.Filter

	// TODO compound filters and / or cases
	filters := []dataframe.F{
		dataframe.F{Colname: "artist", Comparator: series.Eq, Comparando: "John Coltrane"},
		dataframe.F{Colname: "price", Comparator: series.Greater, Comparando: "60"},
	}
	res, err := dfs.ReadRecords(filters, 20)
	if err != nil {
		t.Errorf("cannot read, %v", err)
	}
	log.Println("read", res)
}
