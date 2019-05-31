// Copyright 2019 Aporeto Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manipmongo

import (
	"context"
	"io"
	"net"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"go.aporeto.io/elemental"
	"go.aporeto.io/manipulate"
	"go.aporeto.io/manipulate/internal/backoff"
)

// invertSortKey eventually inverts the given sorting key.
func invertSortKey(k string, revert bool) string {

	if !revert {
		return k
	}

	if strings.HasPrefix(k, "-") {
		return k[1:]
	}

	return "-" + k
}

func applyOrdering(order []string, inverted bool) []string {

	o := []string{} // nolint: prealloc

	for _, f := range order {

		if f == "" {
			continue
		}

		if f == "ID" || f == "id" {
			f = "_id"
		}

		o = append(o, strings.ToLower(invertSortKey(f, inverted)))
	}

	return o
}

func runQueryFunc(
	mctx manipulate.Context,
	operation elemental.Operation,
	identity elemental.Identity,
	operationFunc func() (interface{}, error),
	defaultRetryFunc manipulate.RetryFunc,
) (interface{}, error) {

	var try int

	info := RetryInfo{
		Operation: operation,
		Identity:  identity,
	}

	for {

		out, err := operationFunc()
		if err == nil {
			return out, nil
		}

		err = handleQueryError(err)
		if !manipulate.IsCannotCommunicateError(err) {
			return out, err
		}

		info.try = try
		info.err = err
		info.mctx = mctx

		if rf := mctx.RetryFunc(); rf != nil {
			if rerr := rf(info); rerr != nil {
				return nil, rerr
			}
		} else if defaultRetryFunc != nil {
			if rerr := defaultRetryFunc(info); rerr != nil {
				return nil, rerr
			}
		}

		deadline, ok := mctx.Context().Deadline()
		if ok && deadline.Before(time.Now()) {
			return nil, manipulate.NewErrCannotExecuteQuery(context.DeadlineExceeded.Error())
		}

		<-time.After(backoff.Next(try, deadline))
		try++
	}
}

func handleQueryError(err error) error {

	if _, ok := err.(net.Error); ok {
		return manipulate.NewErrCannotCommunicate(err.Error())
	}

	if err == mgo.ErrNotFound {
		return manipulate.NewErrObjectNotFound("cannot find the object for the given ID")
	}

	if mgo.IsDup(err) {
		return manipulate.NewErrConstraintViolation("duplicate key.")
	}

	if isConnectionError(err) {
		return manipulate.NewErrCannotCommunicate(err.Error())
	}

	// see https://github.com/mongodb/mongo/blob/master/src/mongo/base/error_codes.err
	switch getErrorCode(err) {
	case 6, 7, 71, 74, 91, 109, 189, 202, 216, 262, 10107, 13436, 13435, 11600, 11602:
		// HostUnreachable
		// HostNotFound,
		// ReplicaSetNotFound,
		// NodeNotFound,
		// ConfigurationInProgress,
		// ShutdownInProgress
		// PrimarySteppedDown,
		// NetworkInterfaceExceededTimeLimit
		// ElectionInProgress
		// ExceededTimeLimit
		// NotMaster
		// NotMasterOrSecondary
		// NotMasterNoSlaveOk
		// InterruptedAtShutdown
		// InterruptedDueToStepDown
		return manipulate.NewErrCannotCommunicate(err.Error())
	default:
		return manipulate.NewErrCannotExecuteQuery(err.Error())
	}
}

func getErrorCode(err error) int {

	switch e := err.(type) {

	case *mgo.QueryError:
		return e.Code

	case *mgo.LastError:
		return e.Code

	case *mgo.BulkError:
		// we just get the first
		for _, c := range e.Cases() {
			return getErrorCode(c.Err)
		}
	}

	return 0
}

// Stolen from mongodb code. this is ugly.
const (
	errLostConnection               = "lost connection to server"
	errNoReachableServers           = "no reachable servers"
	errReplTimeoutPrefix            = "waiting for replication timed out"
	errCouldNotContactPrimaryPrefix = "could not contact primary for replica set"
	errWriteResultsUnavailable      = "write results unavailable from"
	errCouldNotFindPrimaryPrefix    = `could not find host matching read preference { mode: "primary"`
	errUnableToTargetPrefix         = "unable to target"
	errNotMaster                    = "not master"
	errConnectionRefusedSuffix      = "connection refused"
)

func isConnectionError(err error) bool {

	if err == nil {
		return false
	}

	lowerCaseError := strings.ToLower(err.Error())
	if lowerCaseError == errNoReachableServers ||
		err == io.EOF ||
		strings.Contains(lowerCaseError, errLostConnection) ||
		strings.Contains(lowerCaseError, errReplTimeoutPrefix) ||
		strings.Contains(lowerCaseError, errCouldNotContactPrimaryPrefix) ||
		strings.Contains(lowerCaseError, errWriteResultsUnavailable) ||
		strings.Contains(lowerCaseError, errCouldNotFindPrimaryPrefix) ||
		strings.Contains(lowerCaseError, errUnableToTargetPrefix) ||
		lowerCaseError == errNotMaster ||
		strings.HasSuffix(lowerCaseError, errConnectionRefusedSuffix) {
		return true
	}
	return false
}

func makeFieldsSelector(fields []string) bson.M {

	if len(fields) == 0 {
		return nil
	}

	sels := bson.M{}
	for _, f := range fields {
		if f == "" {
			continue
		}
		if f == "ID" || f == "id" {
			f = "_id"
		}
		sels[strings.ToLower(f)] = 1
	}

	if len(sels) == 0 {
		return nil
	}

	return sels
}

func convertReadConsistency(c manipulate.ReadConsistency) mgo.Mode {
	switch c {
	case manipulate.ReadConsistencyEventual:
		return mgo.Eventual
	case manipulate.ReadConsistencyMonotonic:
		return mgo.Monotonic
	case manipulate.ReadConsistencyNearest:
		return mgo.Nearest
	case manipulate.ReadConsistencyStrong:
		return mgo.Strong
	default:
		return -1
	}
}

func convertWriteConsistency(c manipulate.WriteConsistency) *mgo.Safe {
	switch c {
	case manipulate.WriteConsistencyNone:
		return nil
	case manipulate.WriteConsistencyStrong:
		return &mgo.Safe{WMode: "majority"}
	case manipulate.WriteConsistencyStrongest:
		return &mgo.Safe{WMode: "majority", J: true}
	default:
		return &mgo.Safe{}
	}
}
