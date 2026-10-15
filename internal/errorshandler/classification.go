package errorshandler

type Classification int

const (
	// NotApplicable — классификатор не распознал ошибку.
	NotApplicable Classification = iota
	// NonRetriable — операцию не следует повторять.
	NonRetriable
	// Retriable — операцию можно повторить.
	Retriable
)

type Classifier interface {
	Classify(error) Classification
}

type BaseClassifier struct {
	classify func(error) Classification
}

func NewBaseClassifier(classify func(error) Classification) BaseClassifier {
	return BaseClassifier{classify: classify}
}

func (c BaseClassifier) Classify(err error) Classification {
	if c.classify == nil {
		return NotApplicable
	}
	return c.classify(err)
}
