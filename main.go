// Copyright 2022 Manlio Perillo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate pigeon -o peg.go peg.peg

// pegcmp command compares two parsing expression grammar (PEG).
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

type Rule struct {
	Name string
	Expr string
	Text string
	Pos  Pos
}

type Pos struct {
	Filename string
	Line     int
	Col      int
	Offset   int
}

var errDuplicateRule = errors.New("duplicate rule")

const usage = "Usage: pegcmp lhs-path rhs-path"

func main() {
	// Setup log.
	log.SetFlags(0)

	// Parse command line.
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()

		os.Exit(2)
	}
	lpath := flag.Arg(0)
	rpath := flag.Arg(1)

	// Parse and compare the lhs and rhs grammars.
	lgrammar, err := parse(lpath)
	if err != nil {
		log.Fatal(err)
	}
	rgrammar, err := parse(rpath)
	if err != nil {
		log.Fatal(err)
	}

	// Use the lhs grammar as reference, assuming that it is a valid PEG grammar.
	rules := make(map[string]Rule)
	for _, lrule := range lgrammar {
		rules[lrule.Name] = lrule
	}

	// Check for duplicates in the rhs grammar.
	if err := validate(rpath, rgrammar); err != nil {
		log.Fatal(err)
	}

	// Compare each rhs rule against lhs.
	for _, rrule := range rgrammar {
		lrule, ok := rules[rrule.Name]
		if !ok {
			fmt.Fprintf(os.Stderr, "! rule %q not found\n", rrule.Name)
			fmt.Fprintf(os.Stderr, "> %s:%d:%d\n", rpath, rrule.Pos.Line, rrule.Pos.Col)
			fmt.Fprintln(os.Stderr, ">", rrule.Expr, "\n")

			continue
		}

		// Rule expressions are compared byte by byte, including whitespace.
		if rrule.Expr != lrule.Expr {
			fmt.Fprintf(os.Stderr, "! rule %q does not match\n", rrule.Name)
			fmt.Fprintf(os.Stderr, "> %s:%d:%d\n", rpath, rrule.Pos.Line, rrule.Pos.Col)
			fmt.Fprintln(os.Stderr, ">", rrule.Expr, "\n")
			fmt.Fprintf(os.Stderr, "< %s:%d:%d\n", lpath, lrule.Pos.Line, lrule.Pos.Col)
			fmt.Fprintln(os.Stderr, "<", lrule.Expr, "\n")
		}
	}
}

func parse(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pn, err := Parse(path, data)
	if err != nil {
		return nil, err
	}

	// Convert interface to concrete type.
	slice := pn.([]interface{})
	rules := make([]Rule, len(slice))
	for i, ent := range slice {
		rules[i] = ent.(Rule)
	}

	return rules, nil
}

func validate(path string, grammar []Rule) error {
	var err error = nil
	rules := make(map[string]Rule)
	for _, rule := range grammar {
		if prule, ok := rules[rule.Name]; ok {
			// Ignore identical duplicate rules.
			if rule.Expr != prule.Expr {
				fmt.Fprintf(os.Stderr, "! duplicate rule %q does not match\n", prule.Name)
				fmt.Fprintf(os.Stderr, "> %s:%d:%d\n", path, rule.Pos.Line, rule.Pos.Col)
				fmt.Fprintln(os.Stderr, ">", rule.Expr, "\n")
				fmt.Fprintf(os.Stderr, "< %s:%d:%d\n", path, prule.Pos.Line, prule.Pos.Col)
				fmt.Fprintln(os.Stderr, "<", prule.Expr, "\n")

				err = errDuplicateRule
			}
		}

		rules[rule.Name] = rule
	}

	return err
}

// strip removes leading and trailing white space and comments.
func strip(s string) string {
	if idx := strings.IndexByte(s, '#'); idx < 0 {
		return strings.TrimSpace(s)
	}

	// Remove comments, assuming eol is LF.
	var b strings.Builder
	for i := 0; ; i++ {
		start := strings.IndexByte(s, '#')
		if start < 0 {
			return strings.TrimSpace(b.String())
		}
		end := strings.IndexByte(s[start:], '\n')
		if end < 0 {
			panic("unterminated comment")
		}

		tmp := s[0:start] + s[start+end+1:]
		b.WriteString(tmp)

		s = tmp
	}

	panic("unreachable")
}
