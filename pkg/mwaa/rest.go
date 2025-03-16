package mwaa

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/mwaa/types"
)

func (c *Client) RestAPIGet(ctx context.Context, environmentName, path string, queryParams map[string]any, response any) error {
	output, err := c.InvokeRestAPI(ctx, types.RestApiMethodGet, environmentName, path, queryParams, nil)
	if err != nil {
		return err
	}

	return output.RestApiResponse.UnmarshalSmithyDocument(response)
}
