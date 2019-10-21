package echo

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type (
	// Binder is the interface that wraps the Bind method.
	Binder interface {
		Bind(interface{}, Context) error
	}

	// BindUnmarshaler is the interface used to wrap the UnmarshalParam method.
	BindUnmarshaler interface {
		// UnmarshalParam decodes and assigns a value from an form or query param.
		UnmarshalParam(param string) error
	}

	binder struct{}
)

func (b *binder) Bind(i interface{}, c Context) (err error) {
	req := c.Request()
	if req.Method() == GET {
		if err = b.bindData(i, c.QueryParams()); err != nil {
			err = NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return
	}
	ctype := req.Header().Get(HeaderContentType)
	if req.Body() == nil {
		err = NewHTTPError(http.StatusBadRequest, "request body can't be empty")
		return
	}
	err = ErrUnsupportedMediaType
	switch {
	case strings.HasPrefix(ctype, MIMEApplicationJSON):
		if err = json.NewDecoder(req.Body()).Decode(i); err != nil {
			if ute, ok := err.(*json.UnmarshalTypeError); ok {
				err = NewHTTPError(http.StatusBadRequest, fmt.Sprintf("unmarshal type error: expected=%v, got=%v, offset=%v", ute.Type, ute.Value, ute.Offset))
			} else if se, ok := err.(*json.SyntaxError); ok {
				err = NewHTTPError(http.StatusBadRequest, fmt.Sprintf("syntax error: offset=%v, error=%v", se.Offset, se.Error()))
			} else {
				err = NewHTTPError(http.StatusBadRequest, err.Error())
			}
		}
	case strings.HasPrefix(ctype, MIMEApplicationXML):
		if err = xml.NewDecoder(req.Body()).Decode(i); err != nil {
			if ute, ok := err.(*xml.UnsupportedTypeError); ok {
				err = NewHTTPError(http.StatusBadRequest, fmt.Sprintf("unsupported type error: type=%v, error=%v", ute.Type, ute.Error()))
			} else if se, ok := err.(*xml.SyntaxError); ok {
				err = NewHTTPError(http.StatusBadRequest, fmt.Sprintf("syntax error: line=%v, error=%v", se.Line, se.Error()))
			} else {
				err = NewHTTPError(http.StatusBadRequest, err.Error())
			}
		}
	case strings.HasPrefix(ctype, MIMEApplicationForm), strings.HasPrefix(ctype, MIMEMultipartForm):
		if err = b.bindData(i, req.FormParams()); err != nil {
			err = NewHTTPError(http.StatusBadRequest, err.Error())
		}
	}
	return
}

func (b *binder) bindData(ptr interface{}, data map[string][]string) error {
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()

	if typ.Kind() != reflect.Struct {
		return errors.New("binding element must be a struct")
	}

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			continue
		}
		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get("form")

		if inputFieldName == "" {
			inputFieldName = typeField.Name
			// If "form" tag is nil, we inspect if the field is a struct.
			if structFieldKind == reflect.Struct {
				err := b.bindData(structField.Addr().Interface(), data)
				if err != nil {
					return err
				}
				continue
			}
		}
		inputValue, exists := data[inputFieldName]
		if !exists {
			continue
		}

		if ok, err := unmarshalField(typeField.Type.Kind(), inputValue[0], structField); ok {
			if err != nil {
				return err
			}
			continue
		}

		numElems := len(inputValue)
		if structFieldKind == reflect.Slice && numElems > 0 {
			sliceOf := structField.Type().Elem().Kind()
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for i := 0; i < numElems; i++ {
				if err := setWithProperType(sliceOf, inputValue[i], slice.Index(i)); err != nil {
					return err
				}
			}
			val.Field(i).Set(slice)
		} else {
			if err := setWithProperType(typeField.Type.Kind(), inputValue[0], structField); err != nil {
				return err
			}
		}
	}
	return nil
}

func setWithProperType(valueKind reflect.Kind, val string, structField reflect.Value) error {

	if ok, err := unmarshalField(valueKind, val, structField); ok {
		return err
	}

	//
	if valueKind == reflect.Ptr {

		switch structField.Type() {
		case reflect.TypeOf(Int):
			intVal, _ := strconv.ParseInt(val, 10, 0)
			structField.Set(reflect.ValueOf(&intVal))
		case reflect.TypeOf(Int8):
			intVal, _ := strconv.ParseInt(val, 10, 8)
			structField.Set(reflect.ValueOf(&intVal))
		case reflect.TypeOf(Int16):
			intVal, _ := strconv.ParseInt(val, 10, 16)
			structField.Set(reflect.ValueOf(&intVal))
		case reflect.TypeOf(Int32):
			intVal, _ := strconv.ParseInt(val, 10, 32)
			structField.Set(reflect.ValueOf(&intVal))
		case reflect.TypeOf(Int64):
			intVal, _ := strconv.ParseInt(val, 10, 64)
			structField.Set(reflect.ValueOf(&intVal))

		case reflect.TypeOf(Uint):
			intVal, _ := strconv.ParseUint(val, 10, 0)
			structField.Set(reflect.ValueOf(&intVal))
		case reflect.TypeOf(Uint8):
			intVal, _ := strconv.ParseUint(val, 10, 8)
			structField.Set(reflect.ValueOf(&intVal))
		case reflect.TypeOf(Uint16):
			intVal, _ := strconv.ParseUint(val, 10, 16)
			structField.Set(reflect.ValueOf(&intVal))
		case reflect.TypeOf(Uint32):
			intVal, _ := strconv.ParseUint(val, 10, 32)
			structField.Set(reflect.ValueOf(&intVal))
		case reflect.TypeOf(Uint64):
			intVal, _ := strconv.ParseUint(val, 10, 64)
			structField.Set(reflect.ValueOf(&intVal))

		case reflect.TypeOf(Float32):
			intVal, _ := strconv.ParseFloat(val, 32)
			structField.Set(reflect.ValueOf(&intVal))
		case reflect.TypeOf(Float64):
			intVal, _ := strconv.ParseFloat(val, 64)
			structField.Set(reflect.ValueOf(&intVal))


		case reflect.TypeOf(String):
			structField.Set(reflect.ValueOf(&val))

		case reflect.TypeOf(Bool):
			boolVal, _ := strconv.ParseBool(val)
			structField.Set(reflect.ValueOf(&boolVal))
		}

	}

	switch valueKind {
	case reflect.Int:
		return setIntField(val, 0, structField)
	case reflect.Int8:
		return setIntField(val, 8, structField)
	case reflect.Int16:
		return setIntField(val, 16, structField)
	case reflect.Int32:
		return setIntField(val, 32, structField)
	case reflect.Int64:
		return setIntField(val, 64, structField)
	case reflect.Uint:
		return setUintField(val, 0, structField)
	case reflect.Uint8:
		return setUintField(val, 8, structField)
	case reflect.Uint16:
		return setUintField(val, 16, structField)
	case reflect.Uint32:
		return setUintField(val, 32, structField)
	case reflect.Uint64:
		return setUintField(val, 64, structField)
	case reflect.Bool:
		return setBoolField(val, structField)
	case reflect.Float32:
		return setFloatField(val, 32, structField)
	case reflect.Float64:
		return setFloatField(val, 64, structField)
	case reflect.String:
		structField.SetString(val)
	default:
		return errors.New("unknown type")
	}
	return nil
}

func setIntField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	intVal, err := strconv.ParseInt(value, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	uintVal, err := strconv.ParseUint(value, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(value string, field reflect.Value) error {
	if value == "" {
		value = "false"
	}
	boolVal, err := strconv.ParseBool(value)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setFloatField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0.0"
	}
	floatVal, err := strconv.ParseFloat(value, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}

func unmarshalField(valueKind reflect.Kind, val string, field reflect.Value) (bool, error) {
	switch valueKind {
	case reflect.Ptr:
		return unmarshalFieldPtr(val, field)
	default:
		return unmarshalFieldNonPtr(val, field)
	}
}

func bindUnmarshaler(field reflect.Value) (BindUnmarshaler, bool) {
	ptr := reflect.New(field.Type())
	if ptr.CanInterface() {
		iface := ptr.Interface()
		if unmarshaler, ok := iface.(BindUnmarshaler); ok {
			return unmarshaler, ok
		}
	}
	return nil, false
}


func unmarshalFieldNonPtr(value string, field reflect.Value) (bool, error) {
	if unmarshaler, ok := bindUnmarshaler(field); ok {
		err := unmarshaler.UnmarshalParam(value)
		field.Set(reflect.ValueOf(unmarshaler).Elem())
		return true, err
	}
	return false, nil
}

func unmarshalFieldPtr(value string, field reflect.Value) (bool, error) {
	if field.IsNil() {
		// Initialize the pointer to a nil value
		field.Set(reflect.New(field.Type().Elem()))
	}
	return unmarshalFieldNonPtr(value, field.Elem())
}
