/*
Copyright 2022 The Kubernetes Authors.

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

package errors

import (
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	// ACS standard error messages of the form "CloudStack API error 431 (CSExceptionErrorCode: 9999):..."
	//  This regexp is used to extract CSExceptionCodes from the message.
	csErrorCodeRegexp, _ = regexp.Compile(".+CSExceptionErrorCode: ([0-9]+).+")

	// List of error codes: https://docs.cloudstack.apache.org/en/latest/developersguide/dev.html#error-handling
	csTerminalErrorCodes = strings.Split(getEnv("CLOUDSTACK_TERMINAL_FAILURE_CODES", "4250,9999"), ",")
)

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

type DeployVMError struct {
	acsError error
}

func IsTerminalDeployVMError(err error) bool {
	deployError := &DeployVMError{}
	return errors.As(err, &deployError) && deployError.IsTerminal()
}

func NewDeployVMError(acsError error) error {
	return &DeployVMError{acsError: acsError}
}

func (e *DeployVMError) Error() string {
	if e.acsError == nil {
		return ""
	}
	return e.acsError.Error()
}

func (e *DeployVMError) IsTerminal() bool {
	errorCode := GetACSErrorCode(e.acsError)
	for _, te := range csTerminalErrorCodes {
		if errorCode == te {
			return true
		}
	}
	return false
}

func GetACSErrorCode(acsError error) string {
	if acsError == nil {
		return ""
	}

	matches := csErrorCodeRegexp.FindStringSubmatch(acsError.Error())
	if len(matches) > 1 {
		return matches[1]
	}

	return "No error code"
}
