//Super simple implementation of a non-concurrent safe LIFO stack for the wanderer.
package go-nets

type LIFO interface {
	Push(*Node)
	Pop() *Node
}

type LLifo struct {
	Len int
	Top *Element
}

type Element struct {
	Node *Node
	Next *Element
}

func (l *LLifo) Push(n *Node) {
	l.Len++
	l.Top = &Element{n, l.Top}
}

func (l *LLifo) Pop() *Node {
	n := l.Top.Node
	l.Top = l.Top.Next
	l.Len--
	return n
}

type SLifo []*Node

func (l *SLifo) Push(n *Node) {
	*l = append(*l, n)
}

func (l *SLifo) Pop() *Node {
	var n *Node
	n, (*l) = (*l)[len(*l)-1], (*l)[:len(*l)-1]
	return n
}
