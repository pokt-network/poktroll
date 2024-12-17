package pokterrors

import "errors"

// Split returns a slice of errors which were joined with errors.Join.
// The returned errs slice will also include elements for any nested wrapped
// errors contained by the joined errors.
func Split(joinedErrs error) (errs []error) {
	if joinedErrs != nil {
		for err := errors.Unwrap(joinedErrs); err != nil; err = errors.Unwrap(err) {
			errs = append(errs, err)
		}
	}
	return errs
}
