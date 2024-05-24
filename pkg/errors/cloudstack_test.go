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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeployVM Error", func() {

	Context("a deploy VM error", func() {
		It("determine an error is terminal", func() {
			vmError := DeployVMError{fmt.Errorf(`CloudStack API error 530 (CSExceptionErrorCode: 4250): Internal error executing command, please contact your system administrator`)}

			Expect(vmError.IsTerminal()).Should(BeTrue())
		})

		It("determine an error is NOT terminal", func() {
			vmError := DeployVMError{fmt.Errorf(`CloudStack API error 400 (CSExceptionErrorCode: 0): Internal error executing command, please contact your system administrator`)}

			Expect(vmError.IsTerminal()).Should(BeFalse())
		})
	})

	Context("a deploy VM error helper", func() {
		It("determine an error is terminal", func() {
			vmError := &DeployVMError{fmt.Errorf(`CloudStack API error 530 (CSExceptionErrorCode: 4250): Internal error executing command, please contact your system administrator`)}

			Expect(IsTerminalDeployVMError(vmError)).Should(BeTrue())
		})

		It("determine an error is terminal when wrapped", func() {
			vmError := &DeployVMError{fmt.Errorf(`CloudStack API error 530 (CSExceptionErrorCode: 4250): Internal error executing command, please contact your system administrator`)}

			Expect(IsTerminalDeployVMError(fmt.Errorf("a wrapped err: %w", vmError))).Should(BeTrue())
		})

		It("determine an error is NOT terminal", func() {
			vmError := &DeployVMError{fmt.Errorf(`CloudStack API error 400 (CSExceptionErrorCode: 0): Internal error executing command, please contact your system administrator`)}

			Expect(IsTerminalDeployVMError(vmError)).Should(BeFalse())
		})
	})

})
