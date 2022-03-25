package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOndatVolumes(t *testing.T) {
	tests := []struct {
		name string

		input string

		expectedVolumes []*Volume
		expectedErr     error
	}{
		{
			name: "success",
			input: `total 262144
-rw-rw---- 1 root disk 2147483648 Feb 25 15:18 d.d613df45-a162-4166-acf2-717a647e1150
brw-rw---- 1 root disk      8, 32 Feb 25 16:07 v.c3561d79-459f-4e5d-b5bb-f71ae7b38672
brw-rw---- 1 root disk      8, 48 Feb 25 15:18 v.78e88095-e690-49be-b0f3-3f735ef084a5
`,
			expectedVolumes: []*Volume{
				{
					Major: 8,
					Minor: 32,
					ID:    "c3561d79-459f-4e5d-b5bb-f71ae7b38672",
					PVC:   "",
				},
				{
					Major: 8,
					Minor: 48,
					ID:    "78e88095-e690-49be-b0f3-3f735ef084a5",
					PVC:   "",
				},
			},
			expectedErr: nil,
		},
		{
			name: "no volumes",
			input: `total 5767168
-rw-rw---- 1 root disk  2147483648 Feb 25 15:18 d.d613df45-a162-4166-acf2-717a647e1150
-rw-rw---- 1 root disk  2147483648 Feb 25 16:07 d.5a1efbf6-2d7a-4f2a-a04e-14fbb4e8894f
-rw-rw---- 1 root disk 10737418240 Feb 25 16:07 d.5fd17ece-93e1-455a-96cf-7150b3eef651
-rw-rw---- 1 root disk 10737418240 Feb 25 16:07 d.804b4a98-4497-48b1-a25f-9312c18c017f
`,
			expectedVolumes: []*Volume{},
			expectedErr:     nil,
		},
		{
			name: "no volumes or deployments",
			input: `total 0
`,
			expectedVolumes: []*Volume{},
			expectedErr:     nil,
		},
		{
			name: "unexpected value in input, invalid minor number",
			input: `total 262144
		-rw-rw---- 1 root disk 2147483648 Feb 25 15:18 d.d613df45-a162-4166-acf2-717a647e1150
		brw-rw---- 1 root disk     8, ops Feb 25 16:07 v.c3561d79-459f-4e5d-b5bb-f71ae7b38672
		`,
			expectedVolumes: []*Volume{},
			expectedErr:     nil,
		},
	}

	for _, tt := range tests {
		var tt = tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vols, err := parseOndatVolumes(strings.Split(tt.input, "\n"))
			if err != nil {
				require.EqualError(t, tt.expectedErr, err.Error())
			}

			require.EqualValues(t, tt.expectedVolumes, vols, "unexpected volumes returned")
		})
	}
}
