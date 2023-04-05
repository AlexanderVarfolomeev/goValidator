package validator

import (
	"github.com/pkg/errors"
	"reflect"
	"strconv"
	"strings"
)

var ErrNotStruct = errors.New("wrong argument given, should be a struct")
var ErrInvalidValidatorSyntax = errors.New("invalid validator syntax")
var ErrValidateForUnexportedFields = errors.New("validation for unexported field is not allowed")

type ValidationError struct {
	Err error
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	var res []string
	for _, validationError := range v {
		res = append(res, validationError.Err.Error())
	}
	return strings.Join(res, ",")
}

func Validate(v any) error {
	var validationErrors ValidationErrors

	if reflect.TypeOf(v).Kind() == reflect.Struct {
		s := reflect.TypeOf(v)
		elem := reflect.ValueOf(&v).Elem().Elem()

		for i := 0; i < s.NumField(); i++ {
			if t := s.Field(i).Tag.Get("validate"); !s.Field(i).IsExported() && len(t) != 0 {
				return ValidationErrors{ValidationError{ErrValidateForUnexportedFields}} // ErrValidateForUnexportedFields
			} else {
				var constraints Constraints
				constraints, validationErrors = ParseConstraints(s.Field(i), validationErrors)
				validationErrors = CheckConstraints(elem.Field(i), s.Field(i).Name, constraints, validationErrors)
			}
		}
	} else {
		return ErrNotStruct
	}

	if len(validationErrors) == 0 {
		return nil
	}
	return validationErrors
}

// validate:"max:2;min:3;len:3;in:2,3,4,"`

func ParseConstraints(f reflect.StructField, validationErrors ValidationErrors) (Constraints, ValidationErrors) {
	constraints := NewConstraints()

	if s := f.Tag.Get("validate"); len(s) != 0 {
		cons := strings.Split(s, ";")

		for _, con := range cons {
			s := strings.Split(con, ":")
			switch s[0] {
			case "max":
				max, err := ParseInt(s[1])
				if err != nil {
					validationErrors = append(validationErrors, ValidationError{err})
				} else {
					constraints.max = max
				}
			case "min":
				min, err := ParseInt(s[1])
				if err != nil {
					validationErrors = append(validationErrors, ValidationError{err})
				} else {
					constraints.min = min
				}
			case "len":
				l, err := ParseInt(s[1])
				if err != nil {
					validationErrors = append(validationErrors, ValidationError{err})
				} else if l < 0 {
					validationErrors = append(validationErrors, ValidationError{errors.New("wrong length")})
				} else {
					constraints.len = l
				}
			case "in":
				constraints.in = strings.Split(s[1], ",")
			}
		}
	}

	return constraints, validationErrors
}

func CheckConstraints(val reflect.Value, fieldName string, constraints Constraints, validationErrors ValidationErrors) ValidationErrors {
	if val.Kind() == reflect.String {
		return checkStringConstraints(val, fieldName, constraints, validationErrors)
	}

	if val.Kind() == reflect.Int {
		return checkIntConstraints(val, fieldName, constraints, validationErrors)
	}

	if val.Kind() == reflect.Slice {
		return checkSliceConstraints(val, fieldName, constraints, validationErrors)
	}

	return validationErrors
}

func checkStringConstraints(val reflect.Value, fieldName string, constraints Constraints, validationErrors ValidationErrors) ValidationErrors {
	if constraints.max != -1 && len(val.String()) > constraints.max {
		validationErrors = append(validationErrors, ValidationError{errors.New("field: " + fieldName + " err: length can't be more than max")})
	}
	if constraints.min != -1 && len(val.String()) < constraints.min {
		validationErrors = append(validationErrors, ValidationError{errors.New("field: " + fieldName + " err: length can't be less than min")})
	}
	if constraints.len != -1 && len(val.String()) != constraints.len {
		validationErrors = append(validationErrors, ValidationError{errors.New("field: " + fieldName + " err: length must be equal to len")})
	}

	if constraints.in != nil {
		var find bool
		for _, s := range constraints.in {
			if val.String() == s {
				find = true
				break
			}
		}
		if !find {
			validationErrors = append(validationErrors, ValidationError{errors.New("field: " + fieldName + " err: value is not contained in the 'in'")})
		}
	}

	return validationErrors
}

func checkIntConstraints(val reflect.Value, fieldName string, constraints Constraints, validationErrors ValidationErrors) ValidationErrors {
	if constraints.max != -1 && val.Int() > int64(constraints.max) {
		validationErrors = append(validationErrors, ValidationError{errors.New("field: " + fieldName + " err: value can't be more than max")})
	}
	if constraints.min != -1 && val.Int() < int64(constraints.min) {
		validationErrors = append(validationErrors, ValidationError{errors.New("field: " + fieldName + " err: value can't be less than min")})
	}
	if constraints.len != -1 {
		validationErrors = append(validationErrors, ValidationError{ErrInvalidValidatorSyntax})
	}

	if constraints.in != nil {
		var find bool
		for _, s := range constraints.in {

			num, err := strconv.Atoi(s)
			if err != nil {
				validationErrors = append(validationErrors, ValidationError{ErrInvalidValidatorSyntax})
			}

			if val.Int() == int64(num) {
				find = true
				break
			}
		}
		if !find {
			validationErrors = append(validationErrors, ValidationError{errors.New("field: " + fieldName + " err: value is not contained in the 'in'")})
		}
	}
	return validationErrors
}

func checkSliceConstraints(val reflect.Value, fieldName string, constraints Constraints, validationErrors ValidationErrors) ValidationErrors {
	for i := 0; i < val.Len(); i++ {
		if val.Index(i).Kind() == reflect.Int {
			validationErrors = checkIntConstraints(val.Index(i), fieldName+" "+strconv.Itoa(i)+"th element", constraints, validationErrors)
		} else {
			validationErrors = checkStringConstraints(val.Index(i), fieldName+" "+strconv.Itoa(i)+"th element", constraints, validationErrors)
		}
	}

	return validationErrors
}

func ParseInt(s string) (int, error) {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0, ErrInvalidValidatorSyntax
	}

	return val, nil
}

func NewConstraints() Constraints {
	return Constraints{len: -1, in: nil, min: -1, max: -1}
}

type Constraints struct {
	len int
	in  []string
	min int
	max int
}
