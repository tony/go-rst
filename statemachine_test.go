package rst

import "testing"
import "fmt"
import "bytes"

func TestViewList(t *testing.T) {
	var buf bytes.Buffer

	// init ViewList
	data := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	v := ViewList{}
	v.Init(data, "s", nil, nil, 0)
	fmt.Fprint(&buf, v)
	if buf.String() != "{[a b c d e f g h] [{s 0} {s 1} {s 2} {s 3} {s 4} {s 5} {s 6} {s 7}] <nil> 0}" {
		t.Error("Init ViewList failed")
	}

	// test Info
	buf.Reset()
	fmt.Fprint(&buf, v.Info(3))
	if buf.String() != "{s 3}" {
		t.Error("Info at index 3 failed")
	}

	if v.Source(2) != "s" {
		t.Error("Source at index 2 failed")
	}

	if v.Offset(5) != 5 {
		t.Error("Offset at index 5 failed")
	}

	// test GetItem
	if v.GetItem(3) != "d" {
		t.Error("GetItem at index 3 failed")
	}

	// test DeleteItem
	v.DeleteItem(3)
	buf.Reset()
	fmt.Fprint(&buf, v)
	if buf.String() != "{[a b c e f g h] [{s 0} {s 1} {s 2} {s 4} {s 5} {s 6} {s 7}] <nil> 0}" {
		t.Error("DeleteItem at index 3 failed")
	}

	v.TrimStart(2)
	buf.Reset()
	fmt.Fprint(&buf, v)
	if buf.String() != "{[c e f g h] [{s 2} {s 4} {s 5} {s 6} {s 7}] <nil> 0}" {
		t.Error("TrimStart(2) failed")
	}

	v.TrimEnd(3)
	buf.Reset()
	fmt.Fprint(&buf, v)
	if buf.String() != "{[c e] [{s 2} {s 4}] <nil> 0}" {
		t.Error("TrimEnd(3) failed")
	}

	v.Pop(1)
	buf.Reset()
	fmt.Fprint(&buf, v)
	if buf.String() != "{[c] [{s 2}] <nil> 0}" {
		t.Error("Pop(1) failed")
	}

	// test InsertItem
	v.InsertItem(1, "a", "d", 1)
	buf.Reset()
	fmt.Fprint(&buf, v)
	if buf.String() != "{[c a] [{s 2} {d 1}] <nil> 0}" {
		t.Error("InsertItem(1, \"a\", \"d\", 1) failed")
	}

	// test InsertItemsSlice
	data2 := []string{"i", "j"}
	v2 := ViewList{}
	v2.Init(data2, "t", nil, nil, 0)
	v.InsertItemsSlice(0, v2)
	buf.Reset()
	fmt.Fprint(&buf, v)
	if buf.String() != "{[i j c a] [{t 0} {t 1} {s 2} {d 1}] <nil> 0}" {
		t.Error("InsertItemsSlice(0, v2) failed")
	}

	// test Extend
	v.Extend(v2)
	buf.Reset()
	fmt.Fprint(&buf, v)
	if buf.String() != "{[i j c a i j] [{t 0} {t 1} {s 2} {d 1} {t 0} {t 1}] <nil> 0}" {
		t.Error("Extend(v2) failed")
	}

	// test AppendItem
	v.AppendItem("e", "f", 2)
	buf.Reset()
	fmt.Fprint(&buf, v)
	if buf.String() != "{[i j c a i j e] [{t 0} {t 1} {s 2} {d 1} {t 0} {t 1} {f 2}] <nil> 0}" {
		t.Error("AppendItem(\"e\", \"f\", 2) failed")
	}
}

func TestStringList(t *testing.T) {
	var buf bytes.Buffer

	// init StringList
	data := []string{"abc", "b", "c", "d", "e", "f", "g", "h"}
	s := StringList{}
	s.Init(data, "t", nil, nil, 0)
	fmt.Fprint(&buf, s)
	if buf.String() != "{{[abc b c d e f g h] [{t 0} {t 1} {t 2} {t 3} {t 4} {t 5} {t 6} {t 7}] <nil> 0}}" {
		t.Error("Init StringList failed")
	}

	s.TrimLeft(2, 0, 1)
	buf.Reset()
	fmt.Fprint(&buf, s)
	if buf.String() != "{{[c b c d e f g h] [{t 0} {t 1} {t 2} {t 3} {t 4} {t 5} {t 6} {t 7}] <nil> 0}}" {
		t.Error("TrimEnd(2, 0, 1) failed")
	}

	s.Replace("c", "aa")
	s.Replace("b", "c")
	buf.Reset()
	fmt.Fprint(&buf, s)
	if buf.String() != "{{[aa c aa d e f g h] [{t 0} {t 1} {t 2} {t 3} {t 4} {t 5} {t 6} {t 7}] <nil> 0}}" {
		t.Error("Repalace(\"c\", \"aa\") failed")
	}
}
