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
	inputLines []string

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
	observers []func()
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
func (s *State) addTransitions(names []string, transitions map[string]Transition) {
	for _, name := range names {
		if _, ok := s.transitions[name]; ok {
			panic("DuplicateTransitionError: " + name)
		}
		if _, ok := transitions[name]; !ok {
			panic("UnknownTransitionError: " + name)
		}
	}

	s.transitionOrder = append(names, s.transitionOrder...)
	for name, transition := range transitions {
		s.transitions[name] = transition
	}
}

/*
   Add a transition to the start of the transition list.

   Parameter `transition`: a ready-made transition 3-tuple.

   Exception: `DuplicateTransitionError`.
*/
func (s *State) addTransition(name string, transition Transition) {
	if _, ok := s.transitions[name]; ok {
		panic("DuplicateTransitionError: " + name)
	}
	s.transitionOrder = append([]string{name}, s.transitionOrder...)
	s.transitions[name] = transition
}

/*
   Remove a transition by `name`.

   Exception: `UnknownTransitionError`.
*/
func (s *State) removeTransition(name string) {
	if _, ok := s.transitions[name]; ok {
		delete(s.transitions, name)
		for i, n := range s.transitionOrder {
			if n == name {
				s.transitionOrder = append(s.transitionOrder[:i], s.transitionOrder[i+1:]...)
				break
			}
		}

	} else {
		panic("UnknownTransitionError: " + name)
	}
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
func (s *State) makeTransition(name, nextState string) Transition {
	if nextState == "" {
		nextState = reflect.TypeOf(s).Name()
	}

	pattern, ok := s.patterns[name]
	if !ok {
		panic("TransitionPatternNotFound: " + name + " not in " + reflect.TypeOf(s).Name())
	}

	method := reflect.New(reflect.TypeOf(s)).FieldByName(name)

	return Transition{pattern, method, nextState}
}

/*
   Return a list of transition names and a transition mapping.

   Parameter `pairs`: a list, where each entry is a 2-tuple (transition name,
   next state name).
*/
func (s *State) makeTransitions(pairs []TransitionNameAndNextState) (names []string, transitions map[string]Transition) {
	for _, pair := range pairs {
		transitions[pair.name] = s.makeTransition(pair.name, pair.nextState)
		names = append(names, pair.name)
	}
	return
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
