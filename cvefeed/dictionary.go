// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cvefeed

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/facebookincubator/nvdtools/cvefeed/internal/iface"
)

// Dictionary is a slice of entries
type Dictionary map[string]CVEItem

// Override amends entries in Dictionary with configurations from Dictionary d2;
// CVE will be matched if it matches the original config of d and does not match the config of d2.
func (d *Dictionary) Override(d2 Dictionary) {
	if d == nil {
		return
	}
	if *d == nil {
		*d = make(Dictionary)
	}
	for k, cve := range d2 {
		if _, ok := (*d)[k]; ok {
			(*d)[k] = iface.MergeCVEItems((*d)[k], cve)
		}
	}
}

// LoadXMLDictionary parses dictionary from multiple NVD vulenrability feed XML files
func LoadXMLDictionary(paths ...string) (Dictionary, error) {
	return LoadFeed(loadXMLFile, paths...)
}

// LoadJSONDictionary parses dictionary from multiple NVD vulenrability feed JSON files
func LoadJSONDictionary(paths ...string) (Dictionary, error) {
	return LoadFeed(loadJSONFile, paths...)
}

// LoadFeed calls loadFunc for each file in paths and returns the combined outputs in a Dictionary.
func LoadFeed(loadFunc func(string) ([]CVEItem, error), paths ...string) (Dictionary, error) {
	dict := make(Dictionary)
	var wg sync.WaitGroup
	done := make(chan struct{})
	errDone := make(chan struct{})
	dictChan := make(chan []CVEItem, 1)
	errChan := make(chan error, 1)
	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			feed, err := loadFunc(path)
			if err != nil {
				errChan <- fmt.Errorf("dictionary: failed to load feed %q: %v", path, err)
				return
			}
			dictChan <- feed
		}(path)
	}
	go func() {
		for d := range dictChan {
			for _, cve := range d {
				dict[cve.CVEID()] = cve
			}
		}
		close(done)
	}()
	var errs []string
	go func() {
		for e := range errChan {
			errs = append(errs, e.Error())
		}
		close(errDone)
	}()
	wg.Wait()
	close(dictChan)
	close(errChan)
	<-done
	<-errDone
	if len(errs) > 0 {
		return dict, errors.New(strings.Join(errs, "\n"))
	}
	return dict, nil
}

// loadXMLFile parses dictionary from NVD vulnerability feed XML file
func loadXMLFile(path string) ([]CVEItem, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("dictionary: failed to load feed %q: %v", path, err)
	}
	defer f.Close()
	return ParseXML(f)
}

// loadJSONFile parses dictionary from NVD vulnerability feed XML file
func loadJSONFile(path string) ([]CVEItem, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("dictionary: failed to load feed %q: %v", path, err)
	}
	defer f.Close()
	return ParseJSON(f)
}
