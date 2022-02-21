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

package cloud

type TagIface interface {
	AddNetworkTags(string, map[string]string) error
	GetNetworkTags(string) (map[string]string, error)
	DeleteNetworkTags(string, map[string]string) error
}

const (
	clusterTagNamePrefix = "CAPC_cluster_"
	createdByCapcTagName = "created_by_CAPC"
	resourceTypeNetwork  = "network"
)

// TagNetwork adds tags to a network by network id.
func (c *client) AddNetworkTags(networkID string, tags map[string]string) error {
	p := c.cs.Resourcetags.NewCreateTagsParams([]string{networkID}, resourceTypeNetwork, tags)
	_, err := c.cs.Resourcetags.CreateTags(p)
	return err
}

// GetNetworkTags gets tags by network id.
func (c *client) GetNetworkTags(networkID string) (map[string]string, error) {
	p := c.cs.Resourcetags.NewListTagsParams()
	p.SetResourceid(networkID)
	p.SetResourcetype(resourceTypeNetwork)
	listTagResponse, err := c.cs.Resourcetags.ListTags(p)
	if err != nil {
		return nil, err
	}
	tags := make(map[string]string, listTagResponse.Count)
	for _, t := range listTagResponse.Tags {
		tags[t.Key] = t.Value
	}
	return tags, nil
}

// DeleteNetworkTags deletes matching tags from a network
func (c *client) DeleteNetworkTags(networkID string, tagsToDelete map[string]string) error {
	p := c.cs.Resourcetags.NewDeleteTagsParams([]string{networkID}, resourceTypeNetwork)
	p.SetTags(tagsToDelete)
	_, err := c.cs.Resourcetags.DeleteTags(p)
	return err
}
