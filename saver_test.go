package go_nets

import (
	"fmt"
	"os"
	"testing"

	"code.google.com/p/go.text/encoding/charmap"
)

func init() {
	initiateMultiCore(1)
}

func TestSaver(t *testing.T) {
	filename := "UMtest2.xml"
	fmt.Println("### TESTING the saver (big file)")
	Parser := XmlParser{
		FileDir:  "_test/",
		FileName: filename,
		Encoding: charmap.Windows1252,
	}
	TestSaver := &SqlSaver{
		DbPath:   "_test/",
		DbName:   filename,
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

func TestDirectSaver(t *testing.T) { // Gets more than above because of the non-discarded filings with " inside the fields.
	filename := "UMtest2.xml"
	fmt.Println("### TESTING the saver (big file)")
	Parser := XmlParser{
		FileDir:  "_test/",
		FileName: filename,
		Encoding: charmap.Windows1252,
	}
	TestSaver := &SqlSaver{
		DbPath:   "_test/",
		DbName:   "direct" + filename,
		DBDriver: "sqlite3",
	}
	cs := make(chan Filing)
	fi, errOs := os.Create("_test/saver_test_parser.log")
	if errOs != nil {
		panic(errOs) //TODO change it to t.Error
	}
	go Parser.Parse(cs, fi)
	ListenAndSaveFilings(cs, TestSaver)
}
