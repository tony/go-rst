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
}
