package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/antoine-lizee/go-nets"
)

////////////
//SECTION 1
//Command-line parsing

type FileNames []string //Flag type

//Define the interface functions
func (fn *FileNames) String() string {
	return fmt.Sprint(*fn)
}

func (fn *FileNames) Set(value string) error {
	for _, dt := range strings.Split(value, ",") {
		*fn = append(*fn, dt)
	}
	return nil
}

var (
	loadArg      = flag.String("loadFile", "", "Provide the name of the file to load the network from")
	parsePathArg = flag.String("parsePath", "/media/FD/MISSIONS/ALEX/UM20140215_X/", "Provide the name of the parsed file")
	parseArgs    = FileNames{}
	nameArg      = flag.String("name", "Total1", "Provide the name of the network")
)

func init() {
	flag.Var(&parseArgs, "parse", "Specify a comma separated list of file names")
}

////////////
//SECTION 2
//Utilities
func merge(cs [](chan go_nets.Filing), out chan<- go_nets.Filing) {
	done := make(chan int)
	for ic, c := range cs {
		c := c
		go func() {
			for p := range c {
				_ = ic
				// fmt.Println("Merging in", p, "from channel", ic)
				out <- p
			}
			// fmt.Println("Sending done for channel", ic)
			done <- 1
		}()
	}
	for ic := range cs {
		_ = ic
		// fmt.Println("listening to channel done for index", ic)
		_ = <-done
	}
	close(out)
}

////////////
//SECTION 3
//Subsections
func Save(network *go_nets.Network) {
	fmt.Println("Saving network")
	t0 := time.Now()
	network.Save()
	fmt.Printf("\n Successfully saved the network in %v \n", time.Now().Sub(t0))
	fmt.Println("### ---------------")
}

func Parse(fileNames []string, network *go_nets.Network) {
	//Prepare the parsers and channels
	parsers := []go_nets.XmlParser{}
	out := make(chan go_nets.Filing)
	cs := []chan go_nets.Filing{}
	for _, fileName := range fileNames {
		parsers = append(parsers, go_nets.XmlParser{
			FileDir:  *parsePathArg,
			FileName: fileName, //"UM20140215_" + strconv.Itoa(i),
		})
		cs = append(cs, make(chan go_nets.Filing))
	}
	//Launch the parsers
	for i, parser := range parsers {
		go parser.Parse(cs[i], ioutil.Discard)
	}
	//Launch fan in
	go merge(cs, out)

	//Consume the filings as Dispatchers
	i := 0
	for p := range out {
		i++
		fmt.Printf("\r Filing number %d (id = %d)loaded.", i, p.OriginalFileNumber)
		// fmt.Println("Trying to add it to the network")
		network.AddDispatcher(&p)
	}
	fmt.Println("")
	network.Summary(os.Stdout)
}

func Load(n *go_nets.Network, fp string) {
	if fp == "" {
		n.Load()
	} else {
		n.LoadFrom(fp)
	}

}

////////////
//SECTION 4
//main
func main() {
	// Parse the command line arguments
	flag.Parse()
	doParse := false
	if (len(parseArgs) > 0) && (*loadArg == "") {
		doParse = true
	}

	//Create Network
	network := go_nets.NewNetwork("Total1", nil, "Networks/")

	//Feed the network
	if doParse { //Parse?
		Parse(parseArgs, &network)
	} else { //Load?
		Load(&network, *loadArg)
	}

	//Save the network
	Save(&network)

}
