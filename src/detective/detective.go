package main

import (
  "encoding/json"
  "flag"
  "fmt"
  "strings"
  "sync/atomic"

  "github.com/golang/glog"
)

type UniqueID int32

func (c *UniqueID) Get() int32 {
  for {
    val := atomic.LoadInt32((*int32)(c))
    if atomic.CompareAndSwapInt32((*int32)(c), val, val+1) {
      return val
    }
  }
}

/*
 * The index map is left as array of indices for an event, to leave future
 * scope for repeated events, thouh the spec has to be more clear and well
 * defined to further support that.
 */
type EventIndexMap map[string][]int

type EventSequence struct {
  id            int32          // give an id to each sequence
  original      []string       // original event sequence..
  sequence      []string       // the event sequence, changes as it gets merged
  mergeDoneMap  map[int32]bool // Indicates which other sequences it was merged
  indexMap      EventIndexMap  // we keep recalculating this as we eapnad/merge.
  transitionSeq [][]string     // for debug/trace, not used yet
}

type EventSequenceMap map[int32]*EventSequence

type DetectiveCase struct {
  eventSeqIdSeed *UniqueID
  eventSeqs      []*EventSequence  // sequences from all.
  eventSeqMap    *EventSequenceMap // a quick id to sequence access/lookup map.
  evStrInput     [][]string        // reference to input given in constructor
}

type MergeStatus int

const (
  MergeUnknown = iota
  MergeComplete
  MergePartial
  MergeNotPossible
)

func (status MergeStatus) String() string {
  description := [...]string{
    "Unknown",
    "Merge is possible",
    "Partial merge is possible",
    "No merge is possible",
  }

  if status < MergeUnknown || status > MergeNotPossible {
    status = MergeUnknown
  }

  return description[status]
}

func newEventIndexMap() EventIndexMap {
  return make(map[string][]int)
}

func NewEventSequenceMapOfLen(len int) EventSequenceMap {
  return make(map[int32]*EventSequence, len)
}

func NewEventSequenceMap() EventSequenceMap {
  return NewEventSequenceMapOfLen(0)
}

// sanitize can do some preprocessing like converting case etc. TODO
func (es *EventSequence) sanitize(str string) string {
  return str // can add sanitization later. TODO
}

func NewEventSequence(eventStrs []string, eventSeqIdSeed *UniqueID) *EventSequence {
  es := &EventSequence{
    id:           eventSeqIdSeed.Get(),
    mergeDoneMap: make(map[int32]bool),
    sequence:     make([]string, 0),
    indexMap:     newEventIndexMap(),
  }

  for kk, ev := range eventStrs {
    sEv := es.sanitize(ev)
    es.sequence = append(es.sequence, sEv)
    es.indexMap[sEv] = append(es.indexMap[sEv], kk)
  }

  // Keep track of the original sequence(sanitized).
  es.original = es.sequence
  glog.V(2).Infof("es - %+v, len of seq - %d\n", es, len(es.sequence))
  return es
}

func NewDetectiveCase(evStrSets [][]string) *DetectiveCase {
  evSeqIdSeed := UniqueID(0)
  evSeqs, evSeqMap := GenerateEventSequencesAndMap(evStrSets, &evSeqIdSeed)
  return &DetectiveCase{
    eventSeqIdSeed: &evSeqIdSeed,
    eventSeqs:      evSeqs,
    eventSeqMap:    evSeqMap,
    evStrInput:     evStrSets,
  }
}

func (es *EventSequence) RegenerateIndexMap() {
  iMap := newEventIndexMap()
  for kk, event := range es.sequence {
    iMap[event] = append(iMap[event], kk)
  }
  // TODO: debug log old eventmap, new event map
  es.indexMap = iMap
}

func GenerateEventSequencesAndMap(evStrSets [][]string,
  evSeqIdSeed *UniqueID) ([]*EventSequence, *EventSequenceMap) {

  len := len(evStrSets)
  evSeqMap := NewEventSequenceMapOfLen(len)
  evSeqs := make([]*EventSequence, len)
  for ii, evStrs := range evStrSets {
    evSeq := NewEventSequence(evStrs, evSeqIdSeed)
    evSeqs[ii] = evSeq
    evSeqMap[evSeq.id] = evSeq
  }
  return evSeqs, &evSeqMap
}

func GenerateEventSequenceMap(evSeqs []*EventSequence) *EventSequenceMap {
  evSeqMap := NewEventSequenceMapOfLen(len(evSeqs))
  for _, evSeq := range evSeqs {
    evSeqMap[evSeq.id] = evSeq
  }
  return &evSeqMap
}

// Using given "esB", get the expanded event strings for
// "this EventSequence - es"
func (es *EventSequence) GetExpandedSequence(esB *EventSequence) []string {
  esA := es // let's name this as esA
  lenA := len(esA.sequence)
  lenB := len(esB.sequence)

  newSeqA := make([]string, 0, lenA)
  newSeqA = append(newSeqA, esA.sequence...) /* TODO can do a lazy copy */

  // Keep track how many prior-expansions taken place needed for right offset
  priorCount := 0
  suffixCount := 0

  glog.V(2).Infof("esA - %+v \n", esA)
  glog.V(2).Infof("esB - %+v \n", esB)
  glog.V(2).Infof("lenA - %d, lenB - %d\n", lenA, lenB)

  for ii, event := range esA.sequence {
    // check if the event is there in the second.
    // We track items in pairs, so in left to right scan, the first one
    // is left and the second one is right
    leftIndicesB, found := esB.indexMap[event]
    glog.V(2).Infof("0. ii - %d, event - %v, found - %v, leftIndices - %+v, "+
      "newSeqA - %+v \n", ii, event, found, leftIndicesB, newSeqA)

    if leftIndicesB != nil {
      // First the special cases.., and this is not about the pair mentioned
      // above.

      // Handle first Event: seqA = "A..", seqB = "xyzA...", newSeqA = "xyzA.."
      if ii == 0 && leftIndicesB[0] != 0 {
        glog.V(2).Infof("1. ii - %d, leftIndices - %+v \n", ii, leftIndicesB)
        tmp := make([]string, 0)
        tmp = append(tmp, esB.sequence[0:leftIndicesB[0]]...)
        newSeqA = append(tmp, newSeqA...)
        priorCount += leftIndicesB[0]
        glog.V(2).Infof("1.1 newSeqA - %+v \n", newSeqA)
      }

      // Handle last Event: seqA = "..A", seqB = "...Axyz", newSeqA = "..Axyz"
      if ii == (lenA-1) && leftIndicesB[0] < (lenB-1) {
        glog.V(2).Infof("2. ii - %d, leftIndices - %+v \n", ii, leftIndicesB)

        newSeqA = append(newSeqA, esB.sequence[leftIndicesB[0]+1:]...)

        glog.V(2).Infof("2.1 newSeqA - %+v \n", newSeqA)
        suffixCount += (lenB - leftIndicesB[0])
      }

      // continue with the pair case...
      // Check if the next element is a match, for the non-terminal event
      // if this is the last event, no more events next
      if ii == (lenA - 1) {
        break
      }
      glog.V(2).Infof("3. ii - %d, leftIndices - %+v \n", ii, leftIndicesB)

      rightIndicesB, found := esB.indexMap[esA.sequence[ii+1]]
      glog.V(2).Infof("4. ii - %d, found - %v, rightIndices - %+v \n", ii, found,
        rightIndicesB)
      if rightIndicesB != nil {
        lenR := len(rightIndicesB)
        if rightIndicesB[lenR-1] > leftIndicesB[0]-1 {
          // we have  seqA = "..123AB...", seqB = "..lmnAxByzB.."
          // seqA expands to "..123AxByzB..."
          // this gets complicated with repetition; in the absence
          // of a clear spec, and given the guidance to get the largest
          // sequence, the above is implemented.
          tmp := make([]string, 0)
          tmp = append(tmp, newSeqA[:ii+priorCount]...)
          tmp = append(tmp,
            esB.sequence[leftIndicesB[0]:rightIndicesB[lenR-1]]...)
          tmp = append(tmp, newSeqA[ii+priorCount+1:]...)
          newSeqA = tmp
          priorCount += (rightIndicesB[lenR-1] - leftIndicesB[0] - 1)
        }
      }
    }
  }

  glog.V(2).Infof("newSeq for id - %d = %v\n", esA.id, newSeqA)

  // If there were no expansions done (in esA), return nil, otherwise
  // return the expanded event sequence set
  if priorCount > 0 || suffixCount > 0 {
    return newSeqA
  } else {
    return nil
  }
}

// MergeTwo merges two given EventSequences with each other.
// It expands "each" sequence using any sub sequence euality from the "other"
func (dc *DetectiveCase) MergeTwo(esA, esB *EventSequence) {
  if esA.mergeDoneMap[esB.id] || esB.mergeDoneMap[esA.id] {
    esB.mergeDoneMap[esA.id] = true
    esA.mergeDoneMap[esB.id] = true
    return
  }

  // First try to expand the two - esA and esB using the other i.e esB & esA
  // respectively.
  newSeqA := esA.GetExpandedSequence(esB)
  glog.V(2).Infof("========================\n")
  newSeqB := esB.GetExpandedSequence(esA)

  // Evaluate now if both the sequences need to be updated.
  // So Future merges with other sequences will use the updated.
  // The order of merges may have some impact, at this time that is
  // not considered, based on the statement that all witnesses are truthful..

  if newSeqA != nil && len(newSeqA) > 0 {
    esA.sequence = newSeqA
    esA.RegenerateIndexMap()
  }
  if newSeqB != nil && len(newSeqB) > 0 {
    esB.sequence = newSeqB
    esB.RegenerateIndexMap()
  }
  esB.mergeDoneMap[esA.id] = true
  esA.mergeDoneMap[esB.id] = true
  return
}

func (dc *DetectiveCase) Merge() []EventSequenceMap {
  numSeqs := len(dc.eventSeqs)

  for ii := 0; ii < numSeqs; ii++ {
    for jj := ii + 1; jj < numSeqs; jj++ {
      dc.MergeTwo(dc.eventSeqs[ii], dc.eventSeqs[jj])
    }
  }
  return nil
}

// Analyze, typically be called after a merge, will go through all
// the merged/expanded sequences and eliminate duplicates, and list
// the unique merged sequences, and a merge status of "merge possible",
// "no merge possible", "partial merge is possible"
func (dc *DetectiveCase) Analyze() (MergeStatus, [][]string) {

  numSeqs := len(dc.eventSeqs)
  changeCount := 0

  concatMap := make(map[string][]int32, numSeqs)

  for _, eventSeq := range dc.eventSeqs {
    if len(eventSeq.original) != len(eventSeq.sequence) {
      changeCount++
    }
    catStr := strings.Join(eventSeq.sequence, "")
    concatMap[catStr] = append(concatMap[catStr], eventSeq.id)
  }

  numExpanded := len(concatMap)

  var status MergeStatus
  if changeCount == 0 {
    status = MergeNotPossible
  } else if numExpanded == 1 && changeCount >= (numSeqs-1) {
    status = MergeComplete
  } else {
    status = MergePartial
  }

  mergedSeqences := make([][]string, numExpanded)

  ii := 0
  for _, val := range concatMap {
    strSeq := (*dc.eventSeqMap)[val[0]].sequence
    mergedSeqences[ii] = strSeq
    ii++
  }

  return status, mergedSeqences
}

func (dc *DetectiveCase) Print() {
  numSeqs := len(dc.eventSeqs)

  for ii := 0; ii < numSeqs; ii++ {
    seqJson, err := json.Marshal(dc.eventSeqs[ii].sequence)
    if err != nil {
      glog.Errorf("Error: %s", err)
      return
    }
    glog.Infof("evSeqs %d - id:%d %+v\n", ii, dc.eventSeqs[ii].id,
      string(seqJson))
  }
}

func (dc *DetectiveCase) PrintAnalysis() {
  mStatus, eventSeqs := dc.Analyze()
  fmt.Println("Input:")
  inputJson, err := json.Marshal(dc.evStrInput)
  if err != nil {
    fmt.Printf("Error: %s", err)
    return
  }
  fmt.Println(string(inputJson))
  fmt.Println("Output:")
  fmt.Println(mStatus)
  seqJson, err := json.Marshal(eventSeqs)
  if err != nil {
    fmt.Printf("Error: %s", err)
    return
  }
  fmt.Println(string(seqJson))
}

// basic unit test.
func main() {
  flag.Parse()
  //flag.Lookup("logtostderr").Value.Set("true")
  evStrSet := [][]string{{"fight", "gunshot", "fleeing"}, {"gunshot", "falling", "fleeing"}}
  glog.V(1).Infof("evStrSet - %+v\n ###### \n", evStrSet)
  dc := NewDetectiveCase(evStrSet)
  if glog.V(1) {
    dc.Print()
  }
  dc.Merge()
  if glog.V(1) {
    glog.Infof("evSeqs after merge - \n")
    dc.Print()
  }
  dc.PrintAnalysis()
  glog.Flush()
}
