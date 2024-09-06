package services

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/sebasegovia01/base-template-go-gin/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// MockStorageClient is a mock for the storage.Client
type MockStorageClient struct {
	mock.Mock
}

func (m *MockStorageClient) Bucket(name string) BucketHandleInterface {
	args := m.Called(name)
	return args.Get(0).(BucketHandleInterface)
}

func (m *MockStorageClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockBucketHandle struct {
	mock.Mock
}

func (m *MockBucketHandle) Object(name string) ObjectHandleInterface {
	args := m.Called(name)
	return args.Get(0).(ObjectHandleInterface)
}

type MockObjectHandle struct {
	mock.Mock
}

func (m *MockObjectHandle) Attrs(ctx context.Context) (*storage.ObjectAttrs, error) {
	args := m.Called(ctx)
	return args.Get(0).(*storage.ObjectAttrs), args.Error(1)
}

func (m *MockObjectHandle) NewReader(ctx context.Context) (io.ReadCloser, error) {
	args := m.Called(ctx)
	rc := args.Get(0)
	if rc == nil {
		return nil, args.Error(1)
	}
	return rc.(io.ReadCloser), args.Error(1)
}

// MockReader is a mock for storage.Reader
type MockReader struct {
	mock.Mock
	io.Reader
}

func (m *MockReader) Read(p []byte) (n int, err error) {
	return m.Reader.Read(p)
}

func (m *MockReader) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewStorageService(t *testing.T) {
	cfg := &config.Config{
		BucketName: "test-bucket",
		GCPCredentials: &google.Credentials{
			ProjectID: "test-project",
			JSON:      []byte(`{"type": "service_account"}`),
		},
	}

	// Create a mock function for storage.NewClient
	originalNewClient := storageNewClient
	storageNewClient = func(ctx context.Context, opts ...option.ClientOption) (*storage.Client, error) {
		return &storage.Client{}, nil
	}
	t.Cleanup(func() { storageNewClient = originalNewClient })

	service, err := NewStorageService(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, "test-bucket", service.bucketName)
}

func TestProcessFile(t *testing.T) {
	mockClient := new(MockStorageClient)
	mockBucket := new(MockBucketHandle)
	mockObject := new(MockObjectHandle)

	service := &StorageService{
		client:     mockClient,
		bucketName: "test-bucket",
	}

	mockClient.On("Bucket", "test-bucket").Return(mockBucket)
	mockBucket.On("Object", "test-file.json").Return(mockObject)
	mockObject.On("Attrs", mock.Anything).Return(&storage.ObjectAttrs{}, nil)

	jsonData := `{"key1": "value1"}
{"key2": "value2"}`
	mockReader := &MockReader{Reader: strings.NewReader(jsonData)}
	mockObject.On("NewReader", mock.Anything).Return(mockReader, nil)
	mockReader.On("Close").Return(nil)

	result, err := service.ProcessFile("test-file.json")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "value1", (*result[0])["key1"])
	assert.Equal(t, "value2", (*result[1])["key2"])

	mockClient.AssertExpectations(t)
	mockBucket.AssertExpectations(t)
	mockObject.AssertExpectations(t)
	mockReader.AssertExpectations(t)
}

func TestProcessFileNotFound(t *testing.T) {
	mockClient := new(MockStorageClient)
	mockBucket := new(MockBucketHandle)
	mockObject := new(MockObjectHandle)

	service := &StorageService{
		client:     mockClient,
		bucketName: "test-bucket",
	}

	mockClient.On("Bucket", "test-bucket").Return(mockBucket)
	mockBucket.On("Object", "non-existent-file.json").Return(mockObject)
	mockObject.On("Attrs", mock.Anything).Return((*storage.ObjectAttrs)(nil), storage.ErrObjectNotExist)

	_, err := service.ProcessFile("non-existent-file.json")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "object does not exist")

	mockClient.AssertExpectations(t)
	mockBucket.AssertExpectations(t)
	mockObject.AssertExpectations(t)
}

func TestProcessFileInvalidJSON(t *testing.T) {
	mockClient := new(MockStorageClient)
	mockBucket := new(MockBucketHandle)
	mockObject := new(MockObjectHandle)

	service := &StorageService{
		client:     mockClient,
		bucketName: "test-bucket",
	}

	mockClient.On("Bucket", "test-bucket").Return(mockBucket)
	mockBucket.On("Object", "invalid-json.json").Return(mockObject)
	mockObject.On("Attrs", mock.Anything).Return(&storage.ObjectAttrs{}, nil)

	invalidJSON := `{"key": "value"
{"invalid": json}`
	mockReader := &MockReader{Reader: strings.NewReader(invalidJSON)}
	mockObject.On("NewReader", mock.Anything).Return(mockReader, nil)
	mockReader.On("Close").Return(nil)

	_, err := service.ProcessFile("invalid-json.json")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal JSON line")

	mockClient.AssertExpectations(t)
	mockBucket.AssertExpectations(t)
	mockObject.AssertExpectations(t)
	mockReader.AssertExpectations(t)
}

func TestClose(t *testing.T) {
	mockClient := new(MockStorageClient)
	service := &StorageService{
		client: mockClient,
	}

	mockClient.On("Close").Return(nil)

	err := service.Close()

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestStorageClientWrapper(t *testing.T) {
	mockClient := &storage.Client{}
	wrapper := &storageClientWrapper{mockClient}

	result := wrapper.Bucket("test-bucket")

	assert.NotNil(t, result)
	assert.IsType(t, &bucketHandleWrapper{}, result)
}

func TestBucketHandleWrapper(t *testing.T) {
	mockBucketHandle := &storage.BucketHandle{}
	wrapper := &bucketHandleWrapper{mockBucketHandle}

	result := wrapper.Object("test-object")

	assert.NotNil(t, result)
	assert.IsType(t, &objectHandleWrapper{}, result)
}

func TestObjectHandleWrapper(t *testing.T) {
	mockObjectHandle := &storage.ObjectHandle{}
	wrapper := &objectHandleWrapper{mockObjectHandle}

	ctx := context.Background()
	_, err := wrapper.NewReader(ctx)

	// This will fail because we can't create a real reader, but it exercises the method
	assert.Error(t, err)
}

func TestNewStorageServiceError(t *testing.T) {
	cfg := &config.Config{
		BucketName: "test-bucket",
		GCPCredentials: &google.Credentials{
			ProjectID: "test-project",
			JSON:      []byte(`{"type": "service_account"}`),
		},
	}

	// Mock an error when creating the client
	originalNewClient := storageNewClient
	storageNewClient = func(ctx context.Context, opts ...option.ClientOption) (*storage.Client, error) {
		return nil, errors.New("mock error")
	}
	t.Cleanup(func() { storageNewClient = originalNewClient })

	_, err := NewStorageService(cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create storage client")
}

func TestProcessFileErrors(t *testing.T) {
	mockClient := new(MockStorageClient)
	mockBucket := new(MockBucketHandle)
	mockObject := new(MockObjectHandle)

	service := &StorageService{
		client:     mockClient,
		bucketName: "test-bucket",
	}

	mockClient.On("Bucket", "test-bucket").Return(mockBucket)
	mockBucket.On("Object", "error-file.json").Return(mockObject)

	// Test error checking object attributes
	t.Run("Error checking object attributes", func(t *testing.T) {
		mockObject.On("Attrs", mock.Anything).Return((*storage.ObjectAttrs)(nil), errors.New("mock error")).Once()
		_, err := service.ProcessFile("error-file.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error checking object attributes")
	})

	// Test error creating reader
	t.Run("Error creating reader", func(t *testing.T) {
		mockObject.On("Attrs", mock.Anything).Return(&storage.ObjectAttrs{}, nil).Once()
		mockObject.On("NewReader", mock.Anything).Return(nil, errors.New("mock error")).Once()
		_, err := service.ProcessFile("error-file.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create reader")
	})

	// Test nil reader
	t.Run("Nil reader", func(t *testing.T) {
		mockObject.On("Attrs", mock.Anything).Return(&storage.ObjectAttrs{}, nil).Once()
		mockObject.On("NewReader", mock.Anything).Return(nil, nil).Once()
		_, err := service.ProcessFile("error-file.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reader is nil")
	})

	// Test error reading file
	t.Run("Error reading file", func(t *testing.T) {
		errorReader := &ErrorReader{err: errors.New("read error")}
		mockReader := &MockReader{Reader: errorReader}
		mockObject.On("Attrs", mock.Anything).Return(&storage.ObjectAttrs{}, nil).Once()
		mockObject.On("NewReader", mock.Anything).Return(mockReader, nil).Once()
		mockReader.On("Close").Return(nil).Once()

		_, err := service.ProcessFile("error-file.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading file")
	})

	mockClient.AssertExpectations(t)
	mockBucket.AssertExpectations(t)
	mockObject.AssertExpectations(t)
}

type ErrorReader struct {
	err error
}

func (e *ErrorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}
