/*
Copyright 2023 The Kubernetes Authors.

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

package kubeconfig_helper

import (
	"errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type Kubeconfig struct {
	content map[string]interface{}
}

func NewKubeconfig() *Kubeconfig {
	return &Kubeconfig{}
}

func (k *Kubeconfig) Load(path string) error {
	rawContent, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var content interface{}
	if err := yaml.Unmarshal(rawContent, &content); err != nil {
		return err
	}

	mapContent, ok := content.(map[string]interface{})
	if ok == false {
		return errors.New("kubeconfig unmarshalling didn't provide expected type map[string]interface{}")
	}

	k.content = mapContent
	return nil
}

func (k *Kubeconfig) Save(path string) error {
	rawContent, err := yaml.Marshal(k.content)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, rawContent, 0644)
	if err != nil {
		return err
	}

	return err
}

func (k *Kubeconfig) GetCurrentContextName() (string, error) {
	value, present := k.content["current-context"]
	if present == false {
		return "", errors.New("current context not present")
	}

	castValue, ok := value.(string)
	if ok == false {
		return "", errors.New("current content not unmarshalled as a string")
	}

	return castValue, nil
}

func (k *Kubeconfig) GetCurrentContext() (map[string]interface{}, error) {
	currentContextName, err := k.GetCurrentContextName()
	if err != nil {
		return nil, err
	}

	contexts := k.content["contexts"].([]interface{})
	var currentContextArrayEntry map[string]interface{} = nil
	for _, ctx := range contexts {
		castCtxArrayEntry, ok := ctx.(map[string]interface{})
		if ok != true {
			return nil, errors.New("unmarshalled kubeconfig context array entry not of expected type map[string]interface{}")
		}
		if castCtxArrayEntry["name"] == currentContextName {
			currentContextArrayEntry = ctx.(map[string]interface{})
			break
		}
	}
	if currentContextArrayEntry == nil {
		return nil, errors.New("no context matching current context name exists in kubeconfig contexts")
	}

	currentContext, present := currentContextArrayEntry["context"]
	if present == false {
		return nil, errors.New("context object not found in matched context array object")
	}

	castCurrentContext, ok := currentContext.(map[string]interface{})
	if ok != true {
		return nil, errors.New("unmarshalled kubeconfig context not of expected type map[string]interface{}")
	}

	return castCurrentContext, nil
}

func (k *Kubeconfig) GetCurrentClusterName() (string, error) {
	currentContext, err := k.GetCurrentContext()
	if err != nil {
		return "", err
	}

	clusterName, present := currentContext["cluster"]
	if present == false {
		return "", errors.New("cluster name not found in current context")
	}

	castClusterName, ok := clusterName.(string)
	if ok == false {
		return "", errors.New("context's cluster name not unmarshalled a string")
	}

	return castClusterName, nil
}

func (k *Kubeconfig) GetCurrentCluster() (map[string]interface{}, error) {
	currentClusterName, err := k.GetCurrentClusterName()
	if err != nil {
		return nil, err
	}

	clusters := k.content["clusters"].([]interface{})
	var currentClusterArrayEntry map[string]interface{} = nil
	for _, clu := range clusters {
		castClusterArrayEntry, ok := clu.(map[string]interface{})
		if ok != true {
			return nil, errors.New("unmarshalled kubeconfig cluster array entry not of expected type map[string]interface{}")
		}
		if castClusterArrayEntry["name"] == currentClusterName {
			currentClusterArrayEntry = clu.(map[string]interface{})
			break
		}
	}
	if currentClusterArrayEntry == nil {
		return nil, errors.New("no cluster matching cluster name specified in current context exists in kubeconfig contexts")
	}

	currentCluster, present := currentClusterArrayEntry["cluster"]
	if present == false {
		return nil, errors.New("cluster object not found in matched cluster array entry")
	}

	castCurrentCluster, ok := currentCluster.(map[string]interface{})
	if ok != true {
		return nil, errors.New("unmarshalled kubeconfig cluster not of expected type map[string]interface{}")
	}

	return castCurrentCluster, nil
}

func (k *Kubeconfig) GetCurrentServer() (string, error) {

	currentCluster, err := k.GetCurrentCluster()
	if err != nil {
		return "", err
	}

	server, present := currentCluster["server"]
	if present == false {
		return "", errors.New("server attribute not present in current cluster")
	}

	castServer, ok := server.(string)
	if ok == false {
		return "", errors.New("unmarshalled server not of expected type string")
	}

	return castServer, nil
}

func (k *Kubeconfig) SetCurrentServer(newServer string) error {
	currentCluster, err := k.GetCurrentCluster()
	if err != nil {
		return err
	}

	var newServerUntyped interface{} = newServer
	currentCluster["server"] = newServerUntyped

	return nil
}
