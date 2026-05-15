package elefant

import (
	"bufio"
	"bytes"
	"debug/elf"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/itchio/headway/state"
	"github.com/itchio/httpkit/eos"
)

type TraceNode struct {
	Name     string
	FullPath string
	Info     *ElfInfo

	Children          []*TraceNode
	UnresolvedImports []string
}

type TraceParams struct {
	Consumer *state.Consumer
}

type traceCache struct {
	nodes map[string]*TraceNode
}

func (c *traceCache) add(tn *TraceNode) {
	c.nodes[tn.FullPath] = tn
}

func Trace(info *ElfInfo, fullPath string, params TraceParams) (*TraceNode, error) {
	fullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("resolving absolute path: %w", err)
	}

	root := &TraceNode{
		Name:     filepath.Base(fullPath),
		FullPath: fullPath,
		Info:     info,
	}

	searchPaths, err := getSearchPaths()
	if err != nil {
		return nil, fmt.Errorf("loading search paths: %w", err)
	}

	cache := &traceCache{
		nodes: make(map[string]*TraceNode),
	}
	cache.add(root)

	if err := root.trace(cache, searchPaths, params); err != nil {
		return nil, fmt.Errorf("tracing %s: %w", fullPath, err)
	}

	return root, nil
}

func (n *TraceNode) trace(cache *traceCache, searchPaths *searchPaths, params TraceParams) error {
	for _, imp := range n.Info.Imports {
		importPath := searchPaths.lookup(imp, n.Info.Arch)
		if importPath == "" {
			n.UnresolvedImports = append(n.UnresolvedImports, imp)
			continue
		}

		if cn, ok := cache.nodes[importPath]; ok {
			n.Children = append(n.Children, cn)
			continue
		}

		if err := n.traceImport(cache, searchPaths, imp, importPath, params); err != nil {
			return err
		}
	}
	return nil
}

func (n *TraceNode) traceImport(cache *traceCache, searchPaths *searchPaths, name, importPath string, params TraceParams) error {
	f, err := eos.Open(importPath)
	if err != nil {
		return fmt.Errorf("opening %s: %w", importPath, err)
	}
	defer f.Close()

	ei, err := Probe(f, ProbeParams{Consumer: params.Consumer})
	if err != nil {
		return fmt.Errorf("probing %s: %w", importPath, err)
	}

	cn := &TraceNode{
		Name:     name,
		FullPath: importPath,
		Info:     ei,
	}
	cache.add(cn)
	n.Children = append(n.Children, cn)

	return cn.trace(cache, searchPaths, params)
}

type stringifyContext struct {
	donePaths map[string]bool
}

func (n *TraceNode) String() string {
	return "\n" + n.stringify(&stringifyContext{
		donePaths: make(map[string]bool),
	})
}

func (n *TraceNode) stringify(ctx *stringifyContext) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("- %s", n.FullPath))

	for _, ui := range n.UnresolvedImports {
		lines = append(lines, fmt.Sprintf("  - MISSING %s", ui))
	}
	for _, c := range n.Children {
		if ctx.donePaths[c.FullPath] {
			lines = append(lines, fmt.Sprintf("  - %s (see above)", c.FullPath))
			continue
		}
		ctx.donePaths[c.FullPath] = true

		for l := range strings.SplitSeq(c.stringify(ctx), "\n") {
			lines = append(lines, fmt.Sprintf("  %s", l))
		}
	}
	return strings.Join(lines, "\n")
}

type searchPaths struct {
	paths []string

	archCache map[string]Arch
}

func (sp *searchPaths) getArch(fullpath string) Arch {
	if sp.archCache == nil {
		sp.archCache = make(map[string]Arch)
	}

	if arch, ok := sp.archCache[fullpath]; ok {
		return arch
	}

	arch := ArchUnknown
	ef, err := elf.Open(fullpath)
	if err == nil {
		defer ef.Close()
		switch ef.Machine {
		case elf.EM_386:
			arch = Arch386
		case elf.EM_X86_64:
			arch = ArchAmd64
		case elf.EM_AARCH64:
			arch = ArchArm64
		}
	}

	sp.archCache[fullpath] = arch
	return arch
}

func (sp *searchPaths) lookup(name string, arch Arch) string {
	for _, dir := range sp.paths {
		candidatePath := filepath.Join(dir, name)
		if sp.getArch(candidatePath) == arch {
			return candidatePath
		}
	}
	return ""
}

func (sp *searchPaths) addPath(path string) {
	sp.paths = append(sp.paths, path)
}

func getSearchPaths() (*searchPaths, error) {
	sp := &searchPaths{}
	sp.addPath("/usr/lib") // this one is standard

	if err := sp.parseConfig("/etc/ld.so.conf"); err != nil {
		return nil, err
	}
	return sp, nil
}

var ldSoConfCommentRe = regexp.MustCompile("#.*$")

// cf. https://www.daemon-systems.org/man/ld.so.conf.5.html
// we do not support hardware-dependent directives
func (sp *searchPaths) parseConfig(configPath string) error {
	contents, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reading %s: %w", configPath, err)
	}

	s := bufio.NewScanner(bytes.NewReader(contents))
	for s.Scan() {
		line := ldSoConfCommentRe.ReplaceAllLiteralString(s.Text(), "")
		line = strings.TrimSpace(line)

		switch {
		case len(line) == 0:
			continue
		case strings.HasPrefix(line, "include "):
			includePath := strings.TrimSpace(strings.TrimPrefix(line, "include "))

			files, err := filepath.Glob(includePath)
			if err != nil {
				return fmt.Errorf("glob %s: %w", includePath, err)
			}

			for _, f := range files {
				if err := sp.parseConfig(f); err != nil {
					return err
				}
			}
		case strings.HasPrefix(line, "/"):
			sp.addPath(line)
		}
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("scanning %s: %w", configPath, err)
	}

	return nil
}
