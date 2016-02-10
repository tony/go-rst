package rst

/*
Implementation of State Machine in Python docutils

URL of Python source code:
http://sourceforge.net/p/docutils/code/HEAD/tree/trunk/docutils/docutils/statemachine.py

Functions:

- `File2lines()`: split file content into a list of one-line strings
*/

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
)

/*
   A finite state machine for text filters using regular expressions.

   The input is provided in the form of a list of one-line strings (no
   newlines). States are subclasses of the `State` class. Transitions consist
   of regular expression patterns and transition methods, and are defined in
   each state.

   The state machine is started with the `run()` method, which returns the
   results of processing in a list.
*/
type StateMachine struct {
	// `StringList` of input lines (without newlines).
	// Filled by `self.run()`.
	inputLines StringList

	// Offset of `self.input_lines` from the beginning of the file.
	inputOffset int

	// Current input line.
	line string

	// Current input line offset from beginning of `self.input_lines`.
	lineOffset int

	// Debugging mode on/off.
	debug bool

	// The name of the initial state (key to `self.states`).
	initialState string

	// The name of the current state (key to `self.states`).
	currentState string

	// Mapping of {state_name: State_object}.
	states map[string]*State

	// List of bound methods or functions to call whenever the current
	// line changes.  Observers are called with one argument, ``self``.
	// Cleared at the end of `run()`.
	observers []func(string, int)
}

/*
   Initialize a `StateMachine` object; add state objects.

   Parameters:

   - `state_classes`: a list of `State` (sub)classes.
   - `initial_state`: a string, the class name of the initial state.
   - `debug`: a boolean; produce verbose output if true (nonzero).
*/
func (s *StateMachine) Init(stateClasses []*State, initialState string, debug bool) {
	s.lineOffset = -1
	s.debug = debug
	s.initialState = initialState
	s.currentState = initialState
	s.addStates(stateClasses)
}

// Remove circular references to objects no longer required.
func (s *StateMachine) unlink() {
	for _, state := range s.states {
		state.unlink()
	}
	s.states = nil
}

/*
   Run the state machine on `input_lines`. Return results (a list).

   Reset `self.line_offset` and `self.current_state`. Run the
   beginning-of-file transition. Input one line at a time and check for a
   matching transition. If a match is found, call the transition method
   and possibly change the state. Store the context returned by the
   transition method to be passed on to the next transition matched.
   Accumulate the results returned by the transition methods in a list.
   Run the end-of-file transition. Finally, return the accumulated
   results.

   Parameters:

   - `input_lines`: a list of strings without newlines, or `StringList`.
   - `input_offset`: the line offset of `input_lines` from the beginning
     of the file.
   - `context`: application-specific storage.
   - `input_source`: name or path of source of `input_lines`.
   - `initial_state`: name of initial state.
*/
func (s *StateMachine) run(inputLines StringList, inputOffset int, context Context, initialState string) {
	s.runtimeInit()

	s.inputLines = inputLines

	s.inputOffset = inputOffset
	s.lineOffset = -1
	if initialState == "" {
		s.currentState = s.initialState
	} else {
		s.currentState = initialState
	}

	if s.debug {
		fmt.Println("\nStateMachine.run: input_lines (line_offset=%d)", s.lineOffset)
		for _, line := range s.inputLines.data {
			fmt.Println(line)
		}
	}

	var transitions []string
	var results []string
	state, _ := s.getState("")

	if s.debug {
		fmt.Println("\nStateMachine.run: bof transition")
	}
	context, result := state.bof(context)
	results = append(results, result...)
	for {
		var nextState string
		_, err := s.nextLine(1)
		if err == nil {
			if s.debug {
				fmt.Println("\nStateMachine.run: line " + s.line)
			}
			context, nextState, result = s.checkLine(context, state, transitions)
		} else {
			// EOFError
			if s.debug {
				fmt.Println("\nStateMachine.run: %s.eof transition", reflect.TypeOf(state).Name())
			}
			result = state.eof(context)
			results = append(results, result...)
			break
		}
		results = append(result, result...)

		// FIXME: implement TransitionCorrection

		// FIXME: implement StateCorrection

		state, _ = s.getState(nextState)
	}
}

/*
   Return current state object; set it first if `next_state` given.

   Parameter `next_state`: a string, the name of the next state.

   Exception: `UnknownStateError` raised if `next_state` unknown.
*/
func (s *StateMachine) getState(nextState string) (*State, error) {
	if nextState != "" {
		if s.debug && nextState != s.currentState {
			fmt.Println("\nStateMachine.get_state: Changing state from %s to %s", s.currentState, nextState)
		}
		s.currentState = nextState
	}

	state, ok := s.states[s.currentState]
	if !ok {
		return nil, &UnknownStateError{"UnknownStateError: " + s.currentState}
	}
	return state, nil
}

// Load `self.line` with the `n`'th next line and return it.
func (s *StateMachine) nextLine(n int) (string, error) {
	s.lineOffset += n
	if s.lineOffset < s.inputLines.Length() {
		s.line = s.inputLines.GetItem(s.lineOffset)
	} else {
		// IndexError
		s.line = ""
		s.notifyObservers()
		return "", &EOFError{"EOFError in StateMachine nextLine"}
	}
	s.notifyObservers()
	return s.line, nil
}

/*
   Examine one line of input for a transition match & execute its method.

   Parameters:

   - `context`: application-dependent storage.
   - `state`: a `State` object, the current state.
   - `transitions`: an optional ordered list of transition names to try,
     instead of ``state.transition_order``.

   Return the values returned by the transition method:

   - context: possibly modified from the parameter `context`;
   - next state name (`State` subclass name);
   - the result output of the transition, a list.

   When there is no match, ``state.no_match()`` is called and its return
   value is returned.
*/
func (s *StateMachine) checkLine(context Context, state *State, transitions []string) (Context, string, []string) {
	if transitions == nil {
		transitions = state.transitionOrder
	}
	//state_correction = None
	if s.debug {
		fmt.Println("\nStateMachine.check_line: state=", reflect.TypeOf(state).Name())
	}
	for _, name := range transitions {
		pattern := state.transitions[name].compiledPattern
		method := state.transitions[name].transitionMethod
		nextState := state.transitions[name].nextStateName
		match := pattern.FindAllString(s.line, -1)
		if match != nil {
			if s.debug {
				fmt.Println("\nStateMachine.check_line: Matched transition")
			}
			retv := method.Call([]reflect.Value{
				reflect.ValueOf(match),
				reflect.ValueOf(context),
				reflect.ValueOf(nextState),
			})
			return retv[0].Interface().(Context), retv[1].Interface().(string), retv[2].Interface().([]string)
		}
	}
	if s.debug {
		fmt.Println("\nStateMachine.check_line: No match in state ", reflect.TypeOf(state).Name())
	}
	return state.noMatch(context, transitions)
}

/*
   Initialize & add a `state_class` (`State` subclass) object.

   Exception: `DuplicateStateError` raised if `state_class` was already
   added.
*/
func (s *StateMachine) addState(stateClass *State) error {
	statename := reflect.TypeOf(stateClass).Name()
	if _, ok := s.states[statename]; ok {
		return &DuplicateStateError{"DuplicateStateError: " + statename}
	}
	stateClass.Init(s, s.debug)
	s.states[statename] = stateClass
	return nil
}

// Add `state_classes` (a list of `State` subclasses).
func (s *StateMachine) addStates(stateClasses []*State) {
	for _, stateClass := range stateClasses {
		s.addState(stateClass)
	}
}

// Initialize `self.states`.
func (s *StateMachine) runtimeInit() {
	for _, state := range s.states {
		state.runtimeInit()
	}
}

func (s *StateMachine) notifyObservers() {
	for _, observer := range s.observers {
		info, err := s.inputLines.Info(s.lineOffset)
		if err == nil {
			observer(info.source, info.offset)
		} else {
			observer("", -1)
		}
	}
}

/*
   State superclass. Contains a list of transitions, and transition methods.

   Transition methods all have the same signature. They take 3 parameters:

   - An `re` match object. ``match.string`` contains the matched input line,
     ``match.start()`` gives the start index of the match, and
     ``match.end()`` gives the end index.
   - A context object, whose meaning is application-defined (initial value
     ``None``). It can be used to store any information required by the state
     machine, and the retured context is passed on to the next transition
     method unchanged.
   - The name of the next state, a string, taken from the transitions list;
     normally it is returned unchanged, but it may be altered by the
     transition method if necessary.

   Transition methods all return a 3-tuple:

   - A context object, as (potentially) modified by the transition method.
   - The next state name (a return value of "" means no state change).
   - The processing result, a list, which is accumulated by the state
     machine.

   Transition methods may raise an `EOFError` to cut processing short.

   There are two implicit transitions, and corresponding transition methods
   are defined: `bof()` handles the beginning-of-file, and `eof()` handles
   the end-of-file. These methods have non-standard signatures and return
   values. `bof()` returns the initial context and results, and may be used
   to return a header string, or do any other processing needed. `eof()`
   should handle any remaining context and wrap things up; it returns the
   final processing result.

   Typical applications need only subclass `State` (or a subclass), set the
   `patterns` and `initial_transitions` class attributes, and provide
   corresponding transition methods. The default object initialization will
   take care of constructing the list of transitions.
*/
type State struct {
	// {Name: pattern} mapping, used by `make_transition()`. Each pattern may
	// be a string or a compiled `re` pattern. Override in subclasses.
	patterns map[string]*regexp.Regexp

	// A list of transitions to initialize when a `State` is instantiated.
	// Each entry is a (transition name, next state name) pair. See
	// `make_transitions()`. Override in subclasses.
	initialTransitions []TransitionNameAndNextState

	// The `StateMachine` class for handling nested processing.
	//
	// If left as nil, `nested_sm` defaults to the class of the state's
	// controlling state machine. Override it in subclasses to avoid the default.
	nestedSm reflect.Type

	// Keyword arguments dictionary, passed to the `nested_sm` constructor.
	//
	// Two keys must have entries in the dictionary:
	//
	// - Key 'state_classes' must be set to a list of `State` classes.
	// - Key 'initial_state' must be set to the name of the initial state class.
	//
	// If `nested_sm_kwargs` is left as nil, 'state_classes' defaults to the
	// class of the current state, and 'initial_state' defaults to the name of
	// the class of the current state. Override in subclasses to avoid the
	// defaults.
	nestedSmKwargs *SmKwargs

	// Debugging mode on/off.
	debug bool

	// A list of transition names in search order.
	transitionOrder []string

	// A mapping of transition names to 3-tuples containing
	// (compiled_pattern, transition_method, next_state_name). Initialized as
	// an instance attribute dynamically (instead of as a class attribute)
	// because it may make forward references to patterns and methods in this
	// or other classes.
	transitions map[string]Transition

	// A reference to the controlling `StateMachine` object.
	stateMachine *StateMachine
}

/*
   Initialize a `State` object; make & add initial transitions.

   Parameters:

   - `statemachine`: the controlling `StateMachine` object.
   - `debug`: a boolean; produce verbose output if true.
*/
func (s *State) Init(sm *StateMachine, debug bool) {
	s.addInitialTransitions()

	s.stateMachine = sm
	s.debug = debug

	if s.nestedSm == nil {
		s.nestedSm = reflect.TypeOf(s.stateMachine)
	}
	if s.nestedSmKwargs == nil {
		s.nestedSmKwargs = &SmKwargs{
			stateClasses: []reflect.Type{reflect.TypeOf(s)},
			initialState: reflect.TypeOf(s).Name(),
		}
	}
}

// Initialize this `State` before running the state machine; called from
// `self.stateMachine.run()`.
func (s *State) runtimeInit() {
}

// Remove circular references to objects no longer required.
func (s *State) unlink() {
	s.stateMachine = nil
}

// Make and add transitions listed in `self.initial_transitions`.
func (s *State) addInitialTransitions() {
	if len(s.initialTransitions) > 0 {
		names, transitions := s.makeTransitions(s.initialTransitions)
		s.addTransitions(names, transitions)
	}
}

/*
   Add a list of transitions to the start of the transition list.

   Parameters:

   - `names`: a list of transition names.
   - `transitions`: a mapping of names to transition tuples.

   Exceptions: `DuplicateTransitionError`, `UnknownTransitionError`.
*/
func (s *State) addTransitions(names []string, transitions map[string]Transition) error {
	for _, name := range names {
		if _, ok := s.transitions[name]; ok {
			return &DuplicateTransitionError{"DuplicateTransitionError: " + name}
		}
		if _, ok := transitions[name]; !ok {
			return &UnknownTransitionError{"UnknownTransitionError: " + name}
		}
	}

	s.transitionOrder = append(names, s.transitionOrder...)
	for name, transition := range transitions {
		s.transitions[name] = transition
	}
	return nil
}

/*
   Add a transition to the start of the transition list.

   Parameter `transition`: a ready-made transition 3-tuple.

   Exception: `DuplicateTransitionError`.
*/
func (s *State) addTransition(name string, transition Transition) error {
	if _, ok := s.transitions[name]; ok {
		return &DuplicateTransitionError{"DuplicateTransitionError: " + name}
	}
	s.transitionOrder = append([]string{name}, s.transitionOrder...)
	s.transitions[name] = transition
	return nil
}

/*
   Remove a transition by `name`.

   Exception: `UnknownTransitionError`.
*/
func (s *State) removeTransition(name string) error {
	if _, ok := s.transitions[name]; ok {
		delete(s.transitions, name)
		for i, n := range s.transitionOrder {
			if n == name {
				s.transitionOrder = append(s.transitionOrder[:i], s.transitionOrder[i+1:]...)
				break
			}
		}

	} else {
		return &UnknownTransitionError{"UnknownTransitionError: " + name}
	}
	return nil
}

/*
   Make & return a transition tuple based on `name`.

   This is a convenience function to simplify transition creation.

   Parameters:

   - `name`: a string, the name of the transition pattern & method. This
     `State` object must have a method called '`name`', and a dictionary
     `self.patterns` containing a key '`name`'.
   - `next_state`: a string, the name of the next `State` object for this
     transition. A value of "" (empty string) implies no state change
     (i.e., continue with the same state).

   Exceptions: `TransitionPatternNotFound`, `TransitionMethodNotFound`.
*/
func (s *State) makeTransition(name, nextState string) (Transition, error) {
	if nextState == "" {
		nextState = reflect.TypeOf(s).Name()
	}

	pattern, ok := s.patterns[name]
	if !ok {
		return Transition{}, &TransitionPatternNotFound{"TransitionPatternNotFound: " + name + " not in " + reflect.TypeOf(s).Name()}
	}

	method := reflect.New(reflect.TypeOf(s)).MethodByName(name)
	if !method.IsValid() {
		return Transition{}, &TransitionMethodNotFound{"TransitionMethodNotFound: " + name + " not in " + reflect.TypeOf(s).Name()}
	}

	return Transition{pattern, method, nextState}, nil
}

/*
   Return a list of transition names and a transition mapping.

   Parameter `pairs`: a list, where each entry is a 2-tuple (transition name,
   next state name).
*/
func (s *State) makeTransitions(pairs []TransitionNameAndNextState) (names []string, transitions map[string]Transition) {
	for _, pair := range pairs {
		transitions[pair.name], _ = s.makeTransition(pair.name, pair.nextState)
		names = append(names, pair.name)
	}
	return
}

/*
   Called when there is no match from `StateMachine.check_line()`.

   Return the same values returned by transition methods:

   - context: unchanged;
   - next state name: "";
   - empty result list.

   Override in subclasses to catch this event.
*/
func (s *State) noMatch(context Context, transitions []string) (Context, string, []string) {
	return context, "", nil
}

/*
   Handle beginning-of-file. Return unchanged `context`, empty result.

   Override in subclasses.

   Parameter `context`: application-defined storage.
*/
func (s *State) bof(context Context) (Context, []string) {
	return context, nil
}

/*
   Handle end-of-file. Return empty result.

   Override in subclasses.

   Parameter `context`: application-defined storage.
*/
func (s *State) eof(context Context) []string {
	return nil
}

/*
   A "do nothing" transition method.

   Return unchanged `context` & `next_state`, empty result. Useful for
   simple state changes (actionless transitions).
*/
func (s *State) nop(match []string, context Context, nextState string) (Context, string, []string) {
	return context, nextState, nil
}

type Transition struct {
	compiledPattern  *regexp.Regexp
	transitionMethod reflect.Value
	nextStateName    string
}

type TransitionNameAndNextState struct {
	name      string
	nextState string
}

type SmKwargs struct {
	stateClasses []reflect.Type
	initialState string
}

type Context string

type ViewListItem struct {
	source string
	offset int
}

/*
   List with extended functionality: slices of ViewList objects are child
   lists, linked to their parents. Changes made to a child list also affect
   the parent list.  A child list is effectively a "view" (in the SQL sense)
   of the parent list.  Changes to parent lists, however, do *not* affect
   active child lists.  If a parent list is changed, any active child lists
   should be recreated.

   The start and end of the slice can be trimmed using the `trim_start()` and
   `trim_end()` methods, without affecting the parent list.  The link between
   child and parent lists can be broken by calling `disconnect()` on the
   child list.

   Also, ViewList objects keep track of the source & offset of each item.
   This information is accessible via the `source()`, `offset()`, and
   `info()` methods.
*/
type ViewList struct {
	//The actual list of data, flattened from various sources.
	data []string

	// A list of (source, offset) pairs, same length as `self.data`: the
	// source of each line and the offset of each line from the beginning of
	// its source.
	items []ViewListItem

	// The parent list.
	parent *ViewList

	// Offset of this list from the beginning of the parent list.
	parentOffset int
}

func (v *ViewList) Init(initlist []string, source string, items []ViewListItem, parent *ViewList, parentOffset int) {
	v.parent = parent
	v.parentOffset = parentOffset
	v.data = initlist
	if items == nil {
		for i, _ := range initlist {
			v.items = append(v.items, ViewListItem{source, i})
		}
	} else {
		v.items = items
	}
	if len(v.data) != len(v.items) {
		panic("data mismatch")
	}
}

func (v *ViewList) Contains(item string) bool {
	for _, d := range v.data {
		if d == item {
			return true
		}
	}
	return false
}

func (v *ViewList) Length() int {
	return len(v.data)
}

func (v *ViewList) GetItem(index int) string {
	return v.data[index]
}

func (v *ViewList) GetItemsSlice(start, stop int) ViewList {
	vl := ViewList{}
	vl.Init(v.data[start:stop], "", v.items, v, start)
	return vl
}

func (v *ViewList) SetItem(index int, item string) {
	v.data[index] = item
	if v.parent != nil {
		v.parent.SetItem(index+v.parentOffset, item)
	}
}

func (v *ViewList) SetItemsSlice(start, stop int, items ViewList) {
	for i := start; i < stop; i++ {
		v.data[i] = items.data[i]
		v.items[i] = items.items[i]
	}

	if v.parent != nil {
		v.parent.SetItemsSlice(start+v.parentOffset, stop+v.parentOffset, items)
	}
}

func (v *ViewList) DeleteItem(index int) {
	v.data = append(v.data[:index], v.data[index+1:]...)
	v.items = append(v.items[:index], v.items[index+1:]...)
	if v.parent != nil {
		v.parent.DeleteItem(index + v.parentOffset)
	}
}

func (v *ViewList) DeleteItemsSlice(start, stop int) {
	v.data = append(v.data[:start], v.data[stop:]...)
	v.items = append(v.items[:start], v.items[stop:]...)
	if v.parent != nil {
		v.parent.DeleteItemsSlice(start+v.parentOffset, stop+v.parentOffset)
	}
}

func (v *ViewList) Add(other ViewList) ViewList {
	data := append(v.data, other.data...)
	items := append(v.items, other.items...)
	result := ViewList{}
	result.Init(data, "", items, nil, 0)
	return result
}

func (v *ViewList) Radd(other ViewList) ViewList {
	data := append(other.data, v.data...)
	items := append(other.items, v.items...)
	result := ViewList{}
	result.Init(data, "", items, nil, 0)
	return result
}

func (v *ViewList) Extend(other ViewList) {
	if v.parent != nil {
		v.parent.InsertItemsSlice(len(v.data)+v.parentOffset, other)
	}
	v.data = append(v.data, other.data...)
	v.items = append(v.items, other.items...)
}

func (v *ViewList) AppendItem(item, source string, offset int) {
	if v.parent != nil {
		v.parent.InsertItem(len(v.data)+v.parentOffset, item, source, offset)
	}
	v.data = append(v.data, item)
	v.items = append(v.items, ViewListItem{source, offset})
}

func (v *ViewList) AppendItemsSlice(vl ViewList) {
	v.Extend(vl)
}

func (v *ViewList) InsertItem(i int, item, source string, offset int) {
	if source == "" {
		panic("source cannot be empty")
	}

	v.data = append(v.data, "")
	copy(v.data[i+1:], v.data[i:])
	v.data[i] = item

	v.items = append(v.items, ViewListItem{})
	copy(v.items[i+1:], v.items[i:])
	v.items[i] = ViewListItem{source, offset}

	if v.parent != nil {
		index := (len(v.data) + i) % len(v.data)
		v.parent.InsertItem(index+v.parentOffset, item, source, offset)
	}
}

func (v *ViewList) InsertItemsSlice(i int, vl ViewList) {
	v.data = append(v.data[:i], append(vl.data, v.data[i:]...)...)
	v.items = append(v.items[:i], append(vl.items, v.items[i:]...)...)
	if v.parent != nil {
		index := (len(v.data) + i) % len(v.data)
		v.parent.InsertItemsSlice(index+v.parentOffset, vl)
	}
}

func (v *ViewList) Pop(i int) string {
	if v.parent != nil {
		index := (len(v.data) + i) % len(v.data)
		v.parent.Pop(index + v.parentOffset)
	}
	v.items = append(v.items[:i], v.items[i+1:]...)
	result := v.data[i]
	v.data = append(v.data[:i], v.data[i+1:]...)
	return result
}

// Remove items from the start of the list, without touching the parent.
func (v *ViewList) TrimStart(n int) error {
	if n > len(v.data) {
		return &IndexError{"Size of trim too large;"}
	}
	if n < 0 {
		return &IndexError{"Trim size must be >= 0."}
	}
	v.data = v.data[n:]
	v.items = v.items[n:]
	if v.parent != nil {
		v.parentOffset += n
	}
	return nil
}

// Remove items from the end of the list, without touching the parent.
func (v *ViewList) TrimEnd(n int) error {
	if n > len(v.data) {
		return &IndexError{"Size of trim too large;"}
	}
	if n < 0 {
		return &IndexError{"Trim size must be >= 0."}
	}
	v.data = v.data[:len(v.data)-n]
	v.items = v.items[:len(v.items)-n]
	return nil
}

// Return source & offset for index `i`.
func (v *ViewList) Info(i int) (ViewListItem, error) {
	if i < len(v.items) {
		return v.items[i], nil
	} else {
		if i == len(v.data) { // Just past the end
			return ViewListItem{v.items[i-1].source, -1}, nil
		} else {
			return ViewListItem{}, &IndexError{"ViewList Info IndexError"}
		}
	}
}

// Return source for index `i`.
func (v *ViewList) Source(i int) (string, error) {
	info, err := v.Info(i)
	return info.source, err
}

// Return offset for index `i`.
func (v *ViewList) Offset(i int) (int, error) {
	info, err := v.Info(i)
	return info.offset, err
}

// Break link between this list and parent list.
func (v *ViewList) Disconnect(i int) {
	v.parent = nil
}

// A `ViewList` with string-specific methods.
type StringList struct {
	ViewList
}

/*
   Trim `length` characters off the beginning of each item, in-place,
   from index `start` to `end`.  No whitespace-checking is done on the
   trimmed text.  Does not affect slice parent.
*/
func (s *StringList) TrimLeft(length, start, end int) {
	for i := start; i < end; i++ {
		s.data[i] = s.data[i][length:]
	}
}

/*
   Return a contiguous block of text.

   If `flush_left` is true, raise `UnexpectedIndentationError` if an
   indented line is encountered before the text block ends (with a blank
   line).
*/
func (s *StringList) GetTextBlock(start int, flushLeft bool) (StringList, error) {
	end := start
	last := len(s.data)
	for end < last {
		line := s.data[end]
		if strings.TrimSpace(line) != "" {
			break
		}
		if flushLeft && line[0] == ' ' {
			return StringList{}, &UnexpectedIndentationError{"UnexpectedIndentationError StringList GetTextBlock"}
		}
		end += 1
	}
	result := StringList{}
	result.Init(s.data[start:end], "", s.items[start:end], s.parent, s.parentOffset)
	return result, nil
}

// Replace all occurrences of substring `oldStr` with `newStr`.
func (s *StringList) Replace(oldStr, newStr string) {
	for i, line := range s.data {
		s.data[i] = strings.Replace(line, oldStr, newStr, -1)
	}
}

func File2lines(filePath string) []string {
	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return lines
}

type IndexError struct {
	msg string
}

func (e *IndexError) Error() string {
	return e.msg
}

type UnexpectedIndentationError struct {
	msg string
}

func (e *UnexpectedIndentationError) Error() string {
	return e.msg
}

type UnknownStateError struct {
	msg string
}

func (e *UnknownStateError) Error() string {
	return e.msg
}

type EOFError struct {
	msg string
}

func (e *EOFError) Error() string {
	return e.msg
}

type DuplicateStateError struct {
	msg string
}

func (e *DuplicateStateError) Error() string {
	return e.msg
}

type DuplicateTransitionError struct {
	msg string
}

func (e *DuplicateTransitionError) Error() string {
	return e.msg
}

type UnknownTransitionError struct {
	msg string
}

func (e *UnknownTransitionError) Error() string {
	return e.msg
}

type TransitionPatternNotFound struct {
	msg string
}

func (e *TransitionPatternNotFound) Error() string {
	return e.msg
}

type TransitionMethodNotFound struct {
	msg string
}

func (e *TransitionMethodNotFound) Error() string {
	return e.msg
}
