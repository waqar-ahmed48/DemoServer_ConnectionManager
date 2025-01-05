package utilities

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/go-playground/validator"
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

/*
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
			fmt.Printf("Skipping field %s: is a struct or pointer to a struct\n", srcFieldName)
			continue
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
*/

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
