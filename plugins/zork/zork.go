// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

// Package zork implements a zork plugin for catbase.
package zork

import (
	"bufio"
	"bytes"
	"go/build"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

// ZorkPlugin is a catbase plugin for playing zork.
type ZorkPlugin struct {
	bot bot.Bot
	sync.Mutex
	// zorks is a map from channels to their corresponding zork instances.
	zorks map[string]io.WriteCloser
}

func New(b bot.Bot) bot.Plugin {
	return &ZorkPlugin{
		bot:   b,
		zorks: make(map[string]io.WriteCloser),
	}
}

func (p *ZorkPlugin) runZork(ch string) error {
	const importString = "github.com/velour/catbase/plugins/zork"
	pkg, err := build.Import(importString, "", build.FindOnly)
	if err != nil {
		return err
	}
	zorkdat := filepath.Join(pkg.Dir, "ZORK1.DAT")
	cmd := exec.Command("dfrotz", zorkdat)

	var r io.ReadCloser
	r, cmd.Stdout = io.Pipe()
	cmd.Stderr = cmd.Stdout

	var w io.WriteCloser
	cmd.Stdin, w = io.Pipe()

	log.Printf("zork running %v\n", cmd)
	if err := cmd.Start(); err != nil {
		w.Close()
		return err
	}
	go func() {
		defer r.Close()
		s := bufio.NewScanner(r)
		// Scan until the next prompt.
		s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			if atEOF && len(data) == 0 {
				return 0, nil, nil
			}
			if i := bytes.Index(data, []byte{'\n', '>'}); i > 0 {
				return i + 1, data[:i], nil
			}
			if atEOF {
				return len(data), data, nil
			}
			return 0, nil, nil
		})
		for s.Scan() {
			// Remove > and quote the whole thing.
			m := strings.Replace(s.Text(), ">", "", -1)
			m = strings.Replace(m, "\n", "\n>", -1)
			m = ">" + m + "\n"
			p.bot.Send(bot.Message, ch, m)
		}
	}()
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("zork exited: %v\n", err)
		}
		p.Lock()
		p.zorks[ch] = nil
		p.Unlock()
	}()
	log.Printf("zork is running in %s\n", ch)
	p.zorks[ch] = w
	return nil
}

func (p *ZorkPlugin) Message(message msg.Message) bool {
	m := strings.ToLower(message.Body)
	log.Printf("got message [%s]\n", m)
	if ts := strings.Fields(m); len(ts) < 1 || ts[0] != "zork" {
		return false
	}
	m = strings.TrimSpace(strings.TrimPrefix(m, "zork"))
	ch := message.Channel

	p.Lock()
	defer p.Unlock()
	if p.zorks[ch] == nil {
		if err := p.runZork(ch); err != nil {
			p.bot.Send(bot.Message, ch, "failed to run zork: "+err.Error())
			return true
		}
	}
	log.Printf("zorking, [%s]\n", m)
	io.WriteString(p.zorks[ch], m+"\n")
	return true
}

func (p *ZorkPlugin) Event(_ string, _ msg.Message) bool { return false }

func (p *ZorkPlugin) BotMessage(_ msg.Message) bool { return false }

func (p *ZorkPlugin) Help(ch string, _ []string) {
	p.bot.Send(bot.Message, ch, "Play zork using 'zork <zork command>'.")
}

func (p *ZorkPlugin) RegisterWeb() *string { return nil }

func (p *ZorkPlugin) ReplyMessage(message msg.Message, identifier string) bool { return false }
