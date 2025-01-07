package utilities

import (
	"DemoServer_ConnectionManager/helper"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type MultiThreadedFunc func(threadId int, opsPerThread int)

func GetFunctionName() string {
	pc, _, _, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	parts := strings.Split(funcName, "/")

	return parts[len(parts)-1]
}

func CallMultiThreadedFunc(f MultiThreadedFunc, count int, threads int) {
	var wg sync.WaitGroup
	wg.Add(threads)

	// Use a channel to signal completion of each thread
	done := make(chan struct{})

	// Divide the work among multiple threads
	opsPerThread := count / threads
	for i := 0; i < threads; i++ {
		go func(threadID int) {
			defer wg.Done()
			f(threadID, opsPerThread)
			done <- struct{}{}
		}(i)
	}

	// Wait for all threads to complete
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for the completion signal
	<-done
}

func CopyMatchingFields(src, tgt interface{}) error {
	srcVal := reflect.ValueOf(src)
	tgtVal := reflect.ValueOf(tgt)

	// Ensure tgt is a pointer and can be dereferenced
	if tgtVal.Kind() != reflect.Ptr || tgtVal.IsNil() {
		return errors.New("target object must be a non-nil pointer to a struct")
	}

	tgtElem := tgtVal.Elem() // Dereference the pointer

	// Ensure tgtElem is a struct
	if tgtElem.Kind() != reflect.Struct {
		return errors.New("target object must be a pointer to a struct")
	}

	// Ensure src is a struct or a pointer to a struct
	if srcVal.Kind() == reflect.Ptr {
		srcVal = srcVal.Elem() // Dereference if it's a pointer
	}

	if srcVal.Kind() != reflect.Struct {
		return errors.New("source object must be a struct or a pointer to a struct")
	}

	// Iterate through the fields of the target struct
	for i := 0; i < tgtElem.NumField(); i++ {
		tgtField := tgtElem.Type().Field(i)
		tgtFieldVal := tgtElem.Field(i)
		srcField := srcVal.FieldByName(tgtField.Name)

		// Ensure srcField exists and is valid
		if !srcField.IsValid() || !tgtFieldVal.CanSet() {
			continue
		}

		srcFieldType := srcField.Type()
		srcFieldName := tgtField.Name

		// Skip if the field is a struct or pointer to a struct
		if srcFieldType.Kind() == reflect.Struct ||
			(srcFieldType.Kind() == reflect.Ptr && srcFieldType.Elem().Kind() == reflect.Struct) {
			// Updated CopyMatchingFields logic
			if srcField.Type().Kind() == reflect.Ptr && !srcField.IsNil() {
				// Source field is a non-nil pointer
				if tgtFieldVal.Kind() == reflect.Ptr {
					// Both source and destination fields are pointers
					if tgtFieldVal.Type() == srcField.Type() {
						// Types match, copy directly
						tgtFieldVal.Set(srcField)
					} else if tgtFieldVal.Type().Elem() == srcField.Type().Elem() {
						// Underlying types match, create a new value and copy
						newVal := reflect.New(tgtFieldVal.Type().Elem())
						newVal.Elem().Set(srcField.Elem())
						tgtFieldVal.Set(newVal)
					} else {
						// Log type mismatch
						fmt.Printf("Skipping field %s: incompatible pointer types (source: %s, target: %s)\n",
							srcFieldName, srcField.Type(), tgtFieldVal.Type())
					}
				} else {
					// Destination is not a pointer, check for direct assignment compatibility
					if tgtFieldVal.Type() == srcField.Type().Elem() {
						tgtFieldVal.Set(srcField.Elem())
					} else {
						// Log type mismatch
						fmt.Printf("Skipping field %s: incompatible types (source: %s, target: %s)\n",
							srcFieldName, srcField.Type().Elem(), tgtFieldVal.Type())
						continue
					}
				}
			} else {
				// Source is not a pointer, handle direct assignment
				if tgtFieldVal.Kind() == reflect.Ptr {
					// Destination is a pointer, create a new value
					if tgtFieldVal.Type().Elem() == srcField.Type() {
						newVal := reflect.New(tgtFieldVal.Type().Elem())
						newVal.Elem().Set(srcField)
						tgtFieldVal.Set(newVal)
					} else {
						// Log type mismatch
						fmt.Printf("Skipping field %s: incompatible types (source: %s, target: %s)\n",
							srcFieldName, srcField.Type(), tgtFieldVal.Type())
						continue
					}
				} else {
					// Direct assignment
					if tgtFieldVal.Type() == srcField.Type() {
						tgtFieldVal.Set(srcField)
					} else {
						// Log type mismatch
						fmt.Printf("Skipping field %s: incompatible types (source: %s, target: %s)\n",
							srcFieldName, srcField.Type(), tgtFieldVal.Type())
						continue
					}
				}
			}

		}

		// Handle pointer-to-value or pointer-to-pointer cases
		if srcField.Kind() == reflect.Ptr {
			if !srcField.IsNil() {
				// Dereference pointer from src and set if tgt is non-pointer
				if tgtFieldVal.Kind() != reflect.Ptr {
					tgtFieldVal.Set(srcField.Elem())
				} else {
					// Both src and tgt are pointers
					tgtFieldVal.Set(srcField)
				}
			}
		} else {
			// Both src and tgt are non-pointers
			if tgtFieldVal.Kind() == srcField.Kind() {
				tgtFieldVal.Set(srcField)
			}
		}
	}

	return nil
}

func ValidateAndWrapPayload(payload map[string]interface{}, target interface{}) error {
	// Ensure target is a pointer
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Ptr || targetVal.IsNil() {
		return errors.New("target must be a non-nil pointer to a struct")
	}

	// Ensure target is a struct
	targetElem := targetVal.Elem()
	if targetElem.Kind() != reflect.Struct {
		return errors.New("target must point to a struct")
	}

	// Marshal the map into JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.New("failed to marshal payload into JSON: " + err.Error())
	}

	// Unmarshal JSON into the target struct
	err = json.Unmarshal(payloadBytes, target)
	if err != nil {
		return errors.New("failed to unmarshal JSON into target struct: " + err.Error())
	}

	// Validate the target struct
	validate := validator.New()

	// Custom tag registration for skipping fields
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	err = validate.Struct(target)
	if err != nil {
		return errors.New("validation failed: " + err.Error())
	}

	return nil
}

// Helper function to set up tracing and logging
func SetupTraceAndLogger(r *http.Request, rw http.ResponseWriter, l *slog.Logger, funcName string, tracerName string) (context.Context, trace.Span, string, *slog.Logger) {
	tr := otel.Tracer(tracerName)
	ctx, span := tr.Start(r.Context(), funcName)
	traceLogger := l.With(
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	)
	requestID, cl := helper.PrepareContext(r, &rw, traceLogger)

	return ctx, span, requestID, cl
}

func ParseQueryParam(vars url.Values, key string, defaultValue, maxValue int) int {
	valueStr := vars.Get(key)
	if valueStr != "" {
		value, err := strconv.Atoi(valueStr)
		if err == nil && value >= 0 {
			return int(math.Min(float64(value), float64(maxValue)))
		}
	}
	return defaultValue
}

func WriteResponse(w http.ResponseWriter, cl *slog.Logger, data interface{}, span trace.Span) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		helper.LogError(cl, helper.ErrorJSONEncodingFailed, err, span)
	}
}

func ValidateQueryParam(
	param string,
	minValue int,
	strictGreaterThan bool,
	cl *slog.Logger,
	r *http.Request,
	rw http.ResponseWriter,
	span trace.Span,
	requestid string,
	helperError helper.ErrorTypeEnum) error {
	if param == "" {
		return nil
	}

	value, err := strconv.Atoi(param)
	if err != nil {
		helper.ReturnError(cl, http.StatusBadRequest, helperError, err, requestid, r, &rw, span)
		return err
	}

	if (strictGreaterThan && value <= minValue) || (!strictGreaterThan && value < minValue) {
		helper.ReturnError(cl, http.StatusBadRequest, helperError, fmt.Errorf("no internal error"), requestid, r, &rw, span)
		return fmt.Errorf("invalid value for parameter")
	}

	return nil
}

func UpdateObject[T any](db *gorm.DB, obj *T, ctx context.Context, tracerName string) error {

	tr := otel.Tracer(tracerName)
	_, span := tr.Start(ctx, GetFunctionName())
	defer span.End()

	// Begin a transaction
	tx := db.Begin()

	// Check if the transaction started successfully
	if tx.Error != nil {
		return tx.Error
	}

	result := tx.Save(obj)

	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func CreateObject[T any](db *gorm.DB, obj *T, ctx context.Context, tracerName string) error {

	tr := otel.Tracer(tracerName)
	_, span := tr.Start(ctx, GetFunctionName())
	defer span.End()

	// Begin a transaction
	tx := db.Begin()

	// Check if the transaction started successfully
	if tx.Error != nil {
		return tx.Error
	}

	result := tx.Create(obj)

	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Generalized middleware for validating connection id
func ValidateQueryStringParam(param string, r *http.Request, cl *slog.Logger, rw http.ResponseWriter, span trace.Span) (string, bool) {
	p := mux.Vars(r)[param]
	if len(p) == 0 {
		helper.ReturnError(
			cl,
			http.StatusBadRequest,
			helper.ErrorInvalidParameter,
			fmt.Errorf("invalid parameter value. parameter: %s", param),
			"",
			r,
			&rw,
			span,
		)
		return "", false
	}
	return p, true
}

// Middleware for decoding and validating JSON payloads
func DecodeAndValidate[T any](r *http.Request, cl *slog.Logger, rw http.ResponseWriter, span trace.Span) (*T, bool) {
	var payload T

	body, _ := io.ReadAll(r.Body)
	fmt.Println(string(body))
	r.Body = io.NopCloser(bytes.NewBuffer(body)) // Reset r.Body for further use

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		helper.ReturnError(
			cl,
			http.StatusBadRequest,
			helper.ErrorInvalidJSONSchemaForParameter,
			err,
			"",
			r,
			&rw,
			span,
		)
		return nil, false
	}

	err = validator.New().Struct(payload)
	if err != nil {
		helper.LogDebug(cl, helper.ErrorInvalidJSONSchemaForParameter, err, span)
		helper.ReturnError(
			cl,
			http.StatusBadRequest,
			helper.ErrorInvalidJSONSchemaForParameter,
			err,
			"",
			r,
			&rw,
			span,
		)
		return nil, false
	}
	return &payload, true
}
