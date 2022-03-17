// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package constantfetch

import (
	"errors"
	"io"
	"strings"

	cbtf "github.com/DataDog/btf-internals/btf"
)

type BTFConstantFetcher struct {
	spec      *cbtf.Spec
	constants map[string]uint64
	err       error
}

func NewBTFConstantFetcherFromSpec(spec *cbtf.Spec) *BTFConstantFetcher {
	return &BTFConstantFetcher{
		spec:      spec,
		constants: make(map[string]uint64),
	}
}

func NewBTFConstantFetcherFromReader(btfReader io.ReaderAt) (*BTFConstantFetcher, error) {
	spec, err := cbtf.LoadSpecFromReader(btfReader)
	if err != nil {
		return nil, err
	}
	return NewBTFConstantFetcherFromSpec(spec), nil
}

func NewBTFConstantFetcherFromCurrentKernel() (*BTFConstantFetcher, error) {
	spec, err := cbtf.LoadKernelSpec()
	if err != nil {
		return nil, err
	}
	return NewBTFConstantFetcherFromSpec(spec), nil
}

type constantRequest struct {
	id                  string
	sizeof              bool
	typeName, fieldName string
}

func (f *BTFConstantFetcher) runRequest(r constantRequest) {
	actualTy := getActualTypeName(r.typeName)
	types, err := f.spec.AnyTypesByName(actualTy)
	if err != nil || len(types) == 0 {
		// if it doesn't exist, we can't do anything
		return
	}

	finalValue := ErrorSentinel

	// the spec can contain multiple types for the same name
	// we check that they all return the same value for the same request
	for _, ty := range types {
		value := runRequestOnBTFType(r, ty)
		if value != ErrorSentinel {
			if finalValue != ErrorSentinel && finalValue != value {
				f.err = errors.New("mismatching values in multiple BTF types")
			}
			finalValue = value
		}
	}

	if finalValue != ErrorSentinel {
		f.constants[r.id] = finalValue
	}
}

func (f *BTFConstantFetcher) AppendSizeofRequest(id, typeName, headerName string) {
	f.runRequest(constantRequest{
		id:       id,
		sizeof:   true,
		typeName: getActualTypeName(typeName),
	})
}

func (f *BTFConstantFetcher) AppendOffsetofRequest(id, typeName, fieldName, headerName string) {
	f.runRequest(constantRequest{
		id:        id,
		sizeof:    false,
		typeName:  getActualTypeName(typeName),
		fieldName: fieldName,
	})
}

func (f *BTFConstantFetcher) FinishAndGetResults() (map[string]uint64, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.constants, nil
}

func getActualTypeName(tn string) string {
	prefixes := []string{"struct", "enum"}
	for _, prefix := range prefixes {
		tn = strings.TrimPrefix(tn, prefix+" ")
	}
	return tn
}

func runRequestOnBTFType(r constantRequest, ty cbtf.Type) uint64 {
	sTy, ok := ty.(*cbtf.Struct)
	if !ok {
		return ErrorSentinel
	}

	if r.sizeof {
		return uint64(sTy.Size)
	}

	for _, m := range sTy.Members {
		if m.Name == r.fieldName {
			return uint64(m.OffsetBits) / 8
		}
	}

	return ErrorSentinel
}
