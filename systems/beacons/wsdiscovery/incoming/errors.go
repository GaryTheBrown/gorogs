package incoming

import (
	"fmt"
)

type ErrorFail struct{}

func (e ErrorFail) Error() string { return "Failure error" }

type ErrorDrop struct{}

func (e ErrorDrop) Error() string { return "Drop Error" }

type ErrVersionNotFound ErrorFail

func (e ErrVersionNotFound) Error() string { return "Version not found at this point" }
func (e ErrVersionNotFound) Unwrap() error { return ErrorFail(e) }

type ErrBadSchemaUnmarshalFailed struct{ ErrorDrop, ExternalError error }

func (e ErrBadSchemaUnmarshalFailed) Error() string {
	return fmt.Sprintf("Bad Schema Unmarshal failed: %s", e.ExternalError)
}
func (e ErrBadSchemaUnmarshalFailed) Unwrap() []error {
	return []error{e.ErrorDrop, e.ExternalError}
}

type ErrBadSchemaFailedHeaderRead struct{ ErrorDrop, ExternalError error }

func (e ErrBadSchemaFailedHeaderRead) Error() string {
	return fmt.Sprintf("Bad Schema Header Read failed: %s", e.ExternalError)
}
func (e ErrBadSchemaFailedHeaderRead) Unwrap() []error {
	return []error{e.ErrorDrop, e.ExternalError}
}

type ErrBadSchemaWrongVersion ErrorDrop

func (e ErrBadSchemaWrongVersion) Error() string { return "Bad Schema Wrong Version" }
func (e ErrBadSchemaWrongVersion) Unwrap() error { return ErrorDrop(e) }

type ErrDuplicateNamespacePrefix ErrorDrop

func (e ErrDuplicateNamespacePrefix) Error() string {
	return "Duplicate namespace prefix declaration detected in envelope attributes"
}
func (e ErrDuplicateNamespacePrefix) Unwrap() error { return ErrorDrop(e) }

type ErrBadSchemaTagNotApproved ErrorDrop

func (e ErrBadSchemaTagNotApproved) Error() string { return "Element name space was not approved" }
func (e ErrBadSchemaTagNotApproved) Unwrap() error { return ErrorDrop(e) }

type ErrBadSchemaTagNameBad ErrorDrop

func (e ErrBadSchemaTagNameBad) Error() string { return "Tag Name Bad" }
func (e ErrBadSchemaTagNameBad) Unwrap() error { return ErrorDrop(e) }

type ErrBadSchemaDecoderFailed struct{ ErrorDrop, ExternalError error }

func (e ErrBadSchemaDecoderFailed) Error() string {
	return fmt.Sprintf("Bad Schema Decoder failed: %s", e.ExternalError)
}
func (e ErrBadSchemaDecoderFailed) Unwrap() []error {
	return []error{e.ErrorDrop, e.ExternalError}
}

type ErrBadSchemaBodyUnmarshalFailed struct{ ErrorDrop, ExternalError error }

func (e ErrBadSchemaBodyUnmarshalFailed) Error() string {
	return fmt.Sprintf("Bad Schema Parse Body: %s", e.ExternalError)
}
func (e ErrBadSchemaBodyUnmarshalFailed) Unwrap() []error {
	return []error{e.ErrorDrop, e.ExternalError}
}

type ErrResolveNotForUs ErrorDrop

func (e ErrResolveNotForUs) Error() string { return "Resolve is not for us. Ignoring" }
func (e ErrResolveNotForUs) Unwrap() error { return ErrorDrop(e) }

type ErrTransferSchemaWrong ErrorDrop

func (e ErrTransferSchemaWrong) Error() string { return "Transfer Schema NOt right" }
func (e ErrTransferSchemaWrong) Unwrap() error { return ErrorDrop(e) }
