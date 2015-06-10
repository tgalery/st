package linking

import (
	"encoding/json"
	dumpparser "github.com/semanticize/st/cmd/semanticizest-dumpparser/internal"
	"github.com/semanticize/st/hash"
	"github.com/semanticize/st/hash/countmin"
	"github.com/semanticize/st/storage"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"
)

var sem = makeSemanticizer()

func TestBestPath(t *testing.T) {
	sem.BestPath("   ") // should not crash
}

func makeSemanticizer() Semanticizer {
	cm, _ := countmin.New(10, 4)
	db, _ := storage.MakeDB(":memory:", true, &storage.Settings{MaxNGram: 2})
	sem := Semanticizer{db, cm, 2}

	for _, h := range hash.NGrams([]string{"Hello", "world"}, 2, 2) {
		_, err := db.Exec(`insert into linkstats values (?, 0, 1)`, h)
		if err == nil {
			_, err = db.Exec(`insert into titles values (0, "dmr")`)
		}
		if err != nil {
			panic(err)
		}
	}
	return sem
}

func TestCandidates(t *testing.T) {
	all, err := sem.All("Hello world")
	if err != nil {
		t.Error(err)
	}
	if len(all) != 1 {
		t.Errorf("expected one entity mention, got %v", all)
	} else if tgt := all[0].Target; tgt != "dmr" {
		t.Errorf(`expected target "dmr", got %q`, tgt)
	}
	if all[0].LinkCount == 0 {
		t.Errorf("LinkCount is zero")
	}
}

func TestExactMatch(t *testing.T) {
	all, err := sem.ExactMatch("Hello world")
	if err != nil {
		t.Error(err)
	}
	if len(all) != 1 {
		t.Errorf("expected one entity mention, got %v", all)
	}

	all, err = sem.ExactMatch("Hello world program")
	if err != nil {
		t.Error(err)
	}
	if len(all) != 0 {
		t.Errorf("expected no entity mentions, got %v", all)
	}
}

func TestJSON(t *testing.T) {
	in := Entity{"Wikipedia", 4, 10, .9, 0.0115, 0, 9}
	enc, _ := json.Marshal(in)

	var got Entity
	json.Unmarshal(enc, &got)

	if !reflect.DeepEqual(in, got) {
		t.Errorf("marshalled %v, got %v", in, got)
	}

	enc = []byte(
		`{"offset": 0,"target":"Wikipedia", "commonness":0.9,"ngramcount": 4 ,
		  "linkcount": 10, "length": 9,"senseprob":0.0115}`)
	err := json.Unmarshal(enc, &got)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(got, in) {
		t.Errorf("could not unmarshal %q, got %v", enc, got)
	}
}

func TestViterbi(t *testing.T) {
	cands := []Entity{
		{Target: "foo", Offset: 4, Length: 6, Senseprob: .8},
		{Target: "bar", Offset: 3, Length: 7, Senseprob: .9},
		{Target: "baz", Offset: 1, Length: 2, Senseprob: .1},
	}
	best := bestPath(cands)
	if len(best) != 2 {
		t.Errorf("too many entities in path: %d (wanted 2)", len(best))
	}
	for _, e := range best {
		if e.Target != "foo" && e.Target != "baz" {
			t.Errorf("unexpected entity %q in best path", e.Target)
		}
	}
}

type tWriter struct {
	t *testing.T
}

func (w tWriter) Write(p []byte) (n int, err error) {
	w.t.Logf("%s", p)
	return len(p), nil
}

func testLogger(t *testing.T) *log.Logger {
	return log.New(tWriter{t}, "", 0)
}

func makeSemanticizer2(t *testing.T) (dbname string, sem *Semanticizer,
	settings *storage.Settings, err error) {

	dumppath := "nlwiki-20140927-sample.xml"

	dbfile, err := ioutil.TempFile("", "semanticizer")
	if err != nil {
		return
	}
	dbname = dbfile.Name()

	err = dumpparser.Main(dbname, dumppath, "", countmin.MaxRows, 32, 7,
		testLogger(t))
	if err != nil {
		return
	}
	sem, settings, err = Load(dbname)
	return
}

func TestEndToEnd(t *testing.T) {
	dbname, sem, settings, err := makeSemanticizer2(t)
	_ = settings
	defer os.Remove(dbname)
	if err != nil {
		t.Fatal(err)
	}
	all, err := sem.All("Antwerpen")
	if len(all) == 0 {
		t.Error(`expected to get candidate entities for "Antwerpen"`)
	}
	for _, entity := range all {
		t.Logf("%v", entity)
	}
}
