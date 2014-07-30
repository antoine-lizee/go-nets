package go_nets

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"testing"
	"time"
)

var testFolder = "_test/"

func TestSave(t *testing.T) {
	fmt.Println("### TESTING the saving option")
	Parser := XmlParser{
		FileDir:  testFolder,
		FileName: "UMtest2.xml", //UM20140215_5 UMtest2
	}

	cs := make(chan Filing)
	go Parser.Parse(cs, ioutil.Discard)
	network := NewNetwork("Test", ioutil.Discard, testFolder, "")

	i := 0
	for p := range cs {
		// p := <-cs
		i++
		fmt.Printf("\r Filing number %d (id = %d)loaded.", i, p.OriginalFileNumber)
		// fmt.Println("Trying to add it to the network")
		network.AddDispatcher(&p)
	}
	fmt.Println("")
	network.Summary(os.Stdout)

	fmt.Println("Saving network")
	t0 := time.Now()
	network.Save()
	fmt.Printf("\n Successfully saved the network in %v \n", time.Now().Sub(t0))
	fmt.Println("### ---------------")
}

func TestPipeline(t *testing.T) {
	fmt.Println("### TESTING the pipeline")
	fmt.Println("Total Number of cores:", runtime.NumCPU())
	nCores := 2
	fmt.Println("Enabling parallelisation with", nCores, "cores")
	runtime.GOMAXPROCS(nCores)

	//Preparing files for logging.
	fi, errOs := os.Create("_test/NetworkTest.log")
	fi2, errOs2 := os.Create("_test/ParserTest.log")
	_ = fi2
	if errOs != nil {
		panic(errOs) //TODO change it to t.Error
	}
	if errOs2 != nil {
		panic(errOs2) //TODO change it to t.Error
	}
	defer func() {
		if errOs = fi.Close(); errOs != nil {
			panic(errOs)
		}
		if errOs = fi2.Close(); errOs != nil {
			panic(errOs)
		}
	}()

	//Launching the Parser
	Parser := XmlParser{
		FileDir:  testFolder,    ///media/FD/MISSIONS/ALEX/UM20140215_X/",
		FileName: "UMtest2.xml", //"UMtest2.xml" UM20140215_5
	}
	cs := make(chan Filing)
	go Parser.Parse(cs, fi)
	network := NewNetwork("Test", fi, testFolder, "")
	//Fill the network by adding the filings sent over the channel
	i := 0
	for p := range cs {
		// p := <-cs
		i++
		fmt.Printf("\r Filing number %d (id = %d)loaded.", i, p.OriginalFileNumber)
		//Dispatcher output bloc
		// log.SetOutput(fi)
		// log.Printf("Filing : \n %# v \n", pretty.Formatter(p))
		// ShowOutputs(&p, ioutil.Discard)
		//Adding the filing to the network
		network.AddDispatcher(&p)
		// cs <- Filing{} // Only when using the ParserVerbose()
	}
	fmt.Println("")
	network.Summary(os.Stdout)

	//Saving the network
	fmt.Println("Saving network")
	t0 := time.Now()
	network.Save()
	fmt.Printf("\n Successfully saved the network in %v \n", time.Now().Sub(t0))
	fmt.Println("### ---------------\n")

	//Loading the network
	fmt.Println("Loading network")
	t0 = time.Now()
	network2 := NewNetwork("Test2", fi, testFolder, "")
	network2.Load()
	fmt.Printf("\n Successfully loaded the network in %v \n", time.Now().Sub(t0))
	network2.Summary(os.Stdout)
	fmt.Println("### ---------------\n")

	//Comparing the networks the network
	fmt.Println("Comparing networks")
	t0 = time.Now()
	network.Compare(network2)
	fmt.Printf("\n Successfully compared the networks in %v \n", time.Now().Sub(t0))
	fmt.Println("### ---------------\n")

	//Looking into the networs
	namePattern := "desert.*car"
	fmt.Printf("Looking for matches of %q\n", namePattern)
	t0 = time.Now()
	fmt.Println("In network", network.Name)
	network.Search(namePattern, "edge")
	fmt.Println("In network", network2.Name)
	network2.Search(namePattern, "edge")
	fmt.Printf("\n Successfully searched the networks in %v \n", time.Now().Sub(t0))
	fmt.Println("### ---------------\n")
}

func TestLoad(t *testing.T) {

	//Loading the network
	fmt.Println("Loading network")
	t0 := time.Now()
	fi, _ := os.Create("_test/TestLoad.log")
	network2 := NewNetwork("Test2", fi, testFolder, "")
	network2.Load()
	fmt.Printf("\n Successfully loaded the network in %v \n", time.Now().Sub(t0))
	network2.Summary(os.Stdout)
	fmt.Println("### ---------------\n")

}

var notString = map[bool]string{
	true:  "",
	false: "not",
}

type MajBool bool

func (b MajBool) String() string {
	if b {
		return "TRUE"
	} else {
		return "FALSE"
	}
}

type SubsResults struct {
	method        func(*Node, int) (map[string]bool, bool)
	methodName    string
	maxN          int
	isSub, isSub2 bool
	t1, t2        time.Duration
	sizeSub       int
}

func (sb SubsResults) testSub(sNode *Node, network *Network) SubsResults {
	t0 := time.Now()
	subN, isSub := sb.method(sNode, sb.maxN)
	sb.t1 = time.Now().Sub(t0)
	t0 = time.Now()
	isSub2 := network.CheckSubNetwork(subN)
	sb.t2 = time.Now().Sub(t0)
	sb.isSub, sb.isSub2 = isSub, isSub2
	sb.sizeSub = len(subN)
	// //DEBUG Show the subNEtwork ?
	// for nName, _ := range subN {
	// 	fmt.Printf("%p ", network.Nodes[nName])
	// }
	// fmt.Println("")
	return sb
}

func (sb SubsResults) Print(debug bool) {
	if debug {
		fmt.Println("-- Method #1", sb.methodName, " with maxN =", sb.maxN)
		// fmt.Println("Found a subNetwork with", len(subN), "nodes, that is", notString[isSub], "supposed to be complete in", time.Now().Sub(t0))
		fmt.Printf("NODES: %d - (%v)\n", sb.sizeSub, sb.t1)
		// fmt.Println("Subnetwork verified: it is", notString[isSub2], "complete. (", time.Now().Sub(t0), ")")
		fmt.Printf("COMPLETE:%s, (%v) - (%v)\n", MajBool(sb.isSub2), sb.isSub, sb.t2)
	} else {
		fmt.Printf("%30.30s:\t %15.15s / %15.15s \t  %8d \t %s - (%v)\n",
			sb.methodName,
			sb.t1, sb.t2,
			sb.sizeSub,
			MajBool(sb.isSub2), sb.isSub)
	}

}

func TestSubs(t *testing.T) {
	fmt.Println("Total Number of cores:", runtime.NumCPU())
	nCores := 4
	fmt.Println("Enabling parallelisation with", nCores, "cores")
	runtime.GOMAXPROCS(nCores)

	//Loading the network
	fmt.Println("Loading network")
	t0 := time.Now()
	network := NewNetwork("Test2", nil, testFolder, "Network1.sqlite")
	network.Load()
	fmt.Printf("\n Successfully loaded the network in %v \n", time.Now().Sub(t0))
	network.Summary(os.Stdout)
	fmt.Println("### ---------------\n")

	//Searching for some nodes, and executing the subnetwork research from this node.
	sNodes := network.SearchNodes("wellsfargobank") //wells wellsfargobanna$
	maxN := 20
	maxNodes := 20
	for i, sNode := range sNodes {
		if i == maxNodes { //Limit the number of search
			break
		}
		// //Display only big subnetworks
		// if _, isSub := DetectSubs(sNode, 5); isSub {
		// 	continue
		// }

		fmt.Printf("\n## Selecting node %.50s (...) and searching for subnetwork linked to it...\n", sNode.Name)
		debug := false
		//Method #1
		M1 := SubsResults{method: DetectSubs, methodName: "Width-First", maxN: maxN}
		M1.testSub(sNode, &network).Print(debug)
		//Method #2
		M2 := SubsResults{method: DetectSubsVertical, methodName: "Recursive Depth-First", maxN: 10000 * maxN}
		M2.testSub(sNode, &network).Print(debug)
		//Method #3
		M3 := SubsResults{method: CcrDetectSubsVertical, methodName: "Concurrent Depth-First", maxN: 3 * maxN}
		M3.testSub(sNode, &network).Print(debug)
		//Method #4
		sw := SimpleWanderer{&SLifo{}}
		M4 := SubsResults{method: sw.Wander, methodName: "Iterative Depth-First", maxN: 10000 * maxN}
		M4.testSub(sNode, &network).Print(debug)

	}

}
