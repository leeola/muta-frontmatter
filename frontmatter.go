package frontmatter

import (
	"bytes"
	"errors"
	"io/ioutil"
	"path/filepath"

	"github.com/leeola/muta"
	"gopkg.in/yaml.v2"
)

// Typer returns a struct reference to represent the yaml/json to be
// generated.
type Typer func(string) (interface{}, error)

//
type FrontMatterData struct {
	Type string `yaml:"fmtype"`
	Data []byte
}

type seekStage uint

const (
	seekingOpening seekStage = iota
	seekingClosing seekStage = iota
	notSeeking     seekStage = iota
)

func NewParser(t Typer, pairs ...[][]byte) (Parser, error) {
	for _, pair := range pairs {
		if len(pair) != 2 {
			return Parser{}, errors.New("Frontmatter bytes must be given in " +
				"opening and closing pairs")
		}
	}

	var largestOpen int = 0
	for i, pair := range pairs {
		pOpen := append(pair[0], '\n')
		pClose := append([]byte{'\n'}, append(pair[1], '\n')...)
		pairs[i] = [][]byte{pOpen, pClose}

		if len(pOpen) > largestOpen {
			largestOpen = len(pOpen)
		}
	}

	p := Parser{
		seekPairs:     pairs,
		typer:         t,
		largestOpen:   largestOpen,
		maxBufferSize: 0,
	}
	p.Reset()
	return p, nil
}

type Parser struct {
	// A buffer used to buffer the contents of the frontmatter block.
	buffer     bytes.Buffer
	lineBuffer bytes.Buffer

	// The opening and closing bytes for the ending of the frontmatter
	// block.
	//
	// Example:
	// FrontmatterParser{
	//	SeekPairs: [][][]byte{
	//		[][]byte{[]byte("---"), []byte("---")},
	//		[][]byte{[]byte("```yaml"), []byte("```")},
	//	}
	// }
	seekPairs [][][]byte
	// When parseOpening finds a pair match, this is set to the index
	// of the pair. parseClosing then only looks for a the closing pair
	// of this index
	pairIndex int

	// The openParser only seeks for this length. Once this length (or more)
	// is found, it compares if the buffer starts with any of the pair
	// openings.
	largestOpen int

	// The maximum size that this Parser will buffer, while looking for
	// the closing pair
	maxBufferSize int

	typer Typer

	stage seekStage

	ParsedFrontMatter bool
}

// Get the FrontMatterData by Unmarshaling the buffer data.
func (p *Parser) FrontMatterData() (*FrontMatterData, error) {
	d, _ := ioutil.ReadAll(&p.buffer)
	// First, Marshal the yaml to get the fmtype
	fmd := &FrontMatterData{}
	err := yaml.Unmarshal(d, fmd)
	if err != nil {
		return nil, err
	}
	fmd.Data = d
	return fmd, nil
}

func (p *Parser) FrontMatter() (frontmatter interface{}, err error) {
	fmd, err := p.FrontMatterData()
	if err != nil {
		return nil, err
	}

	t, err := p.typer(fmd.Type)
	if t == nil || err != nil {
		return t, err
	}

	err = yaml.Unmarshal(fmd.Data, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (p *Parser) parseClosing(chunk []byte) []byte {
	// The buffer write/read/check/write method below is an insanely
	// naive implementation, written to make tests pass. This needs
	// to be improved, though i'm not sure how at the moment.
	p.buffer.Write(chunk)
	b, _ := ioutil.ReadAll(&p.buffer)
	pair := p.seekPairs[p.pairIndex]
	i := bytes.Index(b, pair[1])
	if i >= 0 {
		// Note that this split ignores multiple closing brackets,
		// but for now that is okay. We *probably* only care about the
		// first one anyway.
		chunks := bytes.SplitN(b, pair[1], 2)
		// Write the frontmatter back into the buffer, for use in the typer
		p.buffer.Write(chunks[0])
		p.ParsedFrontMatter = true
		p.stage = notSeeking
		// Return any excess data *after* the close.
		return chunks[1]
	}
	// No match was found, add the data back to the buffer if it's smaller
	// than the max buff size
	if p.maxBufferSize == 0 || len(b) < p.maxBufferSize {
		p.buffer.Write(b)
		return nil
	} else {
		// If our maxBufferSize is exceeded, return the bytes.
		p.stage = notSeeking
		return b
	}
}

func (p *Parser) parseOpening(chunk []byte) []byte {
	p.buffer.Write(chunk)

	// We can't look for the pair openings until we've buffered atleast
	// as much as the largest opening.
	if p.buffer.Len() < p.largestOpen {
		return nil
	}

	chunk, _ = ioutil.ReadAll(&p.buffer)

	for _, pair := range p.seekPairs {
		if bytes.HasPrefix(chunk, pair[0]) {
			// We found the opening, so set this Parser to start
			// seeking the closing.
			p.stage = seekingClosing
			// Remove the pair[0] from the chunk, so we can pass the
			// excess data into parseClosing.
			return p.parseClosing(chunk[len(pair[0]):])
		}
	}

	// We did not find the opening, so set the parser to not seeking.
	p.stage = notSeeking
	// Return all of the data, since it does not match any opening
	return chunk
}

// Parse incoming byte chunks for an opening and closing pair
// of bytes. Once a closing pair is found, the data inbetween the
// bytes is marshalled with the help of the given Typer. All
// other data is returned.
func (p *Parser) Parse(chunk []byte) []byte {
	if chunk == nil {
		return nil
	}

	switch p.stage {
	case seekingOpening:
		return p.parseOpening(chunk)
	case seekingClosing:
		return p.parseClosing(chunk)
	default:
		return chunk
	}
}

func (p *Parser) Reset() {
	p.stage = seekingOpening
}

func FrontMatter(typer Typer) muta.Streamer {
	// Our pairs of opening and closing bytes
	// Hardcoded for now.
	pairs := [][][]byte{
		[][]byte{[]byte("---"), []byte("---")},
		[][]byte{[]byte("```yaml"), []byte("```")},
	}

	p, _ := NewParser(typer, pairs...)
	return func(fi *muta.FileInfo, chunk []byte) (*muta.FileInfo,
		[]byte, error) {

		switch {
		case fi == nil:
			return nil, nil, nil

		case filepath.Ext(fi.Name) != ".md":
			return fi, chunk, nil

		case chunk == nil:
			// We're at the EOF, reset our parser
			p.Reset()
			return fi, nil, nil

		default:
			parsedChunk := p.Parse(chunk)

			// If parsedChunk is nil, then Parse is buffering
			if parsedChunk == nil {
				return nil, nil, nil
			}

			if p.ParsedFrontMatter && fi.Ctx["frontmatter"] == nil {
				fm, err := p.FrontMatter()
				if err != nil {
					return fi, parsedChunk, err
				}
				fi.Ctx["frontmatter"] = fm
			}

			return fi, parsedChunk, nil
		}
	}
}
