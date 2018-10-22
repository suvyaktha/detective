# detective
Solution for a "Detective" puzzle in Go language

## Problem Statement

A detective is asked to review timelines from a number of witnesses. Each recalls the order of
the events that they witnessed, and they are all trustworthy. However, witnesses may not have
witnessed or recall witnessing every event.

For instance:

- John remembers: shouting, fight, fleeing
- Steve remembers: fight, gunshot, panic, fleeing
- Jason remembers: anger, shouting

The detective needs to construct a maximal timeline from each of these witness timelines. When
enough of the witness timelines can be merged to form a long enough timeline, then this increases
the likelihood of a successful conviction. The ordering of events must be absolutely correct,
or else the case will be thrown out. If timelines cannot be strictly ordered, then multiple
timelines must be presented.

- If all witnesses remember events in a fully consistent manner, then present a single merged timeline.
- If some of the events they remember can be combined, or if some of the them can be extended
without fully merging them, then present multiple timelines with events merged across them
to the maximum degree possible.
- If none of the events can be combined, or extended, then present the original, unmodified timelines.

The above example can be combined into a single timeline:

    anger, shouting, fight, gunshot, panic, fleeing

An example of multiple possible timelines is:

- Edgar: pouring gas, laughing, lighting match, fire
- Bruce: buying gas, pouring gas, crying, fire, smoke

Since it is not possible to tell if the crying occurred before or after lighting the match, then
two timelines emerge:

    buying gas, pouring gas, laughing, lighting match, fire, smoke
    buying gas, pouring gas, crying, fire, smoke

## Problem Format

The input appears as an array of eyewitness accounts.  Each eyewitness account is represented
as an array of strings.

The output will also be an array of maximal timelines. Each timeline is an array of strings.

Examples:

<table>
  <tr>
    <th>Input</th>
    <th>Output</th>
  </tr>
  <tr>
    <td>[ ["fight", "gunshot" "fleeing"], ["gunshot", "falling", "fleeing"] ]</td>
    <td>Merge is possible </p> [ ["fight", "gunshot", "falling", "fleeing"] ]</td>
  </tr>
  <tr>
    <td>[ ["shadowy figure", "demands", "scream", "siren"], ["shadowy figure", "pointed gun", "scream"] ]</td>
    <td>Partial merge is possible </p> [ ["shadowy figure", "demand", "scream", "siren"], ["shadowy figure", "pointed gun", "scream", "siren"] ]</td>
  </tr>
  <tr>
    <td>[ ["argument", "stuff", "pointing"], 
    ["press brief", "scandal", "pointing"], ["bribe", "coverup"] ]</td>
    <td>No merge is possible </p> [ ["argument", "stuff", "pointing"], ["press brief", "scandal", "pointing"], ["bribe", "coverup"] ]</td>
  </tr>
</table>
