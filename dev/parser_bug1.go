// This is a bug/feature test about the omitempty.
// BEfore go 1.4, parsing a xml with a omitempty non-string field gives an error. (unrelated)
// Here we test the omitempty at parsing (and it doesn't change anything.)

package go_nets

import (
	"encoding/xml"
	"fmt"
)

type Person struct {
	FirstName   string `xml:",omitempty"`
	MiddleName  string `xml:",omitempty"`
	LastName    string `xml:",omitempty"`
	FameLevel   string `xml:",attr,omitempty"`
	WealthLevel string `xml:",attr,omitempty"`
}

var xmlData = `
	<Person WealthLevel="Good">
		<FirstName>Bill</FirstName>
		<MiddleName></MiddleName>
		<LastName>Gates</LastName>
	</Person>
`
var goData = Person{FirstName: "Bill", LastName: "Clinton", WealthLevel: "ok"}

func mainOld() {
	// Write xml, omitempty works :
	xmlS, _ := xml.Marshal(&goData)
	fmt.Printf("%+v\n", goData)
	fmt.Printf("%s\n", xmlS)
	// Read from xml, doesn't work :
	v := Person{FirstName: "none"}
	xml.Unmarshal([]byte(xmlData), &v)
	fmt.Printf("%+v\n", v)

}
