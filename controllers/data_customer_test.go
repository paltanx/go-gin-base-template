package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sebasegovia01/base-template-go-gin/models"
	"github.com/sebasegovia01/base-template-go-gin/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks para los servicios
type MockPubSubService struct {
	mock.Mock
}

func (m *MockPubSubService) ExtractStorageEvent(body io.Reader) (*services.StorageEvent, error) {
	args := m.Called(body)
	return args.Get(0).(*services.StorageEvent), args.Error(1)
}

type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) ProcessFile(filename string) ([]*map[string]interface{}, error) {
	args := m.Called(filename)
	return args.Get(0).([]*map[string]interface{}), args.Error(1)
}

func (m *MockStorageService) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockPubSubPublishService struct {
	mock.Mock
}

func (m *MockPubSubPublishService) PublishMessage(message json.RawMessage) error {
	args := m.Called(message)
	return args.Error(0)
}

func TestHandlePushMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name               string
		setupMocks         func(*MockPubSubService, *MockStorageService, *MockPubSubPublishService)
		expectedStatusCode int
		expectedResponse   gin.H
	}{
		{
			name: "Successful processing",
			setupMocks: func(mps *MockPubSubService, mss *MockStorageService, mpps *MockPubSubPublishService) {
				mps.On("ExtractStorageEvent", mock.Anything).Return(&services.StorageEvent{Name: "test.json"}, nil)
				mss.On("ProcessFile", "test.json").Return([]*map[string]interface{}{
					{
						"payload": map[string]interface{}{
							"BOPERS_MAE_NAT_BSC": map[string]interface{}{
								"PEMNB_GLS_NOM_PEL": "John",
								"PEMNB_GLS_APL_PAT": "Doe",
							},
						},
					},
				}, nil)
				mpps.On("PublishMessage", mock.Anything).Return(nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedResponse: gin.H{
				"status":     "Customer data processed and published successfully",
				"data_count": float64(1),
			},
		},
		{
			name: "Error extracting storage event",
			setupMocks: func(mps *MockPubSubService, mss *MockStorageService, mpps *MockPubSubPublishService) {
				mps.On("ExtractStorageEvent", mock.Anything).Return((*services.StorageEvent)(nil), errors.New("extraction error"))
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse: gin.H{
				"error": "extraction error",
			},
		},
		{
			name: "Error processing file",
			setupMocks: func(mps *MockPubSubService, mss *MockStorageService, mpps *MockPubSubPublishService) {
				mps.On("ExtractStorageEvent", mock.Anything).Return(&services.StorageEvent{Name: "test.json"}, nil)
				mss.On("ProcessFile", "test.json").Return([]*map[string]interface{}{}, errors.New("processing error"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedResponse: gin.H{
				"error": "Error processing file: processing error",
			},
		},
		{
			name: "Error publishing message",
			setupMocks: func(mps *MockPubSubService, mss *MockStorageService, mpps *MockPubSubPublishService) {
				mps.On("ExtractStorageEvent", mock.Anything).Return(&services.StorageEvent{Name: "test.json"}, nil)
				mss.On("ProcessFile", "test.json").Return([]*map[string]interface{}{
					{
						"payload": map[string]interface{}{
							"BOPERS_MAE_NAT_BSC": map[string]interface{}{
								"PEMNB_GLS_NOM_PEL": "John",
								"PEMNB_GLS_APL_PAT": "Doe",
							},
						},
					},
				}, nil)
				mpps.On("PublishMessage", mock.Anything).Return(errors.New("publishing error"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedResponse: gin.H{
				"error": "Error publishing message: publishing error",
			},
		},
		{
			name: "Error transforming customer data",
			setupMocks: func(mps *MockPubSubService, mss *MockStorageService, mpps *MockPubSubPublishService) {
				mps.On("ExtractStorageEvent", mock.Anything).Return(&services.StorageEvent{Name: "test.json"}, nil)
				mss.On("ProcessFile", "test.json").Return([]*map[string]interface{}{
					{
						"payload": map[string]interface{}{
							"BOPERS_MAE_NAT_BSC": map[string]interface{}{
								"PEMNB_GLS_NOM_PEL": "John",
								"PEMNB_GLS_APL_PAT": "Doe",
							},
						},
					},
				}, nil)

				// Simulamos un error en la transformación de datos
				originalTransformFunc := transformCustomerDataFunc
				transformCustomerDataFunc = func(data *map[string]interface{}) (*models.Customer, error) {
					return nil, errors.New("transformation error")
				}
				t.Cleanup(func() { transformCustomerDataFunc = originalTransformFunc })
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedResponse: gin.H{
				"error": "Error transforming customer data: transformation error",
			},
		},
		{
			name: "Error marshaling customer data",
			setupMocks: func(mps *MockPubSubService, mss *MockStorageService, mpps *MockPubSubPublishService) {
				mps.On("ExtractStorageEvent", mock.Anything).Return(&services.StorageEvent{Name: "test.json"}, nil)
				mss.On("ProcessFile", "test.json").Return([]*map[string]interface{}{
					{
						"payload": map[string]interface{}{
							"BOPERS_MAE_NAT_BSC": map[string]interface{}{
								"PEMNB_GLS_NOM_PEL": "John",
								"PEMNB_GLS_APL_PAT": "Doe",
							},
						},
					},
				}, nil)

				// Aseguramos que la transformación se haga correctamente
				originalTransformFunc := transformCustomerDataFunc
				transformCustomerDataFunc = func(_ *map[string]interface{}) (*models.Customer, error) {
					return &models.Customer{
						PersonalIdentification: models.PersonalCustomerIdentification{
							CustomerFirstName: "John",
							CustomerLastName:  "Doe",
						},
					}, nil
				}

				// Simular el error en CustomMarshalJSON
				originalMarshalFunc := customMarshalJSONFunc
				customMarshalJSONFunc = func(_ interface{}) ([]byte, error) {
					return nil, errors.New("marshalling error")
				}

				t.Cleanup(func() {
					transformCustomerDataFunc = originalTransformFunc
					customMarshalJSONFunc = originalMarshalFunc
				})
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedResponse: gin.H{
				"error": "Error marshaling customer data: marshalling error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPubSubService := new(MockPubSubService)
			mockStorageService := new(MockStorageService)
			mockPubSubPublishService := new(MockPubSubPublishService)

			tt.setupMocks(mockPubSubService, mockStorageService, mockPubSubPublishService)

			controller := NewDataCustomerController(mockPubSubService, mockStorageService, mockPubSubPublishService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{}"))

			controller.HandlePushMessage(c)

			assert.Equal(t, tt.expectedStatusCode, w.Code)

			var response gin.H
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResponse, response)

			mockPubSubService.AssertExpectations(t)
			mockStorageService.AssertExpectations(t)
			mockPubSubPublishService.AssertExpectations(t)
		})
	}
}
