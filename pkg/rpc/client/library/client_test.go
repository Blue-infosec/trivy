package library

import (
	"context"
	"testing"
	"time"

	"golang.org/x/xerrors"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	deptypes "github.com/aquasecurity/go-dep-parser/pkg/types"
	dbTypes "github.com/aquasecurity/trivy-db/pkg/types"
	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/aquasecurity/trivy/rpc/detector"
	"github.com/stretchr/testify/mock"
)

type mockDetector struct {
	mock.Mock
}

func (_m *mockDetector) Detect(a context.Context, b *detector.LibDetectRequest) (*detector.DetectResponse, error) {
	ret := _m.Called(a, b)
	ret0 := ret.Get(0)
	if ret0 == nil {
		return nil, ret.Error(1)
	}
	res, ok := ret0.(*detector.DetectResponse)
	if !ok {
		return nil, ret.Error(1)
	}
	return res, ret.Error(1)
}

func TestDetectClient_Detect(t *testing.T) {
	type detectInput struct {
		req *detector.LibDetectRequest
	}
	type detectOutput struct {
		res *detector.DetectResponse
		err error
	}
	type detect struct {
		input  detectInput
		output detectOutput
	}

	type fields struct {
		customHeaders CustomHeaders
	}

	type args struct {
		imageName string
		filePath  string
		created   time.Time
		libs      []deptypes.Library
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		detect  detect
		want    []types.DetectedVulnerability
		wantErr string
	}{
		{
			name: "happy path",
			fields: fields{
				customHeaders: CustomHeaders{
					"Trivy-Token": []string{"token"},
				},
			},
			args: args{
				imageName: "/tmp/alpine.tar",
				filePath:  "app/Pipfile.lock",
				created:   time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
				libs: []deptypes.Library{
					{Name: "django", Version: "3.0.0"},
				},
			},
			detect: detect{
				input: detectInput{req: &detector.LibDetectRequest{
					ImageName: "/tmp/alpine.tar",
					FilePath:  "app/Pipfile.lock",
					Created: func() *timestamp.Timestamp {
						d := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
						t, _ := ptypes.TimestampProto(d)
						return t
					}(),
					Libraries: []*detector.Library{
						{Name: "django", Version: "3.0.0"},
					},
				},
				},
				output: detectOutput{
					res: &detector.DetectResponse{
						Vulnerabilities: []*detector.Vulnerability{
							{
								VulnerabilityId:  "CVE-2019-0001",
								PkgName:          "django",
								InstalledVersion: "3.0.0",
								FixedVersion:     "3.0.1",
								Title:            "RCE",
								Description:      "Remote Code Execution",
								Severity:         detector.Severity_CRITICAL,
							},
						},
					},
				},
			},
			want: []types.DetectedVulnerability{
				{
					VulnerabilityID:  "CVE-2019-0001",
					PkgName:          "django",
					InstalledVersion: "3.0.0",
					FixedVersion:     "3.0.1",
					Vulnerability: dbTypes.Vulnerability{
						Title:       "RCE",
						Description: "Remote Code Execution",
						Severity:    "CRITICAL",
					},
				},
			},
		},
		{
			name:   "Detect returns an error",
			fields: fields{},
			args: args{
				imageName: "/tmp/alpine.tar",
				filePath:  "app/Pipfile.lock",
				created:   time.Date(2019, 2, 1, 0, 0, 0, 0, time.UTC),
				libs: []deptypes.Library{
					{Name: "django", Version: "3.0.0"},
				},
			},
			detect: detect{
				input: detectInput{req: &detector.LibDetectRequest{
					ImageName: "/tmp/alpine.tar",
					FilePath:  "app/Pipfile.lock",
					Libraries: []*detector.Library{
						{Name: "django", Version: "3.0.0"},
					},
					Created: func() *timestamp.Timestamp {
						d := time.Date(2019, 2, 1, 0, 0, 0, 0, time.UTC)
						t, _ := ptypes.TimestampProto(d)
						return t
					}(),
				},
				},
				output: detectOutput{
					err: xerrors.New("error"),
				},
			},
			wantErr: "failed to detect vulnerabilities via RPC",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDetector := new(mockDetector)
			mockDetector.On("Detect", mock.Anything, tt.detect.input.req).Return(
				tt.detect.output.res, tt.detect.output.err)

			d := NewDetector(tt.fields.customHeaders, mockDetector)
			got, err := d.Detect(tt.args.imageName, tt.args.filePath, tt.args.created, tt.args.libs)
			if tt.wantErr != "" {
				require.NotNil(t, err, tt.name)
				assert.Contains(t, err.Error(), tt.wantErr, tt.name)
				return
			} else {
				assert.NoError(t, err, tt.name)
			}
			assert.Equal(t, tt.want, got, tt.name)
			mockDetector.AssertExpectations(t)
		})
	}
}
