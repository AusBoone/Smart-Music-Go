package amazonmusic

import (
	"context"
	"testing"
)

// TestClientErrors ensures the placeholder methods return errors.
func TestClientErrors(t *testing.T) {
	c := &Client{}
	if _, err := c.SearchTrack(context.Background(), "q"); err == nil {
		t.Fatal("expected error from SearchTrack")
	}
	if _, err := c.GetRecommendations(context.Background(), []string{"1"}); err == nil {
		t.Fatal("expected error from GetRecommendations")
	}
}
