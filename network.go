package go_nets

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"regexp"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

//--------------
//SECTION 0: DEFINITION OF THE NETOWRK

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

type EdgeToNode struct {
	*Edge
	ToNode *Node
}

type Node struct {
	Name     string
	Kind     NodeKind
	Edges    []*EdgeToNode //map[string]*Edge
	NodeData AttrGetter
}

// //OLD CODE, inefficient. Created the EdgeToNode to Change that
// func (n *Node) getNeighbor(e *Edge) *Node {
// 	if e.Src == n {
// 		return e.Dst
// 	} else {
// 		return e.Src
// 	}
// }
// //--------------------------------------------------------------

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
	Edges  map[string]*Edge // Remark: see below
	Nedges int
	Nodes  map[string]*Node // Less efficient with '*' but necessary because we want to share the pointer with the edges and you cannot create apointer to a map element (that can change location)
	Nnodes int
	// EdgeNames []string //AL Needed to iterate over all edges... ?
	// NodeNames []string //AL Needed to iterate over all nodes... ?
	//LinksData []AttrGetter //AL to be fixed / looked into.
	// Parameters of the Network
	Symmetrical bool // TODO let it be assymetrical in the nodes and co.
	// Meta parameters
	Folder         string
	Logger         *log.Logger
	PersistingFile string
	DBDriver       string
}

func NewNetwork(name string, logWriter io.Writer, folder, pf string) Network {
	if logWriter == nil {
		logWriter = os.Stdout
	}
	if pf == "" {
		pf = "Network0.sqlite"
	}
	return Network{
		name,
		make(map[string]*Edge),
		0,
		make(map[string]*Node),
		0,
		true,
		folder,
		log.New(logWriter, "Network: ", log.Lshortfile),
		pf,
		"sqlite3",
	}
}

type Noder interface {
	GetIdentifier() string
	GetKind() NodeKind
	GetData() AttrGetter
	UpdateData(AttrGetter) AttrGetter
}

type SimpleNoder struct {
	Name string
	Kind NodeKind
}

func (s *SimpleNoder) GetIdentifier() string {
	return s.Name
}
func (s *SimpleNoder) GetKind() NodeKind {
	return s.Kind
}
func (s *SimpleNoder) GetData() AttrGetter {
	return 0
}
func (s *SimpleNoder) UpdateData(AttrGetter) AttrGetter {
	return 0
}

type Edger interface {
	GetIdentifier() string
	GetKind() EdgeKind
	GetData() AttrGetter
	GetSrcId() string
	GetDstId() string
}

type SimpleEdger struct {
	Name         string
	Kind         EdgeKind
	SrcId, DstId string
}

func (s *SimpleEdger) GetIdentifier() string {
	return s.Name
}
func (s *SimpleEdger) GetKind() EdgeKind {
	return s.Kind
}
func (s *SimpleEdger) GetData() AttrGetter {
	return 0
}
func (s *SimpleEdger) GetSrcId() string {
	return s.SrcId
}
func (s *SimpleEdger) GetDstId() string {
	return s.DstId
}

type Dispatcher interface {
	Dispatch(*log.Logger) ([]Noder, []Edger)
}

//--------------
//SECTION 1: NETWORK BUILDING

func (n *Network) AddNode(noder Noder) {
	id := noder.GetIdentifier()
	if node, ok := n.Nodes[id]; !ok { // Add Node
		n.Nodes[id] = &Node{
			id,
			noder.GetKind(),
			[]*EdgeToNode{},
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
	id := edger.GetIdentifier()
	_, ok1 := n.Nodes[srcId]
	_, ok2 := n.Nodes[dstId]
	if !(ok1 && ok2) { // Log and return
		n.Logger.Printf("ADD_EDGE ERROR: Edge '%s' could not have been created because of missing Source Node (%t - %s) and/or Destination Node (%t - %s)",
			id, !ok1, srcId, !ok2, dstId)
		return
	}
	data := edger.GetData()
	if _, ok := n.Edges[id]; !ok { // Add Edge
		n.Edges[id] = &Edge{
			id,
			edger.GetKind(),
			n.Nodes[srcId],
			n.Nodes[dstId],
			&data,
		}
		n.Nodes[srcId].Edges = append(n.Nodes[srcId].Edges, &EdgeToNode{n.Edges[id], n.Nodes[dstId]})
		n.Nodes[dstId].Edges = append(n.Nodes[dstId].Edges, &EdgeToNode{n.Edges[id], n.Nodes[srcId]})
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

func compareNetworks(n1, n2 *Network) {
	i := 0
	en := 0
	for _, node1 := range n1.Nodes {
		if node2, ok := n2.Nodes[node1.Name]; ok {
			if node1.Kind == node2.Kind {
				fmt.Printf("\rCompared node number %d, name:%.20s", i, node1.Name)
				i++
			} else {
				log.Print("\nCOMPARE_ERROR: mismatching NodeKind for node number ", i, ", name:", node1.Name, "\n")
			}
		} else {
			log.Printf("\nCOMPARE_ERROR: node %q from network %s is missing in network %s", node1.Name, n1.Name, n2.Name)
			en++
			// break
		}
	}
	i = 0
	ee := 0
	for _, edge1 := range n1.Edges {
		if edge2, ok := n2.Edges[edge1.Name]; ok {
			if edge1.Kind == edge2.Kind && edge1.Src.Name == edge2.Src.Name && edge1.Dst.Name == edge2.Dst.Name {
				fmt.Printf("\rCompared edge number %d, name: %.20s", i, edge1.Name)
				i++
			} else {
				log.Print("COMPARE_ERROR: mismatching EdgeKind for edge number ", i, ", name:", edge1.Name, "\n")
			}
		} else {
			log.Printf("COMPARE_ERROR: edge %q from network %s is missing in network %s\n", edge1.Name, n1.Name, n2.Name)
			ee++
			// break
		}
	}
	fmt.Println("")
	if ee+en > 0 {
		log.Printf("COMPARE_ERROR: FAIL. The Networks are different. %s nodes are missing, %s edges are missing. \n", en, ee)
	} else {
		log.Printf("\n\nNetwoks %s and %s are similar\n", n1.Name, n2.Name)
	}
}

func (n *Network) Compare(ns ...Network) {
	for _, n2 := range ns {
		compareNetworks(n, &n2)
	}
}

// ---------------------
//SECTION 2: PERSISTANCE

const TempSuffix = ".tmp"

func (n *Network) SaveAs(fp string) {
	TempFilePath := fp + TempSuffix
	ch := make(chan string)
	os.Remove(TempFilePath)
	n.SaveNodes(TempFilePath, ch)
	n.SaveEdges(TempFilePath, ch)
	// for i := 0; i < 2; i++ {
	// 	s <- ch
	// }
	os.Rename(TempFilePath, fp)
}

func (n *Network) Save() {
	n.SaveAs(n.Folder + n.PersistingFile)
}

func (n *Network) SaveNodes(fp string, ch chan string) {
	fmt.Printf("Trying to save the nodes of network %q into file %q\n", n.Name, fp)
	// Open/Create the database
	db, err := sql.Open(n.DBDriver, fp)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	//Prepare & execute the table creation statement
	sqlStmt := `CREATE TABLE nodes (name TEXT NOT NULL primary key, kind INT)`
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
			fmt.Println("\nComitting Transaction...")
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
		log.Fatal(err)
	}
	//Prepare & execute the table creation statement
	sqlStmt := `CREATE TABLE edges (name TEXT NOT NULL primary key, kind INT, srcnode TEXT NOT NULL, dstnode TEXT NOT NULL)`
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
			fmt.Println("\nComitting Transaction...")
			tx.Commit()
		}
		i++
	}
	// ch <- "edge"
}

func (n *Network) Load() {
	filePath := n.Folder + n.PersistingFile
	n.LoadNodes(filePath)
	n.LoadEdges(filePath)
}

func (n *Network) LoadNodes(fp string) {
	fmt.Printf("Trying to load the nodes into network %q from file %q\n", n.Name, fp)
	// Open/Create the database
	db, err := sql.Open(n.DBDriver, fp)
	if err != nil {
		log.Fatal(err)
	}
	//Retrieve the data
	rows, err := db.Query("SELECT name, kind FROM nodes")
	if err != nil {
		log.Fatal(err)
	}
	i := 0
	sn := SimpleNoder{}
	for rows.Next() {
		rows.Scan(&sn.Name, &sn.Kind)
		fmt.Print("\r Adding node number ", i, " in the network.")
		n.AddNode(&sn)
		i++
	}
	fmt.Println()
	rows.Close()
}

func (n *Network) LoadEdges(fp string) {
	fmt.Printf("Trying to load the edges into network %q from file %q\n", n.Name, fp)
	// Open/Create the database
	db, err := sql.Open(n.DBDriver, fp)
	if err != nil {
		log.Fatal(err)
	}
	//Retrivee the data
	rows, err := db.Query("SELECT name, kind, srcnode, dstnode FROM edges")
	if err != nil {
		log.Fatal(err)
	}
	se := SimpleEdger{}
	i := 0
	for rows.Next() {
		rows.Scan(&se.Name, &se.Kind, &se.SrcId, &se.DstId)
		fmt.Print("\r Adding edge number ", i, " in the network.")
		n.AddEdge(&se)
		i++
	}
	fmt.Println()
	rows.Close()
}

//-----------------------
//SECTION 3: NON-COMPLEX NETWORK OPERATIONS
func (n *Network) Search(namePattern string, mode string) {
	switch mode {
	case "edge":
		matchingEdges := n.SearchEdges(namePattern)
		fmt.Printf("Found %d edges matching the pattern %s. Here they are:\n", len(matchingEdges), namePattern)
		for i, e := range matchingEdges {
			fmt.Printf("Edge number %d:\n %+v\n", i, *e)
		}
	case "node":
		matchingNodes := n.SearchNodes(namePattern)
		fmt.Printf("Found %d nodes matching the pattern %s. Here they are:\n", len(matchingNodes), namePattern)
		for i, e := range matchingNodes {
			fmt.Printf("Node number %d:\n %+v\n", i, *e)
		}
	default:
		fmt.Println("SEARCH_ERROR: Mode argument of search not recognized")
	}
}
func (n *Network) SearchEdges(namePattern string) []*Edge {
	re := regexp.MustCompile(namePattern)
	matchingEdges := []*Edge{}
	for _, edge := range n.Edges {
		// fmt.Println("French people make code the same way they make love:
		// quick, sloppy, and full of bugs")//Thanks Adam
		if re.MatchString(edge.Name) {
			matchingEdges = append(matchingEdges, edge)
		}
	}
	return matchingEdges

}
func (n *Network) SearchNodes(namePattern string) []*Node {
	re := regexp.MustCompile(namePattern)
	matchingNodes := []*Node{}
	for _, node := range n.Nodes {
		if re.MatchString(node.Name) {
			matchingNodes = append(matchingNodes, node)
		}
	}
	return matchingNodes

}

//Checking subnetworks
func (n *Network) CheckSubNetwork(subNetwork map[string]bool) bool {
	for nName, _ := range subNetwork {
		for _, ne := range n.Nodes[nName].Edges {
			if !subNetwork[ne.ToNode.Name] {
				return false
			}
		}
	}
	return true
}
func (n *Network) CheckSubNetworkNodes(subNetwork map[*Node]bool) bool {
	for nn, _ := range subNetwork {
		for _, ne := range nn.Edges {
			if !subNetwork[ne.ToNode] {
				return false
			}
		}
	}
	return true
}

//-----------------------
//SECTION 4: SUBNETWORK DETECTION
//A.
//"Depth-first": this will not allow a detection of a short wide subnetwork when limited in steps.
//Could be useful for greedy subnetwork search. The most efficient in term of checks.
func DetectSubsVertical(startNode *Node, maxN int) (map[*Node]bool, bool) {
	subNetwork := make(map[*Node]bool)
	subNetwork[startNode] = true
	for _, e := range startNode.Edges {
		if !detectSubsVertical(e.ToNode, maxN-1, subNetwork) {
			// fmt.Println("") //DEBUG
			return subNetwork, false
		}
	}
	// fmt.Println("") //DEBUG
	return subNetwork, true
}

//Subfunction
func detectSubsVertical(startNode *Node, maxN int, subNetwork map[*Node]bool) bool {
	// fmt.Println("new detection on node", startNode, "for maxN", maxN) //DEBUG
	// fmt.Printf("%d ", maxN) //DEBUG
	if maxN == 0 { // End of the research
		return false
	}
	subNetwork[startNode] = true
	if len(startNode.Edges) == 1 { // This is a dead-end (the only edge is the one it comes from)
		return true
	}
	for _, e := range startNode.Edges {
		if !subNetwork[e.ToNode] && !detectSubsVertical(e.ToNode, maxN-1, subNetwork) {
			return false
		}
	}
	return true
}

//B.
//"Depth-first", but concurrent. Should be quite efficient.
//There is no synchronization mechanism (on purpose), so it is not maxN-reproducible.
type ComObject struct {
	cs    []chan bool
	res   *[]bool
	debug *uint64 //DEBUG
}
type ccrSubNetwork struct {
	sync.Mutex //Faster than RWMutex in our case (a lot of very small operations)
	m          map[string]bool
}

func CcrDetectSubsVertical(startNode *Node, maxN int) (map[string]bool, bool) {
	subNetwork := &ccrSubNetwork{m: make(map[string]bool)}
	subNetwork.m[startNode.Name] = true
	co := ComObject{}
	co.res = &[]bool{}
	var init uint64 = 0
	co.debug = &init
	//Prepare the communication channels
	//We need to stage the communications in order to prevent from a bottleneck freeze,
	//i.e. when a waiting routine got fed by a downstream routine, and blocks this branch.
	for i := 0; i < maxN; i++ {
		co.cs = append(co.cs, make(chan bool, 10))
	}
	for _, e := range startNode.Edges {
		// atomic.AddUint64(co.debug, 1) //DEBUG
		subNetwork.Lock()
		subNetwork.m[e.ToNode.Name] = true
		subNetwork.Unlock()
		go ccrDetectSubsVertical(e.ToNode, maxN-1, subNetwork, co)
	}
	for i := 0; i < len(startNode.Edges); i++ {
		*co.res = append(*co.res, <-co.cs[maxN-1])
	}
	// fmt.Println("") //DEBUG
	//Count the results
	nF := 0
	nT := 0
	for _, t := range *co.res {
		if t {
			nT++
		} else {
			nF++
		}
	}
	// fmt.Println(nF, "Falses", nT, "Trues")                    //DEBUG             // DEBUG
	// fmt.Println("Number of go routines launched:", *co.debug) //DEBUG
	isSub := nT > nF //Not a good criterion
	return subNetwork.m, isSub
}

//Subfunction
func ccrDetectSubsVertical(startNode *Node, maxN int, subNetwork *ccrSubNetwork, co ComObject) {
	// fmt.Println("new detection on node", startNode.Name, "for maxN", maxN) //DEBUG
	// fmt.Printf("%d ", maxN) //DEBUG
	if maxN == 0 { // End of the research, with "failure"
		// fmt.Println("Sending 1 FALSE") //DEBUG
		co.cs[maxN] <- false
		return
	}
	nNewNodes := 0
	for _, e := range startNode.Edges {
		n := e.ToNode
		subNetwork.Lock()
		isIn := subNetwork.m[n.Name]
		subNetwork.Unlock()
		if !isIn {
			subNetwork.Lock()
			subNetwork.m[n.Name] = true // Add the current node to the subnetwork
			subNetwork.Unlock()
			if len(n.Edges) != 1 { //Launch next step only if not a dead end
				nNewNodes++
				// atomic.AddUint64(co.debug, 1) //DEBUG
				go ccrDetectSubsVertical(n, maxN-1, subNetwork, co)
			}
		}
	}
	// fmt.Println("this detection on node", startNode.Name, "led to", nNewNodes, "new nodes to discover") //DEBUG
	if nNewNodes == 0 {
		// fmt.Println("[detection on node", startNode.Name, "]: Sending 1 TRUE") //DEBUG
		co.cs[maxN] <- true
	} else { //Undetermined counting!!
		// fmt.Println("[detection on node", startNode.Name, "]: Waiting for", nNewNodes, "answers") //DEBUG
		for i := 0; i < nNewNodes; i++ {
			*co.res = append(*co.res, <-co.cs[maxN-1])
		}
		// fmt.Println("[detection on node", startNode.Name, "]: Gathering answers, Sending 1 TRUE") //DEBUG
		co.cs[maxN] <- true
	}
}

//C.
//"Stepwise Wide first": potentially slow but no false negatives. Useful because MaxSteps has a meaning.
func DetectSubs(startNode *Node, maxN int) (map[*Node]bool, bool) {
	subNetwork := make(map[*Node]bool)
	//Preparation of the first step
	subNetwork[startNode] = true
	counter := 0 // DEBUG
	// nextNodes := []*Node{} // AL: Slices are a very bad idea here. NextNodei will end up with multiple checks of the same guy...
	nextNodes := make(map[*Node]bool)
	// fmt.Printf("Wandering on node %p \n", startNode) //DEBUG
	for _, e := range startNode.Edges {
		// fmt.Printf("%p ", e.ToNode) //DEBUG
		n := e.ToNode
		subNetwork[n] = true // Add the node
		nextNodes[n] = true
		counter++
	}
	fmt.Println("")
	//Launching iteration per step of "depth"
	for i := 0; i < maxN; i++ {
		nextNodesi := make(map[*Node]bool)
		for n, _ := range nextNodes { // Iterate over the nodes we need to discover
			for _, e := range n.Edges { // Check the different edges of the node under discovery
				if n := e.ToNode; !subNetwork[n] { // If not present, add to the nodes to discover.
					subNetwork[n] = true // Add the node
					nextNodesi[n] = true
					counter++
				}
			}
		}
		if len(nextNodesi) == 0 { //No New Nodes to discover, we're done
			return subNetwork, true
		}
		nextNodes = nextNodesi
	}
	return subNetwork, false
}

//Old method with string-keyed maps as subnetwork (NEEDS CLEANING!)
func DetectSubsLegacy(startNode *Node, maxN int) (map[string]bool, bool) {
	subNetwork := make(map[string]bool)
	//Preparation of the first step
	subNetwork[startNode.Name] = true
	counter := 0 // DEBUG
	// nextNodes := []*Node{} // AL: Slices are a very bad idea here. NextNodei will end up with multiple checks of the same guy...
	nextNodes := make(map[*Node]bool)
	// fmt.Printf("Wandering on node %p \n", startNode) //DEBUG
	for _, e := range startNode.Edges {
		// fmt.Printf("%p ", e.ToNode) //DEBUG
		n := e.ToNode
		subNetwork[n.Name] = true // Add the node
		nextNodes[n] = true
		counter++
	}
	fmt.Println("")
	//Launching iteration per step of "depth"
	for i := 0; i < maxN; i++ {
		// fmt.Printf("\rIteration number %d", i+1) // DEBUG
		// fmt.Printf("\nGoing through %v nodes...\n", len(nextNodes)) // DEBUG
		nextNodesi := make(map[*Node]bool)
		for n, _ := range nextNodes { // Iterate over the nodes we need to discover
			// fmt.Printf("Wandering on node %p \n", n) //DEBUG
			for _, e := range n.Edges { // Check the different edges of the node under discovery
				// fmt.Printf("%p ", e.ToNode)             //DEBUG
				if n := e.ToNode; !subNetwork[n.Name] { // If not present, add to the nodes to discover.
					subNetwork[n.Name] = true // Add the node
					nextNodesi[n] = true
					counter++
				}
			}
			// fmt.Println("") //DEBUG
		}
		if len(nextNodesi) == 0 { //No New Nodes to discover, we're done
			// fmt.Println("")                               //DEBUG
			// fmt.Println("Performed", counter, "searches") //DEBUG
			return subNetwork, true
		}
		nextNodes = nextNodesi
	}
	// fmt.Println("") //DEBUG
	return subNetwork, false
}

//D. Crunching the whole Network now.
type Net struct {
	NodeMap     map[*Node]int
	SubNetworks map[int]map[*Node]bool
}

func NewNet() *Net {
	return &Net{make(map[*Node]int), make(map[int]map[*Node]bool)}
}

func (net *Net) Summary(w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	fmt.Fprintln(w, "Summary for the net:")
	fmt.Fprintf(w, "%d nodes in %d subnetworks.\n", len(net.NodeMap), len(net.SubNetworks))
	for iSub, nodes := range net.SubNetworks {
		fmt.Fprintf(w, "SubNetwork %5d: %10d nodes.\n", iSub, len(nodes))
	}
}

func (net *Net) AddSub(subN map[*Node]bool) {
	iSub := len(net.SubNetworks)
	net.SubNetworks[iSub] = subN
	for k, _ := range subN {
		net.NodeMap[k] = iSub
	}
}

func (net *Net) CrunchNetwork(n *Network) {
	maxNratio := 1.2
	for _, node := range n.Nodes {
		if net.NodeMap[node] > 0 {
			continue
		}
		subN, isSub := DetectSubsVertical(node, int(maxNratio*float64(len(n.Nodes))))
		if !isSub {
			panic("CRUNCHNETWORK: maximum iteration number is not enough to detect the biggest subnetwork")
		}
		net.AddSub(subN)
	}
}

//E.
//"Wanderer", is a non recursive re-writing of the DetectSubVertical to make it concurrent
//in the bigger picture.
type SimpleWanderer struct {
	Moignons   *SLifo
	SubNetwork map[*Node]bool
}

func NewSimpleWanderer() *SimpleWanderer {
	return &SimpleWanderer{&SLifo{}, make(map[*Node]bool)}
}

func (sw *SimpleWanderer) DetectSubs(startNode *Node, maxN int) (map[string]bool, bool) {
	//Initialization
	subNetwork := map[string]bool{
		startNode.Name: true,
	}
	if len(startNode.Edges) > 1 {
		for _, e := range startNode.Edges[1:] {
			subNetwork[e.ToNode.Name] = true
			sw.Moignons.Push(e.ToNode)
		}
	}
	// //DEBUG
	// fmt.Printf("Wandering on First Node %p with maxN %v :\n", startNode, maxN)
	// for _, e := range startNode.Edges {
	// 	fmt.Printf("%p ", e.ToNode)
	// }
	// fmt.Println("")
	n := startNode.Edges[0].ToNode
	currentNode := n
	subNetwork[n.Name] = true
	//Wander for maxN steps
	for i := 0; i < maxN; i++ {
		// fmt.Printf("Wandering on Node %p with stack %v :\n", currentNode, sw.Moignons) //DEBUG
		hasNext := false
		for _, e := range currentNode.Edges {
			n := e.ToNode
			// fmt.Printf("Trying Node %p with stack %v :\n", n, sw.Moignons) //DEBUG
			if !subNetwork[n.Name] {
				subNetwork[n.Name] = true
				if len(n.Edges) != 1 { // That would be a dead-end
					if hasNext {
						sw.Moignons.Push(n)
					} else {
						hasNext = true
						currentNode = n
					}
				}
			}
		}
		if !hasNext { //Go back to the most recent Moignon or exit if done
			if len(*sw.Moignons) > 0 {
				currentNode = sw.Moignons.Pop()
			} else {
				return subNetwork, true
			}
		}
	}
	return subNetwork, false
}

//Wandering function, similar to the DetectSub above, but with embedded duplicity to enable
//lightweight communication
func (sw *SimpleWanderer) Wander(startNode *Node, maxN int) (map[*Node]bool, bool) {
	//Initialization
	sw.SubNetwork[startNode] = true
	subNetworkIncrement := map[*Node]bool{
		startNode: true,
	}
	if len(startNode.Edges) > 1 {
		for _, e := range startNode.Edges[1:] {
			n := e.ToNode
			sw.SubNetwork[n] = true
			subNetworkIncrement[n] = true
			sw.Moignons.Push(e.ToNode)
		}
	}
	// Add the first Node
	n := startNode.Edges[0].ToNode
	currentNode := n
	sw.SubNetwork[n] = true
	subNetworkIncrement[n] = true
	//Wander for maxN steps
	for i := 0; i < maxN; i++ {
		// fmt.Printf("Wandering on Node %p with stack %v :\n", currentNode, sw.Moignons) //DEBUG
		hasNext := false
		for _, e := range currentNode.Edges {
			n := e.ToNode
			// fmt.Printf("Trying Node %p with stack %v :\n", n, sw.Moignons) //DEBUG
			if !sw.SubNetwork[n] {
				sw.SubNetwork[n] = true
				subNetworkIncrement[n] = true
				if len(n.Edges) != 1 { // That would be a dead-end
					if hasNext {
						sw.Moignons.Push(n)
					} else {
						hasNext = true
						currentNode = n
					}
				}
			}
		}
		if !hasNext { //Go back to the most recent Moignon or exit if done
			if len(*sw.Moignons) > 0 {
				currentNode = sw.Moignons.Pop()
			} else {
				return subNetworkIncrement, true
			}
		}
	}
	return subNetworkIncrement, false
}

type Order int

const (
	Done Order = iota
	Continue
	Break
	Merge
)

type WandererCom struct {
	cSubN     chan map[*Node]bool
	cOrder    chan Order
	cWanderer chan *SimpleWanderer
}

func NewWandererCom() *WandererCom {
	return &WandererCom{
		make(chan map[*Node]bool),
		make(chan Order),
		make(chan *SimpleWanderer),
	}
}

func (sw *SimpleWanderer) Merge(sw2 *SimpleWanderer) {
	// Merge the stack
	for len(*sw2.Moignons) > 0 {
		sw.Moignons.Push(sw2.Moignons.Pop())
	}
	//Merge the subnetworks
	for n, _ := range sw2.SubNetwork {
		sw.SubNetwork[n] = true
	}
}

func (sw *SimpleWanderer) WanderStep(startNode *Node, stepSize int, com WandererCom) {
	for {
		//Wander for stepSize
		subN, done := sw.Wander(startNode, stepSize)
		//Receive orders
		switch <-com.cOrder {
		case Continue: // Go on!
		case Break: // Stop and pass the subNetwork & the wanderer for merging
			com.cSubN <- sw.SubNetwork
			com.cWanderer <- sw
			return
		case Merge: //Merge with an other wanderer and go on
			sw.Merge(<-com.cWanderer)
		default:
			panic("WANDERSTEP: problem of communication")
		}
		if done {
			com.cOrder <- Done
			com.cSubN <- sw.SubNetwork
			return
		} else {
			//Ask for Status
			com.cOrder <- Continue //Ask for permission to continue
			com.cSubN <- subN
		}
	}
}

// func (net *Net) CcrCrunchNetwork(n *Network, maxW int, stepSize int) {
// 	//Initialize the stuff
// 	comObjects := []*WandererCom{}
// 	wanderers := []*SimpleWanderer{}
// 	i:=0
// 	for _, node := range n.Nodes { // Populate with the number of wanderers
// 		if i == maxW { // Stop at maxW
// 			break
// 		}
// 		comObjects[i] = NewWandererCom()
// 		wanderers[i] = NewSimpleWanderer()
// 		go wanderers[i].WanderStep()
// 		i++
// 		}
// 	}
// 	// Listen and dispatch.
// 	for {
// 		for i:=0; i < maxW; i++ {
// 			select
// 			net.Dispatch(comObjects[i])
// 		}
// 	}
// }
