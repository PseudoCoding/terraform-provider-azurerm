package privateendpointconnections

import (
	"context"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

type DeleteOperationResponse struct {
	HttpResponse *http.Response
	Model        *PrivateEndpointConnection
}

type DeleteOperationOptions struct {
	XMsClientRequestId *string
}

func DefaultDeleteOperationOptions() DeleteOperationOptions {
	return DeleteOperationOptions{}
}

func (o DeleteOperationOptions) toHeaders() map[string]interface{} {
	out := make(map[string]interface{})

	if o.XMsClientRequestId != nil {
		out["x-ms-client-request-id"] = *o.XMsClientRequestId
	}

	return out
}

func (o DeleteOperationOptions) toQueryString() map[string]interface{} {
	out := make(map[string]interface{})

	return out
}

// Delete ...
func (c PrivateEndpointConnectionsClient) Delete(ctx context.Context, id PrivateEndpointConnectionId, options DeleteOperationOptions) (result DeleteOperationResponse, err error) {
	req, err := c.preparerForDelete(ctx, id, options)
	if err != nil {
		err = autorest.NewErrorWithError(err, "privateendpointconnections.PrivateEndpointConnectionsClient", "Delete", nil, "Failure preparing request")
		return
	}

	result.HttpResponse, err = c.Client.Send(req, azure.DoRetryWithRegistration(c.Client))
	if err != nil {
		err = autorest.NewErrorWithError(err, "privateendpointconnections.PrivateEndpointConnectionsClient", "Delete", result.HttpResponse, "Failure sending request")
		return
	}

	result, err = c.responderForDelete(result.HttpResponse)
	if err != nil {
		err = autorest.NewErrorWithError(err, "privateendpointconnections.PrivateEndpointConnectionsClient", "Delete", result.HttpResponse, "Failure responding to request")
		return
	}

	return
}

// preparerForDelete prepares the Delete request.
func (c PrivateEndpointConnectionsClient) preparerForDelete(ctx context.Context, id PrivateEndpointConnectionId, options DeleteOperationOptions) (*http.Request, error) {
	queryParameters := map[string]interface{}{
		"api-version": defaultApiVersion,
	}

	for k, v := range options.toQueryString() {
		queryParameters[k] = autorest.Encode("query", v)
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsDelete(),
		autorest.WithBaseURL(c.baseUri),
		autorest.WithHeaders(options.toHeaders()),
		autorest.WithPath(id.ID()),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// responderForDelete handles the response to the Delete request. The method always
// closes the http.Response Body.
func (c PrivateEndpointConnectionsClient) responderForDelete(resp *http.Response) (result DeleteOperationResponse, err error) {
	err = autorest.Respond(
		resp,
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result.Model),
		autorest.ByClosing())
	result.HttpResponse = resp
	return
}
