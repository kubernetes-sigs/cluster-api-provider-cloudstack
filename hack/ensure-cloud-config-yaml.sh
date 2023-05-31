#!/usr/bin/env bash

# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

# This ensures that cloud-config.yaml exists which is required for e2e smoke test
if [ ! -f "cloud-config.yaml" ];then
    echo "cloud-config.yaml is not found, creating"
    cat >cloud-config.yaml <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: secret1
  namespace: default
type: Opaque
stringData:
  api-key: XXXX
  secret-key: XXXX
  api-url: http://1.2.3.4:8080/client/api
  verify-ssl: "false"
EOF

fi
