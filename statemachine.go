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
func (s *StateMachine) run(inputLines StringList, inputOffset int, context Context, initialState string) []string {
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

		transitions = nil
		state, _ = s.getState(nextState)
	}
	s.observers = nil
	return results
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
	var err error
	s.line, err = s.inputLines.GetItem(s.lineOffset)
	if err != nil {
		// IndexError
		s.line = ""
		s.notifyObservers()
		return "", &EOFError{"EOFError in StateMachine nextLine"}
	}
	s.notifyObservers()
	return s.line, nil
}

// Return true if the next line is blank or non-existant.
func (s *StateMachine) isNextLineBlank() bool {
	line, err := s.inputLines.GetItem(s.lineOffset + 1)
	if err == nil {
		line = strings.TrimSpace(line)
		if line == "" {
			return true
		}
		return false
	}
	return true
}

// Return true if the input is at or past end-of-file.
func (s *StateMachine) AtEof() bool {
	return s.lineOffset >= (s.inputLines.Length() - 1)
}

// Return true if the input is at or before beginning-of-file.
func (s *StateMachine) AtBof() bool {
	return s.lineOffset <= 0
}

// Load `self.line` with the `n`'th previous line and return it.
func (s *StateMachine) previousLine(n int) string {
	s.lineOffset -= n
	if s.lineOffset < 0 {
		s.line = ""
	} else {
		s.line, _ = s.inputLines.GetItem(s.lineOffset)
	}
	s.notifyObservers()
	return s.line
}

// Jump to absolute line offset `line_offset`, load and return it.
func (s *StateMachine) GotoLine(lineOffset int) (string, error) {
	s.lineOffset = lineOffset - s.inputOffset
	var err error
	s.line, err = s.inputLines.GetItem(s.lineOffset)
	if err != nil {
		s.line = ""
		s.notifyObservers()
		return s.line, err
	}
	s.notifyObservers()
	return s.line, nil
}

// Return source of line at absolute line offset `line_offset`.
func (s *StateMachine) GetSource(lineOffset int) (string, error) {
	return s.inputLines.Source(lineOffset - s.inputOffset)
}

// Return line offset of current line, from beginning of file.
func (s *StateMachine) AbsLineOffset() int {
	return s.lineOffset + s.inputOffset
}

// Return line number of current line (counting from 1).
func (s *StateMachine) AbsLineNumber() int {
	return s.lineOffset + s.inputOffset + 1
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
