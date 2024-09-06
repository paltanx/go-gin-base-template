package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/sebasegovia01/base-template-go-gin/models"
)

var FormatRut = formatRUT

var IsEmptyValue = isEmptyValue

var OmitEmptyFields = omitEmptyFields

func TransformCustomerData(data *map[string]interface{}) (*models.Customer, error) {
	payload, ok := (*data)["payload"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload structure")
	}

	customer := &models.Customer{}
	personalIdentification := models.PersonalCustomerIdentification{}
	personalAdditionalInfo := models.PersonalCustomerAdditionalInfo{}

	// BOPERS_MAE_NAT_BSC
	if maeBsc, ok := payload["BOPERS_MAE_NAT_BSC"].(map[string]interface{}); ok {
		if fullName, ok := maeBsc["PEMNB_GLS_NOM_PEL"].(string); ok {
			names := strings.Fields(fullName)
			if len(names) > 0 {
				personalIdentification.CustomerFirstName = names[0]
			}
			if len(names) > 1 {
				personalIdentification.CustomerMiddleName = strings.Join(names[1:], " ")
			}
		}
		if lastName, ok := maeBsc["PEMNB_GLS_APL_PAT"].(string); ok {
			personalIdentification.CustomerLastName = lastName
		}
		if secondLastName, ok := maeBsc["PEMNB_GLS_APL_MAT"].(string); ok {
			personalIdentification.CustomerSecondLastName = secondLastName
		}
	}

	// BOPERS_MAE_PRD_CLI
	if maePrdCli, ok := payload["BOPERS_MAE_PRD_CLI"].(map[string]interface{}); ok {
		if initDate, ok := maePrdCli["PEMPC_FCH_ALS_CTT"].(string); ok {
			if t, err := time.Parse(time.RFC3339, initDate); err == nil && !t.IsZero() {
				personalIdentification.CustomerInitDate = t
			}
		}
	}

	// BOPERS_MAE_IDE
	if maeIde, ok := payload["BOPERS_MAE_IDE"].(map[string]interface{}); ok {
		var rut, dv string
		if rutValue, ok := maeIde["PEMID_GLS_NRO_DCT_IDE_K"].(string); ok {
			rut = rutValue
		}
		if dvValue, ok := maeIde["PEMID_DVR_NRO_DCT_IDE"].(string); ok {
			dv = dvValue
		}
		if rut != "" || dv != "" {
			personalIdentification.CustomerIdentification = formatRUT(rut, dv)
		}
	}

	// BOPERS_MAE_DML
	if maeDml, ok := payload["BOPERS_MAE_DML"].(map[string]interface{}); ok {
		if streetName, ok := maeDml["PEMDM_GLS_CAL_DML"].(string); ok {
			personalAdditionalInfo.CustomerStreetName = streetName
		}
		if buildingNumber, ok := maeDml["PEMDM_NRO_DML"].(string); ok {
			personalAdditionalInfo.CustomerBuildingNumber = buildingNumber
		}
		if districtName, ok := maeDml["PEMDM_COD_DVS_3ER"].(string); ok {
			personalAdditionalInfo.CustomerDistrictName = districtName
		}
		if regionName, ok := maeDml["PEMDM_COD_DVS_1ER"].(string); ok {
			personalAdditionalInfo.CustomerCountrySubDivisionMajorName = regionName
		}
	}

	// BOPERS_MAE_MAI
	if maeMai, ok := payload["BOPERS_MAE_MAI"].(map[string]interface{}); ok {
		if emailAddress, ok := maeMai["PEMMA_GLS_DML_MAI"].(string); ok {
			personalAdditionalInfo.CustomerEmailAddress = emailAddress
		}
	}

	// Only set PersonalIdentification if it's not empty
	if personalIdentification != (models.PersonalCustomerIdentification{}) {
		customer.PersonalIdentification = personalIdentification
	}

	// Only set PersonalAdditionalInfo if it's not empty
	if personalAdditionalInfo != (models.PersonalCustomerAdditionalInfo{}) {
		customer.PersonalAdditionalInfo = personalAdditionalInfo
	}

	// Remove empty fields from the entire customer struct
	customerJSON, _ := CustomMarshalJSON(customer)
	json.Unmarshal(customerJSON, customer)

	return customer, nil
}

// formatRUT formatea el RUT chileno con puntos y guión
func formatRUT(rut string, dv string) string {
	if rut == "" || dv == "" {
		return ""
	}

	// Eliminar cualquier formato existente
	rut = strings.ReplaceAll(rut, ".", "")
	rut = strings.ReplaceAll(rut, "-", "")

	// Formatear el RUT
	length := len(rut)
	var formatted strings.Builder

	for i := 0; i < length; i++ {
		if i > 0 && (length-i)%3 == 0 {
			formatted.WriteString(".")
		}
		formatted.WriteByte(rut[i])
	}

	// Añadir el guión y el dígito verificador
	formatted.WriteString("-")
	formatted.WriteString(strings.ToUpper(dv)) // Asegurarse de que 'k' sea mayúscula

	return formatted.String()
}

func omitEmptyFields(v reflect.Value) interface{} {
	if !v.IsValid() {
		return nil
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		result := make(map[string]interface{})
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.CanInterface() {
				continue // Omitir campos no exportados
			}
			structField := v.Type().Field(i)
			jsonTag := structField.Tag.Get("json")
			jsonName := strings.Split(jsonTag, ",")[0]
			if jsonName == "-" {
				continue // Omitir campos con json:"-"
			}
			if jsonName == "" {
				jsonName = structField.Name
			}
			fieldValue := omitEmptyFields(field)
			if fieldValue != nil {
				result[jsonName] = fieldValue
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	case reflect.Map:
		result := make(map[string]interface{})
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)

			// Verificar si el valor es inválido o nil
			if !value.IsValid() || value.IsNil() {
				continue // Omitir valores vacíos o nil
			}

			fieldValue := omitEmptyFields(value)
			if fieldValue == nil {
				continue // Omitir valores vacíos
			}
			// Verificar si el valor es una cadena vacía
			if str, ok := fieldValue.(string); ok && str == "" {
				continue // Omitir cadenas vacías
			}
			result[key.String()] = fieldValue
		}
		if len(result) == 0 {
			return nil
		}
		return result
	case reflect.Slice, reflect.Array:
		result := make([]interface{}, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			fieldValue := omitEmptyFields(elem)
			if fieldValue != nil {
				result = append(result, fieldValue)
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	default:
		if isEmptyValue(v) {
			return nil
		}
		return v.Interface()
	}
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return v.Interface().(time.Time).IsZero()
		}
		// Check if all fields of the struct are empty
		for i := 0; i < v.NumField(); i++ {
			if !isEmptyValue(v.Field(i)) {
				return false
			}
		}
		return true
	}
	return false
}

func CustomMarshalJSON(v interface{}) ([]byte, error) {
	result := omitEmptyFields(reflect.ValueOf(v))
	if result == nil {
		return json.Marshal(struct{}{})
	}
	return json.Marshal(result)
}
