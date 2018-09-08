package ui

import (
	"log"

	runewidth "github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

/*
+----------------------------+
|    Notification Area       |
|                            |
|                            |
|                            |
|                            |
|                            |
|                            |
|                            |
|                            |
|                            |
|                            |
|                            |
|                            |
|                            |
|                            |
|                            |
+----------------------------+
|  Input Area(text/command)  |
+----------------------------+
*/

type termui struct {
	notifyCh chan string
	inputCh  chan echo
	lock     chan bool
	isInit   bool
	v        *vim
	logger   *log.Logger
}

var ui termui

type echo struct {
	text       string
	hideCursor bool
}

func Init(l *log.Logger) {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
	termbox.SetInputMode(termbox.InputEsc)
	ui.notifyCh = make(chan string, 1024)
	ui.inputCh = make(chan echo, 1)
	ui.lock = make(chan bool, 1)
	termbox.HideCursor()
	termbox.Flush()
	ui.v = newVim()
	ui.logger = l
	ui.isInit = true
}

func Close() {
	close(ui.notifyCh)
	close(ui.inputCh)
	defer lockFunc()()
	termbox.Close()
	ui.v = nil
	ui.logger = nil
	ui.isInit = false
}

func Draw() {
	for {
		ui.lock <- true
		if !ui.isInit {
			<-ui.lock
			return
		}
		select {
		case msg, ok := <-ui.notifyCh:
			if !ok {
				<-ui.lock
				return
			}
			scrollDownNotice(msg)
		case echoText, ok := <-ui.inputCh:
			if !ok {
				<-ui.lock
				return
			}
			refreshInputArea(echoText)
		}
		termbox.Flush()
		<-ui.lock
	}
}

func lockFunc() func() {
	ui.lock <- true
	return func() {
		<-ui.lock
	}
}

func scrollDownNotice(newStr string) {
	if len(newStr) == 0 {
		return
	}
	w, h := termbox.Size()
	cells := string2Cell(newStr, w)
	msgArea := termbox.CellBuffer()[:w*(h-1)]
	if len(cells) >= len(msgArea) {
		copy(msgArea, cells[len(cells)-len(msgArea):])
	} else {
		copy(msgArea[:len(msgArea)-len(cells)], msgArea[len(cells):])
		copy(msgArea[len(msgArea)-len(cells):], cells)
	}
}

func refreshInputArea(echoText echo) {
	w, h := termbox.Size()
	cells := termbox.CellBuffer()[w*(h-1):]
	rs := []rune(echoText.text)
	j := len(cells) - 2
	for i := len(rs) - 1; i >= 0 && j >= 0; i-- {
		w := runewidth.RuneWidth(rs[i])
		if w == 0 || (w == 2 && runewidth.IsAmbiguousWidth(rs[i])) {
			w = 1
		}
		if j < w-1 {
			for ; j >= 0; j-- {
				cells[j] = termbox.Cell{}
			}
			break
		}
		for k := j; k > j-w+1; k-- {
			cells[k] = termbox.Cell{}
		}
		cells[j-w+1] = termbox.Cell{Ch: rs[i]}
		j = j - w
	}
	if j >= 0 {
		copy(cells, cells[j+1:len(cells)-1])
		for k := len(cells) - 1 - j - 1; k < len(cells); k++ {
			cells[k] = termbox.Cell{}
		}

		if echoText.hideCursor {
			termbox.HideCursor()
		} else {
			termbox.SetCursor(len(cells)-1-j-1, h-1)
		}
		return
	}
	if echoText.hideCursor {
		termbox.HideCursor()
	} else {
		termbox.SetCursor(w-1, h-1)
	}
}

func string2Cell(s string, width int) []termbox.Cell {
	cells := make([]termbox.Cell, 0, len(s))
	var x int
	for _, r := range s {
		if r == '\n' {
			if x > 0 {
				cells = append(cells, make([]termbox.Cell, width-x)...)
			}
			x = 0
			continue
		}
		w := runewidth.RuneWidth(r)
		if w == 0 || (w == 2 && runewidth.IsAmbiguousWidth(r)) {
			w = 1
		}
		if x+w > width {
			cells = append(cells, make([]termbox.Cell, width-x)...)
			x = 0
		}
		cells = append(cells, termbox.Cell{Ch: r})
		if w > 1 {
			cells = append(cells, make([]termbox.Cell, w-1)...)
		}
		x += w
		if x == width {
			x = 0
		}
	}
	if x > 0 && x < width {
		cells = append(cells, make([]termbox.Cell, width-x)...)
	}
	return cells
}

func Notify(s string) {
	ui.notifyCh <- s
}

//Scan 读取输入然后识别命令或普通文本
//out包含命令名及参数(isCmd为true)或者普通文本（isCmd为false），isCmd是否为命令
func Scan() (out []string, isCmd bool, err error) {
	for {
		ev := termbox.PollEvent()
		switch ev.Type {
		case termbox.EventInterrupt:
			return
		case termbox.EventError:
			err = ev.Err
			return
		case termbox.EventKey:
			r := ev.Ch
			if ev.Key == termbox.KeyEnter {
				r = '\n'
			} else if ev.Key == termbox.KeyEsc {
				r = '\x1b'
			} else if ev.Key == termbox.KeySpace {
				r = ' '
			} else if ev.Key == termbox.KeyBackspace || ev.Key == termbox.KeyBackspace2 {
				r = '\x08'
			} else if ev.Key == termbox.KeyTab {
				r = '\t'
			} else if r == 0 {
				continue
			}
			var finished bool
			finished, out, isCmd, err = ui.v.handle(r)
			var echoText echo
			echoText.text = string(ui.v.buf)
			if ui.v.mode == command {
				echoText.hideCursor = true
			}
			if ui.v.mode == lastLine {
				echoText.text = ":" + echoText.text + "_"
				echoText.hideCursor = true
			}
			ui.inputCh <- echoText
			if finished {
				return
			}
		default:

		}
	}
}
