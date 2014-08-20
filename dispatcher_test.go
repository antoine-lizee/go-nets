package go_nets

import (
	"fmt"
	"io"
	"log"
	"os"
	"testing"

	"github.com/kr/pretty"
)

var (
	TestAtomizeData = []struct {
		input    string
		expected string
	}{
		{"On Deck Capital, Inc.", "ondeckcapital"},
		{"ROCKWALL CAPITAL L.L.C.", "rockwallcapital"},
		{"CNH Capital America LLC  ", "cnhcapitalamerica"},
		{" INNOVATIVE EQUIPMENT TECHNOLOGY, INC.", "innovativeequipmenttechnology"},
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
