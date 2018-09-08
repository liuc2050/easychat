package ui

import (
	"testing"

	"github.com/liuc2050/easychat/util"
)

type TStep struct {
	in     string
	out    []string
	isCmd  bool
	errNil bool
	mode   Mode
}
type TCase struct {
	sc    *vim
	steps []TStep
}

func TestScan(t *testing.T) {
	tests := []TCase{
		{newVim(), []TStep{{":create 8081\n", []string{"create", "8081"}, true, true, command}}},
		{newVim(), []TStep{{"iHello world!\n", []string{"Hello world!"}, false, true, insert}}},
		{newVim(), []TStep{{":leave\n", []string{"leave"}, true, true, command}}},
		{newVim(), []TStep{{":crea\x1bistill here\n", []string{"still here"}, false, true, insert}}},
		{newVim(), []TStep{{"\n", []string{}, false, false, command}}},
		{newVim(), []TStep{{"idxk伯不可靠\uFFFD", []string{}, false, false, insert}}},
		{newVim(), []TStep{{":bye\uFFFD", []string{}, false, false, command}}},
		{newVim(), []TStep{
			{":create 8081\n", []string{"create", "8081"}, true, true, command},
			{"ienter :the server\n", []string{"enter :the server"}, false, true, insert},
			{"\x1b:leave\n", []string{"leave"}, true, true, command},
			{":bye\n", []string{"bye"}, true, true, command},
		}},
		{newVim(), []TStep{
			{"i\n", []string{""}, false, true, insert},
			{"s:::kdfjd\x1b:q!\n", []string{"q!"}, true, true, command},
		}},
	}

	for i, test := range tests {
		sc := test.sc
		for n, step := range test.steps {
			in := step.in
			_, s, cmd, err := scan(sc, in)
			if !util.StringSliceEqual(s, step.out) || cmd != step.isCmd || (err == nil) != step.errNil || (sc != nil && sc.mode != step.mode) {
				t.Errorf("test%d step%d input:%q output:%#v, %v, %v, %v   want:%#v", i, n, step.in, s, cmd, err, sc.mode, step)
			}
		}
	}
}

func scan(v *vim, s string) (finished bool, out []string, isCmd bool, err error) {
	for _, r := range s {
		finished, out, isCmd, err = v.handle(r)
		if err != nil {
			return
		}
	}
	return
}

func TestnewVim(t *testing.T) {
	v := newVim()
	if v.mode != command {
		t.Errorf("mode %d, want %d", v.mode, command)
	}
}
