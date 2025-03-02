package validate

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/jj-style/eventpix/internal/data/db"
)

type Validator interface {
	ValidateEvent(evt *db.Event) error
	ValidateSlug(slug string) error
}

type validator struct {
}

func NewValidator() Validator {
	return &validator{}
}

func (v *validator) ValidateEvent(evt *db.Event) error {
	if err := v.ValidateSlug(evt.Slug); err != nil {
		return err
	}
	return nil
}

func (v *validator) ValidateSlug(slug string) error {
	slugRe := regexp.MustCompile(`^[a-z][a-z-]*$`)
	slugErr := fmt.Errorf("slug must match %s", slugRe.String())
	slugRequiredErr := errors.New("slug is required")

	if slug == "" {
		return slugRequiredErr
	}

	if !slugRe.MatchString(slug) {
		return slugErr
	}

	return nil
}
