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
}
