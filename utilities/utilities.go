package utilities

import (
	"errors"
	"reflect"
	"runtime"
	"strings"
	"sync"
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
		srcField := srcVal.FieldByName(tgtField.Name)

		// Copy if srcField exists, is valid, and has the same type
		if srcField.IsValid() && srcField.Type() == tgtField.Type {
			tgtElem.Field(i).Set(srcField)
		}
	}

	return nil
}
