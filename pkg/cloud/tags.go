/*
Copyright 2022.

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

type TagIFace interface {
	TagNetwork(string, map[string]string) error
}

// TagNetwork adds tags to a network by network id.
func (c *client) TagNetwork(networkId string, tag map[string]string) error {
	p := c.cs.Resourcetags.NewCreateTagsParams([]string{"someid"}, "network", map[string]string{"some": "tag"})
	_, err := c.cs.Resourcetags.CreateTags(p)
	return err
}
