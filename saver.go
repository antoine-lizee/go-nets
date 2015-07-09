package go_nets

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

//////////
// Define a Saver interface and the listening function that goes with it
//

type Saver interface {
	SaveBatch([]Saveable) // TODO add error handling
	InitPersistance(Saveable) chan string
}

type Saveable interface {
	GetInitStatements() []string
	GetSavingStatements() []string
}

var BatchSize int = 10000

func ListenAndSave(c chan Saveable, s Saver) {
	// Initialize the saving process
	first := <-c
	statusCh := s.InitPersistance(first)
	batchSize := BatchSize
	// Initialize
	batch := make([]Saveable, batchSize)
	batch[0] = first
	i := 1
	for saveable := range c {
		if i == batchSize {
			log.Println("Saving first batch...")
			s.SaveBatch(batch)
			i = 0
		}
		// log.Printf("Filing number %d received, with id %d.", i, saveable.(Filing).FileNumber)
		batch[i] = saveable
		i++
	}
	s.SaveBatch(batch[:i])
	statusCh <- "done"
	<-statusCh
}

//////////
// Implement a sql saver
//
type SqlSaver struct {
	DbPath, DbName string
	DBDriver       string
	currentDB      *sql.DB
}

func (ss *SqlSaver) InitPersistance(so Saveable) chan string {

	// Prepare
	log.Println("Initializing sqlite db...")
	t0 := time.Now()
	tempFilePath := ss.DbPath + ss.DbName + TempSuffix
	os.Remove(tempFilePath)

	// Open/Create the database
	db, err := sql.Open(ss.DBDriver, tempFilePath)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Fatal(err)
	}
	ss.currentDB = db

	// Initialize the db (Create the tables...)
	for _, sqlStmt := range so.GetInitStatements() {
		_, err = db.Exec(sqlStmt)
		if err != nil {
			// log.Printf("%#v", err) //AL DEBUG
			log.Printf("%q: %s\n", err, sqlStmt)
		}
	}

	// Close the db
	statusCh := make(chan string)
	go func() {
		status := <-statusCh
		// log.Println("receiving %s ...", status)
		if status == "done" { // TODO: Rearrange closing db
			err := db.Close()
			if err != nil {
				log.Fatal(err)
			}
			os.Rename(tempFilePath, ss.DbPath+ss.DbName+".sqlite")
			log.Printf("\n Successfully saved the filings in %v \n", time.Now().Sub(t0))
			log.Println("### ---------------")
			close(statusCh)
		}
	}()

	return (statusCh)
}

func (s *SqlSaver) SaveBatch(ss []Saveable) {
	// Begin transaction
	log.Println("Beginning Transaction...")
	tx, err := s.currentDB.Begin()
	if err != nil {
		log.Fatal(err)
	}
	// Load the statements
	for _, saveable := range ss {
		for _, sqlStmt := range saveable.GetSavingStatements() {
			_, err = tx.Exec(sqlStmt)
			if err != nil {
				log.Printf("%q: %s\n", err, sqlStmt)
			}
		}
	}
	// Commit
	tx.Commit()
}

//////////
// Implement the Filing as a saveable object. Rely on the primary keys mechanics and other sql constraints for uniqueness.
//

func (f Filing) GetInitStatements() []string {
	return []string{
		`CREATE TABLE filings (
				filingid INT PRIMARY KEY NOT NULL,
				original_file_number INT,
				file_number INT NOT NULL,
				original_date TEXT,
				date TEST,
				xmlname VARCHAR(50),
				method VARCHAR(50),
				amendment VARCHAR(50),
				type VARCHAR(50),
				alt_type VARCHAR(50)
			)`, // BTW, string length are not inforced by sqlite. Also, NOT NULL is necessary for primary keys
		`CREATE TABLE agents (
		agentid TEXT PRIMARY KEY NOT NULL,
		organisation_name VARCHAR(250),
		first_name VARCHAR(250),
		middle_name VARCHAR(250),
		last_name VARCHAR(250),
		mail_address VARCHAR(250),
		city VARCHAR(250),
		state VARCHAR(250),
		postal_code VARCHAR(250),
		country VARCHAR(250)
		)`,
		`CREATE TABLE debtors (
		filingid INT,
		agentid TEXT,
		FOREIGN KEY(filingid) REFERENCES filings(filingid),
		FOREIGN KEY(agentid) REFERENCES agents(agentid)
		)`,
		`CREATE TABLE securers (
		filingid INT,
		agentid TEXT,
		FOREIGN KEY(filingid) REFERENCES filings(filingid),
		FOREIGN KEY(agentid) REFERENCES agents(agentid)
		)`,
	}
}

func (f Filing) GetSavingStatements() []string {
	sqlStmts := []string{}
	// Add the filing itself
	sqlStmts = append(sqlStmts,
		fmt.Sprintf("INSERT INTO filings VALUES ("+strings.Repeat("\"%v\", ", 9)+"\"%v\""+")",
			f.FileNumber,
			f.OriginalFileNumber,
			f.FileNumber,
			f.OriginalFileDate,
			f.FileDate,
			f.XMLName.Local,
			f.Method.Attr,
			f.Amendment.Attr,
			f.FilingType.Attr,
			f.AltFilingType.Attr),
	)
	// Add the debtors and their lookups
	for _, d := range f.Debtors {
		sqlStmts = append(sqlStmts,
			fmt.Sprintf("INSERT OR IGNORE INTO agents VALUES ("+strings.Repeat("\"%v\", ", 9)+"\"%v\""+")",
				d.GetIdentifier(),
				d.OrganizationName,
				d.IndividualName.FirstName,
				d.IndividualName.MiddleName,
				d.IndividualName.LastName,
				d.MailAddress,
				d.City,
				d.State,
				d.PostalCode,
				d.Country),
			fmt.Sprintf("INSERT INTO debtors VALUES (\"%v\", \"%v\")",
				f.FileNumber,
				d.GetIdentifier()),
		)
	}
	// Add the securers and their lookups
	for _, sec := range f.Securers {
		sqlStmts = append(sqlStmts,
			fmt.Sprintf("INSERT OR IGNORE INTO agents VALUES ("+strings.Repeat("\"%v\", ", 9)+"\"%v\""+")",
				sec.GetIdentifier(),
				sec.OrganizationName,
				sec.IndividualName.FirstName,
				sec.IndividualName.MiddleName,
				sec.IndividualName.LastName,
				sec.MailAddress,
				sec.City,
				sec.State,
				sec.PostalCode,
				sec.Country),
			fmt.Sprintf("INSERT INTO securers VALUES (\"%v\", \"%v\")",
				f.FileNumber,
				sec.GetIdentifier()),
		)
	}
	return sqlStmts
}

func FilingToSaveable(from <-chan Filing) chan Saveable {
	to := make(chan Saveable)
	go func() {
		for f := range from {
			to <- f
			// fmt.Printf("\n Casting and passing around record number %d", f.FileNumber)
		}
		close(to)
	}()
	return to
}

///////////
// Implement a filing specific version of the SaveBatch in order to speed up greatly the execution thanks to prepared state;ents
// Ends up being the same speed, but more reliable bc no need for value-quoting in the SQL statement.
//

func ListenAndSaveFilings(c chan Filing, s *SqlSaver) {
	// Initialize the saving process
	first := <-c
	statusCh := s.InitPersistance(first)
	batchSize := BatchSize
	// Initialize
	batch := make([]Filing, batchSize)
	batch[0] = first
	i := 1
	for filing := range c {
		if i == batchSize {
			log.Println("Saving first batch...")
			s.SaveFilingBatch(batch)
			i = 0
		}
		// log.Printf("Filing number %d received, with id %d.", i, saveable.(Filing).FileNumber)
		batch[i] = filing
		i++
	}
	s.SaveFilingBatch(batch[:i])
	statusCh <- "done"
	<-statusCh
}

//
func (ss *SqlSaver) SaveFilingBatch(batch []Filing) {
	// Begin Transaction
	log.Println("Beginning Transaction...")
	tx, err := ss.currentDB.Begin()
	if err != nil {
		log.Fatal(err)
	}

	// Prepare Statements
	preparationStmts := map[string]string{
		"filings":  "INSERT INTO filings VALUES (" + strings.Repeat("?, ", 9) + "?" + ")",
		"agents":   "INSERT OR IGNORE INTO agents VALUES (" + strings.Repeat("?, ", 9) + "?" + ")",
		"debtors":  "INSERT INTO debtors VALUES (?, ?)",
		"securers": "INSERT INTO securers VALUES (?, ?)",
	}
	preparedStmts := make(map[string]*sql.Stmt)
	for key, stmt := range preparationStmts {
		preparedStmts[key], err = tx.Prepare(stmt)
		if err != nil {
			log.Fatal(err)
		}
		defer preparedStmts[key].Close()
	}

	// Add Statements
	for _, f := range batch {
		// Add the filing itself
		_, err = preparedStmts["filings"].Exec(
			f.FileNumber,
			f.OriginalFileNumber,
			f.FileNumber,
			f.OriginalFileDate,
			f.FileDate,
			f.XMLName.Local,
			f.Method.Attr,
			f.Amendment.Attr,
			f.FilingType.Attr,
			f.AltFilingType.Attr)
		if err != nil {
			log.Printf("%q: %v\n", err, f.FileNumber)
		}
		// Add the debtors and their lookups
		for _, d := range f.Debtors {
			_, err = preparedStmts["agents"].Exec(
				d.GetIdentifier(),
				d.OrganizationName,
				d.IndividualName.FirstName,
				d.IndividualName.MiddleName,
				d.IndividualName.LastName,
				d.MailAddress,
				d.City,
				d.State,
				d.PostalCode,
				d.Country)
			if err != nil {
				log.Printf("%q. As Debtor: %s\n", err, d.GetIdentifier())
			}
			_, err = preparedStmts["debtors"].Exec(
				f.FileNumber,
				d.GetIdentifier())
			if err != nil {
				log.Printf("%q\n", err)
			}
		}
		// Add the securers and their lookups
		for _, sec := range f.Securers {
			_, err = preparedStmts["agents"].Exec(
				sec.GetIdentifier(),
				sec.OrganizationName,
				sec.IndividualName.FirstName,
				sec.IndividualName.MiddleName,
				sec.IndividualName.LastName,
				sec.MailAddress,
				sec.City,
				sec.State,
				sec.PostalCode,
				sec.Country)
			if err != nil {
				log.Printf("%q. As Securer: %s \n", err, sec.GetIdentifier())
			}
			_, err = preparedStmts["securers"].Exec(
				f.FileNumber,
				sec.GetIdentifier())
			if err != nil {
				log.Printf("%q\n", err)
			}
		}
	}

	// Commit
	log.Println("Commiting Transaction...")
	tx.Commit()

}
