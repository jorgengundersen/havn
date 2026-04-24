package dolt_test

import (
	"context"

	"github.com/jorgengundersen/havn/internal/dolt"
)

// fakeBackend implements dolt.Backend for testing.
type fakeBackend struct {
	inspectInfo  dolt.ContainerInfo
	inspectFound bool
	inspectErr   error

	startErr error
	stopErr  error

	createID     string
	createErr    error
	createFunc   func(opts dolt.ContainerCreateOpts) (string, error)
	createOpts   dolt.ContainerCreateOpts
	createCalled bool

	pullErr   error
	pullCalls []string
	pullFunc  func(image string) error

	execOutput string
	execErr    error
	execCalls  []execCall
	execFunc   func(cmd []string) (string, error) // overrides execOutput/execErr when set

	copyErr    error
	copiedData []byte
	copiedPath string
	copyFunc   func(destPath string, content []byte) error

	copyFromData []byte
	copyFromErr  error

	interactiveCalls []execCall
	interactiveErr   error
}

// execCall records a ContainerExec invocation.
type execCall struct {
	container string
	cmd       []string
}

func (f *fakeBackend) ContainerCreate(_ context.Context, opts dolt.ContainerCreateOpts) (string, error) {
	f.createOpts = opts
	f.createCalled = true
	if f.createFunc != nil {
		return f.createFunc(opts)
	}
	return f.createID, f.createErr
}

func (f *fakeBackend) ImagePull(_ context.Context, image string) error {
	f.pullCalls = append(f.pullCalls, image)
	if f.pullFunc != nil {
		return f.pullFunc(image)
	}
	return f.pullErr
}

func (f *fakeBackend) ContainerStart(_ context.Context, _ string) error {
	return f.startErr
}

func (f *fakeBackend) ContainerStop(_ context.Context, _ string) error {
	return f.stopErr
}

func (f *fakeBackend) ContainerInspect(_ context.Context, _ string) (dolt.ContainerInfo, bool, error) {
	return f.inspectInfo, f.inspectFound, f.inspectErr
}

func (f *fakeBackend) ContainerExec(_ context.Context, container string, cmd []string) (string, error) {
	f.execCalls = append(f.execCalls, execCall{container: container, cmd: cmd})
	if f.execFunc != nil {
		return f.execFunc(cmd)
	}
	return f.execOutput, f.execErr
}

func (f *fakeBackend) CopyToContainer(_ context.Context, _ string, destPath string, content []byte) error {
	f.copiedData = content
	f.copiedPath = destPath
	if f.copyFunc != nil {
		return f.copyFunc(destPath, content)
	}
	return f.copyErr
}

func (f *fakeBackend) CopyFromContainer(_ context.Context, _ string, _ string) ([]byte, error) {
	return f.copyFromData, f.copyFromErr
}

func (f *fakeBackend) ContainerExecInteractive(_ context.Context, container string, cmd []string) error {
	f.interactiveCalls = append(f.interactiveCalls, execCall{container: container, cmd: cmd})
	return f.interactiveErr
}
