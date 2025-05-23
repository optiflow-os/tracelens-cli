package project_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/optiflow-os/tracelens-cli/pkg/heartbeat"
	"github.com/optiflow-os/tracelens-cli/pkg/project"
	"github.com/optiflow-os/tracelens-cli/pkg/regex"
	"github.com/optiflow-os/tracelens-cli/pkg/windows"

	"github.com/gandarez/go-realpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithDetection_EntityNotFile(t *testing.T) {
	ctx := context.Background()

	tests := map[string]struct {
		Heartbeats  []heartbeat.Heartbeat
		Override    string
		Alternative string
		Config      project.Config
		Expected    heartbeat.Heartbeat
	}{
		"entity not file override takes precedence": {
			Heartbeats: []heartbeat.Heartbeat{
				{
					EntityType:       heartbeat.AppType,
					ProjectAlternate: "pci",
					ProjectOverride:  "billing",
				},
			},
			Expected: heartbeat.Heartbeat{
				Branch:           heartbeat.PointerTo(""),
				EntityType:       heartbeat.AppType,
				Project:          heartbeat.PointerTo("billing"),
				ProjectAlternate: "pci",
				ProjectOverride:  "billing",
			},
		},
		"entity not file alternative takes precedence": {
			Heartbeats: []heartbeat.Heartbeat{
				{
					EntityType:       heartbeat.AppType,
					ProjectAlternate: "pci",
				},
			},
			Expected: heartbeat.Heartbeat{
				Branch:           heartbeat.PointerTo(""),
				EntityType:       heartbeat.AppType,
				Project:          heartbeat.PointerTo("pci"),
				ProjectAlternate: "pci",
			},
		},
		"entity not file empty return": {
			Heartbeats: []heartbeat.Heartbeat{
				{
					EntityType: heartbeat.AppType,
				},
			},
			Expected: heartbeat.Heartbeat{
				Branch:     heartbeat.PointerTo(""),
				EntityType: heartbeat.AppType,
				Project:    heartbeat.PointerTo(""),
			},
		},
		"entity not file with project obfuscation": {
			Heartbeats: []heartbeat.Heartbeat{
				{
					Entity:          "github.com",
					EntityType:      heartbeat.AppType,
					ProjectOverride: "billing",
				},
			},
			Config: project.Config{HideProjectNames: []regex.Regex{regex.NewRegexpWrap(regexp.MustCompile(".*"))}},
			Expected: heartbeat.Heartbeat{
				Branch:          heartbeat.PointerTo(""),
				Entity:          "github.com",
				EntityType:      heartbeat.AppType,
				Project:         heartbeat.PointerTo(""),
				ProjectOverride: "billing",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			opt := project.WithDetection(test.Config)

			handle := opt(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
				assert.Equal(t, []heartbeat.Heartbeat{
					test.Expected,
				}, hh)

				return nil, nil
			})

			_, err := handle(ctx, test.Heartbeats)
			require.NoError(t, err)
		})
	}
}

func TestWithDetection_WakatimeProjectTakesPrecedence(t *testing.T) {
	fp := setupTestGitBasic(t)

	ctx := context.Background()

	entity := filepath.Join(fp, "wakatime-cli/src/pkg/file.go")
	projectPath := filepath.Join(fp, "wakatime-cli")
	projectPath = project.FormatProjectFolder(ctx, projectPath)

	if runtime.GOOS == "windows" {
		entity = windows.FormatFilePath(entity)
	}

	copyFile(
		t,
		"testdata/wakatime-project-other",
		filepath.Join(fp, "wakatime-cli", ".wakatime-project"),
	)

	opts := []heartbeat.HandleOption{
		heartbeat.WithFormatting(),
		project.WithDetection(project.Config{
			HideProjectNames: []regex.Regex{regex.MustCompile(".*")},
		}),
	}

	sender := mockSender{
		SendHeartbeatsFn: func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
			assert.NotEmpty(t, hh[0].Project)
			assert.Equal(t, []heartbeat.Heartbeat{
				{
					Branch:           heartbeat.PointerTo("master"),
					Entity:           entity,
					EntityType:       heartbeat.FileType,
					Project:          heartbeat.PointerTo("Rough Surf 20"),
					ProjectAlternate: "alternate",
					ProjectPath:      projectPath,
					ProjectRootCount: heartbeat.PointerTo(project.CountSlashesInProjectFolder(projectPath)),
				},
			}, hh)

			return nil, nil
		},
	}

	handle := heartbeat.NewHandle(&sender, opts...)

	_, err := handle(ctx, []heartbeat.Heartbeat{
		{
			EntityType:       heartbeat.FileType,
			Entity:           entity,
			ProjectAlternate: "alternate",
		},
	})
	require.NoError(t, err)
}

func TestWithDetection_OverrideTakesPrecedence(t *testing.T) {
	fp := setupTestGitBasic(t)

	entity := filepath.Join(fp, "wakatime-cli/src/pkg/file.go")
	projectPath := filepath.Join(fp, "wakatime-cli")
	projectPath = project.FormatProjectFolder(context.Background(), projectPath)

	if runtime.GOOS == "windows" {
		entity = windows.FormatFilePath(entity)
	}

	opt := project.WithDetection(project.Config{})

	handle := opt(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		assert.Equal(t, []heartbeat.Heartbeat{
			{
				Branch:           heartbeat.PointerTo("master"),
				Entity:           entity,
				EntityType:       heartbeat.FileType,
				Project:          heartbeat.PointerTo("override"),
				ProjectOverride:  "override",
				ProjectPath:      projectPath,
				ProjectRootCount: heartbeat.PointerTo(project.CountSlashesInProjectFolder(projectPath)),
			},
		}, hh)

		return nil, nil
	})

	_, err := handle(context.Background(), []heartbeat.Heartbeat{
		{
			EntityType:      heartbeat.FileType,
			Entity:          entity,
			ProjectOverride: "override",
		},
	})
	require.NoError(t, err)
}

func TestWithDetection_OverrideTakesPrecedence_WithProjectPathOverride(t *testing.T) {
	fp := setupTestGitBasic(t)

	entity := filepath.Join(fp, "wakatime-cli/src/pkg/file.go")

	if runtime.GOOS == "windows" {
		entity = windows.FormatFilePath(entity)
	}

	opt := project.WithDetection(project.Config{})

	handle := opt(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		assert.Equal(t, []heartbeat.Heartbeat{
			{
				Branch:              heartbeat.PointerTo("master"),
				Entity:              entity,
				EntityType:          heartbeat.FileType,
				Project:             heartbeat.PointerTo("override"),
				ProjectPath:         fp,
				ProjectOverride:     "override",
				ProjectPathOverride: fp,
				ProjectRootCount:    heartbeat.PointerTo(project.CountSlashesInProjectFolder(fp)),
			},
		}, hh)

		return nil, nil
	})

	_, err := handle(context.Background(), []heartbeat.Heartbeat{
		{
			EntityType:          heartbeat.FileType,
			Entity:              entity,
			ProjectOverride:     "override",
			ProjectPathOverride: fp,
		},
	})
	require.NoError(t, err)
}

func TestWithDetection_NoneDetected(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer tmpFile.Close()

	ctx := context.Background()

	entity := tmpFile.Name()

	projectPath := filepath.Dir(tmpFile.Name())
	projectPath = project.FormatProjectFolder(ctx, projectPath)

	if runtime.GOOS == "windows" {
		entity = windows.FormatFilePath(entity)
	} else {
		entity, err = realpath.Realpath(entity)
		require.NoError(t, err)
	}

	opts := []heartbeat.HandleOption{
		heartbeat.WithFormatting(),
		project.WithDetection(project.Config{}),
	}

	sender := mockSender{
		SendHeartbeatsFn: func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
			assert.Equal(t, []heartbeat.Heartbeat{
				{
					Branch:           heartbeat.PointerTo(""),
					Entity:           entity,
					EntityType:       heartbeat.FileType,
					Project:          heartbeat.PointerTo(""),
					ProjectPath:      projectPath,
					ProjectRootCount: heartbeat.PointerTo(project.CountSlashesInProjectFolder(projectPath)),
				},
			}, hh)

			return nil, nil
		},
	}

	handle := heartbeat.NewHandle(&sender, opts...)

	_, err = handle(ctx, []heartbeat.Heartbeat{
		{
			EntityType: heartbeat.FileType,
			Entity:     tmpFile.Name(),
		},
	})
	require.NoError(t, err)
}

func TestWithDetection_NoneDetected_AlternateTakesPrecedence(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer tmpFile.Close()

	ctx := context.Background()

	entity := tmpFile.Name()

	projectPath := filepath.Dir(tmpFile.Name())
	projectPath = project.FormatProjectFolder(ctx, projectPath)

	if runtime.GOOS == "windows" {
		entity = windows.FormatFilePath(entity)
	} else {
		entity, err = realpath.Realpath(entity)
		require.NoError(t, err)
	}

	opts := []heartbeat.HandleOption{
		heartbeat.WithFormatting(),
		project.WithDetection(project.Config{}),
	}

	sender := mockSender{
		SendHeartbeatsFn: func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
			assert.Equal(t, []heartbeat.Heartbeat{
				{
					Branch:           heartbeat.PointerTo("alternate-branch"),
					BranchAlternate:  "alternate-branch",
					Entity:           entity,
					EntityType:       heartbeat.FileType,
					Project:          heartbeat.PointerTo("alternate-project"),
					ProjectAlternate: "alternate-project",
					ProjectPath:      projectPath,
					ProjectRootCount: heartbeat.PointerTo(project.CountSlashesInProjectFolder(projectPath)),
				},
			}, hh)

			return nil, nil
		},
	}

	handle := heartbeat.NewHandle(&sender, opts...)

	_, err = handle(ctx, []heartbeat.Heartbeat{
		{
			BranchAlternate:  "alternate-branch",
			EntityType:       heartbeat.FileType,
			Entity:           tmpFile.Name(),
			ProjectAlternate: "alternate-project",
		},
	})
	require.NoError(t, err)
}

func TestWithDetection_NoneDetected_OverrideTakesPrecedence(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "")
	require.NoError(t, err)

	defer tmpFile.Close()

	ctx := context.Background()

	entity := tmpFile.Name()

	if runtime.GOOS == "windows" {
		entity = windows.FormatFilePath(entity)
	} else {
		entity, err = realpath.Realpath(entity)
		require.NoError(t, err)
	}

	projectPath := filepath.Dir(tmpFile.Name())
	projectPath = project.FormatProjectFolder(ctx, projectPath)

	opts := []heartbeat.HandleOption{
		heartbeat.WithFormatting(),
		project.WithDetection(project.Config{}),
	}

	sender := mockSender{
		SendHeartbeatsFn: func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
			assert.Equal(t, []heartbeat.Heartbeat{
				{
					Branch:           heartbeat.PointerTo(""),
					Entity:           entity,
					EntityType:       heartbeat.FileType,
					Project:          heartbeat.PointerTo("override"),
					ProjectOverride:  "override",
					ProjectPath:      projectPath,
					ProjectRootCount: heartbeat.PointerTo(project.CountSlashesInProjectFolder(projectPath)),
				},
			}, hh)

			return nil, nil
		},
	}

	handle := heartbeat.NewHandle(&sender, opts...)

	_, err = handle(ctx, []heartbeat.Heartbeat{
		{
			EntityType:      heartbeat.FileType,
			Entity:          tmpFile.Name(),
			ProjectOverride: "override",
		},
	})
	require.NoError(t, err)
}

func TestWithDetection_NoneDetected_WithProjectPathOverride(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "")
	require.NoError(t, err)

	defer tmpFile.Close()

	ctx := context.Background()

	opts := []heartbeat.HandleOption{
		heartbeat.WithFormatting(),
		project.WithDetection(project.Config{}),
	}

	entity := tmpFile.Name()

	if runtime.GOOS == "windows" {
		entity = windows.FormatFilePath(entity)
	} else {
		entity, err = realpath.Realpath(entity)
		require.NoError(t, err)
	}

	projectFolder := project.FormatProjectFolder(ctx, tmpDir)

	sender := mockSender{
		SendHeartbeatsFn: func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
			assert.Equal(t, []heartbeat.Heartbeat{
				{
					Branch:              heartbeat.PointerTo(""),
					Entity:              entity,
					EntityType:          heartbeat.FileType,
					Project:             heartbeat.PointerTo("overridden-project"),
					ProjectOverride:     "overridden-project",
					ProjectPath:         projectFolder,
					ProjectPathOverride: projectFolder,
					ProjectRootCount:    heartbeat.PointerTo(project.CountSlashesInProjectFolder(projectFolder)),
				},
			}, hh)

			return nil, nil
		},
	}

	handle := heartbeat.NewHandle(&sender, opts...)

	_, err = handle(ctx, []heartbeat.Heartbeat{
		{
			EntityType:          heartbeat.FileType,
			Entity:              tmpFile.Name(),
			ProjectOverride:     "overridden-project",
			ProjectPathOverride: tmpDir,
		},
	})
	require.NoError(t, err)
}

func TestWithDetection_ObfuscateProject(t *testing.T) {
	fp := setupTestGitBasic(t)

	ctx := context.Background()

	entity := filepath.Join(fp, "wakatime-cli/src/pkg/file.go")
	projectPath := filepath.Join(fp, "wakatime-cli")
	projectPath = project.FormatProjectFolder(ctx, projectPath)

	if runtime.GOOS == "windows" {
		entity = windows.FormatFilePath(entity)
	}

	opt := project.WithDetection(project.Config{
		HideProjectNames: []regex.Regex{regex.MustCompile(".*")},
	})

	handle := opt(func(_ context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
		assert.NotEmpty(t, hh[0].Project)
		assert.Equal(t, []heartbeat.Heartbeat{
			{
				Branch:           heartbeat.PointerTo("master"),
				Entity:           entity,
				EntityType:       heartbeat.FileType,
				Project:          hh[0].Project,
				ProjectPath:      projectPath,
				ProjectRootCount: heartbeat.PointerTo(project.CountSlashesInProjectFolder(projectPath)),
			},
		}, hh)

		return nil, nil
	})

	_, err := handle(ctx, []heartbeat.Heartbeat{
		{
			EntityType: heartbeat.FileType,
			Entity:     entity,
		},
	})
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(fp, "wakatime-cli/.wakatime-project"))
}

func TestDetect_FileDetected(t *testing.T) {
	tmpDir, err := realpath.Realpath(t.TempDir())
	require.NoError(t, err)

	copyFile(
		t,
		"testdata/wakatime-project",
		filepath.Join(tmpDir, ".wakatime-project"),
	)

	copyFile(
		t,
		"testdata/entity.any",
		filepath.Join(tmpDir, "entity.any"),
	)

	result, detector := project.Detect(context.Background(), []project.MapPattern{}, project.DetecterArg{
		Filepath:  filepath.Join(tmpDir, "entity.any"),
		ShouldRun: true,
	})

	assert.Equal(t, "wakatime-cli", result.Project)
	assert.Equal(t, "master", result.Branch)
	assert.Contains(t, result.Folder, tmpDir)
	assert.Equal(t, detector, project.FileDetector)
}

func TestDetect_EmptyFileDetected(t *testing.T) {
	tmpDir, err := realpath.Realpath(t.TempDir())
	require.NoError(t, err)

	err = os.Mkdir(filepath.Join(tmpDir, "wakatime-cli"), os.FileMode(int(0700)))
	require.NoError(t, err)

	tmpWakatimeProjectEmpty, err := os.Create(filepath.Join(tmpDir, "wakatime-cli", ".wakatime-project"))
	require.NoError(t, err)

	defer tmpWakatimeProjectEmpty.Close()

	copyFile(
		t,
		"testdata/entity.any",
		filepath.Join(tmpDir, "wakatime-cli", "entity.any"),
	)

	result, detector := project.Detect(context.Background(), []project.MapPattern{}, project.DetecterArg{
		Filepath:  filepath.Join(tmpDir, "wakatime-cli", "entity.any"),
		ShouldRun: true,
	})

	assert.Equal(t, "wakatime-cli", result.Project)
	assert.Equal(t, "", result.Branch)
	assert.Contains(t, result.Folder, tmpDir)
	assert.Equal(t, detector, project.FileDetector)
}

func TestDetect_MapDetected(t *testing.T) {
	tmpDir := t.TempDir()

	tmpFile, err := os.CreateTemp(tmpDir, "waka-billing")
	require.NoError(t, err)

	defer tmpFile.Close()

	patterns := []project.MapPattern{
		{
			Name:  "my-project-1",
			Regex: regex.MustCompile(formatRegex(filepath.Join(tmpDir, "path", "to", "otherfolder"))),
		},
		{
			Name:  "my-{0}-project",
			Regex: regex.MustCompile(formatRegex(filepath.Join(tmpDir, "waka-([a-z]+)"))),
		},
	}

	result, detector := project.Detect(context.Background(), patterns, project.DetecterArg{
		Filepath:  tmpFile.Name(),
		ShouldRun: true,
	})

	assert.Equal(t, "my-billing-project", result.Project)
	assert.Empty(t, result.Branch)
	assert.Contains(t, result.Folder, filepath.Dir(tmpFile.Name()))
	assert.Equal(t, detector, project.MapDetector)
}

func TestDetectWithRevControl_GitDetected(t *testing.T) {
	fp := setupTestGitBasic(t)

	result := project.DetectWithRevControl(
		context.Background(),
		[]regex.Regex{},
		[]project.MapPattern{},
		false,
		project.DetecterArg{
			Filepath:  filepath.Join(fp, "wakatime-cli/src/pkg/file.go"),
			ShouldRun: true,
		},
	)

	assert.Contains(t, result.Folder, filepath.Join(fp, "wakatime-cli"))
	assert.Equal(t, project.Result{
		Project: "wakatime-cli",
		Folder:  result.Folder,
		Branch:  "master",
	}, result)
}

func TestDetectWithRevControl_GitRemoteDetected(t *testing.T) {
	fp := setupTestGitBasic(t)

	result := project.DetectWithRevControl(
		context.Background(),
		[]regex.Regex{},
		[]project.MapPattern{},
		true,
		project.DetecterArg{
			Filepath:  filepath.Join(fp, "wakatime-cli/src/pkg/file.go"),
			ShouldRun: true,
		},
	)

	assert.Contains(t, result.Folder, filepath.Join(fp, "wakatime-cli"))
	assert.Equal(t, project.Result{
		Project: "wakatime/wakatime-cli",
		Folder:  result.Folder,
		Branch:  "master",
	}, result)
}

func TestDetect_NoProjectDetected(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "wakatime")
	require.NoError(t, err)

	defer tmpFile.Close()

	result, detector := project.Detect(context.Background(), []project.MapPattern{}, project.DetecterArg{
		Filepath:  tmpFile.Name(),
		ShouldRun: true,
	})

	assert.Empty(t, result.Branch)
	assert.Empty(t, result.Folder)
	assert.Empty(t, result.Project)
	assert.Empty(t, detector)
}

func TestWrite(t *testing.T) {
	tmpDir := t.TempDir()

	err := project.Write(tmpDir, "billing")
	require.NoError(t, err)

	actual, err := os.ReadFile(filepath.Join(tmpDir, ".wakatime-project"))
	require.NoError(t, err)

	assert.Equal(t, string([]byte("billing\n")), string(actual))
}

func TestCountSlashesInProjectFolder(t *testing.T) {
	tests := map[string]struct {
		path     string
		expected int
	}{
		"empty path": {
			path:     "",
			expected: 0,
		},
		"root path": {
			path:     "/",
			expected: 1,
		},
		"home path": {
			path:     "/home",
			expected: 2,
		},
		"home user path": {
			path:     "/home/user",
			expected: 3,
		},
		"home user project path": {
			path:     "/home/user/project",
			expected: 4,
		},
		"windows path": {
			path:     `C:\folder\project`,
			expected: 3,
		},
		"wsl path": {
			path:     `\\wsl$/Ubuntu-22.04/home/folder/project`,
			expected: 5,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			count := project.CountSlashesInProjectFolder(test.path)

			assert.Equal(t, test.expected, count)
		})
	}
}

func detectorIDTests() map[string]project.DetectorID {
	return map[string]project.DetectorID{
		"project-file-detector": project.FileDetector,
		"project-map-detector":  project.MapDetector,
		"git-detector":          project.GitDetector,
		"mercurial-detector":    project.MercurialDetector,
		"svn-detector":          project.SubversionDetector,
		"tfvc-detector":         project.TfvcDetector,
	}
}

func TestDetectorID_String(t *testing.T) {
	for value, category := range detectorIDTests() {
		t.Run(value, func(t *testing.T) {
			s := category.String()
			assert.Equal(t, value, s)
		})
	}
}

func formatRegex(fp string) string {
	if runtime.GOOS != "windows" {
		return fp
	}

	return strings.ReplaceAll(fp, `\`, `\\`)
}

type mockSender struct {
	SendHeartbeatsFn        func(context.Context, []heartbeat.Heartbeat) ([]heartbeat.Result, error)
	SendHeartbeatsFnInvoked bool
}

func (m *mockSender) SendHeartbeats(ctx context.Context, hh []heartbeat.Heartbeat) ([]heartbeat.Result, error) {
	m.SendHeartbeatsFnInvoked = true
	return m.SendHeartbeatsFn(ctx, hh)
}
