package apiclient

import (
	"context"

	"github.com/storageos/kubectl-storageos/api_client/openapi"
	"github.com/storageos/kubectl-storageos/model"
)

// GetNodeByName requests basic information for the node resource which has
// name.
//
// The resource model for the API is build around using unique identifiers,
// so this operation is inherently more expensive than the corresponding
// GetNode() operation.
//
// Retrieving a node resource by name involves requesting a list of all nodes
// in the cluster from the StorageOS API and returning the first node where the
// name matches.
func (c *client) GetNodeByName(ctx context.Context, name string) (*model.Node, error) {
	nodes, err := c.ListNodes(ctx)
	if err != nil {
		return nil, err
	}

	for _, n := range nodes {
		if n.Name == name {
			return n, nil
		}
	}

	return nil, openapi.NewNodeNameNotFoundError(name)
}

// GetListNodesByUID requests a list containing basic information on each
// node resource in the cluster.
//
// The returned list is filtered using uids so that it contains only those
// resources which have a matching ID. Omitting uids will skip the filtering.
func (c *client) GetListNodesByUID(ctx context.Context, uids ...string) ([]*model.Node, error) {
	nodes, err := c.ListNodes(ctx)
	if err != nil {
		return nil, err
	}

	return filterNodesForUIDs(nodes, uids...)
}

// GetListNodesByName requests a list containing basic information on each
// node resource in the cluster.
//
// The returned list is filtered using names so that it contains only those
// resources which have a matching name. Omitting names will skip the filtering.
func (c *client) GetListNodesByName(ctx context.Context, names ...string) ([]*model.Node, error) {
	nodes, err := c.ListNodes(ctx)
	if err != nil {
		return nil, err
	}

	return filterNodesForNames(nodes, names...)
}

// filterNodesForNames will return a subset of nodes containing resources
// which have one of the provided names. If names is not provided, nodes is
// returned as is.
//
// If there is no resource for a given name then an error is returned, thus
// this is a strict helper.
func filterNodesForNames(nodes []*model.Node, names ...string) ([]*model.Node, error) {
	// return everything if no filter names given
	if len(names) == 0 {
		return nodes, nil
	}

	retrieved := map[string]*model.Node{}

	for _, n := range nodes {
		retrieved[n.Name] = n
	}

	filtered := make([]*model.Node, 0, len(names))

	for _, name := range names {
		n, ok := retrieved[name]
		if !ok {
			return nil, openapi.NewNodeNameNotFoundError(name)
		}
		filtered = append(filtered, n)
	}

	return filtered, nil
}

// filterNodesForUIDs will return a subset of nodes containing resources
// which have one of the provided uids. If uids is not provided, nodes is
// returned as is.
//
// If there is no resource for a given uid then an error is returned, thus
// this is a strict helper.
func filterNodesForUIDs(nodes []*model.Node, uids ...string) ([]*model.Node, error) {
	// return everything if no filter uids given
	if len(uids) == 0 {
		return nodes, nil
	}

	retrieved := map[string]*model.Node{}

	for _, n := range nodes {
		retrieved[n.ID] = n
	}

	filtered := make([]*model.Node, 0, len(uids))

	for _, idVar := range uids {
		n, ok := retrieved[idVar]
		if !ok {
			return nil, openapi.NewNodeNotFoundError(idVar)
		}
		filtered = append(filtered, n)
	}

	return filtered, nil
}
