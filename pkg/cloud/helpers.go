/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cloud

import (
	"fmt"
	"os"
	"path"

	"gopkg.in/ini.v1"
)

// Dumb CloudStack API config reader. Works for now.
func readAPIConfig() (string, string, string) {
	dir := os.Getenv("PROJECT_DIR")
	cc_path := path.Join(dir, "cloud-config")
	cfg, err := ini.Load(cc_path)
	if err != nil {
		fmt.Println(err, "could not read cloud-config", dir)
		os.Exit(1)
	}
	g := cfg.Section("Global")
	return g.Key("api-url").Value(), g.Key("api-key").Value(), g.Key("secret-key").Value()
}

type set func(string)

func setIfNotEmpty(str string, setFn set) {
	if str != "" {
		setFn(str)
	}
}
