package repository

type NotFoundError struct {
	Err error
}

func (nfe *NotFoundError) Error() string {
	return nfe.Err.Error()
}

func (nfe *NotFoundError) Unwrap() error {
	return nfe.Err
}

func newNotFoundError(err error) error {
	return &NotFoundError{
		Err: err,
	}
}
