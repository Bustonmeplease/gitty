package gitobj

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// local clone: no network, no packfiles. we copy every loose object and ref
// from src into a fresh repo at dst, set HEAD to match, and check out the
// default branch's worktree. it's basically `git clone /path` for our toy.

func Clone(srcDir, dstDir string) (*Repo, error) {
	src, err := Find(srcDir)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(dstDir); err == nil {
		// allow empty existing dir, else bail
		ents, _ := os.ReadDir(dstDir)
		if len(ents) > 0 {
			return nil, fmt.Errorf("clone: destination %q exists and is not empty", dstDir)
		}
	}
	dst, err := Init(dstDir)
	if err != nil {
		return nil, err
	}

	// copy all loose objects
	objs, err := src.ListObjects()
	if err != nil {
		return nil, err
	}
	for _, sha := range objs {
		if err := copyFile(src.objPath(sha), dst.objPath(sha)); err != nil {
			return nil, err
		}
	}

	// copy refs (heads + tags)
	if err := copyTree(filepath.Join(src.GitDir, "refs"), filepath.Join(dst.GitDir, "refs")); err != nil {
		return nil, err
	}
	// copy HEAD verbatim
	if err := copyFile(src.headPath(), dst.headPath()); err != nil {
		return nil, err
	}
	// copy config if present (best effort)
	_ = copyFile(src.configPath(), dst.configPath())

	// check out whatever HEAD points at so the worktree isn't empty
	head, err := dst.HeadCommit()
	if err == nil && head != "" {
		if err := dst.CheckoutCommit(head); err != nil {
			return nil, err
		}
	}
	return dst, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

