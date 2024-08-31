package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"slices"
	"strings"
)

var (
	output string
)

type Substitution struct {
	regex *regexp.Regexp
	new   string
}

type Substitutions []Substitution

func (s Substitutions) String() string {
	return fmt.Sprintf("%d", len(s))
}

func (s *Substitutions) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return fmt.Errorf("Invalid substitution %s", value)
	}
	r, err := regexp.Compile(parts[0])
	if err != nil {
		return err
	}
	*s = append(*s, Substitution{regex: r, new: parts[1]})
	return nil
}

type regexFlags []*regexp.Regexp

func (i *regexFlags) String() string {
	return fmt.Sprintf("%d", len(*i))
}

func (i *regexFlags) Set(value string) error {
	r, err := regexp.Compile(value)
	if err != nil {
		return err
	}
	*i = append(*i, r)
	return nil
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "['\n\t" + strings.Join(*i, "','\n\t") + "\n']"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var filters regexFlags
var substitutions Substitutions
var verbose bool
var inputs arrayFlags

func init() {
	flag.StringVar(&output, "output", "", "The output file")
	flag.Var(&inputs, "input", "The input files")
	flag.Var(&filters, "filter", "The regex to filter the files")
	flag.Var(&substitutions, "substitution", "Substitution in of the form 'regex:new'")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
}

func main() {
	flag.Parse()

	if verbose {
		log.Println("Output", output)
		log.Println("Inputs", inputs)
		log.Println("Filters", filters)
		log.Println("Substitutions", substitutions)
	}

	var w io.Writer
	if (len(inputs) == 0) || (output == "") {
		log.Fatal("Invalid arguments", inputs, output)
	}

	// Create and add some files to the archive.
	w, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	if strings.HasSuffix(output, ".gz") {
		gzw := gzip.NewWriter(w)
		defer gzw.Close()
		w = gzw
	}
	tw := tar.NewWriter(w)

	var list io.Writer
	list, err = os.Create("/tmp/tar-list.txt")
	defer list.(*os.File).Close()

	// Keep track of what we have copied
	copied := []string{}

	// Process all inputs
	for _, input := range inputs {
		var f io.Reader

		if verbose {
			log.Println("Processing", input)
		}
		f, err := os.Open(input)
		if err != nil {
			log.Fatal(err)
		}
		if strings.HasSuffix(input, ".gz") {
			gz, err := gzip.NewReader(f)
			if err != nil {
				log.Fatal(err)
			}
			defer gz.Close()
			f = gz
		} else {
			defer f.(*os.File).Close()
		}

		// Open and iterate through the files in the archive.
		tr := tar.NewReader(f)
		for {
			hdr, err := tr.Next()

			if err == io.EOF {
				break // End of archive
			}
			if err != nil {
				log.Fatal(err)
			}
			skip := false
			// Apply substitutions
			for _, r := range substitutions {
				newName := r.regex.ReplaceAllString(hdr.Name, r.new)
				if newName != hdr.Name {
					if verbose {
						log.Printf("Substituting %s -> %s:\n", hdr.Name, newName)
					}
					hdr.Name = newName
				}
			}
			// Skip empty files
			if hdr.Name == "" {
				continue
			}
			// Apply filters
			for _, r := range filters {
				if r.MatchString(hdr.Name) {
					if verbose {
						log.Printf("Filtering %s:\n", hdr.Name)
					}
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			// Avoid duplicates
			if slices.Contains(copied, hdr.Name) {
				if verbose {
					log.Printf("Skipping duplicate %s:\n", hdr.Name)
				}
				continue
			}
			copied = append(copied, hdr.Name)
			// Copy the file
			if err := tw.WriteHeader(hdr); err != nil {
				log.Fatal("Cannot write header", err)
			}
			b := make([]byte, hdr.Size)
			_, err = io.ReadFull(tr, b)
			if err != nil && err != io.EOF {
				log.Fatal("Cannot read content", err)
			}

			if _, err := tw.Write(b); err != nil {
				log.Fatal("Cannot write content", err)
			}

			if verbose {
				log.Println("Copied", hdr.Name)
			}
		}

	}

	if err := tw.Close(); err != nil {
		log.Fatal(err)
	}
	if verbose {
		s, err := os.Stat(output)
		if err != nil {
			log.Fatal(err)
		}
		// 24906730
		log.Println("Final tar size", s.Size())
	}
}
