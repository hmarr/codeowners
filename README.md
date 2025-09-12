# codeowners

![build](https://github.com/robandpdx/gh-codeowners/workflows/build/badge.svg)

A [GitHub CLI](https://cli.github.com/) extension for GitHub's [CODEOWNERS file](https://docs.github.com/en/github/creating-cloning-and-archiving-repositories/about-code-owners#codeowners-syntax).

## Command line tool

The `codeowners` [GitHub CLI](https://cli.github.com/) extension identifies the owners for files in a local repository or directory.

## Installation
```bash
gh extension install robandpdx/gh-codeowners
```

### Usage

By default, the command line tool will walk the directory tree, printing the code owners of any files that are found.

```console
$ gh codeowners --help
usage: codeowners <path>...
  -f, --file string     CODEOWNERS file path
  -h, --help            show this help message
  -o, --owner strings   filter results by owner
  -u, --unowned         only show unowned files (can be combined with -o)

$ ls
CODEOWNERS       DOCUMENTATION.md README.md        example.go       example_test.go

$ cat CODEOWNERS
*.go       @example/go-engineers
*.md       @example/docs-writers
README.md  product-manager@example.com

$ gh codeowners
CODEOWNERS                           (unowned)
README.md                            product-manager@example.com
example_test.go                      @example/go-engineers
example.go                           @example/go-engineers
DOCUMENTATION.md                     @example/docs-writers
```

To limit the files the tool looks at, provide one or more paths as arguments.

```console
$ gh codeowners *.md
README.md                            product-manager@example.com
DOCUMENTATION.md                     @example/docs-writers
```

Pass the `--owner` flag to filter results by a specific owner.

```console
$ gh codeowners -o @example/go-engineers
example_test.go                      @example/go-engineers
example.go                           @example/go-engineers
```

Pass the `--unowned` flag to only show unowned files.

```console
$ gh codeowners -u
CODEOWNERS                           (unowned)
```
