# Blackfriday Text
## A text renderer for the [Blackfriday Markdown Processor](http://github.com/russross/blackfriday).

[API Documentation](http://godoc.org/github.com/gholt/blackfridaytext)

This is the latest development area for the package.  
For the latest stable version of the package, switch to the
[v1 branch](https://github.com/gholt/blackfridaytext/tree/v1)  
or use `gopkg.in/gholt/brimtext.v1` as the import path.  
Also, you'd want to use the
[V1 API Documentation](http://godoc.org/gopkg.in/gholt/blackfridaytext.v1).

> Copyright Gregory Holt. All rights reserved.  
> Use of this source code is governed by a BSD-style  
> license that can be found in the LICENSE file.

## Example Code

```go
package main

import (
    "io/ioutil"
    "os"

    "github.com/gholt/blackfridaytext"
)

func main() {
    opt := &blackfridaytext.Options{Color: true}
    if len(os.Args) == 2 && os.Args[1] == "--no-color" {
        opt.Color = false
    }
    markdown, _ := ioutil.ReadAll(os.Stdin)
    metadata, output := blackfridaytext.MarkdownToText(markdown, opt)
    for _, item := range metadata {
        name, value := item[0], item[1]
        os.Stdout.WriteString(name)
        os.Stdout.WriteString(":\n    ")
        os.Stdout.WriteString(value)
        os.Stdout.WriteString("\n")
    }
    os.Stdout.WriteString("\n")
    os.Stdout.Write(output)
    os.Stdout.WriteString("\n")
}
```

---

## Sample Input

To give an idea of what the output looks like, I've run this document through
the renderer and appended the output.

 *  Here is a sample list.
 *  Two
     *  And a sublist.
     *  Two, part B.
 *  Three

*Emphasis*, **Double Emphasis**, ***Triple Emphasis***, ~~Strikethrough~~, and `code spans` along with http://auto/linking and [explicit linking](http://explicit/linking).

Here's a quick table from the blackfriday example:

Name  | Age
------|----
Bob   | 27
Alice | 23

---

## No-Color Output

```
--[ Blackfriday Text ]--

    --[ A text renderer for the [Blackfriday Markdown Processor]
        http://github.com/russross/blackfriday. ]--

        [API Documentation] http://godoc.org/github.com/gholt/blackfridaytext

        This is the latest development area for the package.
        For the latest stable version of the package, switch to the [v1
        branch] https://github.com/gholt/blackfridaytext/tree/v1
        or use "gopkg.in/gholt/brimtext.v1" as the import path.
        Also, you'd want to use the [V1 API Documentation]
        http://godoc.org/gopkg.in/gholt/blackfridaytext.v1.

        > Copyright Gregory Holt. All rights reserved.
        > Use of this source code is governed by a BSD-style
        > license that can be found in the LICENSE file.

    --[ Example Code ]--

        package main

        import (
            "io/ioutil"
            "os"

            "github.com/gholt/blackfridaytext"
        )

        func main() {
            opt := &blackfridaytext.Options{Color: true}
            if len(os.Args) == 2 && os.Args[1] == "--no-color" {
                opt.Color = false
            }
            markdown, _ := ioutil.ReadAll(os.Stdin)
            metadata, output := blackfridaytext.MarkdownToText(markdown, opt)
            for _, item := range metadata {
                name, value := item[0], item[1]
                os.Stdout.WriteString(name)
                os.Stdout.WriteString(":\n    ")
                os.Stdout.WriteString(value)
                os.Stdout.WriteString("\n")
            }
            os.Stdout.WriteString("\n")
            os.Stdout.Write(output)
            os.Stdout.WriteString("\n")
        }

        -----------------------------------------------------------------------

    --[ Sample Input ]--

        To give an idea of what the output looks like, I've run this document
        through the renderer and appended the output.
          * Here is a sample list.
          * Two
              * And a sublist.
              * Two, part B.
          * Three

        *Emphasis*, **Double Emphasis**, ***Triple Emphasis***,
        ~~Strikethrough~~, and "code spans" along with http://auto/linking and
        [explicit linking] http://explicit/linking.

        Here's a quick table from the blackfriday example:

        +-------+-----+
        | Name  | Age |
        +-------+-----+
        | Bob   | 27  |
        | Alice | 23  |
        +-------+-----+
```

---

## Color Output

![](screenshot.png)
