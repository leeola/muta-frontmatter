package frontmatter

import (
	"bytes"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// This logBytes func is used to make comparison failures a bit
// easier to read.
func logBytes(s []string, b []byte) []string {
	if b == nil {
		return append(s, "nil")
	} else {
		return append(s, string(b))
	}
}

func TestNewParser(t *testing.T) {
	Convey("Should create a new Parser", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		pairs := [][][]byte{
			[][]byte{[]byte("---"), []byte("---")},
			[][]byte{[]byte("```yaml"), []byte("```")},
		}
		p, err := NewParser(t, pairs...)
		So(err, ShouldBeNil)
		So(p, ShouldNotBeNil)
		So(len(p.seekPairs), ShouldEqual, 2)
	})

	Convey("Should require byte array pairs", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		pairs := [][][]byte{
			[][]byte{[]byte("---")},
		}
		_, err := NewParser(t, pairs...)
		So(err, ShouldNotBeNil)
	})

	Convey("Should append \\n to byte pairs", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		pairs := [][][]byte{
			[][]byte{[]byte("---"), []byte("---")},
			[][]byte{[]byte("```yaml"), []byte("```")},
		}
		p, _ := NewParser(t, pairs...)
		So(p.seekPairs, ShouldResemble, [][][]byte{
			[][]byte{[]byte("---\n"), []byte("\n---\n")},
			[][]byte{[]byte("```yaml\n"), []byte("\n```\n")},
		})
	})
}

func TestParseClose(t *testing.T) {
	Convey("Should buffer while trying to match closing", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t, [][]byte{[]byte("open"), []byte("close")})
		So(p.parseClosing([]byte("c")), ShouldBeNil)
		So(p.parseClosing([]byte("l")), ShouldBeNil)
		So(p.parseClosing([]byte("o")), ShouldBeNil)
		So(p.buffer.String(), ShouldEqual, "clo")
	})

	Convey("Should return the buffer if maxBufferSize is reached", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t, [][]byte{[]byte("open"), []byte("close")})
		p.maxBufferSize = 3
		So(p.parseClosing([]byte("c")), ShouldBeNil)
		So(p.parseClosing([]byte("l")), ShouldBeNil)
		So(string(p.parseClosing([]byte("o"))), ShouldEqual, "clo")
		So(p.buffer.String(), ShouldEqual, "")
		So(p.stage, ShouldEqual, notSeeking)
	})

	Convey("Should ignore maxBufferSize if it is 0", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t, [][]byte{[]byte("open"), []byte("close")})
		p.maxBufferSize = 0
		So(p.parseClosing([]byte("c")), ShouldBeNil)
		So(p.parseClosing([]byte("l")), ShouldBeNil)
		So(p.parseClosing([]byte("o")), ShouldBeNil)
		So(p.buffer.String(), ShouldEqual, "clo")
	})

	Convey("Should return excess bytes when match is found", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t, [][]byte{[]byte("open"), []byte("close")})
		So(p.parseClosing([]byte("this is ")), ShouldBeNil)
		So(p.parseClosing([]byte("frontmatter")), ShouldBeNil)
		So(string(p.parseClosing([]byte("\nclose\nfoo"))), ShouldEqual, "foo")
	})

	Convey("Should buffer the frontmatter when match is found", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t, [][]byte{[]byte("open"), []byte("close")})
		p.parseClosing([]byte("this is "))
		p.parseClosing([]byte("frontmatter"))
		p.parseClosing([]byte("\nclose\n"))
		So(p.buffer.String(), ShouldEqual, "this is frontmatter")
		So(p.ParsedFrontMatter, ShouldBeTrue)
	})

	Convey("Should only look for the pair of pairIndex", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t,
			[][]byte{[]byte("fake"), []byte("fake")},
			[][]byte{[]byte("open"), []byte("close")},
		)
		p.pairIndex = 1
		p.parseClosing([]byte("foo\nfake\nbar"))
		So(p.buffer.String(), ShouldEqual, "foo\nfake\nbar")
	})
}

func TestParseOpening(t *testing.T) {
	Convey("Should buffer while trying to match opening", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t, [][]byte{[]byte("open"), []byte("close")})
		So(p.parseOpening([]byte("o")), ShouldBeNil)
		So(p.parseOpening([]byte("p")), ShouldBeNil)
		So(p.parseOpening([]byte("e")), ShouldBeNil)
		So(p.buffer.String(), ShouldEqual, "ope")
	})

	Convey("Should buffer frontmatter after finding opening", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t, [][]byte{[]byte("open"), []byte("close")})
		So(p.parseOpening([]byte("ope")), ShouldBeNil)
		So(p.parseOpening([]byte("n\nfoo")), ShouldBeNil)
		So(p.buffer.String(), ShouldEqual, "foo")
	})

	Convey("Should set seekStage to seekingClosing when match "+
		"is found", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t, [][]byte{[]byte("open"), []byte("close")})
		So(p.parseOpening([]byte("ope")), ShouldBeNil)
		So(p.parseOpening([]byte("n\nfoo")), ShouldBeNil)
		So(p.stage, ShouldEqual, seekingClosing)
	})

	Convey("Should return all bytes if no match is found", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t, [][]byte{[]byte("open"), []byte("close")})
		So(p.parseOpening([]byte("noto")), ShouldBeNil)
		b := p.parseOpening([]byte("pen\nfoo"))
		So(string(b), ShouldEqual, "notopen\nfoo")
	})
}

func TestParse(t *testing.T) {
	Convey("Should pass through non-frontmatter", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t, [][]byte{[]byte("open"), []byte("close")})
		So(p.Parse([]byte("this")), ShouldBeNil)
		s := string(p.Parse([]byte(" is not frontmatter")))
		So(s, ShouldEqual, "this is not frontmatter")
	})

	Convey("Should capture frontmatter", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t, [][]byte{[]byte("open"), []byte("close")})
		fm := []byte("this is frontmatter")
		out := p.Parse([]byte("open\n"))
		out = append(out, p.Parse(fm)...)
		out = append(out, p.Parse([]byte("\nclose\nand this isn't"))...)
		So(bytes.Contains(out, fm), ShouldBeFalse)
	})

	Convey("Should pass through bytes after frontmatter", t, func() {
		t := func(s string) (interface{}, error) { return nil, nil }
		p, _ := NewParser(t, [][]byte{[]byte("open"), []byte("close")})
		notFm := []byte("and this isn't")
		out := p.Parse([]byte("open\nthis is frontmatter"))
		out = append(out, p.Parse([]byte("\nclose\n"))...)
		out = append(out, p.Parse(notFm)...)
		So(bytes.Contains(out, notFm), ShouldBeTrue)
	})
}

func TestParserFrontMatterData(t *testing.T) {
	Convey("Should return a populated FrontMatterData", t, func() {
		d := []byte(`fmtype: test
foo: bar
`)
		p := Parser{}
		p.buffer.Write(d)
		fmd, err := p.FrontMatterData()
		So(err, ShouldBeNil)
		So(fmd.Type, ShouldEqual, "test")
		So(fmd.Data, ShouldResemble, d)
	})
}

func TestParserFrontMatter(t *testing.T) {
	Convey("Should pass Type to the Typer", t, func() {
		calls := []string{}
		typer := func(s string) (i interface{}, err error) {
			calls = append(calls, s)
			return nil, nil
		}
		d := []byte(`fmtype: t
foo: bar
`)
		p := Parser{typer: typer}
		p.buffer.Write(d)
		p.FrontMatter()
		So(calls, ShouldContain, "t")
	})

	Convey("Should Unmarshal data into the instance that the "+
		"Typer returns", t, func() {
		type T struct {
			Foo string `yaml:"foo"`
		}
		typer := func(s string) (interface{}, error) {
			return &T{}, nil
		}
		d := []byte(`fmtype: t
foo: bar
`)
		p := Parser{typer: typer}
		p.buffer.Write(d)
		i, err := p.FrontMatter()
		So(err, ShouldBeNil)
		So(i, ShouldNotBeNil)
		t := i.(*T)
		So(t.Foo, ShouldEqual, "bar")
	})
}
