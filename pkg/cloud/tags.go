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

import (
	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"strings"
)

type TagIface interface {
	AddClusterTag(resourceType ResourceType, resourceID string, csCluster *infrav1.CloudStackCluster, addCreatedByCAPCTag bool) error
	DeleteClusterTag(resourceType ResourceType, resourceID string, csCluster *infrav1.CloudStackCluster) error
	DoClusterTagsAllowDisposal(resourceType ResourceType, resourceID string) (bool, error)
	AddTags(resourceType ResourceType, resourceID string, tags map[string]string) error
	GetTags(resourceType ResourceType, resourceID string) (map[string]string, error)
	DeleteTags(resourceType ResourceType, resourceID string, tagsToDelete map[string]string) error
}

type ResourceType string

const (
	clusterTagNamePrefix               = "CAPC_cluster_"
	createdByCAPCTagName               = "created_by_CAPC"
	ResourceTypeNetwork   ResourceType = "Network"
	ResourceTypeIPAddress ResourceType = "PublicIpAddress"
)

func (c *client) AddClusterTag(resourceType ResourceType, resourceID string, csCluster *infrav1.CloudStackCluster, addCreatedByCAPCTag bool) error {
	clusterTagName := generateClusterTagName(csCluster)
	newTags := map[string]string{}

	existingTags, err := c.GetTags(resourceType, resourceID)
	if err != nil {
		return err
	}

	if existingTags[clusterTagName] == "" {
		newTags[clusterTagName] = "1"
	}

	if addCreatedByCAPCTag && existingTags[createdByCAPCTagName] == "" {
		newTags[createdByCAPCTagName] = "1"
	}

	if len(newTags) > 0 {
		return c.AddTags(resourceType, resourceID, newTags)
	}

	return nil
}

func (c *client) DeleteClusterTag(resourceType ResourceType, resourceID string, csCluster *infrav1.CloudStackCluster) error {
	tags, err := c.GetTags(resourceType, resourceID)
	if err != nil {
		return err
	}

	clusterTagName := generateClusterTagName(csCluster)
	if tagValue := tags[clusterTagName]; tagValue != "" {
		return c.DeleteTags(resourceType, csCluster.Status.NetworkID, map[string]string{clusterTagName: tagValue})
	}

	return nil
}

func (c *client) DoClusterTagsAllowDisposal(resourceType ResourceType, resourceID string) (bool, error) {
	tags, err := c.GetTags(resourceType, resourceID)
	if err != nil {
		return false, err
	}

	var clusterTagCount int
	for tagName := range tags {
		if strings.HasPrefix(tagName, clusterTagNamePrefix) {
			clusterTagCount++
		}
	}

	return clusterTagCount == 0 && tags[createdByCAPCTagName] != "", nil
}

func (c *client) AddTags(resourceType ResourceType, resourceID string, tags map[string]string) error {
	p := c.cs.Resourcetags.NewCreateTagsParams([]string{resourceID}, string(resourceType), tags)
	_, err := c.cs.Resourcetags.CreateTags(p)
	return err
}

func (c *client) GetTags(resourceType ResourceType, resourceID string) (map[string]string, error) {
	p := c.cs.Resourcetags.NewListTagsParams()
	p.SetResourceid(resourceID)
	p.SetResourcetype(string(resourceType))
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

func (c *client) DeleteTags(resourceType ResourceType, resourceID string, tagsToDelete map[string]string) error {
	p := c.cs.Resourcetags.NewDeleteTagsParams([]string{resourceID}, string(resourceType))
	p.SetTags(tagsToDelete)
	_, err := c.cs.Resourcetags.DeleteTags(p)
	return err
}

func generateClusterTagName(csCluster *infrav1.CloudStackCluster) string {
	return clusterTagNamePrefix + string(csCluster.UID)
}
