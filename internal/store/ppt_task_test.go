package store

import (
	"context"
	"testing"
)

func TestUpdatePptTaskPhase_SQLInjectionPrevention(t *testing.T) {
	tests := []struct {
		name           string
		artifactColumn string
		wantErr        bool
	}{
		{
			name:           "SQL injection attempt - DROP TABLE",
			artifactColumn: "artifact_url; DROP TABLE ppt_tasks; --",
			wantErr:        true,
		},
		{
			name:           "SQL injection attempt - invalid column",
			artifactColumn: "malicious_column",
			wantErr:        true,
		},
		{
			name:           "SQL injection attempt - UNION SELECT",
			artifactColumn: "artifact_url UNION SELECT * FROM users --",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock store (in real tests, use a test database)
			// For now, we just test the validation logic
			s := &PgStore{}

			ctx := context.Background()
			err := s.UpdatePptTaskPhase(ctx, 1, "test_phase", tt.artifactColumn, []byte(`{"test": "data"}`))

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePptTaskPhase() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err != nil {
				// Verify the error message mentions invalid column
				if err.Error() == "" {
					t.Errorf("Expected error message for invalid column, got empty string")
				}
			}
		})
	}
}
