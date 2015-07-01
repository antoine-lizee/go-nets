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

func ListenAndSave(c <-chan Saveable, s Saver) {
	// Initialize the saving process
	first := <-c
	statusCh := s.InitPersistance(first)
	batchSize := 1000
	// Initialize
	batch := make([]Saveable, batchSize)
	i := 0
	for saveable := range c {
		if i == batchSize {
			log.Println("Saving first batch...")
			s.SaveBatch(batch)
			i = 0
		}
		log.Printf("\r Filing number %d received, with id %d.", i, saveable.(*Filing).OriginalFileNumber)
		batch[i] = saveable
		i++
	}
	s.SaveBatch(batch[:i])
	statusCh <- "done"
}

//////////
// Implement a sql saver
//
type SqlSaver struct {
	dbPath, dbName string
	DBDriver       string
	currentDB      *sql.DB
}

func (ss *SqlSaver) InitPersistance(so Saveable) chan string {

	// Prepare
	log.Println("Initializing sqlite db...")
	t0 := time.Now()
	tempFilePath := ss.dbPath + ss.dbName + TempSuffix
	os.Remove(tempFilePath)

	// Open/Create the database
	db, err := sql.Open(ss.DBDriver, tempFilePath)
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
			os.Rename(tempFilePath, ss.dbPath+ss.dbName+".sqlite")
			close(statusCh)
			log.Printf("\n Successfully saved the filings in %v \n", time.Now().Sub(t0))
			log.Println("### ---------------")
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
			_, err = s.currentDB.Exec(sqlStmt)
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

func (f *Filing) GetInitStatements() []string {
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
			type VARCHAR(50)
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

func (f *Filing) GetSavingStatements() []string {
	sqlStmts := []string{}
	// Add the filing itself
	sqlStmts = append(sqlStmts,
		fmt.Sprintf("INSERT INTO filings VALUES ("+strings.Repeat("%v, ", 8)+"%v"+")",
			f.OriginalFileNumber,
			f.OriginalFileNumber,
			f.FileNumber,
			f.OriginalFileDate,
			f.FileDate,
			f.XMLName.Local,
			f.Method.Attr,
			f.Amendment.Attr,
			f.FilingType.Attr),
	)
	// Add the debtors and their lookups
	for _, d := range f.Debtors {
		sqlStmts = append(sqlStmts,
			fmt.Sprintf("INSERT INTO agents VALUES ("+strings.Repeat("%v, ", 9)+"%v"+")",
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
			fmt.Sprintf("INSERT INTO debtors VALUES (%v, %v)",
				f.OriginalFileNumber,
				d.GetIdentifier()),
		)
	}
	// Add the securers and their lookups
	for _, sec := range f.Securers {
		sqlStmts = append(sqlStmts,
			fmt.Sprintf("INSERT INTO agents VALUES ("+strings.Repeat("%v, ", 9)+"%v"+")",
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
			fmt.Sprintf("INSERT INTO securers VALUES (%v, %v)",
				f.OriginalFileNumber,
				sec.GetIdentifier()),
		)
	}
	return sqlStmts
}

func FilingToSaveable(from <-chan Filing) chan Saveable {
	to := make(chan Saveable)
	go func() {
		for f := range from {
			to <- &f
		}
		close(to)
	}()
	return to
}
