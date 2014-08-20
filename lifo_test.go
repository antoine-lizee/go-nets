package go_nets

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

func newNodeInt(i int) *Node {
	n := &Node{}
	n.Name = strconv.Itoa(i)
	return n
}

func testLifo(l LIFO, nn []int) {
	n := newNodeInt(1)

	for i := 0; i < nn[0]; i++ {
		l.Push(n)
	}
	for k := 0; k < nn[2]; k++ {
		for i := 0; i < nn[1]; i++ {
			l.Push(n)
		}
		for i := 0; i < nn[1]; i++ {
			l.Pop()
		}
	}
}

func TestLifo(t *testing.T) {
	//Quick Check
	fmt.Println("LLifo implementation test...")
	st1 := LLifo{}
	for i := 0; i < 10; i++ {
		st1.Push(newNodeInt(i))
	}
	for st1.Len > 0 {
		fmt.Println(st1.Pop())
	}
	fmt.Println("SLifo implementation test...")
	st2 := &SLifo{}
	for i := 0; i < 10; i++ {
		st2.Push(newNodeInt(i))
	}
	for len(*st2) > 0 {
		fmt.Println(st2.Pop())
	}
	//Benchmark
	nn := []int{1000000, 200000, 100}
	t0 := time.Now()
	testLifo(&LLifo{}, nn)
	fmt.Println("Done the stresstesting for LLifo in", time.Now().Sub(t0))
	fmt.Println("for the folloing parameters:", nn)
	t0 = time.Now()
	testLifo(&SLifo{}, nn)
	fmt.Println("Done the stresstesting for SLifo in", time.Now().Sub(t0))
	fmt.Println("for the folloing parameters:", nn)
}
