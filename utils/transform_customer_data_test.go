package utils_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/sebasegovia01/base-template-go-gin/utils"
	"github.com/stretchr/testify/assert"
)

func TestTransformCustomerData(t *testing.T) {
	// Caso 1: Estructura válida
	data := map[string]interface{}{
		"payload": map[string]interface{}{
			"BOPERS_MAE_NAT_BSC": map[string]interface{}{
				"PEMNB_GLS_NOM_PEL": "John Doe",
				"PEMNB_GLS_APL_PAT": "Doe",
				"PEMNB_GLS_APL_MAT": "Smith",
			},
			"BOPERS_MAE_PRD_CLI": map[string]interface{}{
				"PEMPC_FCH_ALS_CTT": time.Now().Format(time.RFC3339),
			},
			"BOPERS_MAE_IDE": map[string]interface{}{
				"PEMID_GLS_NRO_DCT_IDE_K": "12345678",
				"PEMID_DVR_NRO_DCT_IDE":   "K",
			},
			"BOPERS_MAE_DML": map[string]interface{}{
				"PEMDM_GLS_CAL_DML": "123 Main St",
				"PEMDM_NRO_DML":     "101",
				"PEMDM_COD_DVS_3ER": "District",
				"PEMDM_COD_DVS_1ER": "Region",
			},
			"BOPERS_MAE_MAI": map[string]interface{}{
				"PEMMA_GLS_DML_MAI": "johndoe@example.com",
			},
		},
	}

	customer, err := utils.TransformCustomerData(&data)
	assert.NoError(t, err)
	assert.Equal(t, "John", customer.PersonalIdentification.CustomerFirstName)
	assert.Equal(t, "Doe", customer.PersonalIdentification.CustomerLastName)
	assert.Equal(t, "Smith", customer.PersonalIdentification.CustomerSecondLastName)
	assert.Equal(t, "123 Main St", customer.PersonalAdditionalInfo.CustomerStreetName)
	assert.Equal(t, "johndoe@example.com", customer.PersonalAdditionalInfo.CustomerEmailAddress)

	// Caso 2: Payload vacío
	invalidData := map[string]interface{}{
		"payload": "invalid structure",
	}
	_, err = utils.TransformCustomerData(&invalidData)
	assert.Error(t, err)
	assert.Equal(t, "invalid payload structure", err.Error())

	// Caso 3: Nombre completo con un solo nombre
	dataWithOneName := map[string]interface{}{
		"payload": map[string]interface{}{
			"BOPERS_MAE_NAT_BSC": map[string]interface{}{
				"PEMNB_GLS_NOM_PEL": "John",
			},
		},
	}
	customer, err = utils.TransformCustomerData(&dataWithOneName)
	assert.NoError(t, err)
	assert.Equal(t, "John", customer.PersonalIdentification.CustomerFirstName)
	assert.Empty(t, customer.PersonalIdentification.CustomerMiddleName)
}

func TestFormatRUT(t *testing.T) {
	// Caso 1: RUT con formato
	formattedRUT := utils.FormatRut("12345678", "K")
	assert.Equal(t, "12.345.678-K", formattedRUT)

	// Caso 2: RUT vacío
	formattedRUT = utils.FormatRut("", "")
	assert.Equal(t, "", formattedRUT)

	// Caso 3: RUT con valores incorrectos
	formattedRUT = utils.FormatRut("12", "K")
	assert.Equal(t, "12-K", formattedRUT)
}

func TestCustomMarshalJSON(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Empty string `json:"empty,omitempty"`
	}

	// Caso 1: Estructura con datos
	obj := &TestStruct{
		Name: "John",
	}
	jsonData, err := utils.CustomMarshalJSON(obj)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"name":"John"}`, string(jsonData))

	// Caso 2: Estructura vacía
	obj = &TestStruct{}
	jsonData, err = utils.CustomMarshalJSON(obj)
	assert.NoError(t, err)
	assert.JSONEq(t, `{}`, string(jsonData))
}

func TestIsEmptyValue(t *testing.T) {
	// Caso 1: Cadena vacía
	assert.True(t, utils.IsEmptyValue(reflect.ValueOf("")))

	// Caso 2: Entero no vacío
	assert.False(t, utils.IsEmptyValue(reflect.ValueOf(1)))

	// Caso 3: Estructura con valores
	type TestStruct struct {
		Name string
	}
	testStruct := TestStruct{
		Name: "John",
	}
	assert.False(t, utils.IsEmptyValue(reflect.ValueOf(testStruct)))

	// Caso 4: Estructura vacía
	testStruct = TestStruct{}
	assert.True(t, utils.IsEmptyValue(reflect.ValueOf(testStruct)))

	// Caso 5: Booleano falso (cubre return !v.Bool())
	assert.True(t, utils.IsEmptyValue(reflect.ValueOf(false)))
	assert.False(t, utils.IsEmptyValue(reflect.ValueOf(true)))

	// Caso 6: Uint vacío y no vacío (cubre return v.Uint() == 0)
	assert.True(t, utils.IsEmptyValue(reflect.ValueOf(uint(0))))
	assert.False(t, utils.IsEmptyValue(reflect.ValueOf(uint(1))))

	// Caso 7: Float vacío y no vacío (cubre return v.Float() == 0)
	assert.True(t, utils.IsEmptyValue(reflect.ValueOf(float64(0))))
	assert.False(t, utils.IsEmptyValue(reflect.ValueOf(float64(1.1))))

	// Caso 8: Tiempo vacío y no vacío (cubre return v.Interface().(time.Time).IsZero())
	assert.True(t, utils.IsEmptyValue(reflect.ValueOf(time.Time{})))
	assert.False(t, utils.IsEmptyValue(reflect.ValueOf(time.Now())))

	// Caso 9: Tipo no manejado específicamente (cubre return false al final)
	assert.False(t, utils.IsEmptyValue(reflect.ValueOf(complex(1, 2))))
}

func TestOmitEmptyFields(t *testing.T) {
	// Caso 1: Puntero nulo
	var nilPointer *struct{}
	result := utils.OmitEmptyFields(reflect.ValueOf(nilPointer))
	assert.Nil(t, result)

	// Caso 2: Campo con json:"-"
	type TestStructSkip struct {
		Name  string `json:"name"`
		Email string `json:"-"`
	}
	objSkip := TestStructSkip{
		Name:  "John",
		Email: "johndoe@example.com",
	}
	resultMap := utils.OmitEmptyFields(reflect.ValueOf(objSkip)).(map[string]interface{})
	_, ok := resultMap["email"]
	assert.False(t, ok) // El campo "Email" debe ser omitido

	// Caso 3: Campo sin etiqueta json
	type TestStructNoTag struct {
		NoTagField string
	}
	objNoTag := TestStructNoTag{
		NoTagField: "No Tag",
	}
	resultMap = utils.OmitEmptyFields(reflect.ValueOf(objNoTag)).(map[string]interface{})
	assert.Equal(t, "No Tag", resultMap["NoTagField"]) // El nombre debe ser el del campo, ya que no hay etiqueta json

	// Caso 4: Map vacío
	emptyMap := map[string]interface{}{}
	result = utils.OmitEmptyFields(reflect.ValueOf(emptyMap))
	assert.Nil(t, result) // El map vacío debe ser omitido

	// Caso 5: Map con valores omitidos
	mapWithEmptyValues := map[string]interface{}{
		"key1": "",
		"key2": nil,
	}
	result = utils.OmitEmptyFields(reflect.ValueOf(mapWithEmptyValues))
	assert.Nil(t, result) // Debe retornar nil, ya que todos los valores son considerados vacíos

	// Caso 6: Slice vacío
	emptySlice := []string{}
	result = utils.OmitEmptyFields(reflect.ValueOf(emptySlice))
	assert.Nil(t, result) // El slice vacío debe ser omitido

	// Caso 7: Slice con valores
	sliceWithValues := []string{"value1", ""}
	resultSlice := utils.OmitEmptyFields(reflect.ValueOf(sliceWithValues)).([]interface{})
	assert.Equal(t, 1, len(resultSlice)) // Solo debe haber 1 valor no vacío
	assert.Equal(t, "value1", resultSlice[0])

	// Caso 8: Array vacío
	emptyArray := [0]string{}
	result = utils.OmitEmptyFields(reflect.ValueOf(emptyArray))
	assert.Nil(t, result) // El array vacío debe ser omitido

	// Caso 9: Array con valores
	arrayWithValues := [2]string{"value1", ""}
	resultArray := utils.OmitEmptyFields(reflect.ValueOf(arrayWithValues)).([]interface{})
	assert.Equal(t, 1, len(resultArray)) // Solo debe haber 1 valor no vacío
	assert.Equal(t, "value1", resultArray[0])

	// Caso 10: Reflect value no válido (cubre if !v.IsValid())
	var invalidValue reflect.Value
	result = utils.OmitEmptyFields(invalidValue)
	assert.Nil(t, result) // Debe retornar nil ya que el valor no es válido

	// Caso 11: Campo con valor nil (cubre if fieldValue == nil)
	mapWithNilValues := map[string]interface{}{
		"key1": "value1",
		"key2": nil, // Este debe ser omitido
	}
	resultMap = utils.OmitEmptyFields(reflect.ValueOf(mapWithNilValues)).(map[string]interface{})
	assert.Equal(t, 1, len(resultMap))           // Solo debe haber 1 campo no vacío
	assert.Equal(t, "value1", resultMap["key1"]) // Debe contener solo el campo no vacío
}
