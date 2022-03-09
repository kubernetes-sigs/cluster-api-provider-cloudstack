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

	"github.com/hashicorp/go-multierror"

	infrav1 "github.com/aws/cluster-api-provider-cloudstack/api/v1beta1"
	"github.com/pkg/errors"
)

type TagIface interface {
	AddClusterTag(ResourceType, string, *infrav1.CloudStackCluster) error
	DeleteClusterTag(ResourceType, string, *infrav1.CloudStackCluster) error
	AddCreatedByCAPCTag(ResourceType, string) error
	DeleteCreatedByCAPCTag(ResourceType, string) error
	DoClusterTagsAllowDisposal(ResourceType, string) (bool, error)
	AddTags(ResourceType, string, map[string]string) error
	GetTags(ResourceType, string) (map[string]string, error)
	DeleteTags(ResourceType, string, map[string]string) error
}

type ResourceType string

const (
	ClusterTagNamePrefix               = "CAPC_cluster_"
	CreatedByCAPCTagName               = "created_by_CAPC"
	ResourceTypeNetwork   ResourceType = "Network"
	ResourceTypeIPAddress ResourceType = "PublicIpAddress"
)

// ignoreAlreadyPresentErrors returns nil if the error is an already present tag error.
func ignoreAlreadyPresentErrors(err error, rType ResourceType, rID string) error {
	matchSubString := strings.ToLower("already on " + string(rType) + " with id " + rID)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), matchSubString) {
		return err
	}
	return nil
}

func (c *client) IsCapcManaged(resourceType ResourceType, resourceID string) (bool, error) {
	tags, err := c.GetTags(resourceType, resourceID)
	if err != nil {
		return false, errors.Wrapf(err,
			"error encountered while checking if %s with ID: %s is tagged as CAPC managed", resourceType, resourceID)
	}
	_, CreatedByCAPC := tags[CreatedByCAPCTagName]
	return CreatedByCAPC, nil
}

// AddClusterTag adds cluster tag to a resource. This tag indicates the resource is used by a given the cluster.
func (c *client) AddClusterTag(rType ResourceType, rID string, csCluster *infrav1.CloudStackCluster) error {
	if managedByCAPC, err := c.IsCapcManaged(rType, rID); err != nil {
		return err
	} else if managedByCAPC {
		ClusterTagName := generateClusterTagName(csCluster)
		return c.AddTags(rType, rID, map[string]string{ClusterTagName: "1"})
	}
	return nil
}

// DeleteClusterTag deletes the tag that associates the resource with a given cluster.
func (c *client) DeleteClusterTag(rType ResourceType, rID string, csCluster *infrav1.CloudStackCluster) error {
	if managedByCAPC, err := c.IsCapcManaged(rType, rID); err != nil {
		return err
	} else if managedByCAPC {
		ClusterTagName := generateClusterTagName(csCluster)
		return c.DeleteTags(rType, rID, map[string]string{ClusterTagName: "1"})
	}
	return nil
}

// AddCreatedByCAPCTag adds the tag that indicates that the resource was created by CAPC.
// This is useful when a resource is disassociated but not deleted.
func (c *client) AddCreatedByCAPCTag(rType ResourceType, rID string) error {
	return c.AddTags(rType, rID, map[string]string{CreatedByCAPCTagName: "1"})
}

// DeleteCreatedByCAPCTag deletes the tag that indicates that the resource was created by CAPC.
func (c *client) DeleteCreatedByCAPCTag(rType ResourceType, rID string) error {
	return c.DeleteTags(rType, rID, map[string]string{CreatedByCAPCTagName: "1"})
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
		if strings.HasPrefix(tagName, ClusterTagNamePrefix) {
			clusterTagCount++
		}
	}

	return clusterTagCount == 0 && tags[CreatedByCAPCTagName] != "", nil
}

// AddTags adds arbitrary tags to a resource.
func (c *client) AddTags(resourceType ResourceType, resourceID string, tags map[string]string) error {
	p := c.cs.Resourcetags.NewCreateTagsParams([]string{resourceID}, string(resourceType), tags)
	_, err := c.cs.Resourcetags.CreateTags(p)
	return ignoreAlreadyPresentErrors(err, resourceType, resourceID)
}

// GetTags gets all of a resource's tags.
func (c *client) GetTags(resourceType ResourceType, resourceID string) (map[string]string, error) {
	p := c.cs.Resourcetags.NewListTagsParams()
	p.SetResourceid(resourceID)
	p.SetResourcetype(string(resourceType))
	p.SetListall(true)
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

// DeleteTags deletes the given tags from a resource.
// Ignores errors if the tag is not present.
func (c *client) DeleteTags(resourceType ResourceType, resourceID string, tagsToDelete map[string]string) error {
	for tagkey, tagval := range tagsToDelete {
		p := c.cs.Resourcetags.NewDeleteTagsParams([]string{resourceID}, string(resourceType))
		p.SetTags(tagsToDelete)
		if _, err1 := c.cs.Resourcetags.DeleteTags(p); err1 != nil { // Error in deletion attempt. Check for tag.
			currTag := map[string]string{tagkey: tagval}
			if tags, err2 := c.GetTags(resourceType, resourceID); len(tags) != 0 {
				if _, foundTag := tags[tagkey]; foundTag {
					return errors.Wrapf(multierror.Append(err1, err2),
						"could not remove tag %s from %s with ID %s", currTag, resourceType, resourceID)
				}
			}
		}
	}
	return nil
}

func generateClusterTagName(csCluster *infrav1.CloudStackCluster) string {
	return ClusterTagNamePrefix + string(csCluster.UID)
}
