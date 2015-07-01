package go_nets

import (
	"fmt"
	"os"
	"testing"

	"code.google.com/p/go.text/encoding/charmap"
)

func TestSaver(t *testing.T) {
	filename := "UMtest.xml"
	fmt.Println("### TESTING the saver (small file)")
	Parser := XmlParser{
		FileDir:  "_test/",
		FileName: filename,
		Encoding: charmap.Windows1252,
	}
	TestSaver := &SqlSaver{
		dbPath:   "_test/",
		dbName:   filename,
		DBDriver: "sqlite3",
	}
	cs := make(chan Filing)
	fi, errOs := os.Create("_test/saver_test_parser.log")
	if errOs != nil {
		panic(errOs) //TODO change it to t.Error
	}
	go Parser.Parse(cs, fi)
	ListenAndSave(FilingToSaveable(cs), TestSaver)
}
