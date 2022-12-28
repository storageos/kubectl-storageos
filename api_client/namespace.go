package apiclient

import (
	"context"

	"github.com/storageos/kubectl-storageos/api_client/openapi"
	"github.com/storageos/kubectl-storageos/model"
)

// GetNamespaceByName requests basic information for the namespace resource
// which has the given name.
//
// The resource model for the API is build around using unique identifiers,
// so this operation is inherently more expensive than the corresponding
// GetNamespace() operation.
//
// Retrieving a namespace resource by name involves requesting a list of all
// namespaces from the StorageOS API and returning the first one where the
// name matches.
func (c *client) GetNamespaceByName(ctx context.Context, name string) (*model.Namespace, error) {
	namespaces, err := c.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	for _, ns := range namespaces {
		if ns.Name == name {
			return ns, nil
		}
	}

	return nil, openapi.NewNamespaceNameNotFoundError(name)
}

// GetListNamespacesByUID requests a list of namespace resources present in the
// cluster.
//
// The returned list is filtered using uids so that it contains only those
// namespace resources which have a matching ID. If no uids are given then
// all namespaces are returned.
func (c *client) GetListNamespacesByUID(ctx context.Context, uids ...string) ([]*model.Namespace, error) {
	resources, err := c.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	return filterNamespacesForUIDs(resources, uids...)
}

// GetListNamespacesByName requests a list of namespace resources present in
// the cluster.
//
// The returned list is filtered using names so that it contains only those
// namespaces resources which have a matching name. If no names are given then
// all namespaces are returned.
func (c *client) GetListNamespacesByName(ctx context.Context, names ...string) ([]*model.Namespace, error) {
	resources, err := c.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	return filterNamespacesForNames(resources, names...)
}

// filterNamespacesForNames will return a subset of namespaces containing
// resources which have one of the provided names. If names is not provided,
// namespaces is returned as is.
//
// If there is no resource for a given name then an error is returned, thus
// this is a strict helper.
func filterNamespacesForNames(namespaces []*model.Namespace, names ...string) ([]*model.Namespace, error) {
	if len(names) == 0 {
		return namespaces, nil
	}

	retrieved := map[string]*model.Namespace{}

	for _, ns := range namespaces {
		retrieved[ns.Name] = ns
	}

	filtered := make([]*model.Namespace, 0, len(names))
	for _, name := range names {
		ns, ok := retrieved[name]
		if !ok {
			return nil, openapi.NewNamespaceNameNotFoundError(name)
		}
		filtered = append(filtered, ns)
	}

	return filtered, nil
}

// filterNamespacesForUIDS will return a subset of namespaces containing
// resources which have one of the provided uids. If uids is not provided,
// namespaces is returned as is.
//
// If there is no resource for a given uid then an error is returned, thus
// this is a strict helper.
func filterNamespacesForUIDs(namespaces []*model.Namespace, uids ...string) ([]*model.Namespace, error) {
	if len(uids) == 0 {
		return namespaces, nil
	}

	retrieved := map[string]*model.Namespace{}

	for _, ns := range namespaces {
		retrieved[ns.ID] = ns
	}

	filtered := make([]*model.Namespace, 0, len(uids))

	for _, idVar := range uids {
		ns, ok := retrieved[idVar]
		if !ok {
			return nil, openapi.NewNamespaceNotFoundError(idVar)
		}
		filtered = append(filtered, ns)
	}

	return filtered, nil
}
