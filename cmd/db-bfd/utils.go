package main

import (
	"strings"
)

type StringsFlag []string

func (f *StringsFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *StringsFlag) Set(s string) error {
	*f = strings.Split(s, ",")
	return nil
}
