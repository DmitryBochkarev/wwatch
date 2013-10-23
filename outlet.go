/*
original author David Dollar(https://github.com/ddollar)
github.com/ddollar/forego

The MIT License (MIT)

Copyright (c) 2013 David Dollar

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/daviddengcn/go-colortext"
	"sync"
)

type OutletFactory struct {
	Outlets map[string]*Outlet
	Padding int
	mx      sync.Mutex
}

var colors = []ct.Color{
	ct.Cyan,
	ct.Yellow,
	ct.Green,
	ct.Magenta,
	ct.Red,
	ct.Blue,
}

func NewOutletFactory() (of *OutletFactory) {
	of = new(OutletFactory)
	of.Outlets = make(map[string]*Outlet)
	return
}

func (of *OutletFactory) CreateOutlet(name string, index int, isError bool) *Outlet {
	of.Outlets[name] = &Outlet{name, colors[index%len(colors)], isError, of}
	if of.Padding < len(name) {
		of.Padding = len(name)
	}
	return of.Outlets[name]
}

func (of *OutletFactory) Write(b []byte) (num int, err error) {
	of.mx.Lock()
	defer of.mx.Unlock()
	ct.ChangeColor(ct.White, true, ct.None, false)
	formatter := fmt.Sprintf("%%-%ds | ", of.Padding)
	fmt.Printf(formatter, "wwatch")
	ct.ResetColor()
	fmt.Print(string(b))
	ct.ResetColor()
	num = len(b)
	return
}

type Outlet struct {
	Name    string
	Color   ct.Color
	IsError bool
	Factory *OutletFactory
}

func (o *Outlet) Write(b []byte) (num int, err error) {
	o.Factory.mx.Lock()
	defer o.Factory.mx.Unlock()
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		formatter := fmt.Sprintf("%%-%ds | ", o.Factory.Padding)
		ct.ChangeColor(o.Color, true, ct.None, false)
		fmt.Printf(formatter, o.Name)
		if o.IsError {
			ct.ChangeColor(ct.Red, true, ct.None, true)
		} else {
			ct.ResetColor()
		}
		fmt.Println(scanner.Text())
		ct.ResetColor()
	}
	num = len(b)
	return
}
