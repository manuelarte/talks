---
# try also 'default' to start simple
theme: seriph
# random image from a curated Unsplash collection by Anthony
# like them? see https://unsplash.com/collections/94734566/slidev
background: https://cover.sli.dev
# some information about your slides (markdown enabled)
title: Welcome to Slidev
info: |
  ## Slidev Starter Template
  Create Your First Linter in Rust.

  Learn more at [Sli.dev](https://sli.dev)
# apply UnoCSS classes to the current slide
class: text-center
# https://sli.dev/features/drawing
drawings:
  persist: false
# slide transition: https://sli.dev/guide/animations.html#slide-transitions
transition: slide-left
# enable Comark Syntax: https://comark.dev/syntax/markdown
comark: true
# duration of the presentation
duration: 35min
---

# Create Your First Linter in Rust

From a Go developer

<div class="abs-br m-6 text-xl">
  <a href="https://github.com/manuelarte/talks/" target="_blank" class="slidev-icon-btn">
    <carbon:logo-github />
  </a>
</div>

<!--
The last comment block of each slide will be treated as slide notes. It will be visible and editable in Presenter Mode along with the slide. [Read more in the docs](https://sli.dev/guide/syntax.html#notes)
-->

---
layout: statement
hideInToc: true
---

# What is a linter?

<div v-click class="bg-gray-100 dark:bg-gray-800 p-8 rounded-xl text-2xl shadow-lg mt-8">
  Automatically analyze source code for potential errors, 
  <span v-mark.circle.red="2">stylistic issues</span>, and violations of coding conventions
</div>

---
layout: center
transition: fade-out
---

<div class="slidev-vclick-target" v-click.fade-in>
    <img src="/among_us.png">
</div>

<style>
.slidev-vclick-target {
  transition: opacity 5000ms ease-out;
}
</style>

---
layout: image-right
image: /qr-github-manuelarte.jpeg
backgroundSize: 20em 60%
hideInToc: true
---

# About Me

```go [me.go] {2|all}
var Me = Developer{
	Name: "Manuel Doncel Martos",
	Skills: [][]string {
        {"☕Java", "Spring Boot"},
        {"🦫Go", "🐍Python"},
        {"Kubernetes", "Docker"},	
    },
    Interests: []string {
        "Open Source",
        "Domain Driven Design",
        "⚽Football",
    },
}
```

---
layout: two-cols-header
transition: slide-down
hideInToc: true
---

# About Me

<br>

::left::

<img src="/gopher.png">

::right::

<img src="/ferris.svg">

---
layout: center
hideInToc: true
---

# Abstract Syntax Tree (AST)

- TODO: In Go really simple
- In Rust you have, AST, but you also have HIR and MIR, which are more complex and powerful

---
transition: slide-up
hideInToc: true
---

# Go

```go
package main

import "fmt"

type User struct {
    Name    string
    Surname string
}

func main() {
	u := User{Name: "John", Surname: "Doe"}
	fmt.Printf("Hello, %+v!\n", u)
}
```

---
transition: slide-up
hideInToc: true
---

# Go

```plantuml
@startwbs
skinparam backgroundColor transparent
skinparam monochrome reverse
!option handwritten true

* ast.File
** ast.GenDecl
*** ast.ImportSpec
**** ast.BasicLit
** ast.GenDecl
*** ast.ValueSpec
**** ast.Ident
**** ast.BasicLit
** ast.GenDecl
*** ast.TypeSpec
*** ast.StructType
**** ast.FieldList
***** ast.Field
***** ast.Field
** ast.FuncDecl
*** ast.FuncType
*** ast.FieldList
**** ast.Field
**** ast.Field
** ...
@endwbs
```

---
transition: slide-up
hideInToc: true
---

# Rust

<v-click>
<img src="/star-trek-desperate.avif">
</v-click>

---

# Rust

- AST
- Mid-Level Intermediate Representation (MIR)
- High-level Intermediate Representation (HIR)

---
transition: slide-up
level: 2
---

# Rust

<div class="text-sm">

```rust
fn main() {
    let john = User::new("John", "Doe");
    println!("Hello, {:?}!", john);
}

#[derive(Debug)]
struct User {
    name: String,
    surname: String,
}

impl User {
    pub fn new(name: impl Into<String>, surname: impl Into<String>) -> Self {
        User {
            name: name.into(),
            surname: surname.into(),
        }
    }
}
```

</div>

---

# TODO

- Explain the linter we are going to build.
- Prepare the code to build it live.
- Explain clippy and dylint.

---
layout: center
class: text-center
---

# Learn More

[Documentation](https://sli.dev) · [GitHub](https://github.com/slidevjs/slidev) · [Showcases](https://sli.dev/resources/showcases)

<PoweredBySlidev mt-10 />
