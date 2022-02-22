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
	AddClusterTag(resourceType ResourceType, resourceId string, csCluster *infrav1.CloudStackCluster, addCreatedByCapcTag bool) error
	DeleteClusterTag(resourceType ResourceType, resourceId string, csCluster *infrav1.CloudStackCluster) error
	DoClusterTagsAllowDisposal(resourceType ResourceType, resourceId string) (bool, error)
	AddTags(resourceType ResourceType, resourceId string, tags map[string]string) error
	GetTags(resourceType ResourceType, resourceId string) (map[string]string, error)
	DeleteTags(resourceType ResourceType, resourceId string, tagsToDelete map[string]string) error
}

type ResourceType string

const (
	clusterTagNamePrefix               = "CAPC_cluster_"
	createdByCapcTagName               = "created_by_CAPC"
	ResourceTypeNetwork   ResourceType = "network"
	ResourceTypeIpAddress ResourceType = "ipaddress"
)

func (c *client) AddClusterTag(resourceType ResourceType, resourceId string, csCluster *infrav1.CloudStackCluster, addCreatedByCapcTag bool) error {
	clusterTagName := generateClusterTagName(csCluster)
	newTags := map[string]string{}

	existingTags, err := c.GetTags(resourceType, resourceId)
	if err != nil {
		return err
	}

	if existingTags[clusterTagName] == "" {
		newTags[clusterTagName] = "1"
	}

	if addCreatedByCapcTag && existingTags[createdByCapcTagName] == "" {
		newTags[createdByCapcTagName] = "1"
	}

	if len(newTags) > 0 {
		return c.AddTags(resourceType, resourceId, newTags)
	}

	return nil
}

func (c *client) DeleteClusterTag(resourceType ResourceType, resourceId string, csCluster *infrav1.CloudStackCluster) error {
	tags, err := c.GetTags(resourceType, csCluster.Status.NetworkID)
	if err != nil {
		return err
	}

	clusterTagName := generateClusterTagName(csCluster)
	if tagValue := tags[clusterTagName]; tagValue != "" {
		return c.DeleteTags(resourceType, csCluster.Status.NetworkID, map[string]string{clusterTagName: tagValue})
	}

	return nil
}

func (c *client) DoClusterTagsAllowDisposal(resourceType ResourceType, resourceId string) (bool, error) {
	tags, err := c.GetTags(resourceType, resourceId)
	if err != nil {
		return false, err
	}

	var clusterTagCount int
	for tagName := range tags {
		if strings.HasPrefix(tagName, clusterTagNamePrefix) {
			clusterTagCount++
		}
	}

	return clusterTagCount == 0 && tags[createdByCapcTagName] != "", nil
}

func (c *client) AddTags(resourceType ResourceType, resourceId string, tags map[string]string) error {
	p := c.cs.Resourcetags.NewCreateTagsParams([]string{resourceId}, string(resourceType), tags)
	_, err := c.cs.Resourcetags.CreateTags(p)
	return err
}

func (c *client) GetTags(resourceType ResourceType, resourceId string) (map[string]string, error) {
	p := c.cs.Resourcetags.NewListTagsParams()
	p.SetResourceid(resourceId)
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

func (c *client) DeleteTags(resourceType ResourceType, resourceId string, tagsToDelete map[string]string) error {
	p := c.cs.Resourcetags.NewDeleteTagsParams([]string{resourceId}, string(resourceType))
	p.SetTags(tagsToDelete)
	_, err := c.cs.Resourcetags.DeleteTags(p)
	return err
}

func generateClusterTagName(csCluster *infrav1.CloudStackCluster) string {
	return clusterTagNamePrefix + string(csCluster.UID)
}
