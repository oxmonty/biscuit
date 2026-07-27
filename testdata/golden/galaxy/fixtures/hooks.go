// Hand-written extension exercising the generated client's exported API.
// biscuit's compile-the-output CI builds this file inside the galaxy golden
// repo, so a template change that breaks the internal/custom contract —
// client construction, request shapes, operation signatures, response
// fields — fails in biscuit's own CI before it ever reaches a user's repo.
package custom

import (
	"context"
	"net/url"

	"github.com/oxmonty/galaxy-cli/internal/client"
)

// PlanetBody fetches one planet and returns the raw response body — a
// representative custom helper touching every stable surface: Client,
// a Request struct with a path field and Query, an operation method, and
// Response.Status/Body.
func PlanetBody(ctx context.Context, baseURL, planetID string) ([]byte, error) {
	c := &client.Client{BaseURL: baseURL}
	resp, err := c.PlanetsGet(ctx, &client.PlanetsGetRequest{
		PlanetId: planetID,
		Query:    url.Values{},
	})
	if err != nil {
		return nil, err
	}
	if resp.Status >= 400 {
		return nil, context.Canceled
	}
	return resp.Body, nil
}
