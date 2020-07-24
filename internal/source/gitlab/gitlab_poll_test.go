package gitlab

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
)

func TestClient_Poll(t *testing.T) {
	tests := []struct {
		name     string
		retries  int
		interval time.Duration
		wantErr  bool
	}{
		{
			name:     "success_with_no_retry",
			retries:  0,
			interval: 5 * time.Millisecond,
			wantErr:  false,
		},
		{
			name:     "success_after_N_retries",
			retries:  3,
			interval: 10 * time.Millisecond,
			wantErr:  false,
		},
		{
			name:     "fail_with_no_retries",
			retries:  0,
			interval: 5 * time.Millisecond,
			wantErr:  true,
		},
		{
			name:     "fail_after_N_retries",
			retries:  3,
			interval: 5 * time.Millisecond,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var counter int
			checkerMock := checkerMock{StatusErr: func() error {
				if tt.wantErr {
					return fmt.Errorf(client.ConnectionErrorMsg)
				}

				if counter < tt.retries {
					counter++
					return fmt.Errorf(client.ConnectionErrorMsg)
				}

				return nil
			}}

			glClient := Gitlab{checker: checkerMock}

			err := glClient.Poll(tt.retries, tt.interval)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "polling failed after")
				require.Contains(t, err.Error(), client.ConnectionErrorMsg)
				return
			}

			require.NoError(t, err)
		})
	}
}

type checkerMock struct {
	StatusErr func() error
}

func (c checkerMock) Status() error {
	return c.StatusErr()
}
