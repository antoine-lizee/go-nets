package go_nets

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"math"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type NodeKind int

const (
	Emitter NodeKind = iota
	Receiver
)

func (nk NodeKind) String() string {
	NKStrings := []string{
		"Emitter",
		"Receiver",
	}
	return NKStrings[int(nk)]
}

type EdgeKind int

const (
	ER EdgeKind = iota
	EE
	RR
)

func (ek EdgeKind) String() string {
	EKStrings := []string{
		"Emitter-Receiver",
		"Emitter-Emitter",
		"Receiver-Receiver",
	}
	return EKStrings[int(ek)]
}

type Node struct {
	Name     string
	Kind     NodeKind
	Edges    map[string]*Edge
	Nedges   int
	NodeData AttrGetter
}

type Edge struct {
	Name     string
	Kind     EdgeKind
	Src      *Node
	Dst      *Node
	LinkData *AttrGetter
}

type AttrGetter interface {
	// GetAttribute(string) interface{}
}

type Network struct {
	// Objects of the network
	Name   string
	Edges  map[string]Edge
	Nedges int
	Nodes  map[string]*Node // Less efficient with '*' but necessary because we want to share the pointer with the edges and you canot create apointer to a map element (that cna change location)
	Nnodes int
	// EdgeNames []string //AL Needed to iterate over all edges... ?
	// NodeNames []string //AL Needed to iterate over all nodes... ?
	//LinksData []AttrGetter //AL to be fixed / looked into.
	// Parameters of the Network
	Symmetrical bool // TODO let it be assymetrical in the nodes and co.
	// Meta parameters
	Logger         *log.Logger
	PersistingFile string
	DBDriver       string
}

func NewNetwork(name string, logWriter io.Writer) Network {
	if logWriter == nil {
		logWriter = os.Stdout
	}
	return Network{
		name,
		make(map[string]Edge),
		0,
		make(map[string]*Node),
		0,
		true,
		log.New(logWriter, "Network: ", log.Lshortfile),
		"Network0.sqlite",
		"sqlite3",
	}
}

type Noder interface {
	GetIdentifier() string
	GetKind() NodeKind
	GetData() AttrGetter
	UpdateData(AttrGetter) AttrGetter
}

type Edger interface {
	GetIdentifier() string
	GetKind() EdgeKind
	GetData() AttrGetter
	GetSrcId() string
	GetDstId() string
}

type Dispatcher interface {
	Dispatch(*log.Logger) ([]Noder, []Edger)
}

func (n *Network) AddNode(noder Noder) {
	id := noder.GetIdentifier()
	if node, ok := n.Nodes[id]; !ok { // Add Node
		n.Nodes[id] = &Node{
			id,
			noder.GetKind(),
			map[string]*Edge{},
			0,
			noder.GetData(),
		}
		n.Nnodes++
	} else { // Log & Update information
		n.Logger.Printf("ADD_NODE WARNING: Node '%s' already present, updating information (or not!)", id)
		node.NodeData = noder.UpdateData(node.NodeData)
	}
}

func (n *Network) AddEdge(edger Edger) {
	srcId := edger.GetSrcId()
	dstId := edger.GetDstId()
	id := srcId + "_" + edger.GetIdentifier() + "_" + dstId
	if _, ok := n.Edges[id]; !ok && n.Symmetrical { // Try the other way around
		id = dstId + "_" + edger.GetIdentifier() + "_" + srcId
	}
	_, ok1 := n.Nodes[srcId]
	_, ok2 := n.Nodes[dstId]
	if !(ok1 && ok2) { // Log and return
		n.Logger.Printf("ADD_EDGE ERROR: Edge '%s' could not have been created because of missing Source Node (%t - %s) and/or Destination Node (%t - %s)",
			id, !ok1, srcId, !ok2, dstId)
		return
	}
	data := edger.GetData()
	if _, ok := n.Edges[id]; !ok { // Add Edge
		n.Edges[id] = Edge{
			id,
			edger.GetKind(),
			n.Nodes[srcId],
			n.Nodes[dstId],
			&data,
		}
		n.Nedges++
	} else { // Update information
		n.Logger.Printf("ADD_EDGE WARNING: Edge %q (kind %s) already present, moving on...", id, edger.GetKind())
	}
}

func (n *Network) AddDispatcher(dispatcher Dispatcher) {
	noders, edgers := dispatcher.Dispatch(n.Logger)
	for _, noder := range noders {
		// fmt.Println("adding node", noder.GetIdentifier()) //DEBUG
		n.AddNode(noder)
	}
	for _, edger := range edgers {
		// fmt.Println("adding edge", edger.GetIdentifier()) //DEBUG
		n.AddEdge(edger)
	}
}

func (n *Network) Summary(w io.Writer) {
	sLog := fmt.Sprintf("## SUMMARY for Network '%s': %d Edges and %d Nodes\n", n.Name, n.Nedges, n.Nnodes)
	if w == nil {
		// r := n.Logger.Writer // "Writer" to get the ouput stream of the Logger is not defined.
		n.Logger.Print(sLog)
	} else {
		fmt.Fprint(w, sLog)
	}
}

const TempSuffix = ".tmp"

func (n *Network) Save() {
	TempFilePath := n.PersistingFile + TempSuffix
	ch := make(chan string)
	n.SaveNodes(TempFilePath, ch)
	n.SaveEdges(TempFilePath, ch)
	// for i := 0; i < 2; i++ {
	// 	s <- ch
	// }
	os.Rename(TempFilePath, n.PersistingFile)
}

func (n *Network) SaveNodes(fp string, ch chan string) {
	fmt.Printf("Trying to save the nodes of network %q into file %q\n", n.Name, fp)
	// Open/Create the database
	os.Remove(fp)
	db, err := sql.Open(n.DBDriver, fp)
	if err != nil {
		panic(err)
	}
	//Prepare & execute the table creation statement
	sqlStmt := `CREATE TABLE nodes (name TEXT NOT NULL primary key, kind TEXT)`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		// log.Printf("%#v", err) //AL DEBUG
		log.Printf("%q: %s\n", err, sqlStmt)
	}

	// Prepare and execute the main transaction. 100K Edges at a time
	batchsize := 100000.
	i := 0
	var (
		stmt *sql.Stmt
		tx   *sql.Tx
	)
	for _, node := range n.Nodes {
		// Open transaction and prepare statement
		if math.Mod(float64(i), batchsize) == 0 {
			fmt.Println("Beginning Transaction...")
			tx, err = db.Begin()
			if err != nil {
				log.Fatal(err)
			}
			stmt, err = tx.Prepare("INSERT INTO nodes(name, kind) values(?, ?)")
			if err != nil {
				log.Fatal(err)
			}
			defer stmt.Close()
		}
		// add Statements
		fmt.Print("\r Adding statement for node ", i, "  ")
		_, err = stmt.Exec(node.Name, node.Kind)
		if err != nil {
			log.Fatal(err)
		}
		// Commit transaction
		if math.Mod(float64(i+1), batchsize) == 0 || i == n.Nnodes-1 {
			fmt.Println("Comitting Transaction...")
			tx.Commit()
		}
		i++
	}
	// ch <- "node"
}

func (n *Network) SaveEdges(fp string, ch chan string) {
	fmt.Printf("Trying to save the edges of network %q into file %q\n", n.Name, fp)
	// Open/Create the database
	db, err := sql.Open(n.DBDriver, fp)
	if err != nil {
		panic(err)
	}
	//Prepare & execute the table creation statement
	sqlStmt := `CREATE TABLE edges (name TEXT NOT NULL primary key, kind TEXT, srcNode TEXT NOT NULL, dstnode TEXT NOT NULL)`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		// log.Printf("%#v", err) //AL DEBUG
		log.Printf("%q: %s\n", err, sqlStmt)
	}

	// Prepare and execute the main transaction. 100K Edges at a time
	batchsize := 100000.
	i := 0
	var (
		stmt *sql.Stmt
		tx   *sql.Tx
	)
	for _, edge := range n.Edges {
		// Open transaction and prepare statement
		if math.Mod(float64(i), batchsize) == 0 {
			fmt.Println("Beginning Transaction...")
			tx, err = db.Begin()
			if err != nil {
				log.Fatal(err)
			}
			stmt, err = tx.Prepare("INSERT INTO edges(name, kind, srcnode, dstnode) values(?, ?, ?, ?)")
			if err != nil {
				log.Fatal(err)
			}
			defer stmt.Close()
		}
		// add Statements
		fmt.Print("\r Adding statement for edge ", i, "  ")
		_, err = stmt.Exec(edge.Name, edge.Kind, edge.Src.Name, edge.Dst.Name)
		if err != nil {
			log.Fatal(err)
		}
		// Commit transaction
		if math.Mod(float64(i+1), batchsize) == 0 || i == n.Nedges-1 {
			fmt.Println("Comitting Transaction...")
			tx.Commit()
		}
		i++
	}
	// ch <- "edge"
}

func (n *Network) Load() {

}
