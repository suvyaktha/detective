package main

import (
  "github.com/stretchr/testify/assert"
  "strings"
  "testing"
)

type DetectiveTest struct {
  input  [][]string
  output [][]string
  result MergeStatus
}

var testCases = []DetectiveTest{
  {
    [][]string{
      {"fight", "gunshot", "fleeing"},
      {"gunshot", "falling", "fleeing"},
    },
    [][]string{
      {"fight", "gunshot", "falling", "fleeing"},
    },
    MergeComplete,
  },
  {
    [][]string{
      {"shadowy figure", "demands", "scream", "siren"},
      {"shadowy figure", "pointed gun", "scream"},
    },
    [][]string{
      {"shadowy figure", "demands", "scream", "siren"},
      {"shadowy figure", "pointed gun", "scream", "siren"},
    },
    MergePartial,
  },
  {
    [][]string{
      {"argument", "stuff", "pointing"},
      {"press brief", "scandal", "pointing"},
      {"bribe", "coverup"},
    },
    [][]string{
      {"argument", "stuff", "pointing"},
      {"press brief", "scandal", "pointing"},
      {"bribe", "coverup"},
    },
    MergeNotPossible,
  },
  // combinations of above
  {
    [][]string{
      {"argument", "stuff", "pointing"},
      {"bribe", "coverup"},
      {"press brief", "scandal", "pointing"},
    },
    [][]string{
      {"argument", "stuff", "pointing"},
      {"bribe", "coverup"},
      {"press brief", "scandal", "pointing"},
    },
    MergeNotPossible,
  },
  {
    [][]string{
      {"shouting", "fight", "fleeing"},
      {"fight", "gunshot", "panic", "fleeing"},
      {"anger", "shouting"},
    },
    [][]string{
      {"anger", "shouting", "fight", "gunshot", "panic", "fleeing"},
    },
    MergeComplete,
  },
  {
    // combinations of above
    [][]string{
      {"fight", "gunshot", "panic", "fleeing"},
      {"shouting", "fight", "fleeing"},
      {"anger", "shouting"},
      {"anger", "shouting"},
      {"shouting", "fight", "fleeing"},
    },
    [][]string{
      {"anger", "shouting", "fight", "gunshot", "panic", "fleeing"},
    },
    MergeComplete,
  },
}

func TestDetective(t *testing.T) {
  var es *EventSequence
  for _, testCase := range testCases {
    dc := NewDetectiveCase(testCase.input)
    dc.Merge()
    mStatus, eventSeqs := dc.Analyze()
    assert.Equal(t, mStatus, testCase.result)

    resultsStrMap := make(map[string]int, len(testCase.input))

    for _, eventSeq := range eventSeqs {
      catStr := strings.Join(eventSeq, ";")
      cnt, found := resultsStrMap[catStr]
      if found {
        resultsStrMap[catStr] = cnt + 1
      } else {
        resultsStrMap[catStr] = 1
      }
    }
    for _, outputSeqStrs := range testCase.output {
      var sanitizedOutStrs = make([]string, len(outputSeqStrs))
      for ii, oStr := range outputSeqStrs {
        sanitizedOutStrs[ii] = es.sanitize(oStr)
      }
      catStr := strings.Join(sanitizedOutStrs, ";")
      cnt, found := resultsStrMap[catStr]
      assert.Equal(t, found, true)
      assert.Equal(t, cnt, 1)
    }
    dc.PrintAnalysis()
  }
}
