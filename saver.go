package go_nets

import (
	"database/sql"
	"log"
	"os"
	"time"
)

// Define a Saver interface and the listening function that goes with it

type Saver interface {
	SaveFilingBatch([]Filing) // TODO add error handling
	InitPersistance(chan string)
}

func ListenAndSave(c <-chan Filing, s Saver) {
	ch := make(chan string)
	s.InitPersistance(ch)
	batchSize := 1000
	// Initialize
	batch := make([]Filing, batchSize)
	i := 0
	for f := range c {
		if i == batchSize {
			s.SaveFilingBatch(batch)
			i = 0
		}
		batch[i] = f
		i++
	}
	s.SaveFilingBatch(batch[:i])
	ch <- "done"
}

// Implement a sqlite saver
type SqlSaver struct {
	dbPath, dbName string
	DBDriver       string
	currentDB      *sql.DB
}

func (ss *SqlSaver) InitPersistance(ch chan string) {

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

	// Close the db
	go func() {
		status := <-ch
		if status == "done" { // TODO: Rearrange closing db
			err := db.Close()
			if err != nil {
				log.Fatal(err)
			}
			os.Rename(tempFilePath, ss.dbPath+ss.dbName+".sqlite")
			close(ch)
			log.Printf("\n Successfully saved the filings in %v \n", time.Now().Sub(t0))
			log.Println("### ---------------")
		}
	}()
}

// Define Filing as a Saver
func (f *Filing) Persist(logger *log.Logger) {
	noders := []Noder{}
	nodeIds := map[string]bool{} //For checking
	edgers := []Edger{}
	// First check duplicates... [See the code of clean() in the parser file]
	// We have to do that now to prevent from sending the useless stuff over the wire to the network and log wrong warnings...
	// It may be inefficient to do this kind of things at three different places (parser removes empty agents, here + Network check against existing data.)
	i := 0
	for {
		if i == len(f.Debtors) {
			break
		}
		d := f.Debtors[i]
		if nodeIds[d.GetIdentifier()] {
			// a.Data := a.UpdateData // Not implemented yet. (+ not straightforward implementation since there is no data field yet)
			logger.Println("DISPATCHER: removing debtor node", d.GetIdentifier(), "because of duplication.")
			f.Debtors = DeleteAgent(f.Debtors, i)
		} else {
			nodeIds[d.GetIdentifier()] = true
			i++
		}
	}
	i = 0
	for {
		if i == len(f.Securers) {
			break
		}
		s := f.Securers[i]
		if nodeIds[s.GetIdentifier()] {
			// a.Data := a.UpdateData // Not implemented yet. (+ not straightforward implementation since there is no data field yet)
			logger.Println("DISPATCHER: removing securer node", s.GetIdentifier(), "because of duplication.")
			f.Securers = DeleteAgent(f.Securers, i)
		} else {
			nodeIds[s.GetIdentifier()] = true
			i++
		}
	}
	// Do the actual dispatching now that it's clean...
	for i, d := range f.Debtors {
		d := d
		noders = append(noders, &d)
		// Add the RR Edges
		for j := i + 1; j < len(f.Debtors); j++ {
			edgers = append(edgers, f.NewFilingEdger(RR, d.GetIdentifier(), f.Debtors[j].GetIdentifier()))
		}
	}
	for i, s := range f.Securers {
		s := s
		noders = append(noders, &s)
		// Add the EE Edges
		for j := i + 1; j < len(f.Securers); j++ {
			edgers = append(edgers, f.NewFilingEdger(EE, s.GetIdentifier(), f.Securers[j].GetIdentifier()))
		}
		// Add the ER Edges :
		for _, d := range f.Debtors {
			edgers = append(edgers, f.NewFilingEdger(ER, s.GetIdentifier(), d.GetIdentifier()))
		}
	}
}
