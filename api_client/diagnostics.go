package apiclient

import (
	"context"
	"io"
)

// GetSingleNodeDiagnosticsByName requests a local diagnostics bundle from the given
// node.
func (c *Client) GetSingleNodeDiagnosticsByName(ctx context.Context, name string) (io.ReadCloser, string, error) {
	node, err := c.GetNodeByName(ctx, name)
	if err != nil {
		return nil, "", err
	}

	return c.Transport.GetSingleNodeDiagnostics(ctx, node.ID)
}
