package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bustonmeplease/gitty/gitobj"
)

func printObject(r *gitobj.Repo, o *gitobj.Object) {
	switch o.Type {
	case gitobj.TTree:
		ents, err := gitobj.DecodeTree(o.Data)
		if err != nil {
			fmt.Fprintln(os.Stderr, "bad tree:", err)
			return
		}
		for _, e := range ents {
			typ := "blob"
			if e.Mode == gitobj.ModeDir {
				typ = "tree"
			}
			fmt.Printf("%s %s %s\t%s\n", e.Mode, typ, e.Sha, e.Name)
		}
	default:
		// blob / commit / tag all print raw
		os.Stdout.Write(o.Data)
	}
}

func doStatus(r *gitobj.Repo) error {
	st, err := r.ComputeStatus()
	if err != nil {
		return err
	}
	br := r.CurrentBranch()
	if br == "" {
		fmt.Println("HEAD detached")
	} else {
		fmt.Printf("on branch %s\n", br)
	}

	if len(st.Staged) > 0 {
		fmt.Println("\nchanges to be committed:")
		for _, p := range st.Staged {
			fmt.Printf("\tstaged:    %s\n", p)
		}
	}
	if len(st.Modified) > 0 || len(st.Deleted) > 0 {
		fmt.Println("\nchanges not staged:")
		for _, p := range st.Modified {
			fmt.Printf("\tmodified:  %s\n", p)
		}
		for _, p := range st.Deleted {
			fmt.Printf("\tdeleted:   %s\n", p)
		}
	}
	if len(st.Untracked) > 0 {
		fmt.Println("\nuntracked files:")
		for _, p := range st.Untracked {
			fmt.Printf("\t%s\n", p)
		}
	}
	if len(st.Staged)+len(st.Modified)+len(st.Deleted)+len(st.Untracked) == 0 {
		fmt.Println("nothing to commit, working tree clean")
	}
	return nil
}

func doLog(r *gitobj.Repo) error {
	head, err := r.HeadCommit()
	if err != nil {
		return err
	}
	if head == "" {
		return fmt.Errorf("no commits yet")
	}
	chain, err := r.LogChain(head, 0)
	if err != nil {
		return err
	}
	for _, sha := range chain {
		o, err := r.Read(sha)
		if err != nil {
			return err
		}
		c, _ := gitobj.ParseCommit(o.Data)
		fmt.Printf("commit %s\n", sha)
		fmt.Printf("Author: %s <%s>\n", c.Author.Name, c.Author.Email)
		if !c.Author.When.IsZero() {
			fmt.Printf("Date:   %s\n", c.Author.When.Format(time.RFC1123Z))
		}
		fmt.Println()
		for _, ln := range splitMsg(c.Message) {
			fmt.Printf("    %s\n", ln)
		}
		fmt.Println()
	}
	return nil
}

func splitMsg(m string) []string {
	var out []string
	cur := ""
	for _, ch := range m {
		if ch == '\n' {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(ch)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func doDiff(r *gitobj.Repo, only string) error {
	// diff = index (or HEAD) blob vs working tree, per modified file
	idx, err := r.LoadIndex()
	if err != nil {
		return err
	}
	if only != "" {
		only = toRel(r, only)
	}
	any := false
	for _, e := range idx.Entries {
		if only != "" && e.Path != only {
			continue
		}
		fp := filepath.Join(r.Root, filepath.FromSlash(e.Path))
		wd, err := os.ReadFile(fp)
		if err != nil {
			continue // gone; rm handles that, skip in diff
		}
		if gitobj.HashRaw(gitobj.TBlob, wd) == e.Sha {
			continue // unchanged
		}
		o, err := r.Read(e.Sha)
		if err != nil {
			return err
		}
		fmt.Print(gitobj.UnifiedDiff(e.Path, string(o.Data), string(wd)))
		any = true
	}
	if !any {
		// nothing changed; stay quiet like git does
		_ = any
	}
	return nil
}

func doTag(r *gitobj.Repo, args []string) error {
	if len(args) == 0 {
		tags, err := r.ListTags()
		if err != nil {
			return err
		}
		for _, t := range tags {
			fmt.Println(t)
		}
		return nil
	}
	annotate := false
	msg := ""
	name := ""
	target := "HEAD"
	// positionals can come in any order relative to flags; first positional is
	// the tag name, second (if any) is the target.
	var pos []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-a":
			annotate = true
		case "-m":
			if i+1 < len(args) {
				msg = args[i+1]
				i++
				annotate = true
			}
		default:
			pos = append(pos, args[i])
		}
	}
	if len(pos) == 0 {
		return fmt.Errorf("tag: need a name")
	}
	name = pos[0]
	if len(pos) > 1 {
		target = pos[1]
	}
	return r.MakeTag(name, target, annotate, whoami(), msg)
}
