package acl

import (
	"context"
	"fmt"
	"net/http"
)

// CatalogClient operates on the /rest/security/acl/catalog endpoint
// and the related /rest/security/acl/catalog/reload endpoint.
//
// The catalog mode is a singleton configuration value — there is no
// per-rule list shape here. Three modes are documented:
// [CatalogModeHide], [CatalogModeMixed], [CatalogModeChallenge]. See
// each constant's doc for the operational difference.
type CatalogClient struct {
	core Core
}

// catalogModeWire is the JSON envelope GeoServer uses for both GET
// and PUT against /rest/security/acl/catalog.
type catalogModeWire struct {
	Mode CatalogMode `json:"mode"`
}

// Get returns the current catalog mode.
func (c *CatalogClient) Get(ctx context.Context) (CatalogMode, error) {
	const op = "ACL.Catalog.Get"
	u, err := c.core.URL("rest", "security", "acl", "catalog")
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	var out catalogModeWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &out); err != nil {
		return "", err
	}
	return out.Mode, nil
}

// Update sets the catalog mode. mode must be one of
// [CatalogModeHide], [CatalogModeMixed], or [CatalogModeChallenge];
// other values are rejected by GeoServer with 422 Unprocessable
// Entity.
func (c *CatalogClient) Update(ctx context.Context, mode CatalogMode) error {
	const op = "ACL.Catalog.Update"
	u, err := c.core.URL("rest", "security", "acl", "catalog")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := catalogModeWire{Mode: mode}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// Reload reloads the GeoServer Security Manager catalog and
// configuration from disk. Used after an external tool has modified
// the on-disk security configuration. GeoServer accepts both PUT
// and POST for this endpoint; this client uses POST.
func (c *CatalogClient) Reload(ctx context.Context) error {
	const op = "ACL.Catalog.Reload"
	u, err := c.core.URL("rest", "security", "acl", "catalog", "reload")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, nil, nil, nil)
}
