package service

import (
	"testing"

	"github.com/ikennarichard/insighta/internal/domain"
)

func TestNormalizeFilters(t *testing.T) {
	tests := []struct {
		name     string
		a        domain.ProfileFilters
		b        domain.ProfileFilters
		sameKey  bool
	}{
		{
			name:    "male vs males",
			a:       domain.ProfileFilters{Gender: "male"},
			b:       domain.ProfileFilters{Gender: "males"},
			sameKey: true,
		},
		{
			name:    "female vs women",
			a:       domain.ProfileFilters{Gender: "female"},
			b:       domain.ProfileFilters{Gender: "women"},
			sameKey: true,
		},
		{
			name:    "country lowercase vs uppercase",
			a:       domain.ProfileFilters{CountryID: "ng"},
			b:       domain.ProfileFilters{CountryID: "NG"},
			sameKey: true,
		},
		{
			name:    "asc vs ascending",
			a:       domain.ProfileFilters{Order: "asc"},
			b:       domain.ProfileFilters{Order: "ascending"},
			sameKey: true,
		},
		{
			name:    "teenager vs youth",
			a:       domain.ProfileFilters{AgeGroup: "teenager"},
			b:       domain.ProfileFilters{AgeGroup: "youth"},
			sameKey: true,
		},
		{
			name:    "male vs female should differ",
			a:       domain.ProfileFilters{Gender: "male"},
			b:       domain.ProfileFilters{Gender: "female"},
			sameKey: false,
		},
		{
			name:    "invalid sort falls back to created_at",
			a:       domain.ProfileFilters{SortBy: "invalid_col"},
			b:       domain.ProfileFilters{SortBy: "created_at"},
			sameKey: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyA := NormalizedCacheKey(tt.a, 1, 10)
			keyB := NormalizedCacheKey(tt.b, 1, 10)
			if tt.sameKey && keyA != keyB {
				t.Errorf("expected same key\n  A: %s\n  B: %s", keyA, keyB)
			}
			if !tt.sameKey && keyA == keyB {
				t.Errorf("expected different keys but got same: %s", keyA)
			}
		})
	}
}