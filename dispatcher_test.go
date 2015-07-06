package go_nets

import (
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"code.google.com/p/go.text/encoding/charmap"

	"github.com/kr/pretty"
)

var (
	TestAtomizeData = []struct {
		input    string
		expected string
	}{
		{"On Deck Capital, Inc.", "on_deck_capital"},
		{"ROCKWALL CAPITAL L.L.C.", "rockwall_capital"},
		{"CNH Capital   America LLC  ", "cnh_capital_america"},
		{"CNH Capital America, LLC  AS \"office\". Yeah", "cnh_capital_america_as_office_yeah"},
		{" INNOVATIVE EQUIPMENT TECHNOLOGY, INC.", "innovative_equipment_technology"},
		{" RABOBANK N.A., A NATIONAL BANKING ASSOCIATION, ON BEHALF OF ITSELF AND, TO THE EXTENT APPPLICABLE, AS AGENT FOR ANY OTHER SECURED PARTIES UNDER THE DEED OF TRUST DATED AS OF JUNE 11, 2013, BY DEBTOR FOR THE BENEFIT OF SECURED PARTY (COLLECTIVELY",
			"rabobank_na_a_national_banking_association_on_beha"},
	}
)

func TestAtomize(t *testing.T) {
	for _, testEl := range TestAtomizeData {
		if res := Atomize(testEl.input); res != testEl.expected {
			t.Errorf("Error when atomizing '%s': Got '%s', expected '%s'", testEl.input, res, testEl.expected)
		}
	}

}

// Create a debug function for the dispatcher
func ShowOutputs(d Dispatcher, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	log.Println("Dispatcher includes:")
	noders, edgers := d.Dispatch(log.New(w, "[ShowOutputsLog]", log.Lshortfile))
	for _, n := range noders {
		log.Printf("Node '%s' of kind %s with data: %+v\n", n.GetIdentifier(), n.GetKind(), n.GetData())
	}
	for _, e := range edgers {
		log.Printf("Edge '%s' of kind %s connecting '%s' to '%s'\n", e.GetIdentifier(), e.GetKind(), e.GetSrcId(), e.GetDstId())
	}
}

func TestDispatcher(t *testing.T) {
	fmt.Println("### TESTING the dispatcher")
	Parser := XmlParser{
		FileDir:  "_test/", ///media/FD/MISSIONS/ALEX/UM20140215_X/",
		FileName: "UMtest.xml",
		Encoding: charmap.Windows1252,
	}
	cs := make(chan Filing)
	go Parser.Parse(cs, nil)

	i := 0
	for p := range cs {
		// p := <-cs
		i++
		fmt.Printf("\r Filing number %d parsed, with id %d.\n", i, p.OriginalFileNumber)
		pretty.Printf("%# v\n", &p)
		fmt.Println("After munching of the dispatcher:")
		ShowOutputs(&p, nil)
	}
}
