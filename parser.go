package go_nets

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/kr/pretty"

	"code.google.com/p/go.text/encoding"
	"code.google.com/p/go.text/transform"
)

var dummy interface{}

func init() {
	//Main Logger Shit
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type AttrMethodContainer struct {
	Attr string `xml:"Method,attr"`
}

type AttrTypeContainer struct {
	Attr string `xml:"Type,attr"`
}

type AttrVersionContainer struct {
	Attr string `xml:"Version,attr"`
}

type Filing struct {
	XMLName            xml.Name            `xml:"FileDetail"`
	Method             AttrMethodContainer `xml:"FilingMethod"`
	Amendment          AttrTypeContainer   `xml:"AmendmentType"`
	FilingType         AttrTypeContainer   `xml:"AltFilingType"`
	OriginalFileNumber int
	FileNumber         int
	OriginalFileDate   string
	FileDate           string
	Debtors            []Agent `xml:"Debtors>DebtorName>Names"`
	Securers           []Agent `xml:"Secured>Names"`
}

func DeleteAgent(agents []Agent, ind int) []Agent {
	if l := len(agents); ind > l {
		return nil // Not necessary: fmt.Errorf("DeleteAgent() ERROR: Asked for index %d in slice of size %d", ind, l)
	}
	if len(agents) == 1 {
		return []Agent{}
	}
	// Delete in place
	// agents[ind], agents[len(agents)-1], agents = agents[len(agents)-1], nil, agents[:len(agents)-1] //Pointer version, useless & not working
	agents[ind], agents = agents[len(agents)-1], agents[:len(agents)-1]
	return agents
}

func (f *Filing) clean() {
	// NullAgent := Agent{}
	i := 0
	for {
		if i == len(f.Debtors) {
			break
		}
		d := f.Debtors[i]
		if d.OrganizationName == "" && d.IndividualName.LastName == "" {
			f.Debtors = DeleteAgent(f.Debtors, i)
		} else {
			i++
		}
	}
	i = 0
	for {
		if i == len(f.Securers) {
			break
		}
		s := f.Securers[i]
		if s.OrganizationName == "" && s.IndividualName.LastName == "" {
			f.Securers = DeleteAgent(f.Securers, i)
		} else {
			i++
		}
	}
}

type Agent struct {
	OrganizationName string         `xml:",omitempty"`
	IndividualName   IndividualName `xml:"IndividualName,omitempty"`
	MailAddress      string
	City             string
	State            string
	PostalCode       string
	// County           string
	Country string
}

type IndividualName struct {
	FirstName  string `xml:",omitempty"`
	MiddleName string `xml:",omitempty"`
	LastName   string `xml:",omitempty"`
}

type XmlParser struct {
	FileDir  string
	FileName string
	Encoding encoding.Encoding
}

func (p *XmlParser) Parse(c chan Filing, logDst io.Writer) {

	// Unpack arguments & Initialize
	// Logger
	if logDst == nil {
		logDst = os.Stdout
	}
	logger := log.New(logDst, "parserLog", log.Lshortfile)

	// Open input file and defer closing
	// var fi *os.File //non necessary if errors with different names
	fi, errOs := os.Open(p.FileDir + p.FileName)
	if errOs != nil {
		panic(errOs)
	}
	defer func() {
		if errOs = fi.Close(); errOs != nil {
			panic(errOs)
		}
	}()
	// Transform the encoding of the reading pipe
	var fiUTF8 = io.Reader(fi)
	if enc := p.Encoding; enc != nil {
		fiUTF8 = transform.NewReader(fi, enc.NewDecoder())
	} else {
		fiUTF8 = fi
	}
	// Parse the xml
	decoder := xml.NewDecoder(fiUTF8)
	i := 0
	t0 := time.Now()
	for {
		// Read tokens from the XML document in a stream.
		t, err := decoder.Token()
		if t == nil {
			close(c)
			if err != io.EOF {
				log.Fatal(err)
			}
			break
		}
		// Inspect the type of the token just read.
		switch se := t.(type) {
		case xml.StartElement:
			// If we just read a StartElement token
			// ...and its name is "FileDetail"
			if se.Name.Local == "FileDetail" {
				var p Filing
				// decode a whole chunk of following XML into the
				// variable p which is a Filing (see above)
				decoder.DecodeElement(&p, &se)
				p.clean()
				// Check and Send the element
				if n := len(p.Debtors) + len(p.Securers); n > 1 {
					c <- p
				} else {
					logger.Printf("Record %d as been discarded because less than 2 debtor/secured party found.\n", p.OriginalFileNumber)
				}
				i++
			}
		}
	}
	t1 := time.Now()
	fmt.Printf("\n Successfully parsed %d filings in %v \n", i, t1.Sub(t0))
}

type OnOffWriter struct {
	writer  io.Writer
	Writing bool
}

func (w OnOffWriter) Write(p []byte) (n int, err error) {
	if w.Writing {
		return w.writer.Write(p)
	} else {
		return ioutil.Discard.Write(p)
	}
}

func (p *XmlParser) ParseVerbose(c chan Filing, logDst io.Writer) { // Only for Debugging

	// Unpack arguments & Initialize
	// Logger
	if logDst == nil {
		logDst = os.Stdout
	}
	logger := log.New(logDst, "parserLog", log.Lshortfile)

	// Open input file and defer closing
	// var fi *os.File //non necessary if errors with different names
	fi, errOs := os.Open(p.FileDir + p.FileName)
	if errOs != nil {
		panic(errOs)
	}
	defer func() {
		if errOs = fi.Close(); errOs != nil {
			panic(errOs)
		}
	}()
	// Transform the encoding of the reading pipe
	var fiUTF8 = io.Reader(fi)
	if enc := p.Encoding; enc != nil {
		fiUTF8 = transform.NewReader(fi, enc.NewDecoder())
	} else {
		fiUTF8 = fi
	}
	// Parse the xml
	// buffer := bytes.NewBuffer([]byte{})
	// finalReader := io.TeeReader(fiUTF8, buffer)
	// OOWriter := OnOffWriter{logDst, false}
	finalReader := io.TeeReader(fiUTF8, logDst)
	decoder := xml.NewDecoder(finalReader)
	i := 0
	t0 := time.Now()
	for {
		// Read tokens from the XML document in a stream.
		t, _ := decoder.Token()
		if t == nil {
			close(c)
			break
		}
		// Inspect the type of the token just read.
		switch se := t.(type) {
		case xml.StartElement:
			// If we just read a StartElement token
			// ...and its name is "page"
			if se.Name.Local == "FileDetail" {
				var p Filing
				// decode a whole chunk of following XML into the
				// variable p which is a Filing (se above)
				// buffer.Read() // Empty the buffer // Method not used
				// OOWriter.Writing = true // Doesn't really work, because already is already read when 'DecodeElement' is called.
				decoder.DecodeElement(&p, &se)
				p.clean()
				// time.Sleep(100 * time.Millisecond) // I don't understand why after all that shit, I  cannot have a sequential printing. I think the parser reads a lot more at once than just one line !
				// OOWriter.Writing = false
				// io.Copy(buffer, os.Stdout) // Output what has been read.
				// Check and Send the element
				if n := len(p.Debtors) + len(p.Securers); n > 1 {
					fmt.Fprintln(logDst, "\nSending the filing over the Channel")
					c <- p
					<-c
					fmt.Fprintln(logDst, "\nReceived Go-ahead, parsing on...")
				} else {
					logger.Printf("Record %d as been discarded because less than 2 debtor/secured party found.\n", p.OriginalFileNumber)
					logger.Printf("Here is the record: %# v", pretty.Formatter(p))
				}
				i++
			}
		}
	}
	t1 := time.Now()
	fmt.Printf("\n Successfully parsed %d filings in %v \n", i, t1.Sub(t0))
}
