
# Muta Frontmatter

A muta Streamer to pull Frontmatter out of Markdown files, parse the data 
*(yaml only, currently)*, and put the generated data into the FileInfo 
Context.

## Example

```go
type BlogPost struct {
  Title   string `yaml:"title"`
  Author  string `yaml:"autor"`
}

func fmTyper(st string) (t *interface{}, err error) {
  switch st {
  case "blogpost":
    t = BlogPost{}
  default:
    err = errors.New(fmt.Sprintf("Unknown FrontMatter type '%s'", st))
  }
}

func main() {
  muta.Task("markdown", func() (*muta.Stream, error) {
    s := muta.Src("./*.md").
      Pipe(frontmatter.FrontMatter(fmTyper)).
      Pipe(muta.Dest("./build"))
    return s, nil
  })

  muta.Task("default", "markdown")
  muta.Te()
}
```

In this example, `frontmatter.FrontMatter(fmTyper)` will parse all files 
beginning with one of the following two blocks:

    ---
    fmtype: blogpost
    title: foo
    author: me
    ---

or

    ```yaml
    fmtype: blogpost
    title: foo
    author: me
    ```

It will then call the `fmTyper` func with the yaml value of `fmtype`.  
The purpose of this is to allow you, the FrontMatter caller, to return
a struct with fields that coorispond to the FrontMatter yaml.

After Unmarshalling the FrontMatter, `FileInfo.Ctx["frontmatter"]` will
contain the struct that `Typer` returned.
