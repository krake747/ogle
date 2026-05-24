package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra/doc"

	"github.com/ma-tf/ogle/cmd"
)

func main() {
	out := flag.String("out", "./docs/cli", "output directory")
	format := flag.String("format", "markdown", "markdown|man|rest")
	front := flag.Bool(
		"frontmatter",
		false,
		"prepend simple YAML front matter to markdown",
	)

	flag.Parse()

	if err := os.MkdirAll(*out, 0o750); err != nil {
		log.Fatal(err)
	}

	root := cmd.Root()
	root.DisableAutoGenTag = true // stable, reproducible files (no timestamp footer)

	switch *format {
	case "markdown":
		if *front {
			err := doc.GenMarkdownTreeCustom(
				root,
				*out,
				prep,
				link,
			)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			if err := doc.GenMarkdownTree(root, *out); err != nil {
				log.Fatal(err)
			}
		}

		postProcessDir(*out)

	case "man":
		hdr := &doc.GenManHeader{
			Title:   strings.ToUpper(root.Name()),
			Section: "1",
			Date:    nil,
			Source:  "",
			Manual:  "",
		}
		if err := doc.GenManTree(root, hdr, *out); err != nil {
			log.Fatal(err)
		}
	case "rest":
		if err := doc.GenReSTTree(root, *out); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown format: %s", *format)
	}
}

func prep(filename string) string {
	base := filepath.Base(filename)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	title := strings.ReplaceAll(name, "_", " ")

	return fmt.Sprintf(
		"---\ntitle: %q\nslug: %q\ndescription: \"CLI reference for %s\"\n---\n\n",
		title,
		name,
		title,
	)
}

func link(name string) string { return strings.ToLower(name) }

func postProcessDir(dir string) {
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		log.Fatalf("resolve doc dir: %v", err)
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		log.Fatalf("open doc root: %v", err)
	}

	entries, err := readDir(root)
	if err != nil {
		_ = root.Close()

		log.Fatalf("read doc dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		processFile(root, entry.Name())
	}

	_ = root.Close()
}

func readDir(root *os.Root) ([]os.FileInfo, error) {
	f, err := root.Open(".")
	if err != nil {
		return nil, fmt.Errorf("open root: %w", err)
	}
	defer func() { _ = f.Close() }()

	entries, err := f.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("readdir: %w", err)
	}

	return entries, nil
}

const permDocFile os.FileMode = 0o600

func processFile(root *os.Root, name string) {
	f, err := root.Open(name)
	if err != nil {
		log.Fatalf("open %s: %v", name, err)
	}

	data, err := io.ReadAll(f)
	_ = f.Close()

	if err != nil {
		log.Fatalf("read %s: %v", name, err)
	}

	output := fixMarkdown(data)

	f, err = root.OpenFile(name, os.O_WRONLY|os.O_TRUNC, permDocFile)
	if err != nil {
		log.Fatalf("open %s for write: %v", name, err)
	}

	werr := writeAll(f, output)
	if werr != nil {
		_ = f.Close()

		log.Fatalf("write %s: %v", name, werr)
	}

	_ = f.Close()
}

func writeAll(f *os.File, data []byte) error {
	_, err := f.Write(data)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func fixMarkdown(data []byte) []byte {
	lines := bytes.Split(data, []byte("\n"))
	result := make([][]byte, 0, len(lines))
	inCodeBlock := false

	for _, line := range lines {
		// demote all headings one level: ## -> #, ### -> ##
		if len(line) > 2 && line[0] == '#' && line[1] == '#' && line[2] == ' ' {
			line = append([]byte("# "), line[3:]...)
		} else if len(line) > 3 && line[0] == '#' && line[1] == '#' && line[2] == '#' && line[3] == ' ' {
			line = append([]byte("## "), line[4:]...)
		}

		trimmed := bytes.TrimSpace(line)
		if bytes.Equal(trimmed, []byte("```")) {
			if !inCodeBlock {
				line = []byte("```sh")
			}

			inCodeBlock = !inCodeBlock
		}

		result = append(result, line)
	}

	output := bytes.Join(result, []byte("\n"))
	output = bytes.ReplaceAll(output, []byte("\t"), []byte("  "))
	output = bytes.TrimRight(output, "\n")
	output = append(output, '\n')

	return output
}
