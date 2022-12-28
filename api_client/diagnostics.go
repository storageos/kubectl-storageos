package apiclient

import (
	"context"
	"io"
)

func (c *Client) GetSingleNodeDiagnosticsByName(ctx context.Context, name string) (io.ReadCloser, string, error) {
	node, err := c.GetNodeByName(ctx, name)
	if err != nil {
		return nil, "", err
	}

	return c.Transport.GetSingleNodeDiagnostics(ctx, node.ID)
}
