package manipulate

import (
	"context"
	"fmt"

	"go.aporeto.io/elemental"
)

const iterDefaultBlockSize = 10000

// IterFunc calls RetrieveMany on the given Manipulator, and will retrieve the data by block
// of the given blockSize.
//
// IterFunc will naturally ends and return when there is no more data to pull.
//
// For each retrieved block, the given func will be called with the
// current data block. If the function returns an error, the error is returned to the caller
// of IterFunc and the iteration stops.
//
// The given context will be used if the underlying manipulator honors it.
//
// The given manipulate.Context will be derived for each loop and will set the pagination
// information. If the given context already has page information, then it will be ignored.
//
// The identifiablesTemplate parameter is must be an empty elemental.Identifiables that will be used to
// hold the data block. It is reset at every iteration. Do not rely on it to be filled
// once IterFunc is complete.
//
// Finally, if the given blockSize is <= 0, then it will use the default that is 10000.
func IterFunc(
	ctx context.Context,
	manipulator Manipulator,
	identifiablesTemplate elemental.Identifiables,
	mctx Context,
	iteratorFunc func(block elemental.Identifiables) error,
	blockSize int,
) error {

	if manipulator == nil {
		panic("manipulator must not be nil")
	}

	if iteratorFunc == nil {
		panic("iteratorFunc must not be nil")
	}

	if identifiablesTemplate == nil {
		panic("identifiablesTemplate must not be nil")
	}

	if mctx == nil {
		mctx = NewContext(ctx)
	}

	if blockSize <= 0 {
		blockSize = iterDefaultBlockSize
	}

	var page int

	for {
		page++

		objects := identifiablesTemplate.Copy()

		if err := manipulator.RetrieveMany(mctx.Derive(ContextOptionPage(page, blockSize)), objects); err != nil {
			return fmt.Errorf("unable to retrieve objects for page %d: %s", page, err.Error())
		}

		if len(objects.List()) == 0 {
			return nil
		}

		if err := iteratorFunc(objects); err != nil {
			return fmt.Errorf("iter function returned an error on page %d: %s", page, err)
		}
	}
}

// Iter is a helper function for IterFunc.
//
// It will simply iterates on the object with identity of the given elemental.Identifiables.
// Not that this function cannot populate the data in the identifiable parameter. Instead
// It will return the destination.
//
// Always pass an empty elemental.Identifiables to this function
//
// For more information, please check IterFunc documentation.
//
// Example:
//     dest, err := Iter(context.Background(), m, mctx, model.ThingsList{}, 100)
//
func Iter(
	ctx context.Context,
	m Manipulator,
	mctx Context,
	identifiablesTemplate elemental.Identifiables,
	blockSize int,
) (elemental.Identifiables, error) {

	if err := IterFunc(
		ctx,
		m,
		identifiablesTemplate,
		mctx,
		func(block elemental.Identifiables) error {
			identifiablesTemplate = identifiablesTemplate.Append(block.List()...)
			return nil
		},
		blockSize,
	); err != nil {
		return nil, err
	}

	return identifiablesTemplate, nil
}
