Blackfriday Text
================

Package blackfridaytext contains an experimental text renderer for the
Blackfriday Markdown Processor http://github.com/russross/blackfriday.

Example
-------

    package main

    import (
        "github.com/gholt/blackfridaytext"
        "io/ioutil"
        "os"
    )

    func main() {
        markdown, _ := ioutil.ReadAll(os.Stdin)
        metadata, output := blackfridaytext.MarkdownToText(markdown)
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


License
-------

Blackfriday Text is distributed under the Simplified BSD License:

> Copyright © 2014 Gregory Holt  
> All rights reserved.
> 
> Redistribution and use in source and binary forms, with or without
> modification, are permitted provided that the following conditions
> are met:
> 
> 1.  Redistributions of source code must retain the above copyright
>     notice, this list of conditions and the following disclaimer.
> 
> 2.  Redistributions in binary form must reproduce the above
>     copyright notice, this list of conditions and the following
>     disclaimer in the documentation and/or other materials provided with
>     the distribution.
> 
> THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
> "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
> LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS
> FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE
> COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT,
> INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING,
> BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
> LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
> CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
> LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN
> ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
> POSSIBILITY OF SUCH DAMAGE.
