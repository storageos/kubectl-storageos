package apiclient

import (
	"context"

	"github.com/storageos/kubectl-storageos/api_client/openapi"
	"github.com/storageos/kubectl-storageos/model"
)

// GetListPolicyGroupsByUID requests a list containing basic information on each
// policy group configured for the cluster.
//
// The returned list is filtered using gids so that it contains  only those
// resources which have a matching GID. Omitting gids  will skip the filtering.
func (c *Client) GetListPolicyGroupsByUID(ctx context.Context, gids ...string) ([]*model.PolicyGroup, error) {
	policyGroups, err := c.ListPolicyGroups(ctx)
	if err != nil {
		return nil, err
	}

	return filterPolicyGroupsForIDs(policyGroups, gids...)
}

// GetPolicyGroupByName requests a policy group given its name
func (c *Client) GetPolicyGroupByName(ctx context.Context, name string) (*model.PolicyGroup, error) {
	policyGroups, err := c.ListPolicyGroups(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range policyGroups {
		if p.Name == name {
			return p, nil
		}
	}

	return nil, openapi.NewPolicyGroupNameNotFoundError(name)
}

// GetListPolicyGroupsByName requests a list containing basic information on each
// policy group configured for the cluster.
//
// The returned list is filtered using name so that it contains  only those
// resources which have a matching name. Omitting gids  will skip the filtering.
func (c *Client) GetListPolicyGroupsByName(ctx context.Context, names ...string) ([]*model.PolicyGroup, error) {
	policyGroups, err := c.ListPolicyGroups(ctx)
	if err != nil {
		return nil, err
	}

	return filterPolicyGroupsForNames(policyGroups, names...)
}

// filterPolicyGroupsForNames will return a subset of policyGroups containing
// resources which have one of the provided names. If names is not provided,
// policyGroups is returned as is.
//
// If there is no resource for a given name then an error is returned, thus
// this is a strict helper.
func filterPolicyGroupsForNames(policyGroups []*model.PolicyGroup, names ...string) ([]*model.PolicyGroup, error) {
	// return everything if no filter names given
	if len(names) == 0 {
		return policyGroups, nil
	}

	// implicitly removes also duplicates, if any, in the names list
	retrieved := map[string]*model.PolicyGroup{}
	for _, g := range policyGroups {
		retrieved[g.Name] = g
	}

	filtered := make([]*model.PolicyGroup, 0, len(names))

	for _, name := range names {
		g, ok := retrieved[name]
		if !ok {
			return nil, openapi.NewPolicyGroupNameNotFoundError(name)
		}
		filtered = append(filtered, g)
	}

	return filtered, nil
}

// filterPolicyGroupsForIDs will return a subset of policyGroups containing
// resources which have one of the provided gids. If gids is not provided,
// policyGroups is returned as is.
//
// If there is no resource for a given gid then an error is returned, thus
// this is a strict helper.
func filterPolicyGroupsForIDs(policyGroups []*model.PolicyGroup, gids ...string) ([]*model.PolicyGroup, error) {
	if len(gids) == 0 {
		return policyGroups, nil
	}

	// implicitly removes also duplicates, if any, in the gids list
	retrieved := map[string]*model.PolicyGroup{}
	for _, g := range policyGroups {
		retrieved[g.ID] = g
	}

	filtered := make([]*model.PolicyGroup, 0, len(gids))

	for _, gid := range gids {
		g, ok := retrieved[gid]
		if !ok {
			return nil, openapi.NewPolicyGroupIDNotFoundError(gid)
		}
		filtered = append(filtered, g)
	}

	return filtered, nil
}
