package rst

/*
Error types for state machine in Python docutils

URL of Python source code:
http://sourceforge.net/p/docutils/code/HEAD/tree/trunk/docutils/docutils/statemachine.py
*/

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
