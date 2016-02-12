package rst

/*
Implement StringList for state machine in Python docutils

URL of Python source code:
http://sourceforge.net/p/docutils/code/HEAD/tree/trunk/docutils/docutils/statemachine.py
*/

import "strings"

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

func (v *ViewList) GetItem(index int) (string, error) {
	if index < len(v.data) && index >= 0 {
		return v.data[index], nil
	}
	return "", &IndexError{"index error"}
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

func (s *StringList) InsertItemsSlice(i int, vl StringList) {
	s.ViewList.InsertItemsSlice(i, vl.ViewList)
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
