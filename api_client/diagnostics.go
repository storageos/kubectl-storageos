package apiclient

import (
	"context"
	"io"
)

func (c *client) GetSingleNodeDiagnosticsByName(ctx context.Context, name string) (io.ReadCloser, string, error) {
	node, err := c.GetNodeByName(ctx, name)
	if err != nil {
		return nil, "", err
	}

	return c.GetSingleNodeDiagnostics(ctx, node.ID)
}
