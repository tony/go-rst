package rst

/*
Implementation of State Machine in Python docutils

Functions:

- `File2lines()`: split file content into a list of one-line strings
*/

import (
	"bufio"
	"fmt"
	"os"
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
	states map[string]State

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
   - The next state name (a return value of ``None`` means no state change).
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
	patterns map[string]string

	// A list of transitions to initialize when a `State` is instantiated.
	// Each entry is either a transition name string, or a (transition name, next
	// state name) pair. See `make_transitions()`. Override in subclasses.
	initialTransitions []Transition

	// The `StateMachine` class for handling nested processing.
	//
	// If left as ``None``, `nested_sm` defaults to the class of the state's
	// controlling state machine. Override it in subclasses to avoid the default.
	nestedSm StateMachine

	// Keyword arguments dictionary, passed to the `nested_sm` constructor.
	//
	// Two keys must have entries in the dictionary:
	//
	// - Key 'state_classes' must be set to a list of `State` classes.
	// - Key 'initial_state' must be set to the name of the initial state class.
	//
	// If `nested_sm_kwargs` is left as ``None``, 'state_classes' defaults to the
	// class of the current state, and 'initial_state' defaults to the name of
	// the class of the current state. Override in subclasses to avoid the
	// defaults.
	nestedSmKwargs map[string]string
}

type Transition string

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
