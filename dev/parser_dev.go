package go_nets

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"code.google.com/p/go.text/encoding/charmap"
	"code.google.com/p/go.text/transform"
	"github.com/kr/pretty"
)

var dummy interface{}

func init() {
	fmt.Println("package 'fmt' is used now - debugging mode")
	fmt.Printf("package 'log' is used now - debugging mode %s \n", log.Ldate)
	io.WriteString(os.Stdout, "package 'io' and 'os' are used now - debugging mode \n")
}

const (
	FileDir  = "/media/FD/MISSIONS/ALEX/UM20140215_X/"
	FileName = "UMtest2.xml"
)

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

type Results struct {
	ProcessDate string
	ThruDate    string
	Filings     []Filing `xml:"FileDetail"`
}

type Document struct {
	Results    Results `xml:"Record>Results"`
	XMLVersion AttrVersionContainer
}

var data = `<FileDetail>
  <TransType Type="Initial"/>
  <FilingMethod Method="Paper"/>
  <AmendmentType Type="NoType"/>
  <AmendmentActionLoop>
    <AmendmentAction Action=""/>
  </AmendmentActionLoop>
  <AmendmentTypeLoop>
    <AmendmentType Type=""/>
  </AmendmentTypeLoop>
  <OriginalFileNumber>137363375543</OriginalFileNumber>
  <OriginalFileDate>20130522 1700</OriginalFileDate>
  <PreviousFileNumber/>
  <LapseDate>20230522</LapseDate>
  <FileNumber>137363375543</FileNumber>
  <FileDate>20130522 1700</FileDate>
  <FilingOffice>CA</FilingOffice>
  <ActionCode/>
  <AltNameDesignation AltName="NoAltType"/>
  <AltFilingType Type="StateLien"/>
  <MiscInfo/>
  <Debtors>
    <DebtorName>
      <Names>
        <OrganizationName>TMWSF, INC.</OrganizationName>
        <MailAddress>760 MARKET ST STE 817</MailAddress>
        <City>San Francisco</City>
        <State>CA</State>
        <PostalCode>94102</PostalCode>
        <County/>
        <Country>USA</Country>
        <TaxID/>
        <OrganizationType Type=""/>
        <OrganizationJuris/>
        <OrganizationID/>
        <Mark/>
      </Names>
      <DebtorAltCapacity AltCapacity="NOAltCapacity"/>
    </DebtorName>
    <DebtorName>
      <Names>
        <OrganizationName>TRUE MASSAGE &amp; WELLNESS</OrganizationName>
        <MailAddress>760 MARKET ST STE 817</MailAddress>
        <City>San Francisco</City>
        <State>CA</State>
        <PostalCode>94102</PostalCode>
        <County/>
        <Country>USA</Country>
        <TaxID/>
        <OrganizationType Type=""/>
        <OrganizationJuris/>
        <OrganizationID/>
        <Mark/>
      </Names>
      <DebtorAltCapacity AltCapacity="NOAltCapacity"/>
    </DebtorName>
  </Debtors>
  <Secured>
    <Names>
      <OrganizationName>EMPLOYMENT DEVELOPMENT DEPARTMENT</OrganizationName>
      <MailAddress>PO BOX 826880</MailAddress>
      <City>Sacramento</City>
      <State>CA</State>
      <PostalCode>94280</PostalCode>
      <County/>
      <Country>US</Country>
      <TaxID/>
      <OrganizationType Type=""/>
      <OrganizationJuris/>
      <OrganizationID/>
      <Mark/>
    </Names>
  </Secured>
  <Collateral>
    <ColText>
</ColText>
  </Collateral>
  <AuthorizingParty>
    <AuthSecuredParty>
      <OrganizationName/>
    </AuthSecuredParty>
    <AuthDebtor>
      <OrganizationName/>
    </AuthDebtor>
  </AuthorizingParty>
</FileDetail>
	`

func main_old() {

	var err error
	// var fi *os.File

	readFromFile := true
	if readFromFile {

		// Legacy code, testing a different (bad) method
		// if false {

		// 	n, _ := ioutil.ReadFile(FileDir + FileName)
		// 	p := bytes.NewBuffer(n)
		// 	d := xml.NewDecoder(p)
		// 	err = d.Decode(&v)

		// } else {

		v := new(Document)
		// Open input file and defer closing
		// var fi *os.File //non necessary if errors with different names
		fi, errOs := os.Open(FileDir + FileName)
		if errOs != nil {
			panic(errOs)
		}
		defer func() {
			if errOs = fi.Close(); errOs != nil {
				panic(errOs)
			}
		}()
		// Transform the encoding of the reading pipe
		fiUTF8 := transform.NewReader(fi, charmap.Windows1252.NewDecoder())
		dummy = fiUTF8 //TODO remove
		// Write a simple test to stdout
		// io.Copy(os.Stdout, fi)
		// Parse the xml
		d := xml.NewDecoder(fiUTF8)
		t0 := time.Now()
		err = d.Decode(&v)
		t1 := time.Now()
		// }
		l := len(v.Results.Filings)
		if l < 50 {
			// fmt.Printf("%+v\n", v)
			pretty.Printf("%# v\n", v)
		}
		fmt.Printf("Successfully parsed %d filings in %v \n", l, t1.Sub(t0))

	} else { // Read the data provided for testing
		v := new(Filing)
		err = xml.Unmarshal([]byte(data), &v)
		fmt.Printf("%+v\n", v)
	}
	if err != nil {
		if err == io.EOF {
			fmt.Println("### Unexpected EOF at end of parsing ###")
		}
		// log.Fatal(err)
		fmt.Printf("%+v\n", err)
		panic(err)
	}

}
