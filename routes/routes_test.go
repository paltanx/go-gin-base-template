package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sebasegovia01/base-template-go-gin/controllers"
	"github.com/sebasegovia01/base-template-go-gin/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock para PubSubService
type MockPubSubService struct {
	mock.Mock
}

func (m *MockPubSubService) ExtractStorageEvent(body io.Reader) (*services.StorageEvent, error) {
	args := m.Called(body)
	return args.Get(0).(*services.StorageEvent), args.Error(1)
}

// Mock para StorageService
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

// Mock para PubSubPublishService
type MockPubSubPublishService struct {
	mock.Mock
}

func (m *MockPubSubPublishService) PublishMessage(message json.RawMessage) error {
	args := m.Called(message)
	return args.Error(0)
}

func TestSetupRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Crear un router de prueba
	r := gin.New()

	// Crear mocks para los servicios
	mockPubSubService := new(MockPubSubService)
	mockStorageService := new(MockStorageService)
	mockPubSubPublishService := new(MockPubSubPublishService)

	// Crear una instancia real de DataCustomerController con los mocks
	dataCustomerController := controllers.NewDataCustomerController(mockPubSubService, mockStorageService, mockPubSubPublishService)

	// Configurar el comportamiento esperado del mock PubSubService
	mockPubSubService.On("ExtractStorageEvent", mock.Anything).Return(&services.StorageEvent{Name: "test.json"}, nil)

	// Configurar el comportamiento esperado del mock StorageService
	mockStorageService.On("ProcessFile", "test.json").Return([]*map[string]interface{}{
		{
			"payload": map[string]interface{}{
				"BOPERS_MAE_NAT_BSC": map[string]interface{}{
					"PEMNB_GLS_NOM_PEL": "John",
					"PEMNB_GLS_APL_PAT": "Doe",
				},
			},
		},
	}, nil)

	// Configurar el comportamiento esperado del mock PubSubPublishService
	mockPubSubPublishService.On("PublishMessage", mock.Anything).Return(nil)

	// Llamar a `SetupRoutes` para registrar las rutas con el controlador real
	SetupRoutes(r, dataCustomerController)

	// Prueba de la ruta de HealthCheck
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/customer-data-retrieval/v1/api/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"UP","message":"API is healthy"}`, w.Body.String())

	// Prueba de la ruta de /customers/retrieve
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/customer-data-retrieval/v1/api/customers/retrieve", bytes.NewBuffer([]byte(`{}`)))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWithTraceability(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Crear un router de prueba
	r := gin.New()

	// Definir un handler de prueba
	handlerCalled := false
	testHandler := func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"message": "Handler executed"})
	}

	// Registrar la ruta con el middleware `WithTraceability`
	r.GET("/test", WithTraceability(testHandler))

	// Crear una solicitud de prueba
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)

	// Añadir los headers requeridos por el middleware
	req.Header.Set("Consumer-Sys-Code", "CHL-SIT-SEG")
	req.Header.Set("Consumer-Enterprise-Code", "BANCORIPLEY-CHL")
	req.Header.Set("Consumer-Country-Code", "CHL")
	req.Header.Set("Trace-Client-Req-Timestamp", "2024-09-05 12:00:00.123456+0000")
	req.Header.Set("Trace-Source-Id", "123e4567-e89b-12d3-a456-426614174000")
	req.Header.Set("Channel-Name", "SEGUROS")
	req.Header.Set("Channel-Mode", "PRESENCIAL")

	// Ejecutar la solicitud
	r.ServeHTTP(w, req)

	// Verificar que el middleware no haya abortado y que el handler fue ejecutado
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled, "El handler debería haber sido llamado")

	// Verificar que la respuesta sea JSON válida
	assert.JSONEq(t, `{"message": "Handler executed"}`, w.Body.String())
}
