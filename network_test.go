package go_nets

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"math"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"code.google.com/p/go.text/encoding/charmap"

	"github.com/gonum/matrix/mat64"
)

var testFolder = "_test/"

func initiateMultiCore(nCores int) {
	fmt.Println("Total Number of cores:", runtime.NumCPU())
	fmt.Println("Enabling parallelisation with", nCores, "cores")
	runtime.GOMAXPROCS(nCores)
}

func loadNetwork(name string, writer io.Writer) Network {
	fmt.Println("Loading network")
	t0 := time.Now()
	network := NewNetwork(name, writer, testFolder)
	network.Load()
	fmt.Printf("\n Successfully loaded the network in %v \n", time.Now().Sub(t0))
	network.Summary(os.Stdout)
	fmt.Println("### ---------------\n")
	return network
}

func TestSave(t *testing.T) {
	fmt.Println("### TESTING the saving option")
	Parser := XmlParser{
		FileDir:  testFolder,
		FileName: "UMtest.xml", //UM20140215_5 UMtest2
		Encoding: charmap.Windows1252,
	}

	cs := make(chan Filing)
	go Parser.Parse(cs, ioutil.Discard)
	network := NewNetwork("TestSmall", ioutil.Discard, testFolder)

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
	initiateMultiCore(2)

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
		Encoding: charmap.Windows1252,
	}
	cs := make(chan Filing)
	go Parser.Parse(cs, fi)
	network := NewNetwork("Test", fi, testFolder)
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
	network2 := NewNetwork("Test2", fi, testFolder)
	network2.Load()
	fmt.Printf("\n Successfully loaded the network in %v \n", time.Now().Sub(t0))
	network2.Summary(os.Stdout)
	fmt.Println("### ---------------\n")

	//Comparing the networks
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
	fi, _ := os.Create("_test/TestLoad.log")
	_ = loadNetwork("Test", fi)
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
	methodNodes   func(*Node, int) (map[*Node]bool, bool)
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
	// //DEBUG Show the subNetwork ?
	// for nName, _ := range subN {
	// 	fmt.Printf("%p ", network.Nodes[nName])
	// }
	// fmt.Println("")
	return sb
}

func (sb SubsResults) testSubNode(sNode *Node, network *Network) SubsResults {
	t0 := time.Now()
	subN, isSub := sb.methodNodes(sNode, sb.maxN)
	sb.t1 = time.Now().Sub(t0)
	t0 = time.Now()
	isSub2 := network.CheckSubNetworkNodes(subN)
	sb.t2 = time.Now().Sub(t0)
	sb.isSub, sb.isSub2 = isSub, isSub2
	sb.sizeSub = len(subN)
	return sb
}

func (sb SubsResults) Print(debug bool) {
	if debug {
		fmt.Println("-- Method", sb.methodName, " with maxN =", sb.maxN)
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
	initiateMultiCore(4)

	//Loading the network
	network := loadNetwork("TestSmall", nil)

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
		M1 := SubsResults{method: DetectSubsLegacy, methodName: "Width-First", maxN: maxN}
		M1.testSub(sNode, &network).Print(debug)
		//Method #1BIS
		M12 := SubsResults{methodNodes: DetectSubs, methodName: "Width-First, Node keyed map", maxN: maxN}
		M12.testSubNode(sNode, &network).Print(debug)
		//Method #2
		M2 := SubsResults{methodNodes: DetectSubsVertical, methodName: "Recursive Depth-First", maxN: 10000 * maxN}
		M2.testSubNode(sNode, &network).Print(debug)
		//Method #3
		M3 := SubsResults{method: CcrDetectSubsVertical, methodName: "Concurrent Depth-First", maxN: 3 * maxN}
		M3.testSub(sNode, &network).Print(debug)
		//Method #4
		sw := SimpleWanderer{&SLifo{}, nil}
		M4 := SubsResults{method: sw.DetectSubs, methodName: "Iterative Depth-First", maxN: 10000 * maxN}
		M4.testSub(sNode, &network).Print(debug)
		//Method #5
		sw = SimpleWanderer{&SLifo{}, make(map[*Node]bool)}
		M5 := SubsResults{methodNodes: sw.Wander, methodName: "Duplicating Iterative Depth-First", maxN: 10000 * maxN}
		M5.testSubNode(sNode, &network).Print(debug)
		//

	}
}

func TestCrunching(t *testing.T) {
	initiateMultiCore(4)

	//Loading the network
	network := loadNetwork("TestSmall", nil)

	//Crunch the network:
	net := NewNet()
	t0 := time.Now()
	net.CrunchNetwork(&network)
	d1 := time.Now().Sub(t0)
	fmt.Println("Crunched the network in", d1, "- Summary:")
	net.Summary(nil)
}

type myMap map[*Node]float32

type myStore struct {
	k string
	v float32
}

func (pi myMap) summary(nTop int) {
	maxInd := 1 << uint(math.Floor(math.Log2(float64(nTop))))
	elts := make([]myStore, maxInd+1)
	for k, v := range pi {
		if v > elts[maxInd].v {
			if v > elts[0].v {
				elts = append([]myStore{myStore{
					k.Name,
					v,
				}}, elts[0:len(elts)-1]...)
				// fmt.Printf("111 - %# v \n", pretty.Formatter(elts)) // DEBUG
			} else {
				i1 := 0
				i2 := maxInd
				for i2 > i1+1 {
					iMid := i1 + (i2-i1)/2
					if v < elts[iMid].v {
						i1 = iMid
					} else {
						i2 = iMid
					}
				}
				elts = append(elts[0:i2],
					append([]myStore{myStore{
						k.Name,
						v,
					}},
						elts[i2:len(elts)-1]...)...)
				// fmt.Printf("222 %d - \n %# v \n", i2, pretty.Formatter(elts)) //DEBUG
			}
		}
	}
	for i, e := range elts {
		fmt.Printf("%3d  -  %20.20s : %.3e \n", i, e.k, e.v)
	}

}

func ShowMat64Mat(M *mat64.Dense) {
	showImage(createImageFromMat(M))
}

func DumpMat64Mat(M *mat64.Dense, fileName string) {
	dumpImage(createImageFromMat(M), fileName+".png")
	dumpVec(M.RawMatrix().Data, fileName+".txt")
}

func createImageFromMat(M *mat64.Dense) image.Image {
	dx, dy := M.Dims()
	m := image.NewGray(image.Rect(0, 0, dx, dy))
	for y := 0; y < dy; y++ {
		for x := 0; x < dx; x++ {
			// v := data[y][x]
			i := y*m.Stride + x
			m.Pix[i] = uint8(M.At(x, y) * 255 * 2)
		}
	}
	return m
}

func showImage(m image.Image) {
	var buf bytes.Buffer
	err := png.Encode(&buf, m)
	if err != nil {
		panic(err)
	}
	enc := base64.StdEncoding.EncodeToString(buf.Bytes())
	fmt.Println("IMAGE:" + enc)
}

func dumpImage(m image.Image, fileName string) {
	fi, _ := os.Create(fileName)
	defer fi.Close()
	err := png.Encode(fi, m)
	if err != nil {
		panic(err)
	}
}

func dumpVec(data []float64, fileName string) {
	fi, _ := os.Create(fileName)
	defer fi.Close()
	buf := bufio.NewWriter(fi)
	// for _, f := range data {
	// 		fmt.Fprintf(buf, "%.8f", f)
	// }
	fmt.Fprintln(buf, strings.Trim(fmt.Sprintf("%.8f ", data), "[ ]"))
}

func isMat64Symmetric(M *mat64.Dense) bool {
	r, c := M.Dims()
	if r != c {
		return false
	}
	for i := 0; i < r; i++ {
		for j := 0; j < i; j++ {
			if M.At(i, j) != M.At(j, i) {
				return false
			}
		}
	}
	return true
}

func TestAMatrix(t *testing.T) {
	// Loading the network
	network := loadNetwork("TestSmall", nil)
	//Get the A Matrix and it's transposed version
	A, _ := network.GetAMatrix()
	fmt.Println("Symmetric ?", isMat64Symmetric(A))
	//Test for equality
	ShowMat64Mat(A)
	A.Mul(A, A)
	ShowMat64Mat(A)
	A.Mul(A, A)
	ShowMat64Mat(A)
}

func TestPageRank(t *testing.T) {
	// initiateMultiCore(4)

	//Loading the network
	network := loadNetwork("TestSmall", nil)

	//Running pagerank in two different ways
	fmt.Println("Method 1 - Random Walks")
	t0 := time.Now()
	pi := network.PageRankRW(1, 1e5, nil)
	t1 := time.Now().Sub(t0)
	fmt.Println("Summary:")
	myMap(pi).summary(10)
	fmt.Println("Done in", t1)

	fmt.Println("Method 2 s- determinist")
	t0 = time.Now()
	pi = network.PageRankSymmetricRegular()
	t1 = time.Now().Sub(t0)
	fmt.Println("Summary:")
	myMap(pi).summary(10)
	fmt.Println("Done in", t1)

	fmt.Println("Method 3 - Matrix")
	t0 = time.Now()
	pi = network.PageRankMatrix()
	t1 = time.Now().Sub(t0)
	fmt.Println("Summary:")
	myMap(pi).summary(10)
	fmt.Println("Done in", t1)

}
