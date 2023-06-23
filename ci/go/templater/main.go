/*
Copyright 2020 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	varFiles      arrayFlags
	variables		   arrayFlags
	inputs			   arrayFlags
)

func init() {
	flag.Var(&varFiles, "var-file", "Paths to file with VAR value\nVAR2 value2\n.")
	flag.Var(&inputs, "input", "", "The input file")
	flag.Var(&variables, "var", "A variable to be used in the format string (can be used multiple times)")
}

func workspaceStatusDict(filenames []string) map[string]interface{} {
	d := map[string]interface{}{}
	for _, f := range filenames {
		content, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatalf("Unable to read %s: %v", f, err)
		}
		for _, l := range strings.Split(string(content), "\n") {
			sv := strings.SplitN(l, " ", 2)
			if len(sv) == 2 {
				d[sv[0]] = sv[1]
			}
		}
	}
	return d
}

func templateFile(inFile string, vars map[string]interface{}, outFile string) error {
	content, err := ioutil.ReadFile(inFile)
	if err != nil {
		return err
	}
	for k, v := range vars {
		content = content.ReplaceAll([]byte("{{"+k+"}}"), []byte(v.(string)))
	}
	_, err = ioutil.WriteFile(outFile, content, 0666)
	return err
}

func main() {
	var err error
	flag.Parse()
	vars := workspaceStatusDict(varFiles)
	for _, v := range variables {
		sv := strings.SplitN(v, "=", 2)
		if len(sv) == 2 {
			vars[sv[0]] = sv[1]
		}
	}
	/*
	if formatFile != "" {
		if format != "" {
			log.Fatal("only one of --format or --format-file should be used")
		}
		imp, err := ioutil.ReadFile(formatFile)
		if err != nil {
			log.Fatalf("Unable to read file %s: %v", formatFile, err)
		}
		format = string(imp)
	}
	*/

	outf := os.Stdout
	if output != "" {
		outf, err = os.OpenFile(output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			log.Fatalf("Unable to create output file %s: %v", output, err)
		}
		defer outf.Close()
	}
	log.Printf("Vars: %v", stamps)
	log.Printf("Input: %v", inputs)

	for _, input := range inputs {
		// Copy file
		if strings.Contains(input, ":") {
			inOut := strings.SplitN(input, ":", 2)
			log.Printf("Template file from %s to %s", inOut[0], inOut[1])
			templateFile(inOut[0], vars, inOut[1])
		} else {
			// Inplace replacement
			log.Printf("Modifying template file %s", input)
			templateFile(input, vars, input)
		}
	}
}
