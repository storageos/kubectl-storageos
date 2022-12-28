package apiclient

import (
	"context"

	"github.com/storageos/kubectl-storageos/api_client/openapi"
	"github.com/storageos/kubectl-storageos/model"
)

func (c *client) GetUserByName(ctx context.Context, username string) (*model.User, error) {
	list, err := c.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	for _, u := range list {
		if u.Username == username {
			return u, nil
		}
	}

	return nil, openapi.NewUserNameNotFoundError(username)
}

func (c *client) GetListUsersByUID(ctx context.Context, uids []string) ([]*model.User, error) {
	list, err := c.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	toMap := make(map[string]*model.User)
	for _, u := range list {
		toMap[u.ID] = u
	}

	filtered := make([]*model.User, 0)
	for _, idVar := range uids {
		u, ok := toMap[idVar]
		if !ok {
			return nil, openapi.NewUserNotFoundError("user not found", idVar)
		}
		filtered = append(filtered, u)
	}

	return filtered, nil
}

func (c *client) GetListUsersByUsername(ctx context.Context, usernames []string) ([]*model.User, error) {
	list, err := c.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	toMap := make(map[string]*model.User)
	for _, u := range list {
		toMap[u.Username] = u
	}

	filtered := make([]*model.User, 0)
	for _, username := range usernames {
		u, ok := toMap[username]
		if !ok {
			return nil, openapi.NewUserNameNotFoundError(username)
		}
		filtered = append(filtered, u)
	}

	return filtered, nil
}
