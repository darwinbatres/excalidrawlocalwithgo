package validate

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var hasUpper = regexp.MustCompile(`[A-Z]`)
var hasDigitOrSpecial = regexp.MustCompile(`[0-9\W_]`)

// V is the package-level validator instance.
var V *validator.Validate

func init() {
	V = validator.New(validator.WithRequiredStructEnabled())

	// Register custom validation tags.
	V.RegisterValidation("slug", func(fl validator.FieldLevel) bool {
		return slugRegex.MatchString(fl.Field().String())
	})

	V.RegisterValidation("trimmed", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		return s == strings.TrimSpace(s)
	})

	V.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		return hasUpper.MatchString(s) && hasDigitOrSpecial.MatchString(s)
	})
}

// Struct validates a struct and returns an API error with field details on failure.
func Struct(v any) *apierror.Error {
	if err := V.Struct(v); err != nil {
		validationErrors, ok := err.(validator.ValidationErrors)
		if !ok {
			return apierror.ErrBadRequest
		}

		details := make(map[string]any, len(validationErrors))
		for _, fe := range validationErrors {
			field := strings.ToLower(fe.Field()[:1]) + fe.Field()[1:]
			details[field] = formatValidationError(fe)
		}
		return apierror.ErrValidationFailed.WithDetails(details)
	}
	return nil
}

// DecodeAndValidate decodes JSON from a request body and validates the struct.
func DecodeAndValidate(r *http.Request, v any) *apierror.Error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return apierror.ErrInvalidJSON
	}
	return Struct(v)
}

func formatValidationError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "min":
		return "Must be at least " + fe.Param() + " characters"
	case "max":
		return "Must be at most " + fe.Param() + " characters"
	case "gte":
		return "Must be greater than or equal to " + fe.Param()
	case "lte":
		return "Must be less than or equal to " + fe.Param()
	case "oneof":
		return "Must be one of: " + fe.Param()
	case "url":
		return "Must be a valid URL"
	case "slug":
		return "Must contain only lowercase letters, numbers, and hyphens"
	case "trimmed":
		return "Must not have leading or trailing whitespace"
	case "alphanum":
		return "Must contain only letters and numbers"
	case "uuid":
		return "Must be a valid UUID"
	case "len":
		return "Must be exactly " + fe.Param() + " characters"
	case "gt":
		return "Must be greater than " + fe.Param()
	case "lt":
		return "Must be less than " + fe.Param()
	case "strongpassword":
		return "Must contain at least one uppercase letter and one number or special character"
	default:
		return "Invalid value"
	}
}
