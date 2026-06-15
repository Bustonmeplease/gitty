package gitobj

import (
	"fmt"
	"strings"
)

// blame: for each line of a file at HEAD, find the commit that last touched it.
// done the slow honest way: walk history oldest->newest, and whenever a line
// first appears (vs the previous version) attribute it to that commit. not as
// clever as git's real blame (no rename detection, no -C) but it's correct for
// straightforward histories.

type BlameLine struct {
	Commit string
	Author string
	Text   string
}

func (r *Repo) Blame(path string) ([]BlameLine, error) {
	head, err := r.HeadCommit()
	if err != nil {
		return nil, err
	}
	if head == "" {
		return nil, fmt.Errorf("no commits")
	}
	rp, err := r.rel(path)
	if err != nil {
		return nil, err
	}

	chain, err := r.LogChain(head, 0)
	if err != nil {
		return nil, err
	}
	// reverse to oldest-first
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	// current attribution: text -> commit, keyed by line content+position is
	// fragile, so track a slice of (text, commit) and re-diff each revision.
	type entry struct {
		text   string
		commit string
		author string
	}
	var cur []entry
	var prevText string

	for _, csha := range chain {
		flat, err := r.commitTreeFlat(csha)
		if err != nil {
			return nil, err
		}
		blobSha, ok := flat[rp]
		if !ok {
			continue // file not present in this revision
		}
		o, err := r.Read(blobSha)
		if err != nil {
			return nil, err
		}
		text := string(o.Data)
		co, _ := r.Read(csha)
		cm, _ := ParseCommit(co.Data)
		author := cm.Author.Name

		newLines := splitLines(text)
		// diff prevText -> text; keep attribution for kept lines, attribute
		// added lines to this commit.
		var next []entry
		ops := DiffLines(prevText, text)
		ci := 0 // index into cur (old lines), advances on keep/del
		_ = newLines
		for _, op := range ops {
			switch op.Kind {
			case ' ':
				if ci < len(cur) {
					next = append(next, cur[ci])
				} else {
					next = append(next, entry{op.Line, csha, author})
				}
				ci++
			case '-':
				ci++ // dropped
			case '+':
				next = append(next, entry{op.Line, csha, author})
			}
		}
		cur = next
		prevText = text
	}

