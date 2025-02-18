// Copyright © 2022 Meroxa, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//nolint:revive,dogsled,stylecheck // this is a test file
package plugin

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/conduitio/conduit-connector-protocol/cpluginv1"
	"github.com/conduitio/conduit-connector-protocol/cpluginv1/mock"
	"github.com/conduitio/conduit/pkg/foundation/assert"
	"github.com/conduitio/conduit/pkg/foundation/cerrors"
	"github.com/conduitio/conduit/pkg/record"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

// AcceptanceTestV1 is the acceptance test that all implementations of v1
// plugins should pass. It should manually be called from a test case in each
// implementation:
//
//   func TestPlugin(t *testing.T) {
//       testDispenser := func() {...}
//       plugin.AcceptanceTestV1(t, testDispenser)
//   }
//
func AcceptanceTestV1(t *testing.T, tdf testDispenserFunc) {
	// specifier tests
	run(t, tdf, testSpecifier_Specify_Success)
	run(t, tdf, testSpecifier_Specify_Fail)

	// source tests
	run(t, tdf, testSource_Configure_Success)
	run(t, tdf, testSource_Configure_Fail)
	run(t, tdf, testSource_Start_WithPosition)
	run(t, tdf, testSource_Start_EmptyPosition)
	run(t, tdf, testSource_Read_Success)
	run(t, tdf, testSource_Read_WithoutStart)
	run(t, tdf, testSource_Read_AfterStop)
	run(t, tdf, testSource_Read_CancelContext)
	run(t, tdf, testSource_Ack_Success)
	run(t, tdf, testSource_Ack_WithoutStart)
	run(t, tdf, testSource_Run_Fail)
	run(t, tdf, testSource_Teardown_Success)

	// destination tests
	run(t, tdf, testDestination_Configure_Success)
	run(t, tdf, testDestination_Configure_Fail)
	run(t, tdf, testDestination_Start_Success)
	run(t, tdf, testDestination_Start_Fail)
	run(t, tdf, testDestination_Write_Success)
	run(t, tdf, testDestination_Write_WithoutStart)
	run(t, tdf, testDestination_Ack_Success)
	run(t, tdf, testDestination_Ack_WithError)
	run(t, tdf, testDestination_Ack_WithoutStart)
	run(t, tdf, testDestination_Run_Fail)
	run(t, tdf, testDestination_Teardown_Success)
}

func run(t *testing.T, tdf testDispenserFunc, test func(*testing.T, testDispenserFunc)) {
	name := runtime.FuncForPC(reflect.ValueOf(test).Pointer()).Name()
	name = name[strings.LastIndex(name, ".")+1:]
	t.Run(name, func(t *testing.T) { test(t, tdf) })
}

type testDispenserFunc func(*testing.T) (Dispenser, *mock.SpecifierPlugin, *mock.SourcePlugin, *mock.DestinationPlugin)

// ---------------
// -- SPECIFIER --
// ---------------

func testSpecifier_Specify_Success(t *testing.T, tdf testDispenserFunc) {
	dispenser, mockSpecifier, _, _ := tdf(t)

	want := Specification{
		Name:        "test-name",
		Summary:     "A short summary",
		Description: "A long description",
		Version:     "v1.2.3",
		Author:      "Donald Duck",
		SourceParams: map[string]Parameter{
			"param1.1": {Default: "foo", Type: "string", Description: "Param 1.1 description"},
			"param2.1": {Default: "bar", Type: "string", Description: "Param 1.2 description", Validations: []Validation{{Type: ValidationTypeRequired}}},
		},
		DestinationParams: map[string]Parameter{
			"param2.1": {Default: "baz", Type: "string", Description: "Param 2.1 description"},
			"param2.2": {Default: "qux", Type: "string", Description: "Param 2.2 description", Validations: []Validation{{Type: ValidationTypeRequired}}},
		},
	}

	mockSpecifier.EXPECT().
		Specify(gomock.Any(), cpluginv1.SpecifierSpecifyRequest{}).
		Return(cpluginv1.SpecifierSpecifyResponse{
			Name:        want.Name,
			Summary:     want.Summary,
			Description: want.Description,
			Version:     want.Version,
			Author:      want.Author,
			SourceParams: map[string]cpluginv1.SpecifierParameter{
				"param1.1": {Default: "foo", Required: false, Description: "Param 1.1 description"},
				"param2.1": {Default: "bar", Required: true, Description: "Param 1.2 description"},
			},
			DestinationParams: map[string]cpluginv1.SpecifierParameter{
				"param2.1": {Default: "baz", Required: false, Description: "Param 2.1 description"},
				"param2.2": {Default: "qux", Required: true, Description: "Param 2.2 description"},
			},
		}, nil)

	specifier, err := dispenser.DispenseSpecifier()
	if err != nil {
		t.Fatalf("error dispensing specifier: %+v", err)
	}

	got, err := specifier.Specify()
	if err != nil {
		t.Fatalf("error dispensing specifier: %+v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("expected specification: %s", diff)
	}
}

func testSpecifier_Specify_Fail(t *testing.T, tdf testDispenserFunc) {
	dispenser, mockSpecifier, _, _ := tdf(t)

	want := cerrors.New("specify error")
	mockSpecifier.EXPECT().
		Specify(gomock.Any(), cpluginv1.SpecifierSpecifyRequest{}).
		Return(cpluginv1.SpecifierSpecifyResponse{}, want)

	specifier, err := dispenser.DispenseSpecifier()
	if err != nil {
		t.Fatalf("error dispensing specifier: %+v", err)
	}

	_, got := specifier.Specify()
	if got.Error() != want.Error() {
		t.Fatalf("want: %+v, got: %+v", want, got)
	}
}

// ------------
// -- SOURCE --
// ------------

func testSource_Configure_Success(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, mockSource, _ := tdf(t)

	cfg := map[string]string{
		"foo":   "bar",
		"empty": "",
	}
	want := cerrors.New("init error")
	mockSource.EXPECT().
		Configure(gomock.Any(), cpluginv1.SourceConfigureRequest{Config: cfg}).
		Return(cpluginv1.SourceConfigureResponse{}, want)

	source, err := dispenser.DispenseSource()
	if err != nil {
		t.Fatalf("error dispensing source: %+v", err)
	}

	got := source.Configure(ctx, cfg)
	if got.Error() != want.Error() {
		t.Fatalf("want: %+v, got: %+v", want, got)
	}
}

func testSource_Configure_Fail(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, mockSource, _ := tdf(t)

	mockSource.EXPECT().
		Configure(gomock.Any(), cpluginv1.SourceConfigureRequest{Config: nil}).
		Return(cpluginv1.SourceConfigureResponse{}, nil)

	source, err := dispenser.DispenseSource()
	if err != nil {
		t.Fatalf("error dispensing source: %+v", err)
	}

	got := source.Configure(ctx, map[string]string{})
	if got != nil {
		t.Fatalf("want: nil, got: %+v", got)
	}
}

func testSource_Start_WithPosition(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, mockSource, _ := tdf(t)

	pos := record.Position("test-position")

	// Function Source.Run is called in a goroutine, we have to wait for it to
	// run to prove this works.
	closeCh := make(chan struct{})
	mockSource.EXPECT().
		Start(gomock.Any(), cpluginv1.SourceStartRequest{Position: pos}).
		Return(cpluginv1.SourceStartResponse{}, nil)
	mockSource.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, cpluginv1.SourceRunStream) error {
			close(closeCh)
			return nil
		})

	source, err := dispenser.DispenseSource()
	if err != nil {
		t.Fatalf("error dispensing source: %+v", err)
	}

	err = source.Start(ctx, pos)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	select {
	case <-closeCh:
	case <-time.After(time.Second):
		t.Fatal("should've received call to source.Run")
	}
}

func testSource_Start_EmptyPosition(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, mockSource, _ := tdf(t)

	// Function Source.Run is called in a goroutine, we have to wait for it to
	// run to prove this works.
	closeCh := make(chan struct{})
	mockSource.EXPECT().
		Start(gomock.Any(), cpluginv1.SourceStartRequest{Position: nil}).
		Return(cpluginv1.SourceStartResponse{}, nil)
	mockSource.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, cpluginv1.SourceRunStream) error {
			close(closeCh)
			return nil
		})

	source, err := dispenser.DispenseSource()
	if err != nil {
		t.Fatalf("error dispensing source: %+v", err)
	}

	err = source.Start(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	select {
	case <-closeCh:
	case <-time.After(time.Second):
		t.Fatal("should've received call to source.Run")
	}
}

func testSource_Read_Success(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, mockSource, _ := tdf(t)

	var want []record.Record
	for i := 0; i < 10; i++ {
		want = append(want, record.Record{
			Position: record.Position(fmt.Sprintf("test-position-%d", i)),
			Metadata: map[string]string{
				"foo":   "bar",
				"empty": "",
			},
			CreatedAt: time.Now().UTC(),
			Key: record.RawData{
				Raw: []byte("test-key"),
			},
			Payload: record.RawData{
				Raw: []byte("test-payload"),
			},
		})
	}

	mockSource.EXPECT().
		Start(gomock.Any(), cpluginv1.SourceStartRequest{Position: nil}).
		Return(cpluginv1.SourceStartResponse{}, nil)
	mockSource.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, stream cpluginv1.SourceRunStream) error {
			for _, r := range want {
				plugRec := cpluginv1.Record{
					Position:  r.Position,
					Metadata:  r.Metadata,
					CreatedAt: r.CreatedAt,
					Key:       cpluginv1.RawData(r.Key.(record.RawData).Raw),
					Payload:   cpluginv1.RawData(r.Payload.(record.RawData).Raw),
				}

				err := stream.Send(cpluginv1.SourceRunResponse{Record: plugRec})
				if err != nil {
					return err
				}
			}
			return nil
		})

	source, err := dispenser.DispenseSource()
	if err != nil {
		t.Fatalf("error dispensing source: %+v", err)
	}

	err = source.Start(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	var got []record.Record
	for i := 0; i < len(want); i++ {
		rec, err := source.Read(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %+v", err)
		}
		// read at is recorded when we receive the record, adjust in the expectation
		want[i].ReadAt = rec.ReadAt
		got = append(got, rec)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("expected record: %s", diff)
	}
}

func testSource_Read_WithoutStart(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, _ := tdf(t)

	source, err := dispenser.DispenseSource()
	if err != nil {
		t.Fatalf("error dispensing source: %+v", err)
	}

	_, err = source.Read(ctx)
	if !cerrors.Is(err, ErrStreamNotOpen) {
		t.Fatalf("unexpected error: %+v", err)
	}
}

func testSource_Read_AfterStop(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, mockSource, _ := tdf(t)

	stopRunCh := make(chan struct{})
	mockSource.EXPECT().
		Start(gomock.Any(), cpluginv1.SourceStartRequest{}).
		Return(cpluginv1.SourceStartResponse{}, nil)
	mockSource.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, cpluginv1.SourceRunStream) error {
			<-stopRunCh
			return nil
		})
	mockSource.EXPECT().
		Stop(gomock.Any(), cpluginv1.SourceStopRequest{}).
		DoAndReturn(func(context.Context, cpluginv1.SourceStopRequest) (cpluginv1.SourceStopResponse, error) {
			close(stopRunCh)
			return cpluginv1.SourceStopResponse{}, nil
		})

	source, err := dispenser.DispenseSource()
	if err != nil {
		t.Fatalf("error dispensing source: %+v", err)
	}

	err = source.Start(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	err = source.Stop(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	_, err = source.Read(ctx)
	if !cerrors.Is(err, ErrStreamNotOpen) {
		t.Fatalf("unexpected error: %+v", err)
	}

	select {
	case <-stopRunCh:
	case <-time.After(time.Second):
		t.Fatal("should've received call to source.Stop")
	}
}

func testSource_Read_CancelContext(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, mockSource, _ := tdf(t)

	stopRunCh := make(chan struct{})
	mockSource.EXPECT().
		Start(gomock.Any(), cpluginv1.SourceStartRequest{Position: nil}).
		Return(cpluginv1.SourceStartResponse{}, nil)
	mockSource.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, stream cpluginv1.SourceRunStream) error {
			<-stopRunCh
			return nil
		})

	source, err := dispenser.DispenseSource()
	if err != nil {
		t.Fatalf("error dispensing source: %+v", err)
	}

	startCtx, startCancel := context.WithCancel(ctx)
	err = source.Start(startCtx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	// calling read when source didn't produce records should block until start
	// ctx is cancelled
	time.AfterFunc(time.Millisecond*50, func() {
		startCancel()
	})

	_, err = source.Read(ctx)
	assert.Error(t, err)
	// TODO see if we can change this error into context.Canceled, right now we
	//  follow the default gRPC behavior
	if cerrors.Is(err, context.Canceled) {
		t.Fatalf("unexpected error: %+v", err)
	}
	if cerrors.Is(err, ErrStreamNotOpen) {
		t.Fatalf("unexpected error: %+v", err)
	}

	close(stopRunCh) // stop run channel
}

func testSource_Ack_Success(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, mockSource, _ := tdf(t)

	want := []byte("test-position")

	// Function Source.Run is called in a goroutine, we have to wait for it to
	// run to prove this works.
	closeCh := make(chan struct{})
	mockSource.EXPECT().
		Start(gomock.Any(), cpluginv1.SourceStartRequest{Position: nil}).
		Return(cpluginv1.SourceStartResponse{}, nil)
	mockSource.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, stream cpluginv1.SourceRunStream) error {
			defer close(closeCh)
			got, err := stream.Recv()
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if diff := cmp.Diff(got.AckPosition, want); diff != "" {
				t.Errorf("expected ack: %s", diff)
			}
			return nil
		})

	source, err := dispenser.DispenseSource()
	if err != nil {
		t.Fatalf("error dispensing source: %+v", err)
	}

	err = source.Start(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	err = source.Ack(ctx, record.Position("test-position"))
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	select {
	case <-closeCh:
	case <-time.After(time.Second):
		t.Fatal("should've received call to source.Ack")
	}

	// wait for stream closing to propagate from plugin to Conduit
	time.Sleep(time.Millisecond * 50)

	// acking after the stream is closed should result in an error
	err = source.Ack(ctx, record.Position("test-position"))
	if !cerrors.Is(err, ErrStreamNotOpen) {
		t.Fatalf("unexpected error: %+v", err)
	}
}

func testSource_Ack_WithoutStart(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, _ := tdf(t)

	source, err := dispenser.DispenseSource()
	if err != nil {
		t.Fatalf("error dispensing source: %+v", err)
	}

	err = source.Ack(ctx, []byte("test-position"))
	if !cerrors.Is(err, ErrStreamNotOpen) {
		t.Fatalf("unexpected error: %+v", err)
	}
}

func testSource_Run_Fail(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, mockSource, _ := tdf(t)

	want := cerrors.New("test-error")

	// Function Source.Run is called in a goroutine, we have to wait for it to
	// run to prove this works.
	closeCh := make(chan struct{})
	mockSource.EXPECT().
		Start(gomock.Any(), cpluginv1.SourceStartRequest{Position: nil}).
		Return(cpluginv1.SourceStartResponse{}, nil)
	mockSource.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, stream cpluginv1.SourceRunStream) error {
			defer close(closeCh)
			_, _ = stream.Recv() // receive ack and fail
			return want
		})

	source, err := dispenser.DispenseSource()
	if err != nil {
		t.Fatalf("error dispensing source: %+v", err)
	}

	err = source.Start(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	err = source.Ack(ctx, record.Position("test-position"))
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	select {
	case <-closeCh:
	case <-time.After(time.Second):
		t.Fatal("should've received call to source.Ack")
	}

	// Error is returned through the Read function, that's the incoming stream.
	_, err = source.Read(ctx)
	// Unwrap inner-most error
	var got error
	for unwrapped := err; unwrapped != nil; {
		got = unwrapped
		unwrapped = cerrors.Unwrap(unwrapped)
	}

	if got.Error() != want.Error() {
		t.Fatalf("want: %+v, got: %+v", want, got)
	}

	// Error is returned through the Ack function, that's the outgoing stream.
	err = source.Ack(ctx, record.Position("test-position"))
	// Unwrap inner-most error
	got = nil
	for unwrapped := err; unwrapped != nil; {
		got = unwrapped
		unwrapped = cerrors.Unwrap(unwrapped)
	}

	if got.Error() != want.Error() {
		t.Fatalf("want: %+v, got: %+v", want, got)
	}
}

func testSource_Teardown_Success(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, mockSource, _ := tdf(t)

	want := cerrors.New("init error")
	closeCh := make(chan struct{})
	stopRunCh := make(chan struct{})
	mockSource.EXPECT().
		Start(gomock.Any(), cpluginv1.SourceStartRequest{Position: nil}).
		Return(cpluginv1.SourceStartResponse{}, nil)
	mockSource.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, stream cpluginv1.SourceRunStream) error {
			defer close(closeCh)
			<-stopRunCh
			return nil
		})
	mockSource.EXPECT().
		Teardown(gomock.Any(), cpluginv1.SourceTeardownRequest{}).
		Return(cpluginv1.SourceTeardownResponse{}, want)

	source, err := dispenser.DispenseSource()
	if err != nil {
		t.Fatalf("error dispensing source: %+v", err)
	}

	err = source.Start(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	got := source.Teardown(ctx)
	if got.Error() != want.Error() {
		t.Fatalf("want: %+v, got: %+v", want, got)
	}

	close(stopRunCh)
	select {
	case <-closeCh:
	case <-time.After(time.Second):
		t.Fatal("should've received call to source.Run")
	}
}

// -----------------
// -- DESTINATION --
// -----------------

func testDestination_Configure_Success(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, mockDestination := tdf(t)

	cfg := map[string]string{
		"foo":   "bar",
		"empty": "",
	}
	want := cerrors.New("init error")
	mockDestination.EXPECT().
		Configure(gomock.Any(), cpluginv1.DestinationConfigureRequest{Config: cfg}).
		Return(cpluginv1.DestinationConfigureResponse{}, want)

	destination, err := dispenser.DispenseDestination()
	if err != nil {
		t.Fatalf("error dispensing destination: %+v", err)
	}

	got := destination.Configure(ctx, cfg)
	if got.Error() != want.Error() {
		t.Fatalf("want: %+v, got: %+v", want, got)
	}
}

func testDestination_Configure_Fail(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, mockDestination := tdf(t)

	mockDestination.EXPECT().
		Configure(gomock.Any(), cpluginv1.DestinationConfigureRequest{Config: nil}).
		Return(cpluginv1.DestinationConfigureResponse{}, nil)

	destination, err := dispenser.DispenseDestination()
	if err != nil {
		t.Fatalf("error dispensing destination: %+v", err)
	}

	err = destination.Configure(ctx, map[string]string{})
	if err != nil {
		t.Fatalf("want: nil, got: %+v", err)
	}
}

func testDestination_Start_Success(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, mockDestination := tdf(t)

	// Function Destination.Run is called in a goroutine, we have to wait for it to
	// run to prove this works.
	closeCh := make(chan struct{})
	mockDestination.EXPECT().
		Start(gomock.Any(), cpluginv1.DestinationStartRequest{}).
		Return(cpluginv1.DestinationStartResponse{}, nil)
	mockDestination.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, stream cpluginv1.DestinationRunStream) error {
			defer close(closeCh)
			return nil
		})

	destination, err := dispenser.DispenseDestination()
	if err != nil {
		t.Fatalf("error dispensing destination: %+v", err)
	}

	err = destination.Start(ctx)
	if err != nil {
		t.Fatalf("want: nil, got: %+v", err)
	}

	select {
	case <-closeCh:
	case <-time.After(time.Second):
		t.Fatal("should've received call to destination.Run")
	}
}

func testDestination_Start_Fail(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, mockDestination := tdf(t)

	want := cerrors.New("test error")

	mockDestination.EXPECT().
		Start(gomock.Any(), cpluginv1.DestinationStartRequest{}).
		Return(cpluginv1.DestinationStartResponse{}, want)

	destination, err := dispenser.DispenseDestination()
	if err != nil {
		t.Fatalf("error dispensing destination: %+v", err)
	}

	got := destination.Start(ctx)
	if got.Error() != want.Error() {
		t.Fatalf("want: %+v, got: %+v", want, got)
	}
}

func testDestination_Write_Success(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, mockDestination := tdf(t)

	want := cpluginv1.Record{
		Position:  []byte("test-position"),
		Metadata:  map[string]string{"foo": "bar"},
		CreatedAt: time.Now().UTC(),
		Key:       cpluginv1.RawData("raw-key"),
		Payload:   cpluginv1.StructuredData{"baz": "qux"},
	}

	// Function Destination.Run is called in a goroutine, we have to wait for it to
	// run to prove this works.
	closeCh := make(chan struct{})
	mockDestination.EXPECT().
		Start(gomock.Any(), cpluginv1.DestinationStartRequest{}).
		Return(cpluginv1.DestinationStartResponse{}, nil)
	mockDestination.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, stream cpluginv1.DestinationRunStream) error {
			defer close(closeCh)
			got, err := stream.Recv()
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if diff := cmp.Diff(got.Record, want); diff != "" {
				t.Errorf("expected ack: %s", diff)
			}
			return nil
		})

	destination, err := dispenser.DispenseDestination()
	if err != nil {
		t.Fatalf("error dispensing destination: %+v", err)
	}

	err = destination.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	err = destination.Write(ctx, record.Record{
		Position:  want.Position,
		Metadata:  want.Metadata,
		CreatedAt: want.CreatedAt,
		Key:       record.RawData{Raw: want.Key.(cpluginv1.RawData)},
		Payload:   record.StructuredData{"baz": "qux"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	select {
	case <-closeCh:
	case <-time.After(time.Second):
		t.Fatal("should've received call to destination.Write")
	}

	// wait for stream closing to propagate from plugin to Conduit
	time.Sleep(time.Millisecond * 50)

	err = destination.Write(ctx, record.Record{})
	if !cerrors.Is(err, ErrStreamNotOpen) {
		t.Fatalf("unexpected error: %+v", err)
	}
}

func testDestination_Write_WithoutStart(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, _ := tdf(t)

	destination, err := dispenser.DispenseDestination()
	if err != nil {
		t.Fatalf("error dispensing destination: %+v", err)
	}

	err = destination.Write(ctx, record.Record{})
	if !cerrors.Is(err, ErrStreamNotOpen) {
		t.Fatalf("unexpected error: %+v", err)
	}
}

func testDestination_Ack_Success(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, mockDestination := tdf(t)

	var want []record.Position
	for i := 0; i < 10; i++ {
		want = append(want, []byte(fmt.Sprintf("position-%d", i)))
	}

	mockDestination.EXPECT().
		Start(gomock.Any(), cpluginv1.DestinationStartRequest{}).
		Return(cpluginv1.DestinationStartResponse{}, nil)
	mockDestination.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, stream cpluginv1.DestinationRunStream) error {
			for _, p := range want {
				err := stream.Send(cpluginv1.DestinationRunResponse{
					AckPosition: p,
				})
				if err != nil {
					t.Fatalf("unexpected error: %+v", err)
				}
			}
			return nil
		})

	destination, err := dispenser.DispenseDestination()
	if err != nil {
		t.Fatalf("error dispensing destination: %+v", err)
	}

	err = destination.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	var got []record.Position
	for i := 0; i < len(want); i++ {
		pos, err := destination.Ack(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %+v", err)
		}
		got = append(got, pos)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("expected position: %s", diff)
	}
}

func testDestination_Ack_WithError(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, mockDestination := tdf(t)

	wantPos := record.Position("test-position")
	wantErr := cerrors.New("test error")

	mockDestination.EXPECT().
		Start(gomock.Any(), cpluginv1.DestinationStartRequest{}).
		Return(cpluginv1.DestinationStartResponse{}, nil)
	mockDestination.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, stream cpluginv1.DestinationRunStream) error {
			err := stream.Send(cpluginv1.DestinationRunResponse{
				AckPosition: wantPos,
				Error:       wantErr.Error(),
			})
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			return nil
		})

	destination, err := dispenser.DispenseDestination()
	if err != nil {
		t.Fatalf("error dispensing destination: %+v", err)
	}

	err = destination.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	gotPos, gotErr := destination.Ack(ctx)
	if diff := cmp.Diff(gotPos, wantPos); diff != "" {
		t.Errorf("expected position: %s", diff)
	}
	if gotErr.Error() != wantErr.Error() {
		t.Fatalf("want: %+v, got: %+v", wantErr, gotErr)
	}
}

func testDestination_Ack_WithoutStart(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, _ := tdf(t)

	destination, err := dispenser.DispenseDestination()
	if err != nil {
		t.Fatalf("error dispensing destination: %+v", err)
	}

	_, err = destination.Ack(ctx)
	if !cerrors.Is(err, ErrStreamNotOpen) {
		t.Fatalf("unexpected error: %+v", err)
	}
}

func testDestination_Run_Fail(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, mockDestination := tdf(t)

	want := cerrors.New("test-error")

	// Function Destination.Run is called in a goroutine, we have to wait for it to
	// run to prove this works.
	closeCh := make(chan struct{})
	mockDestination.EXPECT().
		Start(gomock.Any(), cpluginv1.DestinationStartRequest{}).
		Return(cpluginv1.DestinationStartResponse{}, nil)
	mockDestination.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, stream cpluginv1.DestinationRunStream) error {
			defer close(closeCh)
			_, _ = stream.Recv() // receive record and fail
			return want
		})

	destination, err := dispenser.DispenseDestination()
	if err != nil {
		t.Fatalf("error dispensing destination: %+v", err)
	}

	err = destination.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	err = destination.Write(ctx, record.Record{})
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	select {
	case <-closeCh:
	case <-time.After(time.Second):
		t.Fatal("should've received call to destination.Write")
	}

	// Error is returned through the Ack function, that's the incoming stream.
	_, err = destination.Ack(ctx)
	// Unwrap inner-most error
	var got error
	for unwrapped := err; unwrapped != nil; {
		got = unwrapped
		unwrapped = cerrors.Unwrap(unwrapped)
	}

	if got.Error() != want.Error() {
		t.Fatalf("want: %+v, got: %+v", want, got)
	}

	// Error is returned through the Write function, that's the outgoing stream.
	err = destination.Write(ctx, record.Record{})
	// Unwrap inner-most error
	got = nil
	for unwrapped := err; unwrapped != nil; {
		got = unwrapped
		unwrapped = cerrors.Unwrap(unwrapped)
	}

	if got.Error() != want.Error() {
		t.Fatalf("want: %+v, got: %+v", want, got)
	}
}

func testDestination_Teardown_Success(t *testing.T, tdf testDispenserFunc) {
	ctx := context.Background()
	dispenser, _, _, mockDestination := tdf(t)

	want := cerrors.New("init error")
	closeCh := make(chan struct{})
	stopRunCh := make(chan struct{})
	mockDestination.EXPECT().
		Start(gomock.Any(), cpluginv1.DestinationStartRequest{}).
		Return(cpluginv1.DestinationStartResponse{}, nil)
	mockDestination.EXPECT().
		Run(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, stream cpluginv1.DestinationRunStream) error {
			defer close(closeCh)
			<-stopRunCh
			return nil
		})
	mockDestination.EXPECT().
		Teardown(gomock.Any(), cpluginv1.DestinationTeardownRequest{}).
		Return(cpluginv1.DestinationTeardownResponse{}, want)

	destination, err := dispenser.DispenseDestination()
	if err != nil {
		t.Fatalf("error dispensing destination: %+v", err)
	}

	err = destination.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	got := destination.Teardown(ctx)
	if got.Error() != want.Error() {
		t.Fatalf("want: %+v, got: %+v", want, got)
	}

	close(stopRunCh)
	select {
	case <-closeCh:
	case <-time.After(time.Second):
		t.Fatal("should've received call to destination.Run")
	}
}
