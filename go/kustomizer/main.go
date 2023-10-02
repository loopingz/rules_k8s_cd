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
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
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
	paths        arrayFlags
	images       arrayFlags
	vars         arrayFlags
	output       string
	input        string
	repository   string
	relativePath string
)

// https://github.com/kubernetes-sigs/kustomize/blob/master/api/types/image.go#L8
// Image contains an image name, a new name, a new tag or digest,
// which will replace the original name and tag.
type Image struct {
	// Name is a tag-less image name.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// NewName is the value used to replace the original name.
	NewName string `json:"newName,omitempty" yaml:"newName,omitempty"`

	// TagSuffix is the value used to suffix the original tag
	// If Digest and NewTag is present an error is thrown
	TagSuffix string `json:"tagSuffix,omitempty" yaml:"tagSuffix,omitempty"`

	// NewTag is the value used to replace the original tag.
	NewTag string `json:"newTag,omitempty" yaml:"newTag,omitempty"`

	// Digest is the value used to replace the original image tag.
	// If digest is present NewTag value is ignored.
	Digest string `json:"digest,omitempty" yaml:"digest,omitempty"`
}

type Patch struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

func init() {
	flag.Var(&vars, "var", "Var to replace in template --var=name:value.")
	flag.Var(&paths, "path", "Paths to file to add to the yaml --path=type:path.")
	flag.StringVar(&output, "output", "", "The output file")
	flag.StringVar(&relativePath, "relativePath", "", "The relative path for resources")
	flag.StringVar(&input, "input", "", "The input file")
	flag.StringVar(&repository, "repository", "", "Repository prefix for images")
	flag.Var(&images, "image", "The image file (can be used multiple times)")
}

func main() {
	var err error
	flag.Parse()

	yfile, err := os.ReadFile(input)

	if err != nil {
		log.Fatal(err)
	}

	data := make(map[interface{}]interface{})

	err = yaml.Unmarshal(yfile, &data)

	if err != nil {
		log.Fatal(err)
	}

	for _, p := range paths {
		log.Printf("Adding %s", p)
		info := strings.SplitN(p, ":", 2)
		var value interface{} = info[1]
		if strings.HasPrefix(info[1], relativePath) {
			value = info[1][len(relativePath):]
		}
		// Specific case as patchJson6902 need to be replaced by inline
		if info[0] == "patchesJson6902" {
			info[0] = "patches"
			pfile, err := os.ReadFile(info[1])
			if err != nil {
				log.Fatal(err)
			}
			value = make(map[interface{}]interface{})
			err = yaml.Unmarshal(pfile, &value)

			if err != nil {
				log.Fatal(err)
			}
		} else if info[0] == "patchesStrategicMerge" {
			info[0] = "patches"
			value = Patch{Path: value.(string)}
		} else if info[0] == "stamp" {
			pfile, err := os.ReadFile(info[1])
			if err != nil {
				log.Fatal(err)
			}
			for _, line := range strings.Split(strings.TrimSpace(string(pfile)), "\n") {
				pair := strings.SplitN(line, " ", 2)
				if len(pair) != 2 {
					continue
				}
				vars = append(vars, "{{"+pair[0]+"}}:"+pair[1])
			}
			continue
		}
		if data[info[0]] == nil {
			data[info[0]] = []interface{}{}
		}
		switch v := data[info[0]].(type) {
		case []interface{}:
			data[info[0]] = append(data[info[0]].([]interface{}), value)
		default:
			log.Printf("Type %s is not managed currently", v)
		}
	}

	if data["images"] == nil {
		data["images"] = []Image{}
	}
	for _, i := range images {
		log.Printf("Adding %s", i)
		img := Image{}
		info := strings.SplitN(i, ":", 2)
		img.Name = info[0]
		if strings.HasPrefix(info[1], "oci_push_info://") {
			b, err := os.ReadFile(info[1][len("oci_push_info://"):])
			if err != nil {
				log.Fatal(err)
			}
			img.NewName = repository + info[0]
			img.Digest = strings.Trim(string(b), " \n")
		} else {
			img.NewName = repository + info[0]
			img.Digest = info[1]
		}
		data["images"] = append(data["images"].([]Image), img)
	}

	result, err := yaml.Marshal(&data)

	if err != nil {
		log.Fatal(err)
	}

	content := string(result)
	for _, v := range vars {
		info := strings.SplitN(v, ":", 2)
		content = strings.ReplaceAll(content, info[0], info[1])
	}

	err = os.WriteFile(output, []byte(content), 0)
}
