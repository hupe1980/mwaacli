package mwaa

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/mwaa/types"
)

func (c *Client) ListDags(ctx context.Context, environmentName string, queryParams map[string]any) ([]map[string]any, error) {
	output, err := c.InvokeRestAPI(ctx, types.RestApiMethodGet, environmentName, "/dags", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Dags []map[string]any `json:"dags"`
	}

	if err := output.RestApiResponse.UnmarshalSmithyDocument(&result); err != nil {
		return nil, err
	}

	return result.Dags, nil
}

func (c *Client) GetDag(ctx context.Context, environmentName, dagID string, queryParams map[string]any) (map[string]any, error) {
	output, err := c.InvokeRestAPI(ctx, types.RestApiMethodGet, environmentName, fmt.Sprintf("/dags/%s", dagID), queryParams, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]any

	if err := output.RestApiResponse.UnmarshalSmithyDocument(&result); err != nil {
		return nil, err
	}

	return result, nil
}
