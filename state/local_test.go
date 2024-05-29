package state

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"testing"

	"github.com/camptocamp/terraboard/config"
)

type LocalStateTest struct {
	path   string
	result error
}

func TestNewLocal(t *testing.T) {
	instance, _ := NewLocal(
		config.LocalConfig{
			StatePath: "/path/to/key",
			StateFile: "terraform.tfstate",
		},
	)

	if instance == nil {
		t.Error("Local instance should not be nil")
	}
}

func TestNewLocalNoStatePath(t *testing.T) {
	instance, _ := NewLocal(
		config.LocalConfig{
			StateFile: "terraform.tfstate",
		},
	)

	if instance != nil {
		t.Error("Local instance should be nil")
	}
}

func TestNewLocalNoStateFile(t *testing.T) {
	instance, _ := NewLocal(
		config.LocalConfig{
			StatePath: "/path/to/key",
		},
	)

	if instance != nil {
		t.Error("Local instance should be nil")
	}
}

func TestNewLocalCollection(t *testing.T) {
	config := config.Config{
		Local: []config.LocalConfig{
			{
				StatePath: "/path/to/key",
				StateFile: "terraform.tfstate",
			},
		},
		Version:        false,
		ConfigFilePath: "",
		Provider:       config.ProviderConfig{},
		DB:             config.DBConfig{},
		TFE:            []config.TFEConfig{},
		GCP:            []config.GCPConfig{},
		Gitlab:         []config.GitlabConfig{},
		Web:            config.WebConfig{},
	}
	instances, _ := NewLocalCollection(&config)

	if instances == nil || len(instances) != 1 {
		t.Errorf("Local instances are nil or not the expected number")
	}
}

// func (o *localMock) GetStates(_ *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
// 	return &s3.ListObjectsV2Output{Contents: []*s3.Object{
// 		{Key: aws.String("test.tfstate")}, {Key: aws.String("test2.tfstate")}, {Key: aws.String("test3.tfstate")}},
// 		IsTruncated: func() *bool { b := false; return &b }(),
// 		KeyCount:    func() *int64 { b := int64(3); return &b }(),
// 		MaxKeys:     func() *int64 { b := int64(1000); return &b }()}, nil
// }
// func (s *localMock) ListObjectVersions(_ *s3.ListObjectVersionsInput) (*s3.ListObjectVersionsOutput, error) {
// 	return &s3.ListObjectVersionsOutput{
// 		Versions: []*s3.ObjectVersion{
// 			{Key: aws.String("testId"), VersionId: aws.String("test"), LastModified: aws.Time(time.Now())},
// 			{Key: aws.String("testId2"), VersionId: aws.String("test2"), LastModified: aws.Time(time.Now())},
// 		},
// 	}, nil
// }
// func (s *localMock) GetObjectWithContext(_ aws.Context, _ *s3.GetObjectInput, _ ...request.Option) (*s3.GetObjectOutput, error) {
// 	return &s3.GetObjectOutput{
// 		Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"Version": 4, "Serial": 3, "TerraformVersion": "0.12.0"}`))),
// 	}, nil
// }

type getStatesTest struct {
	id                   string
	path                 []string
	matched              bool
	expected             int
	expectedWalkerError  error
	expectedMatcherError error
}

type mockedDirEntry struct {
	fs.DirEntry
	value bool
}

func (m mockedDirEntry) IsDir() bool { return m.value }

func TestLocalGetStates(t *testing.T) {
	tmpDir := t.TempDir()
	subtests := []getStatesTest{
		{"match", []string{"a", "terraform.tfstate"}, true, 1, nil, nil},
		{"empty", []string{"b", "terraform.tfstate"}, false, 0, nil, nil},
		{"errorNotExist", []string{"c", "terraform.txt"}, false, 0, fs.ErrNotExist, nil},
		{"errorPermission", []string{"d", "terraform.tfstate"}, true, 0, fs.ErrPermission, nil},
		{"errorBadPattern", []string{"e", "terraform.tfstate"}, true, 0, nil, filepath.ErrBadPattern},
	}

	for _, subtest := range subtests {
		t.Run(fmt.Sprintf("GetStates_%s", subtest.id), func(t *testing.T) {
			instance, _ := NewLocal(
				config.LocalConfig{
					StatePath: tmpDir,
					StateFile: "*",
				},
			)
			instance.walkDirFn = func(root string, fn fs.WalkDirFunc) error {
				filePath := path.Join(tmpDir, path.Join(subtest.path...))
				fileInfo := mockedDirEntry{value: false}
				return fn(filePath, fileInfo, subtest.expectedWalkerError)
			}
			instance.matchFn = func(pattern string, name string) (matched bool, err error) {
				return subtest.matched, subtest.expectedMatcherError
			}

			states, err := instance.GetStates()
			if err != nil && (err != subtest.expectedWalkerError && err != subtest.expectedMatcherError) {
				t.Error(err)
			} else if len(states) != subtest.expected {
				t.Errorf("Expected %d states, got %d", subtest.expected, len(states))
			}
		})
	}
}
