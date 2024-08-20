package manageddatabases

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/odata"
)

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type ListByInstanceOperationResponse struct {
	HttpResponse *http.Response
	OData        *odata.OData
	Model        *[]ManagedDatabase
}

type ListByInstanceCompleteResult struct {
	LatestHttpResponse *http.Response
	Items              []ManagedDatabase
}

type ListByInstanceCustomPager struct {
	NextLink *odata.Link `json:"nextLink"`
}

func (p *ListByInstanceCustomPager) NextPageLink() *odata.Link {
	defer func() {
		p.NextLink = nil
	}()

	return p.NextLink
}

// ListByInstance ...
func (c ManagedDatabasesClient) ListByInstance(ctx context.Context, id commonids.SqlManagedInstanceId) (result ListByInstanceOperationResponse, err error) {
	opts := client.RequestOptions{
		ContentType: "application/json; charset=utf-8",
		ExpectedStatusCodes: []int{
			http.StatusOK,
		},
		HttpMethod: http.MethodGet,
		Pager:      &ListByInstanceCustomPager{},
		Path:       fmt.Sprintf("%s/databases", id.ID()),
	}

	req, err := c.Client.NewRequest(ctx, opts)
	if err != nil {
		return
	}

	var resp *client.Response
	resp, err = req.ExecutePaged(ctx)
	if resp != nil {
		result.OData = resp.OData
		result.HttpResponse = resp.Response
	}
	if err != nil {
		return
	}

	var values struct {
		Values *[]ManagedDatabase `json:"value"`
	}
	if err = resp.Unmarshal(&values); err != nil {
		return
	}

	result.Model = values.Values

	return
}

// ListByInstanceComplete retrieves all the results into a single object
func (c ManagedDatabasesClient) ListByInstanceComplete(ctx context.Context, id commonids.SqlManagedInstanceId) (ListByInstanceCompleteResult, error) {
	return c.ListByInstanceCompleteMatchingPredicate(ctx, id, ManagedDatabaseOperationPredicate{})
}

// ListByInstanceCompleteMatchingPredicate retrieves all the results and then applies the predicate
func (c ManagedDatabasesClient) ListByInstanceCompleteMatchingPredicate(ctx context.Context, id commonids.SqlManagedInstanceId, predicate ManagedDatabaseOperationPredicate) (result ListByInstanceCompleteResult, err error) {
	items := make([]ManagedDatabase, 0)

	resp, err := c.ListByInstance(ctx, id)
	if err != nil {
		result.LatestHttpResponse = resp.HttpResponse
		err = fmt.Errorf("loading results: %+v", err)
		return
	}
	if resp.Model != nil {
		for _, v := range *resp.Model {
			if predicate.Matches(v) {
				items = append(items, v)
			}
		}
	}

	result = ListByInstanceCompleteResult{
		LatestHttpResponse: resp.HttpResponse,
		Items:              items,
	}
	return
}
