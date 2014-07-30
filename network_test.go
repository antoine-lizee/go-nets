package go_nets

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestSave(t *testing.T) {
	fmt.Println("### TESTING the saving option")
	Parser := XmlParser{
		FileDir:  "/media/FD/MISSIONS/ALEX/UM20140215_X/",
		FileName: "UMtest2.xml",
	}

	cs := make(chan Filing)
	go Parser.Parse(cs, ioutil.Discard)
	network := NewNetwork("Test", ioutil.Discard)

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
	fmt.Println("\n### ---------------")
}

func TestPipeline(t *testing.T) {
	fmt.Println("### TESTING the pipeline")
	fmt.Println("Total Number of cores:", runtime.NumCPU())
	nCores := 2
	fmt.Println("Enabling parallelisation with", nCores, "cores")
	runtime.GOMAXPROCS(nCores)

	//Preparing files for logging.
	fi, errOs := os.Create("NetworkTest.log")
	fi2, errOs2 := os.Create("ParserTest.log")
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
		FileDir:  "/media/FD/MISSIONS/ALEX/UM20140215_X/",
		FileName: "UM20140215_5.xml", //"UMtest2.xml" UM20140215_5
	}
	cs := make(chan Filing)
	go Parser.Parse(cs, fi)
	network := NewNetwork("Test", fi)
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
	fmt.Println("\n### ---------------")
}
