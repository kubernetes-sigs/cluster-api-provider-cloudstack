#!/bin/bash

# Copyright 2022.
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

# This is a simple script to assist in adding GinkgoRecover() statements to controllers during testing.
# This is necessary as the controllers are run in goroutines when tested under testenv.
# Add to add, remove to remove, and contains exits 1 if the statements are missing.

CONTROLLER_DIR=${PROJECT_DIR:-$(dirname $(dirname "$0"))}/controllers
FILES=${CONTROLLER_DIR}/cloudstack*controller.go

case $1 in
    --add) 
        # Use grep to prevent double addition of ginkgo recover statements.
        grep -i ginkgo ${FILES} 2>&1> /dev/null \
            || (sed -i.bak '/Reconcile(/a\'$'\n'$'\t''defer ginkgo.GinkgoRecover()'$'\n''' ${FILES} && \
                sed -i.bak '/^import (/a\'$'\n'$'\t''"github.com/onsi/ginkgo/v2"'$'\n''' ${FILES} && \
                rm ${CONTROLLER_DIR}/*.bak)
        ;;
    --remove)
            sed -i.bak '/ginkgo/d' ${FILES} && rm ${CONTROLLER_DIR}/*.bak
        ;;
    --contains)
        grep -i ginkgo ${FILES} 2>&1> /dev/null && exit 0
        echo "**************************************************************************************************************"
        echo "******************************************************************************************************************"
        echo "Did not find GinkgoRecover statements present in controllers."
        echo "Please run $0 "
        echo "with the '--add' argument to add to tests."
        echo "Without this, controller test failures will result in a Ginkgo Panic," echo "and failures will be opaque."
        echo "******************************************************************************************************************"
        echo "**************************************************************************************************************"
        exit 1
        ;;
esac

