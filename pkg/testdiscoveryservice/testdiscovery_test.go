// Package testdiscoveryservice is used for discover tests
package testdiscoveryservice

import (
	"context"
	"reflect"
	"testing"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/requestutils"
	"github.com/LambdaTest/test-at-scale/testutils"
	"github.com/LambdaTest/test-at-scale/testutils/mocks"
	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/mock"
)

type args struct {
	ctx        context.Context
	tasConfig  *core.TASConfig
	payload    *core.Payload
	secretData map[string]string
	diff       map[string]int
	diffExists bool
}
type test struct {
	name           string
	args           args
	wantErr        bool
	wantEnvMap     map[string]string
	wantSecretData map[string]string
}

func Test_testDiscoveryService_Discover(t *testing.T) {
	logger, err := testutils.GetLogger()
	if err != nil {
		t.Errorf("Couldn't initialize logger, error: %v", err)
	}
	requests := requestutils.New(logger, global.DefaultAPITimeout, &backoff.StopBackOff{})
	tdResChan := make(chan core.DiscoveryResult)
	global.TestEnv = true
	defer func() { global.TestEnv = false }()

	var PassedEnvMap map[string]string        // envMap which should be passed to call execManager.GetEnvVariables
	var PassedSecretDataMap map[string]string // secretData map which should be passed to call execManager.GetEnvVariables

	execManager := new(mocks.ExecutionManager)
	execManager.On("GetEnvVariables", mock.AnythingOfType("map[string]string"), mock.AnythingOfType("map[string]string")).Return(
		func(envMap, secretData map[string]string) []string {
			PassedEnvMap = envMap
			PassedSecretDataMap = secretData
			return []string{"success", "ss"}
		},
		func(envMap, secretData map[string]string) error {
			PassedEnvMap = envMap
			PassedSecretDataMap = secretData
			return nil
		},
	)
	tds := NewTestDiscoveryService(context.TODO(), tdResChan, execManager, requests, logger)
	tests := getTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tds.Discover(tt.args.ctx, tt.args.tasConfig, tt.args.payload, tt.args.secretData, tt.args.diff, tt.args.diffExists)
			if !reflect.DeepEqual(PassedEnvMap, tt.wantEnvMap) || !reflect.DeepEqual(PassedSecretDataMap, tt.wantSecretData) {
				t.Errorf("expected Envmap: %+v, received: %+v\nexpected SecretDataMap: %+v, received: %+v\n",
					tt.wantEnvMap, PassedEnvMap, tt.wantSecretData, PassedSecretDataMap)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func getTestCases() []*test {
	testCases := []*test{
		{"Test Discover with Premerge pattern",
			args{
				ctx: context.TODO(),
				tasConfig: &core.TASConfig{
					Postmerge: &core.Merge{
						Patterns: []string{"./test/**/*.spec.ts"},
						EnvMap:   map[string]string{"env": "RepoName"},
					},
					Premerge: &core.Merge{
						Patterns: []string{"./test/**/*.spec.ts"},
						EnvMap:   map[string]string{"env": "repo"},
					},
				},
				payload: &core.Payload{
					EventType:   "pull-request",
					TasFileName: "../../tesutils/testdata/tas.yaml",
				},
				secretData: map[string]string{"secret": "data"},
				diff:       map[string]int{},
			},
			true,
			map[string]string{"env": "repo"},
			map[string]string{"secret": "data"},
		},
		{"Test Discover with Postmerge pattern",
			args{
				ctx: context.TODO(),
				tasConfig: &core.TASConfig{
					Postmerge: &core.Merge{
						Patterns: []string{"./test/**/*.spec.ts"},
						EnvMap:   map[string]string{"env": "RepoName"},
					},
					Premerge: &core.Merge{
						Patterns: []string{"./test/**/*.spec.ts"},
						EnvMap:   map[string]string{"env": "repo"},
					},
				},
				payload: &core.Payload{
					EventType:   "push",
					TasFileName: "../../tesutils/testdata/tas.yaml",
				},
				secretData: map[string]string{"this is": "a secret"},
				diff:       map[string]int{"../../tesutils/testdata/tas.yaml": 2},
			},
			true,
			map[string]string{"env": "RepoName"},
			map[string]string{"this is": "a secret"},
		},
		{"Test Discover not to execute discoverAll",
			args{
				ctx: context.TODO(),
				tasConfig: &core.TASConfig{
					Postmerge: &core.Merge{
						Patterns: []string{"./test/**/*.spec.ts"},
						EnvMap:   map[string]string{"env": "RepoName"},
					},
					Premerge: &core.Merge{
						Patterns: []string{"./test/**/*.spec.ts"},
						EnvMap:   map[string]string{"env": "repo"},
					},
					SmartRun: true,
				},
				payload: &core.Payload{
					EventType:                  "push",
					TasFileName:                "../../tesutils/testdata/tas.yaml",
					ParentCommitCoverageExists: true,
				},
				secretData: map[string]string{"secret": "data"},
				diff:       map[string]int{"../../tesutils/testdata/dne.yaml": 4},
			},
			true,
			map[string]string{"env": "RepoName"},
			map[string]string{"secret": "data"},
		},
	}
	return testCases
}
