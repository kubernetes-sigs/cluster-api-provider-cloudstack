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
	"strings"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
)

type TagIface interface {
	AddClusterTag(ResourceType, string, *infrav1.CloudStackCluster, bool) error
	DeleteClusterTag(ResourceType, string, *infrav1.CloudStackCluster) error
	DeleteCreatedByCAPCTag(ResourceType, string) error
	DoClusterTagsAllowDisposal(ResourceType, string) (bool, error)
	AddTags(ResourceType, string, map[string]string) error
	GetTags(ResourceType, string) (map[string]string, error)
	DeleteTags(ResourceType, string, map[string]string) error
}

type ResourceType string

const (
	clusterTagNamePrefix               = "CAPC_cluster_"
	createdByCAPCTagName               = "created_by_CAPC"
	ResourceTypeNetwork   ResourceType = "Network"
	ResourceTypeIPAddress ResourceType = "PublicIpAddress"
)

// AddClusterTag adds cluster-related tags to a resource.  One tag indicates that the resource is used by a given
// cluster. The other tag, if applied, indicates that CAPC created the resource and may dispose of it later.
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

// DeleteClusterTag deletes the tag that associates the resource with a given cluster.
func (c *client) DeleteClusterTag(resourceType ResourceType, resourceID string, csCluster *infrav1.CloudStackCluster) error {
	tags, err := c.GetTags(resourceType, resourceID)
	if err != nil {
		return err
	}

	clusterTagName := generateClusterTagName(csCluster)
	if tagValue := tags[clusterTagName]; tagValue != "" {
		return c.DeleteTags(resourceType, resourceID, map[string]string{clusterTagName: tagValue})
	}

	return nil
}

// DeleteCreatedByCAPCTag deletes the tag that indicates that the resource was created by CAPC.  This is useful when a
// resource is disassociated instead of deleted.  That way the tag won't cause confusion if the resource is reused later.
func (c *client) DeleteCreatedByCAPCTag(resourceType ResourceType, resourceID string) error {
	tags, err := c.GetTags(resourceType, resourceID)
	if err != nil {
		return err
	}

	if tagValue := tags[createdByCAPCTagName]; tagValue != "" {
		return c.DeleteTags(resourceType, resourceID, map[string]string{createdByCAPCTagName: tagValue})
	}

	return nil
}

// DoClusterTagsAllowDisposal checks to see if the resource is in a state that makes it eligible for disposal.  CAPC can
// dispose of a resource if the tags show it was created by CAPC and isn't being used by any clusters.
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

// AddTags adds arbitrary tags to a resource.
func (c *client) AddTags(resourceType ResourceType, resourceID string, tags map[string]string) error {
	p := c.cs.Resourcetags.NewCreateTagsParams([]string{resourceID}, string(resourceType), tags)
	_, err := c.cs.Resourcetags.CreateTags(p)
	return err
}

// GetTags gets all of a resource's tags.
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

// DeleteTags deletes the given tags from a resource.   If the tags don't exist, or if the values don't match, it will
// result in an error.
func (c *client) DeleteTags(resourceType ResourceType, resourceID string, tagsToDelete map[string]string) error {
	p := c.cs.Resourcetags.NewDeleteTagsParams([]string{resourceID}, string(resourceType))
	p.SetTags(tagsToDelete)
	_, err := c.cs.Resourcetags.DeleteTags(p)
	return err
}

func generateClusterTagName(csCluster *infrav1.CloudStackCluster) string {
	return clusterTagNamePrefix + string(csCluster.UID)
}
