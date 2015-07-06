package go_nets

import (
	"fmt"
	"io/ioutil"
	"testing"

	"code.google.com/p/go.text/encoding/charmap"

	"github.com/kr/pretty"
)

func TestParser(t *testing.T) {
	fmt.Println("### TESTING the parser (big file)")
	Parser := XmlParser{
		FileDir:  "_test/",
		FileName: "UMtest2.xml",
		Encoding: charmap.Windows1252, // Comment that line to see it fail, and parse a small subset.
	}
	cs := make(chan Filing)
	go Parser.Parse(cs, ioutil.Discard) // Put nil instead of Discard to get the default stdout logging behaviour
	i := 0
	for p := range cs {
		i++
		fmt.Printf("\r Filing number %d parsed, with id %d.", i, p.OriginalFileNumber)
	}
	fmt.Println("\n### ---------------")
}

func TestParserVerbose(t *testing.T) {
	fmt.Println("### TESTING the parser (debug mode)")
	Parser := XmlParser{
		FileDir:  "_test/", //media/FD/MISSIONS/ALEX/UM20140215_X/",
		FileName: "UMtest.xml",
		Encoding: charmap.Windows1252,
	}
	cs := make(chan Filing)
	go Parser.ParseVerbose(cs, nil)
	i := 0
	for p := range cs {
		fmt.Println("\nReceived the filing object from the channel")
		i++
		fmt.Printf("Filing number %d parsed, with id %d.\n", i, p.OriginalFileNumber)
		pretty.Printf("%# v\n", p)
		fmt.Println("len(Debtors) =", len(p.Debtors))
		fmt.Println("\nSending the Go-ahead !")
		cs <- Filing{} // Ensure sequentiality
	}
	fmt.Println("\n### ---------------")
}

func TestClean(t *testing.T) {
	f := Filing{Debtors: []Agent{Agent{}}, Securers: []Agent{}}
	pretty.Printf("Seeing: %# v\n and len(Debtors) = %d \n", f, len(f.Debtors))
	f.clean()
	pretty.Printf("Seeing: %# v\n and len(Debtors) = %d \n", f, len(f.Debtors))

}
