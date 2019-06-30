package peekconn

import (
	"bytes"
	"io/ioutil"
	"regexp"
	"testing"
)

var (
	testRegex = regexp.MustCompile(`^\x16\x03[\00-\x03]`)
)

func TestPeeker(t *testing.T) {
	cases := map[string]struct {
		data  string
		regex *regexp.Regexp
		len   int
		match bool
	}{
		"no match": {
			"no matching string",
			testRegex,
			3,
			false,
		},
		"match": {
			"\x16\x03\x00hellothere",
			testRegex,
			3,
			true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			conn := New(mockConn{bytes.NewBufferString(tc.data)})
			match, err := conn.PeekMatch(tc.regex, tc.len)
			if err != nil {
				t.Fatal("unexpected error:", err)
			}
			if match != tc.match {
				t.Fatalf("mismatch, expected %v but got %v", tc.match, match)
			}

			first := make([]byte, 1)
			_, err = conn.Read(first)
			if err != nil {
				t.Fatal("unexpected error:", err)
			}
			if first[0] != tc.data[0] {
				t.Fatalf("mismatch, expected %v but got %v", tc.match, match)
			}

			b, err := ioutil.ReadAll(conn)
			if err != nil {
				t.Fatal("unexpected error:", err)
			}
			if string(append(first, b...)) != tc.data {
				t.Fatalf("read data (%s%s) didn't match original (%s)", string(first), string(b), tc.data)
			}
		})
	}
}

func TestMultiplePeekers(t *testing.T) {
	conn := New(mockConn{bytes.NewBufferString("Hello world!")})
	match, err := conn.PeekMatch(regexp.MustCompile("Hell"), 4)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if !match {
		t.Fatalf("mismatch, expected true but got false")
	}

	conn = New(conn)
	match, err = conn.PeekMatch(regexp.MustCompile("Hello"), 5)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if !match {
		t.Fatalf("mismatch, expected true but got false")
	}
}
