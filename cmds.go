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

