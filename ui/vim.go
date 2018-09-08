package ui

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type vim struct {
	mode Mode
	buf  []rune
}

type Mode int

const (
	command Mode = iota
	insert
	lastLine
)

func newVim() *vim {
	return &vim{buf: make([]rune, 0, 1024)}
}

func (v *vim) handle(r rune) (finished bool, out []string, isCmd bool, err error) {
	if v == nil {
		err = errors.New("vim.Scan: v is nil")
		finished = true
		return
	}

	if !utf8.ValidRune(r) {
		r = utf8.RuneError
	}

	switch v.mode {
	case command:
		switch r {
		case ':':
			v.mode = lastLine
		case 'i':
			v.mode = insert
		case '\x1b':
			//esc do nothing
		default:
			err = errors.New("invalid mode input")
			finished = true
		}

	case insert:
		switch r {
		case utf8.RuneError:
			err = errors.New("invalid rune")
			finished = true
			v.buf = v.buf[:0]
		case '\x1b':
			v.mode = command
			v.buf = v.buf[:0]
		case '\n':
			out = []string{string(v.buf)}
			isCmd = false
			finished = true
			v.buf = v.buf[:0]
		case '\x08':
			//backspace
			if len(v.buf) > 0 {
				v.buf = v.buf[:len(v.buf)-1]
			}
		default:
			v.buf = append(v.buf, r)
		}

	case lastLine:
		switch r {
		case utf8.RuneError:
			err = errors.New("invalid rune")
			finished = true
			v.buf = v.buf[:0]
			v.mode = command
		case '\x1b':
			v.mode = command
			v.buf = v.buf[:0]
		case '\n':
			out = strings.Fields(string(v.buf))
			isCmd = true
			finished = true
			v.buf = v.buf[:0]
			v.mode = command
		case '\x08':
			//backspace
			if len(v.buf) > 0 {
				v.buf = v.buf[:len(v.buf)-1]
			} else {
				v.mode = command
			}
		default:
			v.buf = append(v.buf, r)
		}

	default:
		err = fmt.Errorf("invalid mode %d", v.mode)
		v.mode = command
		v.buf = v.buf[:0]
		finished = true
	}
	return
}
