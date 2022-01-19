/*
Copyright 2022.

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
	"bytes"
	"compress/gzip"
	"encoding/base64"
)

type set func(string)

func setIfNotEmpty(str string, setFn set) {
	if str != "" {
		setFn(str)
	}
}

func CompressAndEncodeString(str string) (string, error) {
	buf := &bytes.Buffer{}
	gzipWriter := gzip.NewWriter(buf)
	if _, err := gzipWriter.Write([]byte(str)); err != nil {
		gzipWriter.Close()
		return "", err
	}
	gzipWriter.Close()
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
