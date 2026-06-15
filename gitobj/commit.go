package gitobj

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Sig struct {
	Name  string
	Email string
	When  time.Time
}

func (s Sig) String() string {
	// unix ts + zone offset, the way git stores it
	_, off := s.When.Zone()
	sign := "+"
	if off < 0 {
		sign = "-"
		off = -off
	}
	hh := off / 3600
	mm := (off % 3600) / 60
	return fmt.Sprintf("%s <%s> %d %s%02d%02d",
		s.Name, s.Email, s.When.Unix(), sign, hh, mm)
}

type Commit struct {
	Tree      string
	Parents   []string
	Author    Sig
	Committer Sig
	Message   string
}

func EncodeCommit(c Commit) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "tree %s\n", c.Tree)
	for _, p := range c.Parents {
		fmt.Fprintf(&b, "parent %s\n", p)
	}
	fmt.Fprintf(&b, "author %s\n", c.Author.String())
	fmt.Fprintf(&b, "committer %s\n", c.Committer.String())
	b.WriteByte('\n')
	b.WriteString(c.Message)
	if !strings.HasSuffix(c.Message, "\n") {
		b.WriteByte('\n')
	}
	return b.Bytes()
}

