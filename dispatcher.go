package go_nets

import (
	"log"
	"regexp"
	"strconv"
	"strings"
)

// Bloc Subfunctions
func Atomize(s string) string {
	re := regexp.MustCompile(",? +(inc|l\\.?l\\.?c|as representative|p.c.|co|as agent).?")
	re2 := regexp.MustCompile(" |\\.|\\,")
	return re2.ReplaceAllString(re.ReplaceAllString(strings.ToLower(s), ""), "")
}

func (i *IndividualName) String() string {
	return i.FirstName + "." + i.LastName
}

// Define the agent as an Noder
func (a *Agent) GetIdentifier() string {
	if a.GetKind() == Emitter {
		return Atomize(a.OrganizationName)
	} else {
		return strings.ToLower(a.IndividualName.String() + a.PostalCode)
	}
}

func (a *Agent) GetKind() NodeKind {
	if a.OrganizationName != "" {
		return Emitter
	} else {
		return Receiver
	}

}

func (a *Agent) GetData() AttrGetter {
	return a
}

func (a *Agent) UpdateData(AttrGetter) AttrGetter {
	// Create a more sophisticated attribute getter that can hold more data than that. Then, update with an additional address when there is some for the BUSINESSES only.
	return a.GetData()
}

// Create a new Edger from a Filing
type FilingEdger struct {
	srcId, dstId string
	kind         EdgeKind
	filing       *Filing
}

func (f *Filing) NewFilingEdger(kind EdgeKind, srcId string, dstId string) FilingEdger {
	if dstId < srcId {
		temp := dstId
		dstId = srcId
		srcId = temp
	}
	return FilingEdger{srcId, dstId, kind, f}
}

func (fe FilingEdger) GetIdentifier() string {
	return fe.GetSrcId() + "_" + strconv.Itoa(fe.filing.OriginalFileNumber) + "_" + fe.GetDstId()
}

func (fe FilingEdger) GetKind() EdgeKind {
	return fe.kind
}

func (fe FilingEdger) GetSrcId() string {
	return fe.srcId
}

func (fe FilingEdger) GetDstId() string {
	return fe.dstId
}

func (fe FilingEdger) GetData() AttrGetter {
	return struct {
		FileNumber, OriginalFileNumber    int
		FileDate, OriginalFileDate        string
		AmendmentType, FilingType, Method string
	}{fe.filing.FileNumber, fe.filing.OriginalFileNumber,
		fe.filing.FileDate, fe.filing.OriginalFileDate,
		fe.filing.Amendment.Attr, fe.filing.FilingType.Attr, fe.filing.Method.Attr}
}

// Define Filing as a Dispatcher
func (f *Filing) Dispatch(logger *log.Logger) ([]Noder, []Edger) {
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
			f.Securers = DeleteAgent(f.Securers, i)
			logger.Println("DISPATCHER: removing securer node", s.GetIdentifier(), "because of duplication.")
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
	return noders, edgers
}
