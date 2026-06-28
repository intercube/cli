package inventory

import (
	"encoding/json"
	"testing"
)

func TestPagedSiteResponseMatchesInventoryAPIShape(t *testing.T) {
	payload := []byte(`{
		"items": [
			{
				"id": "58",
				"username": "shop",
				"mainDomain": "shop.example.com",
				"isProduction": true,
				"serverId": "9",
				"serverName": "web-01"
			}
		],
		"meta": {
			"offset": 0,
			"limit": 100,
			"count": 1,
			"total": 1,
			"hasNext": false,
			"hasPrevious": false
		}
	}`)

	var response pagedResponse[SiteServer]
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("failed to decode paged site response: %v", err)
	}

	if len(response.Items) != 1 {
		t.Fatalf("expected one site, got %d", len(response.Items))
	}

	site := response.Items[0]
	if site.ID != "58" || site.Username != "shop" || site.MainDomain != "shop.example.com" || site.ServerID != "9" || site.ServerName != "web-01" {
		t.Fatalf("unexpected decoded site: %+v", site)
	}

	if response.Meta.HasNext {
		t.Fatalf("expected hasNext to decode as false")
	}
}
